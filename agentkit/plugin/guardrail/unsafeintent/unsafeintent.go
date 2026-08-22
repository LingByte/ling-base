//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

// Package unsafeintent provides a runner-scoped unsafe intent guardrail plugin.
package unsafeintent

import (
	"context"
	"fmt"

	log "github.com/LingByte/ling-base/common/logger"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/plugin"
	unsafereview "github.com/LingByte/ling-base/agentkit/plugin/guardrail/unsafeintent/review"
)

// Plugin is the unsafe intent guardrail implementation.
type Plugin struct {
	name         string
	reviewer     unsafereview.Reviewer
	tokenCounter compat.TokenCounter
}

// New creates a new unsafe intent plugin.
func New(options ...Option) (*Plugin, error) {
	opts := newOptions(options...)
	if opts.reviewer == nil {
		return nil, fmt.Errorf("newing unsafe intent plugin: reviewer is nil")
	}
	return &Plugin{
		name:         opts.name,
		reviewer:     opts.reviewer,
		tokenCounter: compat.NewSimpleTokenCounter(),
	}, nil
}

// Name implements plugin.Plugin.
func (p *Plugin) Name() string {
	return p.name
}

// Register implements plugin.Plugin.
func (p *Plugin) Register(r *plugin.Registry) {
	if p == nil || r == nil {
		return
	}
	r.BeforeModel(p.beforeModel())
}

func (p *Plugin) beforeModel() compat.BeforeModelCallbackStructured {
	return func(ctx context.Context, args *compat.BeforeModelArgs) (*compat.BeforeModelResult, error) {
		if p == nil || args == nil || args.Request == nil {
			return nil, nil
		}
		req := p.buildReviewRequest(ctx, args.Request.Messages)
		if req == nil {
			return nil, nil
		}
		decision, err := p.reviewer.Review(ctx, req)
		if err != nil {
			log.ErrorfContext(ctx, "Unsafe intent review denied: %v", err)
			return &compat.BeforeModelResult{CustomResponse: p.blockedResponse("")}, nil
		}
		if decision == nil {
			err = fmt.Errorf("unsafe intent reviewer returned nil decision")
			log.ErrorfContext(ctx, "Unsafe intent review denied: %v", err)
			return &compat.BeforeModelResult{CustomResponse: p.blockedResponse("")}, nil
		}
		if !decision.Blocked {
			return nil, nil
		}
		denyMessage := unsafeIntentDenyMessage(decision)
		log.WarnContext(ctx, denyMessage)
		return &compat.BeforeModelResult{CustomResponse: p.blockedResponse(denyMessage)}, nil
	}
}

func (p *Plugin) blockedResponse(content string) *compat.Response {
	if content == "" {
		content = "The input was blocked by the unsafe intent guardrail."
	}
	return &compat.Response{
		Object: compat.ObjectTypeChatCompletion,
		Done:   true,
		Choices: []compat.Choice{{
			Index:   0,
			Message: compat.NewAssistantMessage(content),
		}},
	}
}

func unsafeIntentDenyMessage(decision *unsafereview.Decision) string {
	return fmt.Sprintf(
		"Unsafe intent detected (category: %s): %s",
		decision.Category,
		decision.Reason,
	)
}
