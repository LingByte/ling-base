//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package llmflow

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/internal/flow/calllimit"
	imodelrequest "github.com/LingByte/ling-base/agentkit/internal/modelrequest"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/require"
)

type modelRetryTestContextKey struct{}
type modelRetryTestCallbacksKey struct{}

type modelRetryTestCallbacks struct {
	before func(context.Context, *compat.Request) (
		context.Context,
		*compat.Response,
		error,
	)
	after func(context.Context, *compat.Request, *compat.Response) (
		context.Context,
		error,
	)
}

type modelRetryTestBinder struct{}

func (modelRetryTestBinder) GenerateContent(
	context.Context,
	*compat.Request,
) (<-chan *compat.Response, error) {
	return nil, nil
}

func (modelRetryTestBinder) Info() compat.Info {
	return compat.Info{Name: "retry-test-binder"}
}

func (modelRetryTestBinder) WithModelRetryCallbacks(
	ctx context.Context,
	before func(context.Context, *compat.Request) (
		context.Context,
		*compat.Response,
		error,
	),
	after func(context.Context, *compat.Request, *compat.Response) (
		context.Context,
		error,
	),
) context.Context {
	return context.WithValue(ctx, modelRetryTestCallbacksKey{},
		modelRetryTestCallbacks{before: before, after: after})
}

func TestModelRetryCallbacks_RunNormalCallbackChain(t *testing.T) {
	const contextValue = "retry-callback-context"
	var beforeCalls int
	var afterCalls int
	callbacks := compat.NewCallbacks().
		RegisterBeforeModel(func(
			ctx context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			beforeCalls++
			require.Equal(t, "retry", args.Request.Messages[0].Content)
			return &compat.BeforeModelResult{Context: context.WithValue(
				ctx,
				modelRetryTestContextKey{},
				contextValue,
			)}, nil
		}).
		RegisterAfterModel(func(
			ctx context.Context,
			args *compat.AfterModelArgs,
		) (*compat.AfterModelResult, error) {
			afterCalls++
			require.Equal(t, contextValue, ctx.Value(modelRetryTestContextKey{}))
			require.Equal(t, "retry", args.Request.Messages[0].Content)
			require.Equal(t, "response", args.Response.ID)
			return nil, nil
		})
	flow := New(nil, nil, Options{ModelCallbacks: callbacks})
	ctx := contextWithModelRetryCallbacks(
		context.Background(),
		flow,
		agent.NewInvocation(),
		modelRetryTestBinder{},
	)
	bound, ok := ctx.Value(modelRetryTestCallbacksKey{}).(modelRetryTestCallbacks)
	require.True(t, ok)
	req := &compat.Request{Messages: []compat.Message{
		compat.NewUserMessage("retry"),
	}}

	ctx, customResponse, err := bound.before(ctx, req)
	require.NoError(t, err)
	require.Nil(t, customResponse)
	ctx, err = bound.after(ctx, req, &compat.Response{ID: "response"})
	require.NoError(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, 1, beforeCalls)
	require.Equal(t, 1, afterCalls)
}

func TestModelRetryCallbacks_FinalizationRemainsToolFree(t *testing.T) {
	const instruction = "finish with available results"
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			ctx context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			require.True(t, imodelrequest.ToolsDisabled(ctx))
			require.Nil(t, args.Request.Tools)
			require.NotContains(t, args.Request.ExtraFields, "tool_choice")

			args.Request.Tools = map[string]tool.Tool{"lookup": nil}
			args.Request.ExtraFields["tool_choice"] = "required"
			return &compat.BeforeModelResult{
				Context: context.Background(),
			}, nil
		},
	)
	flow := New(nil, nil, Options{ModelCallbacks: callbacks})
	invocation := agent.NewInvocation()
	toolInstruction := instruction
	calllimit.Configure(invocation, nil, &toolInstruction)
	calllimit.ScheduleToolFinalization(invocation)
	_, finalizing := calllimit.ActivateForLLM(invocation, false)
	require.True(t, finalizing)

	ctx := contextWithModelRetryCallbacks(
		context.Background(),
		flow,
		invocation,
		modelRetryTestBinder{},
	)
	bound, ok := ctx.Value(modelRetryTestCallbacksKey{}).(modelRetryTestCallbacks)
	require.True(t, ok)
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("question"),
			compat.NewUserMessage(instruction),
		},
		Tools: map[string]tool.Tool{"lookup": nil},
		ExtraFields: map[string]any{
			"tool_choice": "required",
			"keep":        "value",
		},
	}

	ctx, customResponse, err := bound.before(ctx, req)

	require.NoError(t, err)
	require.Nil(t, customResponse)
	require.True(t, imodelrequest.ToolsDisabled(ctx))
	require.Nil(t, req.Tools)
	require.NotContains(t, req.ExtraFields, "tool_choice")
	require.Equal(t, "value", req.ExtraFields["keep"])
	require.Equal(t, instruction, req.Messages[len(req.Messages)-1].Content)
}

func TestModelRetryCallbacks_UnsupportedModel(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, ctx, contextWithModelRetryCallbacks(
		ctx,
		New(nil, nil, Options{}),
		agent.NewInvocation(),
		&mockModel{},
	))
}
