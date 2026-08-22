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

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
)

type modelRetryCallbackBinder interface {
	WithModelRetryCallbacks(
		context.Context,
		func(context.Context, *compat.Request) (
			context.Context,
			*compat.Response,
			error,
		),
		func(context.Context, *compat.Request, *compat.Response) (
			context.Context,
			error,
		),
	) context.Context
}

func contextWithModelRetryCallbacks(
	ctx context.Context,
	flow *Flow,
	invocation *agent.Invocation,
	callModel compat.Model,
) context.Context {
	binder, ok := callModel.(modelRetryCallbackBinder)
	if ctx == nil || flow == nil || !ok {
		return ctx
	}
	return binder.WithModelRetryCallbacks(
		ctx,
		func(
			callbackCtx context.Context,
			req *compat.Request,
		) (context.Context, *compat.Response, error) {
			return flow.runBeforeModelCallbacks(
				callbackCtx,
				invocation,
				req,
			)
		},
		func(
			callbackCtx context.Context,
			req *compat.Request,
			resp *compat.Response,
		) (context.Context, error) {
			updatedCtx, _, err := flow.runAfterModelCallbacks(
				callbackCtx,
				invocation,
				req,
				resp,
			)
			return updatedCtx, err
		},
	)
}
