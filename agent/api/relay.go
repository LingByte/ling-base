package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"

	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/channel/openai"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

// RelayProvider implements Provider by translating the canonical Anthropic
// request into a relay.ChatRequest, calling relay.Client.ChatStream() (streaming
// — relay's streaming channel now carries text deltas, tool-call deltas,
// reasoning deltas, and finish_reason), and assembling the response back into
// an anthropic.BetaMessage so the rest of the agent (loop, tools, sessions,
// compaction) is unchanged.
//
// This lets the agent use any of relay's 40+ provider adaptors (OpenAI, Claude,
// DeepSeek, Gemini, Moonshot, Qiniu-compatible endpoints, etc.) without
// changing the agent loop or tool layer.
type RelayProvider struct {
	client          *relay.Client
	model           string
	temperature     *float64
	maxTokens       int
	providerName    string // for display
	reasoningEffort string  // optional: low/medium/high for reasoning models
}

// NewRelayProvider builds a provider backed by a relay.Client.
// model is the default model ID to use when params.Model is empty.
// providerName is a human-readable label for /doctor and /stats.
func NewRelayProvider(client *relay.Client, model string, temperature *float64, maxTokens int, providerName string) *RelayProvider {
	return &RelayProvider{
		client:       client,
		model:        model,
		temperature:  temperature,
		maxTokens:    maxTokens,
		providerName: providerName,
	}
}

// NewRelayProviderFromConfig is a convenience constructor that builds a
// relay.Client with the OpenAI adaptor pointing at baseURL with apiKey.
// This is the most common path: an OpenAI-compatible endpoint (Qiniu,
// DeepSeek, Moonshot, etc.) accessed through relay's OpenAI channel.
func NewRelayProviderFromConfig(baseURL, apiKey, model string, temperature *float64, maxTokens int) *RelayProvider {
	provider := openai.NewProvider(apiKey, openai.WithBaseURL(baseURL))
	client := relay.New(relay.WithProvider(provider))
	return &RelayProvider{
		client:       client,
		model:        model,
		temperature:  temperature,
		maxTokens:    maxTokens,
		providerName: "relay",
	}
}

// SetReasoningEffort configures the reasoning effort level sent to the
// model. Use reasoning.OpenAIEffort() to map a canonical level to the
// OpenAI-compatible enum. Empty string means no reasoning effort is sent.
func (p *RelayProvider) SetReasoningEffort(effort string) {
	p.reasoningEffort = effort
}

// ProviderName returns the human-readable provider label.
func (p *RelayProvider) ProviderName() string {
	return p.providerName
}

// StreamTurn implements Provider. It uses relay's ChatStream() to get
// incremental text, tool-call, and reasoning deltas, forwarding text to
// sink.OnText for live TUI rendering and synthesizing Anthropic-shaped raw
// events for partial-message consumers. Tool-call deltas are accumulated
// across chunks (OpenAI streams them in fragments) and assembled at the end.
func (p *RelayProvider) StreamTurn(ctx context.Context, params anthropic.BetaMessageNewParams, sink StreamSink) (anthropic.BetaMessage, error) {
	req, err := p.translateRequest(params)
	if err != nil {
		return anthropic.BetaMessage{}, fmt.Errorf("relay: translate request: %w", err)
	}

	result, err := p.client.ChatStream(ctx, req)
	if err != nil {
		return anthropic.BetaMessage{}, fmt.Errorf("relay: chat stream: %w", err)
	}

	var text strings.Builder
	tools := map[int]*toolAccum{}
	finish := ""
	var inTok, outTok int64

	msgID := "msg_" + uuid.NewString()
	started := false
	startBlocks := func() {
		if started {
			return
		}
		started = true
		if sink.OnRawEvent != nil {
			// delivered = true (no return value needed here; sink signals)
		}
		sink.raw(synthEvent(map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id": msgID, "type": "message", "role": "assistant",
				"model": req.Model, "content": []any{}, "stop_reason": nil,
				"usage": map[string]any{"input_tokens": 0, "output_tokens": 0},
			},
		}))
		sink.raw(synthEvent(map[string]any{
			"type": "content_block_start", "index": 0,
			"content_block": map[string]any{"type": "text", "text": ""},
		}))
	}

	for chunk := range result.Ch {
		if chunk.Err != nil {
			return anthropic.BetaMessage{}, fmt.Errorf("relay: stream: %w", chunk.Err)
		}
		if chunk.Done {
			if chunk.Usage != nil {
				inTok = int64(chunk.Usage.InputTokens)
				outTok = int64(chunk.Usage.OutputTokens)
			}
			break
		}
		// Text delta
		if chunk.Delta != "" {
			startBlocks()
			text.WriteString(chunk.Delta)
			sink.text(chunk.Delta)
			sink.raw(synthEvent(map[string]any{
				"type": "content_block_delta", "index": 0,
				"delta": map[string]any{"type": "text_delta", "text": chunk.Delta},
			}))
		}
		// Tool-call deltas (accumulate across fragments by index)
		for _, tc := range chunk.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			acc := tools[idx]
			if acc == nil {
				acc = &toolAccum{}
				tools[idx] = acc
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if name := tc.Function.Name; name != "" {
				acc.name = name
			}
			acc.args.WriteString(tc.Function.Arguments)
		}
		if chunk.FinishReason != "" {
			finish = chunk.FinishReason
		}
	}

	// Close out the synthesized event sequence (only if we opened it — a
	// tool-only turn with no text never emits a text block).
	if started {
		sink.raw(synthEvent(map[string]any{"type": "content_block_stop", "index": 0}))
		sink.raw(synthEvent(map[string]any{
			"type":  "message_delta",
			"delta": map[string]any{"stop_reason": mapFinishReason(finish, len(tools) > 0)},
			"usage": map[string]any{"output_tokens": outTok},
		}))
		sink.raw(synthEvent(map[string]any{"type": "message_stop"}))
	}

	return assembleMessage(req.Model, text.String(), tools, finish, inTok, outTok)
}

// translateRequest converts Anthropic BetaMessageNewParams into a
// relay.ChatRequest. The translation mirrors OpenAIProvider.translateRequest
// but produces relay.Message instead of raw OpenAI JSON.
func (p *RelayProvider) translateRequest(params anthropic.BetaMessageNewParams) (*relay.ChatRequest, error) {
	model := string(params.Model)
	if model == "" {
		model = p.model
	}

	maxTokens := p.maxTokens
	if params.MaxTokens > 0 {
		maxTokens = int(params.MaxTokens)
	}
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	req := &relay.ChatRequest{
		Model:           model,
		Temperature:     p.temperature,
		MaxTokens:       &maxTokens,
		ReasoningEffort: p.reasoningEffort,
	}

	// System prompt → a leading system message.
	var sys strings.Builder
	for _, blk := range params.System {
		sys.WriteString(blk.Text)
	}
	if sys.Len() > 0 {
		req.Messages = append(req.Messages, relay.Message{
			Role:    "system",
			Content: sys.String(),
		})
	}

	// Conversation messages: marshal through JSON to extract the structured
	// content blocks (same approach as OpenAIProvider.translateRequest).
	rawMsgs, _ := json.Marshal(params.Messages)
	var amsgs []aMsg
	if err := json.Unmarshal(rawMsgs, &amsgs); err != nil {
		return nil, fmt.Errorf("translate messages: %w", err)
	}
	for _, m := range amsgs {
		req.Messages = append(req.Messages, relayMessage(m)...)
	}

	// Custom tools → relay.Tool (function type).
	rawTools, _ := json.Marshal(params.Tools)
	var atools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"input_schema"`
	}
	_ = json.Unmarshal(rawTools, &atools)
	for _, t := range atools {
		if t.Name == "" || len(t.InputSchema) == 0 {
			continue
		}
		var schema any
		_ = json.Unmarshal(t.InputSchema, &schema)
		req.Tools = append(req.Tools, relay.Tool{
			Type: "function",
			Function: dto.FunctionRequest{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}

	return req, nil
}

// relayMessage converts one Anthropic message (in the aMsg intermediate form)
// into one or more relay.Message values. Tool results become separate
// role:"tool" messages, matching the OpenAI Chat Completions convention.
func relayMessage(m aMsg) []relay.Message {
	var text strings.Builder
	var toolCallsJSON []byte
	var toolResults []relay.Message

	// Build tool_calls as JSON array if any tool_use blocks exist.
	type tcJSON struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	var tcList []tcJSON

	for _, b := range m.Content {
		switch b.Type {
		case "text":
			text.WriteString(b.Text)
		case "tool_use":
			args := string(b.Input)
			if args == "" {
				args = "{}"
			}
			tc := tcJSON{ID: b.ID, Type: "function"}
			tc.Function.Name = b.Name
			tc.Function.Arguments = args
			tcList = append(tcList, tc)
		case "tool_result":
			toolResults = append(toolResults, relay.Message{
				Role:       "tool",
				ToolCallId: b.ToolUseID,
				Content:    toolResultContent(b.Content),
			})
		}
	}

	if len(tcList) > 0 {
		toolCallsJSON, _ = json.Marshal(tcList)
	}

	var msgs []relay.Message
	if m.Role == "assistant" {
		am := relay.Message{
			Role:    "assistant",
			Content: text.String(),
		}
		if toolCallsJSON != nil {
			am.ToolCalls = toolCallsJSON
		}
		msgs = append(msgs, am)
	} else { // user
		if text.Len() > 0 {
			msgs = append(msgs, relay.Message{
				Role:    "user",
				Content: text.String(),
			})
		}
		msgs = append(msgs, toolResults...)
	}
	return msgs
}
