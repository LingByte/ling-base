//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package toolcall

import (
	"context"
	"fmt"
	"strings"
	"testing"

	log "github.com/LingByte/ling-base/common/logger"
	compat "github.com/LingByte/ling-base/relay/compat"
)

var sanitizeMessagesBenchmarkSink []compat.Message

func BenchmarkSanitizeMessagesWithTools(b *testing.B) {
	originalWarnfContext := log.WarnfContext
	log.WarnfContext = func(context.Context, string, ...any) {}
	b.Cleanup(func() {
		log.WarnfContext = originalWarnfContext
	})

	benchmarks := []struct {
		name     string
		messages []compat.Message
		bytes    int64
	}{
		{
			name:     "text/messages=32",
			messages: sanitizeBenchmarkTextMessages(32, 64),
		},
		{
			name:     "text/messages=512",
			messages: sanitizeBenchmarkTextMessages(512, 64),
		},
		{
			name:     "text/messages=2048",
			messages: sanitizeBenchmarkTextMessages(2048, 64),
		},
		{
			name:     "large_text/messages=512/payload_bytes=3145728",
			messages: sanitizeBenchmarkTextMessages(512, 6*1024),
			bytes:    3 * 1024 * 1024,
		},
		{
			name:     "valid_tool_rounds/messages=32",
			messages: sanitizeBenchmarkValidToolRounds(16),
		},
		{
			name:     "valid_tool_rounds/messages=512",
			messages: sanitizeBenchmarkValidToolRounds(256),
		},
		{
			name:     "valid_tool_rounds/messages=2048",
			messages: sanitizeBenchmarkValidToolRounds(1024),
		},
		{
			name:     "mixed_tool_rounds/messages=32",
			messages: sanitizeBenchmarkMixedToolRounds(8),
		},
		{
			name:     "mixed_tool_rounds/messages=512",
			messages: sanitizeBenchmarkMixedToolRounds(128),
		},
		{
			name:     "mixed_tool_rounds/messages=2048",
			messages: sanitizeBenchmarkMixedToolRounds(512),
		},
	}
	ctx := context.Background()
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			if benchmark.bytes > 0 {
				b.SetBytes(benchmark.bytes)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sanitizeMessagesBenchmarkSink = SanitizeMessagesWithTools(
					ctx,
					benchmark.messages,
					nil,
				)
			}
		})
	}
}

func sanitizeBenchmarkTextMessages(count int, contentBytes int) []compat.Message {
	messages := make([]compat.Message, count)
	content := strings.Repeat("t", contentBytes)
	for i := range messages {
		role := compat.RoleUser
		if i%2 == 1 {
			role = compat.RoleAssistant
		}
		messages[i] = compat.Message{Role: role, Content: content}
	}
	return messages
}

func sanitizeBenchmarkValidToolRounds(rounds int) []compat.Message {
	messages := make([]compat.Message, 0, rounds*2)
	for i := 0; i < rounds; i++ {
		callID := fmt.Sprintf("valid-call-%d", i)
		messages = append(messages,
			compat.Message{
				Role: compat.RoleAssistant,
				ToolCalls: []compat.ToolCall{{
					ID: callID,
					Function: compat.FunctionDefinitionParam{
						Name:      "benchmark_tool",
						Arguments: []byte(`{"value":"benchmark"}`),
					},
				}},
			},
			compat.Message{
				Role:     compat.RoleTool,
				ToolID:   callID,
				ToolName: "benchmark_tool",
				Content:  "benchmark result",
			},
		)
	}
	return messages
}

func sanitizeBenchmarkMixedToolRounds(rounds int) []compat.Message {
	messages := make([]compat.Message, 0, rounds*4)
	for i := 0; i < rounds; i++ {
		validID := fmt.Sprintf("valid-call-%d", i)
		invalidID := fmt.Sprintf("invalid-call-%d", i)
		orphanID := fmt.Sprintf("orphan-call-%d", i)
		messages = append(messages,
			compat.Message{
				Role: compat.RoleAssistant,
				ToolCalls: []compat.ToolCall{
					{
						ID: validID,
						Function: compat.FunctionDefinitionParam{
							Name:      "benchmark_tool",
							Arguments: []byte(`{"value":"benchmark"}`),
						},
					},
					{
						ID: invalidID,
						Function: compat.FunctionDefinitionParam{
							Arguments: []byte(`{"value":"benchmark"}`),
						},
					},
					{
						ID: orphanID,
						Function: compat.FunctionDefinitionParam{
							Name:      "benchmark_tool",
							Arguments: []byte(`{"value":"benchmark"}`),
						},
					},
				},
			},
			compat.Message{
				Role:     compat.RoleTool,
				ToolID:   validID,
				ToolName: "benchmark_tool",
				Content:  "valid result",
			},
			compat.Message{
				Role:     compat.RoleTool,
				ToolID:   invalidID,
				ToolName: "benchmark_tool",
				Content:  "invalid result",
			},
			compat.Message{
				Role:     compat.RoleTool,
				ToolID:   fmt.Sprintf("unknown-call-%d", i),
				ToolName: "benchmark_tool",
				Content:  "orphan result",
			},
		)
	}
	return messages
}
