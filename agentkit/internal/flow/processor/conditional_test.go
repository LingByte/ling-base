//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package processor

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

type conditionalRequestProcessorStub struct {
	called bool
}

func (p *conditionalRequestProcessorStub) ProcessRequest(
	ctx context.Context,
	invocation *agent.Invocation,
	req *compat.Request,
	ch chan<- *event.Event,
) {
	p.called = true
}

type conditionalResponseProcessorStub struct {
	called bool
}

func (p *conditionalResponseProcessorStub) ProcessResponse(
	ctx context.Context,
	invocation *agent.Invocation,
	req *compat.Request,
	rsp *compat.Response,
	ch chan<- *event.Event,
) {
	p.called = true
}

func TestConditionalRequestProcessor_ProcessRequest(t *testing.T) {
	delegate := &conditionalRequestProcessorStub{}
	processor := NewConditionalRequestProcessor(
		func(ctx context.Context, inv *agent.Invocation) bool {
			_ = ctx
			return inv != nil && inv.InvocationID == "allowed"
		},
		delegate,
	)

	processor.ProcessRequest(
		context.Background(),
		&agent.Invocation{InvocationID: "blocked"},
		&compat.Request{},
		make(chan *event.Event, 1),
	)
	if delegate.called {
		t.Fatalf("expected delegate to be skipped when predicate blocks invocation")
	}

	processor.ProcessRequest(
		context.Background(),
		&agent.Invocation{InvocationID: "allowed"},
		&compat.Request{},
		make(chan *event.Event, 1),
	)
	if !delegate.called {
		t.Fatalf("expected delegate to run when predicate allows invocation")
	}
}

func TestConditionalResponseProcessor_ProcessResponse(t *testing.T) {
	delegate := &conditionalResponseProcessorStub{}
	processor := NewConditionalResponseProcessor(
		func(ctx context.Context, inv *agent.Invocation) bool {
			_ = ctx
			return inv != nil && inv.InvocationID == "allowed"
		},
		delegate,
	)

	processor.ProcessResponse(
		context.Background(),
		&agent.Invocation{InvocationID: "blocked"},
		&compat.Request{},
		&compat.Response{},
		make(chan *event.Event, 1),
	)
	if delegate.called {
		t.Fatalf("expected delegate to be skipped when predicate blocks invocation")
	}

	processor.ProcessResponse(
		context.Background(),
		&agent.Invocation{InvocationID: "allowed"},
		&compat.Request{},
		&compat.Response{},
		make(chan *event.Event, 1),
	)
	if !delegate.called {
		t.Fatalf("expected delegate to run when predicate allows invocation")
	}
}
