//
// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT
//

// Package relay provides a [model.Model] implementation backed by the
// ling-base/relay multi-provider layer.
//
// It bridges the agentkit model.Model interface (GenerateContent returning
// a <-chan *model.Response) to relay's RichChatStream API, so agentkit
// flows can use any of relay's 40+ provider adaptors (OpenAI, Anthropic,
// Gemini, Ollama, DeepSeek, ...) without depending on individual SDKs.
//
// # Quick start
//
//	m, err := relay.New("gpt-4o",
//	    relay.WithAPIKey("sk-..."),
//	    relay.WithBaseURL("https://api.openai.com"),
//	    relay.WithChannel("openai"),
//	)
//	respCh, err := m.GenerateContent(ctx, &model.Request{
//	    Messages: []model.Message{model.NewUserMessage("Hello")},
//	    GenerationConfig: model.GenerationConfig{Stream: true},
//	})
//	for resp := range respCh {
//	    // ...
//	}
package relay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/agentkit/model"
	"github.com/LingByte/ling-base/relay"
	relayclaude "github.com/LingByte/ling-base/relay/channel/claude"
	relayopenai "github.com/LingByte/ling-base/relay/channel/openai"
)

// Channel identifies the relay provider channel to use.
type Channel string

const (
	// ChannelOpenAI selects the OpenAI-compatible channel (also works for
	// any OpenAI-compatible endpoint like rightapi, deepseek, etc.).
	ChannelOpenAI Channel = "openai"
	// ChannelClaude selects the native Anthropic Messages API channel.
	ChannelClaude Channel = "claude"
)

// Model implements [model.Model] by delegating to a [relay.Client].
type Model struct {
	client       *relay.Client
	name         string
	channel      Channel
	baseURL      string
	apiKey       string
	contextWin   int
	channelBuf   int
	extraHeaders map[string]string
	extraFields  map[string]any

	mu     sync.RWMutex
	cached *relay.Client // built lazily from channel/baseURL/apiKey
}

// Option configures a [Model].
type Option func(*Model)

// WithAPIKey sets the API key for the relay provider.
func WithAPIKey(key string) Option {
	return func(m *Model) { m.apiKey = key }
}

// WithBaseURL sets the base URL for the relay provider.
// For ChannelOpenAI, the adaptor appends /v1/chat/completions.
// For ChannelClaude, the adaptor appends /v1/messages.
func WithBaseURL(url string) Option {
	return func(m *Model) { m.baseURL = url }
}

// WithChannel selects the relay channel (default: ChannelOpenAI).
func WithChannel(ch Channel) Option {
	return func(m *Model) { m.channel = ch }
}

// WithContextWindow sets the context window size (in tokens) reported by Info().
func WithContextWindow(tokens int) Option {
	return func(m *Model) { m.contextWin = tokens }
}

// WithChannelBufferSize sets the buffer size for the response channel.
func WithChannelBufferSize(size int) Option {
	return func(m *Model) { m.channelBuf = size }
}

// WithHeaders adds static HTTP headers to outbound requests.
func WithHeaders(headers map[string]string) Option {
	return func(m *Model) {
		if m.extraHeaders == nil {
			m.extraHeaders = make(map[string]string)
		}
		for k, v := range headers {
			m.extraHeaders[k] = v
		}
	}
}

// WithExtraFields merges provider-specific top-level request body fields.
func WithExtraFields(fields map[string]any) Option {
	return func(m *Model) {
		if m.extraFields == nil {
			m.extraFields = make(map[string]any)
		}
		for k, v := range fields {
			m.extraFields[k] = v
		}
	}
}

// WithClient uses a pre-constructed [relay.Client] instead of building
// one from channel/baseURL/apiKey. This is useful when you need custom
// relay options (fallback, circuit breaker, etc.).
func WithClient(c *relay.Client) Option {
	return func(m *Model) {
		m.client = c
		m.cached = c
	}
}

// New creates a new relay-backed [Model].
func New(name string, opts ...Option) *Model {
	m := &Model{
		name:    name,
		channel: ChannelOpenAI,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Info implements [model.Model].
func (m *Model) Info() model.Info {
	return model.Info{
		Name:          m.name,
		ContextWindow: m.contextWin,
	}
}

// GenerateContent implements [model.Model]. It translates the agentkit
// model.Request into a relay.RichChatRequest, streams the response, and
// translates relay rich stream chunks back into model.Response objects
// on the returned channel.
func (m *Model) GenerateContent(
	ctx context.Context,
	request *model.Request,
) (<-chan *model.Response, error) {
	if request == nil {
		return nil, fmt.Errorf("relay: request cannot be nil")
	}

	client, err := m.clientOrBuild()
	if err != nil {
		return nil, err
	}

	richReq, err := m.translateRequest(request)
	if err != nil {
		return nil, fmt.Errorf("relay: translate request: %w", err)
	}

	bufSize := m.channelBuf
	if bufSize <= 0 {
		bufSize = 64
	}
	responseChan := make(chan *model.Response, bufSize)

	go func() {
		defer close(responseChan)
		m.streamToChannel(ctx, client, richReq, request, responseChan)
	}()

	return responseChan, nil
}

// clientOrBuild lazily constructs a relay.Client from the configured
// channel/baseURL/apiKey, or returns the pre-set client.
func (m *Model) clientOrBuild() (*relay.Client, error) {
	m.mu.RLock()
	if m.cached != nil {
		m.mu.RUnlock()
		return m.cached, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock.
	if m.cached != nil {
		return m.cached, nil
	}

	if m.apiKey == "" {
		return nil, fmt.Errorf("relay: apiKey is required (use WithAPIKey or WithClient)")
	}

	var provider relay.Provider
	switch m.channel {
	case ChannelClaude:
		var opts []relayclaude.Option
		if m.baseURL != "" {
			opts = append(opts, relayclaude.WithBaseURL(m.baseURL))
		}
		provider = relayclaude.NewProvider(m.apiKey, opts...)
	default: // ChannelOpenAI
		var opts []relayopenai.Option
		if m.baseURL != "" {
			opts = append(opts, relayopenai.WithBaseURL(strings.TrimSuffix(m.baseURL, "/")))
		}
		provider = relayopenai.NewProvider(m.apiKey, opts...)
	}

	client := relay.New(relay.WithProvider(provider))
	m.cached = client
	return client, nil
}

// translateRequest converts a [model.Request] to a [relay.RichChatRequest].
func (m *Model) translateRequest(req *model.Request) (*relay.RichChatRequest, error) {
	richReq := &relay.RichChatRequest{
		Model:     m.name,
		MaxTokens: 4096, // default; overridden below if set
		Stream:    req.Stream,
	}

	if req.MaxTokens != nil {
		richReq.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		richReq.Temperature = req.Temperature
	}
	if req.ReasoningEffort != nil {
		richReq.ReasoningEffort = *req.ReasoningEffort
	}

	// Extract system message(s) into the System field; relay uses a
	// top-level System string rather than a system role in Messages.
	var nonSystemMsgs []relay.RichMessage
	for _, msg := range req.Messages {
		if msg.Role == model.RoleSystem {
			if richReq.System != "" {
				richReq.System += "\n\n"
			}
			richReq.System += msg.Content
			continue
		}
		richMsg, err := modelMessageToRich(msg)
		if err != nil {
			return nil, fmt.Errorf("convert message role=%s: %w", msg.Role, err)
		}
		nonSystemMsgs = append(nonSystemMsgs, richMsg)
	}
	richReq.Messages = nonSystemMsgs

	// Convert tools.
	for name, t := range req.Tools {
		decl := t.Declaration()
		if decl == nil {
			continue
		}
		var schemaBytes json.RawMessage
		if decl.InputSchema != nil {
			b, err := json.Marshal(decl.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("marshal tool %s schema: %w", name, err)
			}
			schemaBytes = b
		}
		richReq.Tools = append(richReq.Tools, relay.RichTool{
			Name:        decl.Name,
			Description: decl.Description,
			InputSchema: schemaBytes,
			Type:        "function",
		})
	}

	return richReq, nil
}

// modelMessageToRich converts a [model.Message] to a [relay.RichMessage].
func modelMessageToRich(msg model.Message) (relay.RichMessage, error) {
	role := string(msg.Role)
	// model.RoleTool → relay "user" with tool_result blocks.
	if msg.Role == model.RoleTool {
		role = "user"
		blocks := []relay.ContentBlock{
			relay.NewToolResultBlock(msg.ToolID, msg.Content, false),
		}
		return relay.RichMessage{Role: role, Content: blocks}, nil
	}

	var blocks []relay.ContentBlock

	// Multimodal content parts.
	for _, part := range msg.ContentParts {
		switch part.Type {
		case model.ContentTypeText:
			if part.Text != nil {
				blocks = append(blocks, relay.NewTextBlock(*part.Text))
			}
		case model.ContentTypeImage:
			if part.Image != nil {
				blocks = append(blocks, imageToBlock(part.Image))
			}
		case model.ContentTypeAudio, model.ContentTypeVideo, model.ContentTypeFile:
			// For non-text/image parts, fall back to a text description.
			// Full audio/video/file support can be added later.
			if part.File != nil && part.File.URL != "" {
				blocks = append(blocks, relay.NewTextBlock(part.File.URL))
			}
		}
	}

	// Plain text content (when no ContentParts).
	if len(blocks) == 0 && msg.Content != "" {
		blocks = append(blocks, relay.NewTextBlock(msg.Content))
	}

	// Tool calls (assistant message with tool_use).
	for _, tc := range msg.ToolCalls {
		args := json.RawMessage(tc.Function.Arguments)
		if len(args) == 0 || strings.TrimSpace(string(args)) == "" {
			args = json.RawMessage("{}")
		}
		blocks = append(blocks, relay.NewToolUseBlock(tc.ID, tc.Function.Name, args))
	}

	// Reasoning content → thinking block.
	if msg.ReasoningContent != "" {
		blocks = append(blocks, relay.NewThinkingBlock(msg.ReasoningContent, msg.ReasoningSignature))
	}

	if len(blocks) == 0 {
		blocks = []relay.ContentBlock{relay.NewTextBlock("")}
	}

	return relay.RichMessage{Role: role, Content: blocks}, nil
}

// imageToBlock converts a [model.Image] to a [relay.ContentBlock].
func imageToBlock(img *model.Image) relay.ContentBlock {
	if img.URL != "" {
		// URL-based image: represent as text for now (relay's flat
		// translation will handle URL images in future versions).
		return relay.NewTextBlock(fmt.Sprintf("![image](%s)", img.URL))
	}
	if len(img.Data) > 0 {
		// Inline base64 image.
		mime := img.Format
		if mime == "" {
			mime = "image/png"
		}
		if !strings.HasPrefix(mime, "image/") {
			mime = "image/" + mime
		}
		return relay.ContentBlock{
			Type: relay.BlockTypeImage,
			Source: &relay.ImageSource{
				Type:      "base64",
				MediaType: mime,
				Data:      base64.StdEncoding.EncodeToString(img.Data),
			},
		}
	}
	return relay.NewTextBlock("")
}

// streamToChannel drains the relay rich stream and emits model.Response
// objects on responseChan.
func (m *Model) streamToChannel(
	ctx context.Context,
	client *relay.Client,
	richReq *relay.RichChatRequest,
	origReq *model.Request,
	responseChan chan<- *model.Response,
) {
	result, err := client.RichChatStream(ctx, richReq)
	if err != nil {
		responseChan <- errorResponse(err, model.ErrorTypeAPIError)
		return
	}

	var (
		textBuf       strings.Builder
		toolCalls     []model.ToolCall
		toolCallMu    sync.Mutex
		thinkingBuf   strings.Builder
		startTime     = time.Now()
		firstToken    = true
		// Accumulate tool calls by index for streaming deltas.
		toolAccum = map[int]*struct {
			id   string
			name string
			args strings.Builder
		}{}
	)

	for chunk := range result.Ch {
		if chunk.Err != nil {
			responseChan <- errorResponse(chunk.Err, model.ErrorTypeStreamError)
			return
		}

		switch chunk.Type {
		case relay.ChunkTypeTextDelta:
			if firstToken {
				firstToken = false
			}
			textBuf.WriteString(chunk.Text)
			responseChan <- deltaResponse(chunk.Text, "", nil)

		case relay.ChunkTypeThinkingDelta:
			thinkingBuf.WriteString(chunk.Thinking)
			responseChan <- deltaResponse("", chunk.Thinking, nil)

		case relay.ChunkTypeToolUseDelta:
			toolCallMu.Lock()
			acc := toolAccum[chunk.ToolUseIndex]
			if acc == nil {
				acc = &struct {
					id   string
					name string
					args strings.Builder
				}{}
				toolAccum[chunk.ToolUseIndex] = acc
			}
			if chunk.ToolUseID != "" {
				acc.id = chunk.ToolUseID
			}
			if chunk.ToolUseName != "" {
				acc.name = chunk.ToolUseName
			}
			acc.args.WriteString(chunk.InputFragment)
			toolCallMu.Unlock()

		case relay.ChunkTypeFinish:
			// Finalize tool calls from accumulators.
			toolCallMu.Lock()
			for i := 0; i < len(toolAccum); i++ {
				acc := toolAccum[i]
				if acc == nil {
					continue
				}
				idx := i
				toolCalls = append(toolCalls, model.ToolCall{
					Type:  "function",
					ID:    acc.id,
					Index: &idx,
					Function: model.FunctionDefinitionParam{
						Name:      acc.name,
						Arguments: []byte(acc.args.String()),
					},
				})
			}
			toolCallMu.Unlock()

		case relay.ChunkTypeUsage:
			// Build the final response.
			final := result.Final
			resp := &model.Response{
				ID:        final.ID,
				Object:    model.ObjectTypeChatCompletion,
				Created:   time.Now().Unix(),
				Model:     m.name,
				Timestamp: time.Now(),
				Done:      true,
				Usage: &model.Usage{
					PromptTokens:     int(final.InputTokens),
					CompletionTokens: int(final.OutputTokens),
					TotalTokens:      int(final.InputTokens + final.OutputTokens),
				},
			}
			// Timing info.
			if !firstToken {
				resp.Usage.TimingInfo = &model.TimingInfo{
					FirstTokenDuration: time.Since(startTime),
				}
			}
			// Build the final Choice with the complete message.
			finalMsg := model.Message{
				Role:            model.RoleAssistant,
				Content:         textBuf.String(),
				ReasoningContent: thinkingBuf.String(),
				ToolCalls:        toolCalls,
			}
			// If we have tool calls, set stop reason to "tool_calls".
			finishReason := final.StopReason
			if finishReason == "" {
				finishReason = "stop"
			}
			// Map relay stop reasons to OpenAI-style finish reasons.
			finishReason = mapStopReason(finishReason)
			resp.Choices = []model.Choice{{
				Index:        0,
				Message:      finalMsg,
				FinishReason: &finishReason,
			}}
			responseChan <- resp
		}
	}
}

// mapStopReason converts relay stop reasons to OpenAI-style finish reasons.
func mapStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "content_filter":
		return "content_filter"
	default:
		return reason
	}
}

// deltaResponse creates a streaming delta [model.Response].
func deltaResponse(textDelta, thinkingDelta string, toolCalls []model.ToolCall) *model.Response {
	resp := &model.Response{
		Object:    model.ObjectTypeChatCompletionChunk,
		Created:   time.Now().Unix(),
		Timestamp: time.Now(),
		IsPartial: true,
	}
	delta := model.Message{
		Role: model.RoleAssistant,
	}
	if textDelta != "" {
		delta.Content = textDelta
	}
	if thinkingDelta != "" {
		delta.ReasoningContent = thinkingDelta
	}
	if len(toolCalls) > 0 {
		delta.ToolCalls = toolCalls
	}
	resp.Choices = []model.Choice{{
		Index: 0,
		Delta: delta,
	}}
	return resp
}

// errorResponse creates a [model.Response] carrying an error.
func errorResponse(err error, errType string) *model.Response {
	return &model.Response{
		Object:    model.ObjectTypeError,
		Timestamp: time.Now(),
		Done:      true,
		Error: &model.ResponseError{
			Message: err.Error(),
			Type:    errType,
		},
	}
}
