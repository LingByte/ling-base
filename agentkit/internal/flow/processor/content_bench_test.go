//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
)

var contentRequestMessageCountSink int

func BenchmarkContentRequestProcessorProcessRequest(b *testing.B) {
	for _, historySize := range []int{16, 256, 1024} {
		b.Run(fmt.Sprintf("history=%d", historySize), func(b *testing.B) {
			ctx := context.Background()
			invocation := contentRequestBenchmarkInvocation(historySize)
			processor := NewContentRequestProcessor()
			events := make(chan *event.Event, 1)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				request := &compat.Request{
					Messages: []compat.Message{
						compat.NewSystemMessage("benchmark system prompt"),
					},
				}
				processor.ProcessRequest(ctx, invocation, request, events)
				contentRequestMessageCountSink = len(request.Messages)
				<-events
			}
		})
	}
}

func contentRequestBenchmarkInvocation(historySize int) *agent.Invocation {
	const agentName = "benchmark-agent"
	events := make([]event.Event, historySize)
	for i := range events {
		role := compat.RoleAssistant
		if i%2 == 0 {
			role = compat.RoleUser
		}
		events[i] = event.Event{
			ID:           fmt.Sprintf("event-%d", i),
			RequestID:    fmt.Sprintf("request-%d", i/2),
			InvocationID: fmt.Sprintf("history-invocation-%d", i/2),
			Author:       agentName,
			Branch:       agentName,
			Response: &compat.Response{
				Done: true,
				Choices: []compat.Choice{{
					Message: compat.Message{
						Role:    role,
						Content: fmt.Sprintf("history message %d", i),
					},
				}},
			},
		}
	}
	invocation := agent.NewInvocation(
		agent.WithInvocationID("benchmark-invocation"),
		agent.WithInvocationBranch(agentName),
		agent.WithInvocationEventFilterKey(agentName),
		agent.WithInvocationSession(&session.Session{Events: events}),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	invocation.AgentName = agentName
	return invocation
}
