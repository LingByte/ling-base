//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package llmflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/internal/state/summaryview"
	itelemetry "github.com/LingByte/ling-base/agentkit/internal/telemetry"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
)

var (
	// benchCountSink and benchRespSink prevent the compiler from optimizing away the benchmarked work.
	benchRespSink  *compat.Response
	benchCountSink int
)

type benchChanModel struct {
	responses []*compat.Response
}

func (m *benchChanModel) GenerateContent(ctx context.Context, request *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response)
	go func() {
		for _, resp := range m.responses {
			ch <- resp
		}
		close(ch)
	}()
	return ch, nil
}

func (m *benchChanModel) Info() compat.Info {
	return compat.Info{Name: "benchChanModel"}
}

type benchIterModel struct {
	responses []*compat.Response
}

func (m *benchIterModel) GenerateContent(ctx context.Context, request *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response)
	close(ch)
	return ch, nil
}

func (m *benchIterModel) GenerateContentIter(ctx context.Context, request *compat.Request) (compat.Seq[*compat.Response], error) {
	return func(yield func(*compat.Response) bool) {
		for _, resp := range m.responses {
			if !yield(resp) {
				return
			}
		}
	}, nil
}

func (m *benchIterModel) Info() compat.Info {
	return compat.Info{Name: "benchIterModel"}
}

func makeBenchResponses(n int) []*compat.Response {
	responses := make([]*compat.Response, n)
	for i := 0; i < n; i++ {
		responses[i] = &compat.Response{Created: int64(i + 1)}
	}
	return responses
}

func consumeSeq(seq compat.Seq[*compat.Response]) (checksum int, last *compat.Response) {
	seq(func(resp *compat.Response) bool {
		checksum += int(resp.Created)
		last = resp
		return true
	})
	return checksum, last
}

func BenchmarkGenerateContentSeq(b *testing.B) {
	ctx := context.Background()
	f := new(Flow)
	invocation := &agent.Invocation{AgentName: "bench"}
	request := &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("benchmark request")},
	}

	for _, n := range []int{1, 16, 256, 1024} {
		b.Run(fmt.Sprintf("Channel/n=%d", n), func(b *testing.B) {
			invocation.Model = &benchChanModel{responses: makeBenchResponses(n)}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				seq, err := f.generateContentSeq(ctx, invocation, request, invocation.Model)
				if err != nil {
					b.Fatal(err)
				}
				benchCountSink, benchRespSink = consumeSeq(seq)
			}
		})

		b.Run(fmt.Sprintf("Iter/n=%d", n), func(b *testing.B) {
			invocation.Model = &benchIterModel{responses: makeBenchResponses(n)}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				seq, err := f.generateContentSeq(ctx, invocation, request, invocation.Model)
				if err != nil {
					b.Fatal(err)
				}
				benchCountSink, benchRespSink = consumeSeq(seq)
			}
		})
	}
}

func BenchmarkStreamingResponseProcessorUpdateMetricsState(b *testing.B) {
	for _, itemCount := range []int{16, 256} {
		b.Run(fmt.Sprintf("SummaryItems/%d", itemCount), func(b *testing.B) {
			base := &agent.Invocation{
				InvocationID: "invocation-id",
				AgentName:    "bench-agent",
				Session: &session.Session{
					ID:      "session-id",
					UserID:  "user-id",
					AppName: "app-name",
				},
			}
			items := make([]summaryview.Item, itemCount)
			for i := range items {
				items[i] = summaryview.Item{
					Message: compat.Message{
						Role:    compat.RoleUser,
						Content: "benchmark model-visible history",
					},
					EffectiveEvent: event.Event{
						ID: fmt.Sprintf("event-%d", i),
						StateDelta: map[string][]byte{
							"benchmark": make([]byte, 1024),
						},
					},
					RequestIndex: i,
				}
			}
			summaryview.AttachProjection(base, &summaryview.View{
				SessionID: "session-id",
				Items:     items,
			})
			current := &agent.Invocation{
				Session: &session.Session{
					ID:      "updated-session-id",
					UserID:  "updated-user-id",
					AppName: "updated-app-name",
				},
			}
			processor := &streamingResponseProcessor{
				currentInvocation:       current,
				observabilityInvocation: base,
				tracker: itelemetry.NewChatMetricsTracker(
					context.Background(),
					base,
					&compat.Request{},
					nil,
					nil,
					nil,
				),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processor.updateMetricsState()
			}
		})
	}
}
