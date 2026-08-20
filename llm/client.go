// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package llm is the unified entry point for calling AI/LLM providers.
// It wraps the relay adaptor system with a clean, framework-agnostic API
// and integrates usage metering.
//
// Quick start:
//
//	import (
//		"github.com/LingByte/ling-base/llm"
//		"github.com/LingByte/ling-base/llm/meter"
//		"github.com/LingByte/ling-base/llm/provider/openai"
//	)
//
//	client := llm.New(
//		llm.WithProvider(openai.New("sk-xxx")),
//		llm.WithMeter(meter.NewMemoryMeter(nil)),
//	)
//	resp, err := client.Chat(ctx, &llm.ChatRequest{
//		Model:    "gpt-4o",
//		Messages: []llm.Message{{Role: "user", Content: json.RawMessage(`"Hello"`)}},
//	})
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/ling-base/llm/constant"
	"github.com/LingByte/ling-base/llm/meter"
	"github.com/LingByte/ling-base/llm/relay"
	"github.com/LingByte/ling-base/llm/relaykit/dto"
	relaymode "github.com/LingByte/ling-base/llm/relaymode"
	"github.com/LingByte/ling-base/llm/relaykit/types"
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
// It wraps the low-level relay.Adaptor with a simpler, type-safe API.
type Provider interface {
	Name() string
	ApiType() int
	Adaptor() relay.Adaptor
}

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
		return fmt.Errorf("llm: no provider configured")
	}
	return nil
}

// buildRelayInfo creates a RelayInfo for the given request.
func (c *Client) buildRelayInfo(mode int, model string, stream bool) *relay.RelayInfo {
	info := relay.NewRelayInfo()
	info.RelayMode = mode
	info.OriginModelName = model
	info.IsStream = stream
	info.ApiType = c.provider.ApiType()
	info.ChannelMeta = &relay.ChannelMeta{
		ApiType:           c.provider.ApiType(),
		UpstreamModelName: model,
		ApiKey:            c.providerApiKey(),
	}
	return info
}

func (c *Client) providerApiKey() string {
	// Providers expose their API key through the ChannelMeta in Adaptor.Init.
	// This is a placeholder; each provider sets its key in its constructor.
	return ""
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
		return nil, fmt.Errorf("llm: convert request: %w", err)
	}

	// Marshal request body.
	var body io.Reader
	switch v := converted.(type) {
	case io.Reader:
		body = v
	default:
		jsonData, err := json.Marshal(converted)
		if err != nil {
			return nil, fmt.Errorf("llm: marshal request: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	// Build URL and send request.
	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, fmt.Errorf("llm: build URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("llm: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := adaptor.SetupRequestHeader(ctx, &httpReq.Header, info); err != nil {
		return nil, fmt.Errorf("llm: setup headers: %w", err)
	}

	resp, err := c.doRequest(ctx, adaptor, info, httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp, c.provider.Name())
	}

	// Parse response using a dummy ResponseWriter (we don't stream).
	var dummyW dummyResponseWriter
	usageRaw, apiErr := adaptor.DoResponse(ctx, resp, info, &dummyW)
	if apiErr != nil {
		return nil, fmt.Errorf("llm: parse response: %w", apiErr)
	}

	usage := extractUsage(usageRaw)
	c.record(ctx, req.Model, usage)

	// Parse the response body that the adaptor wrote to dummyW.
	chatResp, err := parseChatResponse(dummyW.Bytes(), req.Model, c.provider.Name(), usage)
	if err != nil {
		return nil, fmt.Errorf("llm: parse chat response: %w", err)
	}
	return chatResp, nil
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
		return nil, fmt.Errorf("llm: convert embed request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal embed request: %w", err)
	}

	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		return nil, fmt.Errorf("llm: build URL: %w", err)
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
		return nil, fmt.Errorf("llm: parse embed response: %w", apiErr)
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
		return nil, fmt.Errorf("llm: convert image request: %w", err)
	}

	jsonData, err := json.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal image request: %w", err)
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
		return nil, fmt.Errorf("llm: parse image response: %w", apiErr)
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

// ─── Helpers ─────────────────────────────────────────────────────

func (c *Client) doRequest(ctx context.Context, adaptor relay.Adaptor, info *relay.RelayInfo, httpReq *http.Request) (*http.Response, error) {
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
	return fmt.Errorf("llm: provider=%s HTTP %d: %s", provider, resp.StatusCode, string(body))
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
