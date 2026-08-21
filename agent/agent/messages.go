// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package agent messages: these are thin aliases of relay's rich content-block
// types, so the agent loop has zero dependency on anthropic-sdk-go.
//
// The agent loop speaks ContentBlock/RichMessage/RichResponse instead of
// anthropic.BetaContentBlockParamUnion/BetaMessageParam/BetaMessage.
// The relay layer handles the wire format (Anthropic Messages API natively,
// OpenAI Chat Completions via translation).

package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/LingByte/ling-base/relay"
)

// Message is a conversation message (alias of relay.RichMessage).
type Message = relay.RichMessage

// ContentBlock is a content block within a message (alias of relay.ContentBlock).
type ContentBlock = relay.ContentBlock

// Response is the assembled model response (alias of relay.RichResponse).
type Response = relay.RichResponse

// StreamChunk is one chunk in a streaming response (alias of relay.RichStreamChunk).
type StreamChunk = relay.RichStreamChunk

// Tool is a tool definition (alias of relay.RichTool).
type Tool = relay.RichTool

// ─── Constructors (mirror anthropic.NewBeta* but relay-native) ───────────

// NewTextBlock creates a text content block.
func NewTextBlock(text string) ContentBlock { return relay.NewTextBlock(text) }

// NewToolUseBlock creates a tool_use content block.
func NewToolUseBlock(id, name string, input json.RawMessage) ContentBlock {
	return relay.NewToolUseBlock(id, name, input)
}

// NewToolResultBlock creates a tool_result content block.
func NewToolResultBlock(toolUseID, content string, isError bool) ContentBlock {
	return relay.NewToolResultBlock(toolUseID, content, isError)
}

// NewToolResultBlockRaw creates a tool_result with raw JSON content.
func NewToolResultBlockRaw(toolUseID string, content json.RawMessage, isError bool) ContentBlock {
	return relay.NewToolResultBlockRaw(toolUseID, content, isError)
}

// NewThinkingBlock creates a thinking content block.
func NewThinkingBlock(thinking, signature string) ContentBlock {
	return relay.NewThinkingBlock(thinking, signature)
}

// NewImageBlock creates an image content block.
func NewImageBlock(mediaType, base64Data string) ContentBlock {
	return relay.NewImageBlock(mediaType, base64Data)
}

// NewUserMessage creates a user message with text content.
func NewUserMessage(text string) Message { return relay.NewUserMessage(text) }

// NewUserMessageBlocks creates a user message from content blocks.
func NewUserMessageBlocks(blocks ...ContentBlock) Message {
	return relay.NewUserMessageBlocks(blocks...)
}

// NewAssistantMessageBlocks creates an assistant message from content blocks.
func NewAssistantMessageBlocks(blocks ...ContentBlock) Message {
	return relay.NewAssistantMessageBlocks(blocks...)
}

// ─── Block type constants (re-exported for convenience) ─────────────────

const (
	BlockText       = relay.BlockTypeText
	BlockToolUse    = relay.BlockTypeToolUse
	BlockToolResult = relay.BlockTypeToolResult
	BlockThinking   = relay.BlockTypeThinking
	BlockImage      = relay.BlockTypeImage
)

// ─── Stream chunk type constants ─────────────────────────────────────────

const (
	ChunkTextDelta     = relay.ChunkTypeTextDelta
	ChunkToolUseDelta  = relay.ChunkTypeToolUseDelta
	ChunkThinkingDelta = relay.ChunkTypeThinkingDelta
	ChunkFinish        = relay.ChunkTypeFinish
	ChunkUsage         = relay.ChunkTypeUsage
	ChunkError         = relay.ChunkTypeError
)

// ─── Model type (replaces anthropic.Model) ───────────────────────────────

// Model is a model identifier (e.g. "claude-sonnet-4-5-20250929").
type Model string

// FriendlyError returns a human-readable error message for common API errors.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "429") {
		return "Rate limited or insufficient quota (429). " + msg
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") {
		return "Authentication failed. Check your API key. " + msg
	}
	return msg
}

// ContextWindow returns the effective input-token limit for a model.
// override > model table > 0 (unknown).
func ContextWindow(model string, override int) (limit int, source string) {
	if override > 0 {
		return override, "config override"
	}
	if n, ok := modelContextWindows[model]; ok {
		return n, "model default"
	}
	return 0, "unknown — using compaction fallback"
}

// modelContextWindows is the per-model input-token limit.
var modelContextWindows = map[string]int{
	"claude-opus-4-8":   200_000,
	"claude-opus-4-7":   200_000,
	"claude-sonnet-4-6": 200_000,
	"claude-haiku-4-5":  200_000,
}

// ─── Provider interface (replaces api.Provider) ──────────────────────────

// Provider runs one model turn and returns the assembled response.
// This is the agent's own abstraction; relay.Client implements it via
// RichChatStream.
type Provider interface {
	// StreamTurn issues one model call, forwarding incremental updates to
	// the sink, and returns the fully assembled response.
	StreamTurn(ctx context.Context, req *relay.RichChatRequest, sink StreamSink) (*relay.RichResponse, error)
}

// StreamSink receives streaming updates during one model turn.
type StreamSink struct {
	// OnText is called for each text delta.
	OnText func(string)
	// OnChunk is called for every raw stream chunk (for partial-message
	// consumers). May be nil.
	OnChunk func(relay.RichStreamChunk)
}

// Text forwards a text delta to the sink (nil-safe).
func (s StreamSink) Text(delta string) {
	if s.OnText != nil {
		s.OnText(delta)
	}
}

// Chunk forwards a raw stream chunk to the sink (nil-safe).
func (s StreamSink) Chunk(chunk relay.RichStreamChunk) {
	if s.OnChunk != nil {
		s.OnChunk(chunk)
	}
}
