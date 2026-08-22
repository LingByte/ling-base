//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package summary

import (
	"context"

	"github.com/LingByte/ling-base/agentkit/internal/jsonmap"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	"github.com/LingByte/ling-base/agentkit/tool"
)

type cacheSafeForkRequestContextKey struct{}

// ContextWithCacheSafeForkRequest attaches the parent model request used for a
// cache-safe summary fork. Framework code supplies this context when it already
// has the request that would be sent to the parent conversation.
func ContextWithCacheSafeForkRequest(ctx context.Context, req *compat.Request) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return ctx
	}
	return context.WithValue(ctx, cacheSafeForkRequestContextKey{}, req)
}

// CacheSafeForkRequestFromContext returns the parent model request attached by
// ContextWithCacheSafeForkRequest, if any.
func CacheSafeForkRequestFromContext(ctx context.Context) (*compat.Request, bool) {
	if ctx == nil {
		return nil, false
	}
	req, ok := ctx.Value(cacheSafeForkRequestContextKey{}).(*compat.Request)
	return req, ok && req != nil
}

type cacheSafeForkingResolver interface {
	CacheSafeForkingEnabled(context.Context, *session.Session) bool
}

func cacheSafeForkingEnabled(
	ctx context.Context,
	s SessionSummarizer,
	sess *session.Session,
) bool {
	if s == nil {
		return false
	}
	if resolver, ok := s.(cacheSafeForkingResolver); ok {
		return resolver.CacheSafeForkingEnabled(ctx, sess)
	}
	metadata := s.Metadata()
	enabled, _ := metadata[metadataKeyCacheSafeForking].(bool)
	return enabled
}

func cloneRequestForCacheSafeFork(req *compat.Request) *compat.Request {
	if req == nil {
		return nil
	}

	cloned := *req
	cloned.Messages = cloneMessagesForCacheSafeFork(req.Messages)
	cloned.GenerationConfig = cloneGenerationConfigForCacheSafeFork(req.GenerationConfig)
	cloned.StructuredOutput = cloneStructuredOutputForCacheSafeFork(req.StructuredOutput)
	cloned.ExtraFields = jsonmap.Clone(req.ExtraFields)
	cloned.Headers = cloneHeadersForCacheSafeFork(req.Headers)
	if reqTools, ok := req.Tools.(map[string]tool.Tool); ok {
		cloned.Tools = cloneToolsForCacheSafeFork(reqTools)
	}
	return &cloned
}

func cloneMessagesForCacheSafeFork(messages []compat.Message) []compat.Message {
	if messages == nil {
		return nil
	}
	cloned := make([]compat.Message, len(messages))
	for i := range messages {
		cloned[i] = cloneMessageForCacheSafeFork(messages[i])
	}
	return cloned
}

func cloneMessageForCacheSafeFork(msg compat.Message) compat.Message {
	cloned := msg
	cloned.ContentParts = cloneContentPartsForCacheSafeFork(msg.ContentParts)
	cloned.ToolCalls = cloneToolCallsForCacheSafeFork(msg.ToolCalls)
	return cloned
}

func cloneContentPartsForCacheSafeFork(parts []compat.ContentPart) []compat.ContentPart {
	if parts == nil {
		return nil
	}
	cloned := make([]compat.ContentPart, len(parts))
	for i := range parts {
		cloned[i] = cloneContentPartForCacheSafeFork(parts[i])
	}
	return cloned
}

func cloneContentPartForCacheSafeFork(part compat.ContentPart) compat.ContentPart {
	cloned := part
	if part.Text != nil {
		text := *part.Text
		cloned.Text = &text
	}
	if part.Image != nil {
		image := *part.Image
		if part.Image.Data != nil {
			image.Data = append([]byte(nil), part.Image.Data...)
		}
		cloned.Image = &image
	}
	if part.Audio != nil {
		audio := *part.Audio
		if part.Audio.Data != nil {
			audio.Data = append([]byte(nil), part.Audio.Data...)
		}
		cloned.Audio = &audio
	}
	if part.Video != nil {
		video := *part.Video
		if part.Video.Data != nil {
			video.Data = append([]byte(nil), part.Video.Data...)
		}
		cloned.Video = &video
	}
	if part.File != nil {
		file := *part.File
		if part.File.Data != nil {
			file.Data = append([]byte(nil), part.File.Data...)
		}
		cloned.File = &file
	}
	return cloned
}

func cloneToolCallsForCacheSafeFork(toolCalls []compat.ToolCall) []compat.ToolCall {
	if toolCalls == nil {
		return nil
	}
	cloned := make([]compat.ToolCall, len(toolCalls))
	for i := range toolCalls {
		cloned[i] = toolCalls[i]
		if toolCalls[i].Function.Arguments != nil {
			cloned[i].Function.Arguments = append([]byte(nil), toolCalls[i].Function.Arguments...)
		}
		if toolCalls[i].Index != nil {
			index := *toolCalls[i].Index
			cloned[i].Index = &index
		}
		cloned[i].ExtraFields = jsonmap.Clone(toolCalls[i].ExtraFields)
	}
	return cloned
}

func cloneGenerationConfigForCacheSafeFork(cfg compat.GenerationConfig) compat.GenerationConfig {
	cloned := cfg
	if cfg.Stop != nil {
		cloned.Stop = append([]string(nil), cfg.Stop...)
	}
	return cloned
}

func cloneStructuredOutputForCacheSafeFork(out *compat.StructuredOutput) *compat.StructuredOutput {
	if out == nil {
		return nil
	}
	cloned := *out
	if out.JSONSchema != nil {
		schema := *out.JSONSchema
		schema.Schema = jsonmap.Clone(out.JSONSchema.Schema)
		cloned.JSONSchema = &schema
	}
	return &cloned
}

func cloneHeadersForCacheSafeFork(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	cloned := make(map[string]string, len(headers))
	for k, v := range headers {
		cloned[k] = v
	}
	return cloned
}

func cloneToolsForCacheSafeFork(tools map[string]tool.Tool) map[string]tool.Tool {
	if tools == nil {
		return nil
	}
	cloned := make(map[string]tool.Tool, len(tools))
	for name, t := range tools {
		cloned[name] = t
	}
	return cloned
}
