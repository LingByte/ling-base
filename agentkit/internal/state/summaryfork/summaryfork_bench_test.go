//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package summaryfork

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
)

var summaryForkBenchmarkSink *compat.Request

func BenchmarkAttach(b *testing.B) {
	for _, historySize := range []int{16, 256, 1024} {
		b.Run(fmt.Sprintf("history=%d", historySize), func(b *testing.B) {
			invocation := agent.NewInvocation()
			request := summaryForkBenchmarkRequest(historySize)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Attach(invocation, request)
			}
		})
	}
}

func BenchmarkRequest(b *testing.B) {
	for _, historySize := range []int{16, 256, 1024} {
		b.Run(fmt.Sprintf("history=%d", historySize), func(b *testing.B) {
			invocation := agent.NewInvocation()
			Attach(invocation, summaryForkBenchmarkRequest(historySize))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				request, ok := Request(invocation)
				if !ok {
					b.Fatal("summary fork request is missing")
				}
				summaryForkBenchmarkSink = request
			}
		})
	}
}

func summaryForkBenchmarkRequest(historySize int) *compat.Request {
	messages := make([]compat.Message, historySize)
	for i := range messages {
		messages[i] = compat.Message{
			Role:    compat.RoleAssistant,
			Content: fmt.Sprintf("cache-safe history item %d", i),
			ToolCalls: []compat.ToolCall{{
				ID: fmt.Sprintf("call-%d", i),
				Function: compat.FunctionDefinitionParam{
					Name:      "benchmark_tool",
					Arguments: bytes.Repeat([]byte{'a' + byte(i%26)}, 64),
				},
			}},
		}
	}
	return &compat.Request{Messages: messages}
}
