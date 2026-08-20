// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package relay is the unified entry point for calling AI/LLM providers.
// It wraps the relay adaptor system with a clean, framework-agnostic API
// and integrates usage metering.
//
// Quick start:
//
//	import (
//		"github.com/LingByte/ling-base/relay"
//		"github.com/LingByte/ling-base/relay/meter"
//		"github.com/LingByte/ling-base/relay/channel/openai"
//	)
//
//	client := relay.New(
//		relay.WithProvider(openai.New("sk-xxx")),
//		relay.WithMeter(meter.NewMemoryMeter()),
//	)
//	resp, err := client.Chat(ctx, &relay.ChatRequest{
//		Model:    "gpt-4o",
//		Messages: []relay.Message{{Role: "user", Content: json.RawMessage(`"Hello"`)}},
//	})
package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/ling-base/relay/constant"
	"github.com/LingByte/ling-base/relay/meter"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/types"
)

// ─── Message types (wrappers around relaykit DTOs) ──────────────

// Message is a chat message. Content is any to support both
// plain string content and multimodal content arrays.
type Message = dto.Message

// ToolCall represents a tool/function call request.
type ToolCall = dto.ToolCallRequest

// Tool represents a tool the model may call (alias to ToolCallRequest).
type Tool = dto.ToolCallRequest

// ─── Request types ───────────────────────────────────────────────

// ChatRequest is the unified chat completion request.
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
	Stop        json.RawMessage `json:"stop,omitempty"`
	N           *int            `json:"n,omitempty"`
	User        string          `json:"user,omitempty"`
}

// EmbedRequest is the unified embedding request.
type EmbedRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
	User  string          `json:"user,omitempty"`
}

// ImageRequest is the unified image generation request.
type ImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              *int   `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Style          string `json:"style,omitempty"`
	User           string `json:"user,omitempty"`
}

// ─── Response types ──────────────────────────────────────────────

// ChatResponse is the unified chat completion response.
type ChatResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []ChatChoice   `json:"choices"`
	Usage   meter.Usage    `json:"usage"`
	Provider string       `json:"provider"`
}

// ChatChoice is one choice in a chat completion response.
type ChatChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// EmbedResponse is the unified embedding response.
type EmbedResponse struct {
	Model    string      `json:"model"`
	Data     []EmbedData `json:"data"`
	Usage    meter.Usage `json:"usage"`
	Provider string      `json:"provider"`
}

// EmbedData is one embedding vector.
type EmbedData struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// ImageResponse is the unified image generation response.
type ImageResponse struct {
	Created  int64       `json:"created"`
	Data     []ImageData `json:"data"`
	Usage    meter.Usage `json:"usage"`
	Provider string      `json:"provider"`
}

// ImageData is one generated image.
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ─── Provider interface ──────────────────────────────────────────

// Provider is the high-level interface for AI API providers.
// It wraps the low-level common.Adaptor with a simpler, type-safe API.
type Provider interface {
	Name() string
	ApiType() int
	Adaptor() common.Adaptor
}

// ConfiguredProvider is an optional interface implemented by providers
// that expose their API endpoint and key. The Client uses this to
// populate RelayInfo.ChannelBaseUrl and RelayInfo.ApiKey, which most
// adaptors rely on for URL construction and authentication.
type ConfiguredProvider interface {
	BaseURL() string
	APIKey() string
}

// GenericProvider wraps any common.Adaptor with a name, API type,
// base URL, and API key. It is the standard way to use non-OpenAI/Claude/
// Gemini providers with the Client.
type GenericProvider struct {
	name    string
	apiType int
	adaptor common.Adaptor
	baseURL string
	apiKey  string
}

// NewProvider creates a GenericProvider from the given adaptor and config.
// This is the primary constructor for providers that don't have their own
// Provider type (i.e. all providers except openai, claude, and gemini).
func NewProvider(name string, apiType int, adaptor common.Adaptor, baseURL string, apiKey string) *GenericProvider {
	return &GenericProvider{
		name:    name,
		apiType: apiType,
		adaptor: adaptor,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (p *GenericProvider) Name() string             { return p.name }
func (p *GenericProvider) ApiType() int             { return p.apiType }
func (p *GenericProvider) Adaptor() common.Adaptor  { return p.adaptor }
func (p *GenericProvider) BaseURL() string          { return p.baseURL }
func (p *GenericProvider) APIKey() string           { return p.apiKey }

// ─── Client ──────────────────────────────────────────────────────

// Client is the unified entry point. It routes requests to the configured
// provider, records usage via the Meter, and returns results.
type Client struct {
	provider Provider
	meter    meter.Meter
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithProvider sets the provider.
func WithProvider(p Provider) Option {
	return func(c *Client) { c.provider = p }
}

// WithMeter sets the usage meter.
func WithMeter(m meter.Meter) Option {
	return func(c *Client) { c.meter = m }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new Client.
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetProvider replaces the provider at runtime.
func (c *Client) SetProvider(p Provider) { c.provider = p }

// SetMeter replaces the meter at runtime.
func (c *Client) SetMeter(m meter.Meter) { c.meter = m }

// Provider returns the current provider.
func (c *Client) Provider() Provider { return c.provider }

// Meter returns the current meter.
func (c *Client) Meter() meter.Meter { return c.meter }

func (c *Client) ensureProvider() error {
	if c.provider == nil {
		return fmt.Errorf("relay: no provider configured")
	}
	return nil
}

// buildRelayInfo creates a RelayInfo for the given request.
func (c *Client) buildRelayInfo(mode int, model string, stream bool) *common.RelayInfo {
	info := common.NewRelayInfo()
	info.RelayMode = mode
	info.OriginModelName = model
	info.UpstreamModelName = model
	info.IsStream = stream
	info.ApiType = c.provider.ApiType()

	// Populate channel config from the provider if it implements ConfiguredProvider.
	var baseURL, apiKey string
	if cp, ok := c.provider.(ConfiguredProvider); ok {
		baseURL = cp.BaseURL()
		apiKey = cp.APIKey()
	}

	info.ChannelBaseUrl = baseURL
	info.ApiKey = apiKey
	info.ChannelMeta = &common.ChannelMeta{
		ApiType:           c.provider.ApiType(),
		UpstreamModelName: model,
		ApiKey:            apiKey,
		ChannelBaseUrl:    baseURL,
	}
	return info
}

func (c *Client) record(ctx context.Context, model string, usage meter.Usage) {
	if c.meter == nil {
		return
	}
	_ = c.meter.Record(ctx, &meter.UsageRecord{
		Provider: c.provider.Name(),
		Model:    model,
		Usage:    usage,
	})
}

// ─── Chat ────────────────────────────────────────────────────────

// Chat sends a chat completion request.
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeChatCompletions, req.Model, req.Stream)
	adaptor.Init(info)

	// Convert to relaykit DTO.
	openaiReq := c.toOpenAIRequest(req)

	// Convert request to provider's native format.
	converted, err := adaptor.ConvertOpenAIRequest(ctx, info, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert request: %w", err)
	}

	// Marshal request body.
	var body io.Reader
	switch v := converted.(type) {
	case io.Reader:
		body = v
	default:
		jsonData, err := json.Marshal(converted)
		if err != nil {
			return nil, fmt.Errorf("relay: marshal request: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	// Build URL and send request.
	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, fmt.Errorf("relay: build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("relay: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := adaptor.SetupRequestHeader(ctx, &httpReq.Header, info); err != nil {
		return nil, fmt.Errorf("relay: setup headers: %w", err)
	}

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	// Parse response using a dummy ResponseWriter (we don't stream).
	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	// Parse the response body that the adaptor wrote to dummyW.
	chatResp, err := parseChatResponse(dummyW.Bytes(), req.Model, c.provider.Name(), usage)
	if err != nil {
		return nil, fmt.Errorf("relay: parse chat response: %w", err)
	}
	return chatResp, nil
}

// ─── Chat Stream ────────────────────────────────────────────────

// ChatStreamChunk is one chunk in a streaming chat response.
type ChatStreamChunk struct {
	Delta    string // content delta
	Err      error
	Done     bool
	Usage    *meter.Usage // present on final chunk if provider reports it
}

// ChatStreamResult holds the stream channel and final usage.
type ChatStreamResult struct {
	Ch    chan ChatStreamChunk
	Usage meter.Usage // filled after channel closes
}

// ChatStream sends a streaming chat completion request.
// It returns a channel of chunks. The channel closes when the stream ends.
// After the channel closes, ChatStreamResult.Usage contains the final usage.
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (*ChatStreamResult, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	req.Stream = true
	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeChatCompletions, req.Model, true)
	adaptor.Init(info)

	openaiReq := c.toOpenAIRequest(req)

	converted, err := adaptor.ConvertOpenAIRequest(ctx, info, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, fmt.Errorf("relay: build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if err := adaptor.SetupRequestHeader(ctx, &httpReq.Header, info); err != nil {
		return nil, fmt.Errorf("relay: setup headers: %w", err)
	}

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: request failed: %w", err)
	}

	result := &ChatStreamResult{
		Ch: make(chan ChatStreamChunk, 100),
	}

	go func() {
		defer resp.Body.Close()
		defer close(result.Ch)

		// Use a pipe-based ResponseWriter to capture SSE chunks.
		pw := &streamResponseWriter{ch: result.Ch}
		usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, pw)
		if apiErr != nil {
			result.Ch <- ChatStreamChunk{Err: apiErr}
			return
		}

		usage := extractUsage(usageRaw)
		result.Usage = usage
		c.record(ctx, req.Model, usage)

		// Send final chunk with usage.
		result.Ch <- ChatStreamChunk{Done: true, Usage: &usage}
	}()

	return result, nil
}

// streamResponseWriter implements http.ResponseWriter by forwarding
// SSE data chunks to a channel.
type streamResponseWriter struct {
	ch       chan ChatStreamChunk
	header   http.Header
}

func (w *streamResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *streamResponseWriter) Write(data []byte) (int, error) {
	// Parse SSE data lines and forward as chunks.
	text := string(data)
	for _, line := range splitSSELines(text) {
		if line == "[DONE]" {
			continue
		}
		// Try to extract content delta from the SSE JSON.
		if delta := extractSSEDelta(line); delta != "" {
			w.ch <- ChatStreamChunk{Delta: delta}
		}
	}
	return len(data), nil
}

func (w *streamResponseWriter) WriteHeader(statusCode int) {}

func splitSSELines(text string) []string {
	var lines []string
	for _, line := range splitLines(text) {
		line = trimSpace(line)
		if hasPrefix(line, "data: ") {
			lines = append(lines, trimPrefix(line, "data: "))
		}
	}
	return lines
}

func splitLines(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func extractSSEDelta(data string) string {
	// Try to parse as OpenAI streaming chunk and extract delta content.
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ""
	}
	if len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}

// ─── Embed ───────────────────────────────────────────────────────

// Embed sends an embedding request.
func (c *Client) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeEmbeddings, req.Model, false)
	adaptor.Init(info)

	embedReq := dto.EmbeddingRequest{
		Model: req.Model,
		Input: req.Input,
		User:  req.User,
	}

	converted, err := adaptor.ConvertEmbeddingRequest(ctx, info, embedReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert embed request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal embed request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, fmt.Errorf("relay: build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	_ = adaptor.SetupRequestHeader(ctx, &httpReq.Header, info)

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse embed response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	embedResp, err := parseEmbedResponse(dummyW.Bytes(), req.Model, c.provider.Name(), usage)
	if err != nil {
		return nil, err
	}
	return embedResp, nil
}

// ─── Image ───────────────────────────────────────────────────────

// Image sends an image generation request.
func (c *Client) Image(ctx context.Context, req *ImageRequest) (*ImageResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeImagesGenerations, req.Model, false)
	adaptor.Init(info)

	imageReq := dto.ImageRequest{
		Model:          req.Model,
		Prompt:         req.Prompt,
		Size:           req.Size,
		Quality:        req.Quality,
		ResponseFormat: req.ResponseFormat,
		Style:          json.RawMessage(marshalString(req.Style)),
		User:           json.RawMessage(marshalString(req.User)),
	}
	if req.N != nil {
		imageReq.N = ptrUint(uint(*req.N))
	}

	converted, err := adaptor.ConvertImageRequest(ctx, info, imageReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert image request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal image request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	_ = adaptor.SetupRequestHeader(ctx, &httpReq.Header, info)

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse image response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	if usage.ImageCount == 0 && req.N != nil {
		usage.ImageCount = *req.N
	} else if usage.ImageCount == 0 {
		usage.ImageCount = 1
	}
	c.record(ctx, req.Model, usage)

	imageResp, err := parseImageResponse(dummyW.Bytes(), req.Model, c.provider.Name(), usage)
	if err != nil {
		return nil, err
	}
	return imageResp, nil
}

// ─── Audio (TTS / ASR) ──────────────────────────────────────────

// AudioRequest is the unified audio request for TTS and ASR.
type AudioRequest struct {
	Model          string
	Input          json.RawMessage // text for TTS, file data for ASR
	Voice          string
	ResponseFormat string
	Speed          *float64
	Language       string
}

// Audio sends an audio request (TTS or ASR) to the provider.
// The response body is returned as raw bytes (audio data for TTS, transcript JSON for ASR).
func (c *Client) Audio(ctx context.Context, req *AudioRequest, isTranscription bool) (*AudioResponse, error) {
	return c.audioWithMode(ctx, req, isTranscription, false)
}

// AudioTranslation sends an audio translation request (ASR translation) to the provider.
// Similar to transcription but translates the audio to English.
func (c *Client) AudioTranslation(ctx context.Context, req *AudioRequest) (*AudioResponse, error) {
	return c.audioWithMode(ctx, req, false, true)
}

func (c *Client) audioWithMode(ctx context.Context, req *AudioRequest, isTranscription, isTranslation bool) (*AudioResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	mode := relaymode.RelayModeAudioSpeech
	switch {
	case isTranslation:
		mode = relaymode.RelayModeAudioTranslation
	case isTranscription:
		mode = relaymode.RelayModeAudioTranscription
	}
	info := c.buildRelayInfo(mode, req.Model, false)
	adaptor.Init(info)

	audioReq := dto.AudioRequest{
		Model:          req.Model,
		Input:          string(req.Input),
		Voice:          req.Voice,
		ResponseFormat: req.ResponseFormat,
		Language:       json.RawMessage(marshalString(req.Language)),
	}
	if req.Speed != nil {
		audioReq.Speed = req.Speed
	}

	body, err := adaptor.ConvertAudioRequest(ctx, info, audioReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert audio request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, err
	}

	var httpReq *http.Request
	if body != nil {
		httpReq, err = http.NewRequestWithContext(ctx, "POST", url, body)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, "POST", url, nil)
	}
	if err != nil {
		return nil, err
	}
	_ = adaptor.SetupRequestHeader(ctx, &httpReq.Header, info)

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse audio response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	return &AudioResponse{
		Data:    dummyW.Bytes(),
		Usage:   usage,
		Headers: resp.Header,
	}, nil
}

// AudioResponse holds the audio response data.
type AudioResponse struct {
	Data    []byte
	Usage   meter.Usage
	Headers http.Header
}

// ─── Rerank ──────────────────────────────────────────────────────

// RerankRequest is the unified rerank request.
type RerankRequest struct {
	Model     string
	Query     string
	Documents []string
	TopN      *int
}

// Rerank sends a rerank request to the provider.
func (c *Client) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeRerank, req.Model, false)
	adaptor.Init(info)

	docs := make([]any, len(req.Documents))
	for i, d := range req.Documents {
		docs[i] = d
	}
	rerankReq := dto.RerankRequest{
		Model:     req.Model,
		Query:     req.Query,
		Documents: docs,
	}
	if req.TopN != nil {
		rerankReq.TopN = req.TopN
	}

	converted, err := adaptor.ConvertRerankRequest(ctx, relaymode.RelayModeRerank, rerankReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert rerank request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal rerank request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	_ = adaptor.SetupRequestHeader(ctx, &httpReq.Header, info)

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse rerank response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	var rerankResp dto.RerankResponse
	if err := json.Unmarshal(dummyW.Bytes(), &rerankResp); err != nil {
		return nil, fmt.Errorf("relay: unmarshal rerank response: %w", err)
	}

	return &RerankResponse{
		Results: rerankResp.Results,
		Usage:   usage,
	}, nil
}

// RerankResponse holds the rerank response.
type RerankResponse struct {
	Results []dto.RerankResponseResult
	Usage   meter.Usage
}

// ─── Responses (OpenAI Responses API) ───────────────────────────

// ResponsesRequest is the unified OpenAI Responses API request.
type ResponsesRequest struct {
	Model   string
	Input   json.RawMessage
	Stream  bool
	Tools   json.RawMessage
	Options json.RawMessage
}

// Responses sends an OpenAI Responses API request.
func (c *Client) Responses(ctx context.Context, req *ResponsesRequest) (*ResponsesResponse, error) {
	if err := c.ensureProvider(); err != nil {
		return nil, err
	}

	adaptor := c.provider.Adaptor()
	info := c.buildRelayInfo(relaymode.RelayModeResponses, req.Model, req.Stream)
	adaptor.Init(info)

	stream := req.Stream
	responsesReq := dto.OpenAIResponsesRequest{
		Model:  req.Model,
		Input:  req.Input,
		Stream: &stream,
		Tools:  req.Tools,
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(ctx, info, responsesReq)
	if err != nil {
		return nil, fmt.Errorf("relay: convert responses request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal responses request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	_ = adaptor.SetupRequestHeader(ctx, &httpReq.Header, info)

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("relay: parse responses response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	return &ResponsesResponse{
		Data:  dummyW.Bytes(),
		Usage: usage,
	}, nil
}

// ResponsesResponse holds the Responses API response.
type ResponsesResponse struct {
	Data  []byte
	Usage meter.Usage
}

// ─── Task (async video/music) ────────────────────────────────────

// TaskProvider is the interface for async task providers.
type TaskProvider interface {
	Name() string
	ApiType() int
	TaskAdaptor() common.TaskAdaptor
}

// SubmitTask submits an async task (video/music generation).
func (c *Client) SubmitTask(ctx context.Context, tp TaskProvider, body io.Reader) (*TaskSubmitResult, error) {
	adaptor := tp.TaskAdaptor()
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeVideoSubmit
	info.ChannelType = tp.ApiType()
	if cp, ok := c.provider.(ConfiguredProvider); ok {
		info.ApiKey = cp.APIKey()
		info.ChannelBaseUrl = cp.BaseURL()
	}
	adaptor.Init(info)

	if taskErr := adaptor.ValidateRequestAndSetAction(ctx, info); taskErr != nil {
		return nil, fmt.Errorf("relay: validate task: %w", taskErr)
	}

	url, err := adaptor.BuildRequestURL(info)
	if err != nil {
		return nil, err
	}

	if body == nil {
		body, err = adaptor.BuildRequestBody(ctx, info)
		if err != nil {
			return nil, fmt.Errorf("relay: build task body: %w", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	_ = adaptor.BuildRequestHeader(ctx, httpReq, info)

	resp, err := c.doTaskRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	taskID, taskData, taskErr := adaptor.DoResponse(ctx, resp, info)
	if taskErr != nil {
		return nil, fmt.Errorf("relay: task submit: %w", taskErr)
	}

	// Record usage for task submission (request count).
	c.record(ctx, info.UpstreamModelName, meter.Usage{
		RequestCount: 1,
	})

	return &TaskSubmitResult{
		TaskID:   taskID,
		TaskData: taskData,
	}, nil
}

// FetchTask polls the status of an async task.
func (c *Client) FetchTask(ctx context.Context, tp TaskProvider, baseURL, key string, body map[string]any, proxy string) (*common.TaskInfo, error) {
	adaptor := tp.TaskAdaptor()
	resp, err := adaptor.FetchTask(baseURL, key, body, proxy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	taskInfo, err := adaptor.ParseTaskResult(respBody)
	if err != nil {
		return nil, err
	}

	// Record usage when task completes (video/audio seconds if available).
	if taskInfo != nil && taskInfo.Status == "SUCCESS" {
		usage := meter.Usage{
			RequestCount: 1,
		}
		c.record(ctx, "", usage)
	}

	return taskInfo, nil
}

// TaskSubmitResult holds the result of submitting an async task.
type TaskSubmitResult struct {
	TaskID   string
	TaskData []byte
}

func (c *Client) doTaskRequest(ctx context.Context, adaptor common.TaskAdaptor, info *common.RelayInfo, httpReq *http.Request) (*http.Response, error) {
	if c.httpClient != nil {
		return c.httpClient.Do(httpReq)
	}
	return http.DefaultClient.Do(httpReq)
}

// ─── Midjourney ──────────────────────────────────────────────────

// MidjourneyRequest represents a Midjourney task submission request.
type MidjourneyRequest struct {
	Prompt      string   `json:"prompt,omitempty"`
	CustomId    string   `json:"customId,omitempty"`
	BotType     string   `json:"bot_type,omitempty"`
	NotifyHook  string   `json:"notifyHook,omitempty"`
	Action      string   `json:"action,omitempty"`
	Index       int      `json:"index,omitempty"`
	State       string   `json:"state,omitempty"`
	TaskId      string   `json:"taskId,omitempty"`
	Base64Array []string `json:"base64Array,omitempty"`
	Content     string   `json:"content,omitempty"`
	MaskBase64  string   `json:"maskBase64,omitempty"`
}

// MidjourneyResponse represents the Midjourney API response.
type MidjourneyResponse struct {
	Code        int            `json:"code"`
	Description string         `json:"description,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	Result      string         `json:"result,omitempty"`
}

// MidjourneyTask represents a fetched Midjourney task state.
type MidjourneyTask struct {
	ID          string         `json:"id"`
	Action      string         `json:"action,omitempty"`
	Status      string         `json:"status,omitempty"`
	Progress    string         `json:"progress,omitempty"`
	Prompt      string         `json:"prompt,omitempty"`
	PromptEn    string         `json:"promptEn,omitempty"`
	ImageUrl    string         `json:"imageUrl,omitempty"`
	VideoUrl    string         `json:"videoUrl,omitempty"`
	FailReason  string         `json:"failReason,omitempty"`
	Buttons     []map[string]any `json:"buttons,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	SubmitTime  int64          `json:"submitTime,omitempty"`
	StartTime   int64          `json:"startTime,omitempty"`
	FinishTime  int64          `json:"finishTime,omitempty"`
}

// MidjourneySubmit submits a Midjourney task.
// The mode should be one of the relaymode.RelayModeMidjourney* constants.
func (c *Client) MidjourneySubmit(ctx context.Context, baseURL, apiKey string, mode int, req *MidjourneyRequest) (*MidjourneyResponse, error) {
	var path string
	switch mode {
	case relaymode.RelayModeMidjourneyImagine:
		path = "/mj/submit/imagine"
	case relaymode.RelayModeMidjourneyDescribe:
		path = "/mj/submit/describe"
	case relaymode.RelayModeMidjourneyBlend:
		path = "/mj/submit/blend"
	case relaymode.RelayModeMidjourneyChange:
		path = "/mj/submit/change"
	case relaymode.RelayModeMidjourneySimpleChange:
		path = "/mj/submit/simple-change"
	case relaymode.RelayModeMidjourneyAction:
		path = "/mj/submit/action"
	case relaymode.RelayModeMidjourneyShorten:
		path = "/mj/submit/shorten"
	case relaymode.RelayModeMidjourneyModal:
		path = "/mj/submit/modal"
	case relaymode.RelayModeMidjourneyVideo:
		path = "/mj/submit/video"
	case relaymode.RelayModeMidjourneyEdits:
		path = "/mj/submit/edits"
	case relaymode.RelayModeMidjourneyUpload:
		path = "/mj/submit/upload-discord-images"
	case relaymode.RelayModeSwapFace:
		path = "/mj/insight-face/swap"
	default:
		return nil, fmt.Errorf("relay: unsupported midjourney mode: %d", mode)
	}

	url := strings.TrimSuffix(baseURL, "/") + path

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal midjourney request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("mj-api-secret", apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: midjourney request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mjResp MidjourneyResponse
	if err := json.Unmarshal(body, &mjResp); err != nil {
		return nil, fmt.Errorf("relay: parse midjourney response: %w", err)
	}

	c.record(ctx, "midjourney", meter.Usage{RequestCount: 1})

	return &mjResp, nil
}

// MidjourneyFetch fetches a Midjourney task by ID.
func (c *Client) MidjourneyFetch(ctx context.Context, baseURL, apiKey, taskID string) (*MidjourneyTask, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/mj/task/" + taskID + "/fetch"

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("mj-api-secret", apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: midjourney fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var task MidjourneyTask
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("relay: parse midjourney task: %w", err)
	}

	return &task, nil
}

// MidjourneyFetchByCondition fetches Midjourney tasks by condition (e.g. user_id, status).
func (c *Client) MidjourneyFetchByCondition(ctx context.Context, baseURL, apiKey string, condition map[string]any) ([]MidjourneyTask, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/mj/task/list-by-condition"

	jsonData, err := json.Marshal(condition)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal condition: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("mj-api-secret", apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: midjourney fetch-by-condition failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tasks []MidjourneyTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("relay: parse midjourney tasks: %w", err)
	}

	return tasks, nil
}

// ─── Suno ────────────────────────────────────────────────────────

// SunoSubmitRequest is the request body for Suno music/lyrics generation.
type SunoSubmitRequest struct {
	GptDescriptionPrompt string  `json:"gpt_description_prompt,omitempty"`
	Prompt               string  `json:"prompt,omitempty"`
	Mv                   string  `json:"mv,omitempty"`
	Title                string  `json:"title,omitempty"`
	Tags                 string  `json:"tags,omitempty"`
	ContinueAt           float64 `json:"continue_at,omitempty"`
	TaskID               string  `json:"task_id,omitempty"`
	ContinueClipId       string  `json:"continue_clip_id,omitempty"`
	MakeInstrumental     bool    `json:"make_instrumental"`
}

// SunoSubmitResponse is the response from a Suno submit request.
type SunoSubmitResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

// SunoTaskData represents one Suno task entry returned by the fetch endpoint.
type SunoTaskData struct {
	TaskID     string          `json:"task_id,omitempty"`
	Action     string          `json:"action,omitempty"`
	Status     string          `json:"status,omitempty"`
	FailReason string          `json:"fail_reason,omitempty"`
	SubmitTime int64           `json:"submit_time,omitempty"`
	StartTime  int64           `json:"start_time,omitempty"`
	FinishTime int64           `json:"finish_time,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// SunoActionMusic and SunoActionLyrics are the supported Suno actions.
const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"
)

// SubmitSunoTask submits a Suno music or lyrics generation task.
// action must be SunoActionMusic or SunoActionLyrics.
func (c *Client) SubmitSunoTask(ctx context.Context, baseURL, apiKey, action string, req *SunoSubmitRequest) (*SunoSubmitResponse, error) {
	switch action {
	case SunoActionMusic, SunoActionLyrics:
	default:
		return nil, fmt.Errorf("relay: invalid suno action: %s", action)
	}

	if action == SunoActionMusic && req.Mv == "" {
		req.Mv = "chirp-v3-0"
	}
	if action == SunoActionLyrics && req.Prompt == "" {
		return nil, fmt.Errorf("relay: suno lyrics requires prompt")
	}

	url := strings.TrimSuffix(baseURL, "/") + "/suno/submit/" + strings.ToLower(action)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal suno request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: suno submit failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sunoResp SunoSubmitResponse
	if err := json.Unmarshal(body, &sunoResp); err != nil {
		return nil, fmt.Errorf("relay: parse suno response: %w", err)
	}

	c.record(ctx, "suno", meter.Usage{RequestCount: 1})

	return &sunoResp, nil
}

// FetchSunoTask fetches Suno task status by task IDs (batch supported).
func (c *Client) FetchSunoTask(ctx context.Context, baseURL, apiKey string, taskIDs []string) ([]SunoTaskData, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/suno/fetch"

	payload := map[string]any{"ids": taskIDs}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("relay: marshal suno fetch payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("relay: suno fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tasks []SunoTaskData
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("relay: parse suno tasks: %w", err)
	}

	return tasks, nil
}

// ─── Helpers ─────────────────────────────────────────────────────

func (c *Client) doRequest(ctx context.Context, adaptor common.Adaptor, info *common.RelayInfo, httpReq *http.Request) (*http.Response, error) {
	// Use the adaptor's DoRequest if it handles the full request;
	// otherwise fall back to our HTTP client.
	resp, err := adaptor.DoRequest(ctx, info, httpReq.Body)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return resp, nil
	}
	// Fallback: send the request ourselves.
	return c.httpClient.Do(httpReq)
}

func (c *Client) toOpenAIRequest(req *ChatRequest) *dto.GeneralOpenAIRequest {
	r := &dto.GeneralOpenAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      &req.Stream,
		Tools:       req.Tools,
		Stop:        req.Stop,
		N:           req.N,
		User:        json.RawMessage(marshalString(req.User)),
	}
	if req.MaxTokens != nil {
		r.MaxTokens = ptrUint(uint(*req.MaxTokens))
	}
	if len(req.ToolChoice) > 0 {
		r.ToolChoice = json.RawMessage(req.ToolChoice)
	}
	// For streaming requests, request usage in the final chunk.
	if req.Stream {
		r.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}
	return r
}

func ptrUint(v uint) *uint { return &v }

func extractUsage(usageRaw any) meter.Usage {
	if usageRaw == nil {
		return meter.Usage{}
	}
	// Try different usage types.
	switch u := usageRaw.(type) {
	case meter.Usage:
		return u
	case *meter.Usage:
		return *u
	case dto.Usage:
		return meter.Usage{
			InputTokens:  u.PromptTokens,
			OutputTokens: u.CompletionTokens,
			TotalTokens:  u.TotalTokens,
			CachedTokens: u.PromptCacheHitTokens,
		}
	case *dto.Usage:
		if u == nil {
			return meter.Usage{}
		}
		return meter.Usage{
			InputTokens:  u.PromptTokens,
			OutputTokens: u.CompletionTokens,
			TotalTokens:  u.TotalTokens,
			CachedTokens: u.PromptCacheHitTokens,
		}
	case dto.ClaudeUsage:
		return meter.Usage{
			InputTokens:  u.InputTokens,
			OutputTokens: u.OutputTokens,
			TotalTokens:  u.InputTokens + u.OutputTokens,
			CachedTokens: u.CacheReadInputTokens,
		}
	case *dto.ClaudeUsage:
		if u == nil {
			return meter.Usage{}
		}
		return meter.Usage{
			InputTokens:  u.InputTokens,
			OutputTokens: u.OutputTokens,
			TotalTokens:  u.InputTokens + u.OutputTokens,
			CachedTokens: u.CacheReadInputTokens,
		}
	case dto.GeminiUsageMetadata:
		return meter.Usage{
			InputTokens:  u.PromptTokenCount,
			OutputTokens: u.CandidatesTokenCount,
			TotalTokens:  u.TotalTokenCount,
			CachedTokens: u.CachedContentTokenCount,
		}
	case *dto.GeminiUsageMetadata:
		if u == nil {
			return meter.Usage{}
		}
		return meter.Usage{
			InputTokens:  u.PromptTokenCount,
			OutputTokens: u.CandidatesTokenCount,
			TotalTokens:  u.TotalTokenCount,
			CachedTokens: u.CachedContentTokenCount,
		}
	}
	return meter.Usage{}
}

// dummyResponseWriter captures response body written by adaptor.DoResponse.
type dummyResponseWriter struct {
	buf []byte
	hdr http.Header
}

func (w *dummyResponseWriter) Header() http.Header {
	if w.hdr == nil {
		w.hdr = make(http.Header)
	}
	return w.hdr
}

func (w *dummyResponseWriter) Write(data []byte) (int, error) {
	w.buf = append(w.buf, data...)
	return len(data), nil
}

func (w *dummyResponseWriter) WriteHeader(statusCode int) {}

func (w *dummyResponseWriter) Bytes() []byte { return w.buf }

// parseChatResponse parses the JSON response body into a ChatResponse.
func parseChatResponse(body []byte, model, provider string, usage meter.Usage) (*ChatResponse, error) {
	var raw struct {
		ID      string         `json:"id"`
		Model   string         `json:"model"`
		Choices []ChatChoice   `json:"choices"`
		Usage   *dto.Usage     `json:"usage"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse chat response: %w", err)
	}
	if raw.Model == "" {
		raw.Model = model
	}
	resp := &ChatResponse{
		ID:       raw.ID,
		Model:    raw.Model,
		Choices:  raw.Choices,
		Usage:    usage,
		Provider: provider,
	}
	return resp, nil
}

// parseEmbedResponse parses the JSON response body into an EmbedResponse.
func parseEmbedResponse(body []byte, model, provider string, usage meter.Usage) (*EmbedResponse, error) {
	var raw struct {
		Model string      `json:"model"`
		Data  []EmbedData `json:"data"`
		Usage *dto.Usage  `json:"usage"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse embed response: %w", err)
	}
	if raw.Model == "" {
		raw.Model = model
	}
	return &EmbedResponse{
		Model:    raw.Model,
		Data:     raw.Data,
		Usage:    usage,
		Provider: provider,
	}, nil
}

// parseImageResponse parses the JSON response body into an ImageResponse.
func parseImageResponse(body []byte, model, provider string, usage meter.Usage) (*ImageResponse, error) {
	var raw struct {
		Created int64       `json:"created"`
		Data    []ImageData `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse image response: %w", err)
	}
	return &ImageResponse{
		Created:  raw.Created,
		Data:     raw.Data,
		Usage:    usage,
		Provider: provider,
	}, nil
}

// parseError reads an error response body and returns an error.
func parseError(resp *http.Response, provider string) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("relay: provider=%s HTTP %d: %s", provider, resp.StatusCode, string(body))
}

// marshalString wraps a string as a JSON string.
func marshalString(s string) string {
	if s == "" {
		return ""
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// Ensure imports are used.
var _ = constant.APITypeOpenAI
var _ = types.RelayFormatOpenAI
