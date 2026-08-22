//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package llmflow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/agent"
	iflow "github.com/LingByte/ling-base/agentkit/internal/flow"
	"github.com/LingByte/ling-base/agentkit/internal/flow/processor"
	"github.com/LingByte/ling-base/agentkit/internal/state/summaryview"
	compat "github.com/LingByte/ling-base/relay/compat"
)

type fixedSummaryViewTokenCounter struct {
	tokens int
	err    error
}

func (c fixedSummaryViewTokenCounter) CountTokens(
	context.Context,
	compat.Message,
) (int, error) {
	return c.tokens, c.err
}

func (c fixedSummaryViewTokenCounter) CountTokensRange(
	context.Context,
	[]compat.Message,
	int,
	int,
) (int, error) {
	return c.tokens, c.err
}

func TestFinalizeSummaryViewUsesFinalRequest(t *testing.T) {
	invocation := agent.NewInvocation()
	summaryview.AttachProjection(invocation, &summaryview.View{
		ContentRequestLength: 1,
		Items: []summaryview.Item{{
			Message:      compat.NewUserMessage("visible"),
			RequestIndex: 0,
		}},
	})
	request := &compat.Request{Messages: []compat.Message{
		compat.NewSystemMessage("added after content processing"),
		compat.NewUserMessage("visible"),
	}}
	counter := fixedSummaryViewTokenCounter{tokens: 42}
	flow := &Flow{requestProcessors: []iflow.RequestProcessor{
		processor.NewContentRequestProcessor(
			processor.WithContextCompactionTokenCounter(counter),
		),
	}}

	finalizeSummaryView(
		context.Background(),
		invocation,
		request,
		flow.summaryViewTokenCounter(),
	)

	view, ok := summaryview.Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Equal(t, 42, view.RequestTokens)
	require.Equal(t, 1, view.Items[0].RequestIndex)
}

func TestFinalizeSummaryViewHandlesUnavailableInputs(t *testing.T) {
	counter := fixedSummaryViewTokenCounter{tokens: 42}
	require.NotPanics(t, func() {
		finalizeSummaryView(context.Background(), nil, &compat.Request{}, counter)
	})

	nilRequest := agent.NewInvocation()
	finalizeSummaryView(context.Background(), nilRequest, nil, counter)
	_, ok := summaryview.Snapshot(nilRequest)
	require.False(t, ok)

	missingProjection := agent.NewInvocation()
	finalizeSummaryView(
		context.Background(),
		missingProjection,
		&compat.Request{Messages: []compat.Message{
			compat.NewUserMessage("visible"),
		}},
		counter,
	)
	_, ok = summaryview.Snapshot(missingProjection)
	require.False(t, ok)
}

func TestFinalizeSummaryViewLeavesProjectionOnCountFailure(t *testing.T) {
	invocation := agent.NewInvocation()
	summaryview.AttachProjection(invocation, &summaryview.View{
		ContentRequestLength: 1,
		Items: []summaryview.Item{{
			Message:      compat.NewUserMessage("visible"),
			RequestIndex: 0,
		}},
	})

	finalizeSummaryView(
		context.Background(),
		invocation,
		&compat.Request{Messages: []compat.Message{
			compat.NewUserMessage("visible"),
		}},
		fixedSummaryViewTokenCounter{err: errors.New("count failed")},
	)

	view, ok := summaryview.Snapshot(invocation)
	require.True(t, ok)
	require.False(t, view.Bound)
	require.Zero(t, view.RequestTokens)
}

func TestFinalizeSummaryViewUsesDefaultCounter(t *testing.T) {
	invocation := agent.NewInvocation()
	summaryview.AttachProjection(invocation, &summaryview.View{
		ContentRequestLength: 1,
		Items: []summaryview.Item{{
			Message:      compat.NewUserMessage("visible"),
			RequestIndex: 0,
		}},
	})

	finalizeSummaryView(
		context.Background(),
		invocation,
		&compat.Request{Messages: []compat.Message{
			compat.NewUserMessage("visible"),
		}},
		nil,
	)

	view, ok := summaryview.Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Positive(t, view.RequestTokens)
}
