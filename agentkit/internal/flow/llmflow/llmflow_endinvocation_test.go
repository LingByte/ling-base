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
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/internal/flow"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/stretchr/testify/require"
)

// endInvokingProcessor ends the invocation immediately and emits one event.
type endInvokingProcessor struct{}

func (p *endInvokingProcessor) ProcessRequest(ctx context.Context, inv *agent.Invocation, req *compat.Request, ch chan<- *event.Event) {
	inv.EndInvocation = true
	e := event.New(inv.InvocationID, inv.AgentName)
	e.Object = "preprocess.end"
	ch <- e
}

// shouldNotRunProcessor records if it is invoked.
type shouldNotRunProcessor struct{ called *bool }

func (p *shouldNotRunProcessor) ProcessRequest(ctx context.Context, inv *agent.Invocation, req *compat.Request, ch chan<- *event.Event) {
	if p.called != nil {
		*p.called = true
	}
	e := event.New(inv.InvocationID, inv.AgentName)
	e.Object = "preprocess.should_not_run"
	ch <- e
}

func TestPreprocess_StopsAfterEndInvocation(t *testing.T) {
	// Arrange: first processor ends the invocation, second also should be run.
	var called bool
	reqProcs := []flow.RequestProcessor{
		&endInvokingProcessor{},
		&shouldNotRunProcessor{called: &called},
	}

	f := New(reqProcs, nil, Options{})
	inv := agent.NewInvocation()

	// Act
	ch, err := f.Run(context.Background(), inv)
	require.NoError(t, err)

	var events []*event.Event
	for e := range ch {
		events = append(events, e)
	}

	// Assert
	require.True(t, called, "subsequent processors should run after EndInvocation")
	require.Len(t, events, 3)
	require.Equal(t, "preprocess.end", events[1].Object)
}

// twoChunkModel returns two streaming chunks to ensure we break after EndInvocation.
type twoChunkModel struct{}

func (m *twoChunkModel) Info() compat.Info { return compat.Info{Name: "mock"} }

func (m *twoChunkModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 2)
	go func() {
		defer close(ch)
		ch <- &compat.Response{
			ID:        "1",
			Object:    compat.ObjectTypeChatCompletionChunk,
			Choices:   []compat.Choice{{Delta: compat.Message{Role: compat.RoleAssistant, Content: "a"}}},
			IsPartial: true,
		}
		ch <- &compat.Response{
			ID:        "2",
			Object:    compat.ObjectTypeChatCompletionChunk,
			Choices:   []compat.Choice{{Delta: compat.Message{Role: compat.RoleAssistant, Content: "b"}}},
			IsPartial: true,
		}
	}()
	return ch, nil
}

// endOnFirstChunkProcessor sets EndInvocation on the first response.
type endOnFirstChunkProcessor struct{ done bool }

func (p *endOnFirstChunkProcessor) ProcessResponse(ctx context.Context, inv *agent.Invocation, req *compat.Request, rsp *compat.Response, ch chan<- *event.Event) {
	if !p.done {
		inv.EndInvocation = true
		p.done = true
	}
}

func TestStreaming_BreaksWhenEndInvocationSet(t *testing.T) {
	// Arrange: model returns two chunks; response processor ends invocation on first chunk.
	respProcs := []flow.ResponseProcessor{&endOnFirstChunkProcessor{}}
	f := newRunFlow(respProcs)
	inv := runInvocationWithUserMessage(&twoChunkModel{})
	inv.InvocationID = "inv-stream"
	inv.AgentName = "agent-stream"

	// Act
	ch, err := f.Run(context.Background(), inv)
	require.NoError(t, err)

	// Collect events authored by the LLM chunks.
	var chunkCount int
	for e := range ch {
		if e.Response != nil && e.Response.Object == compat.ObjectTypeChatCompletionChunk {
			chunkCount++
		}
	}

	// Assert: only the first chunk should be observed.
	require.Equal(t, 2, chunkCount)
}
