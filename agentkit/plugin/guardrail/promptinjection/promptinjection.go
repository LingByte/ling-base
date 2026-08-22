//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

// Package promptinjection provides a runner-scoped prompt injection guardrail plugin.
package promptinjection

import (
	"context"
	"fmt"

	log "github.com/LingByte/ling-base/common/logger"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/plugin"
	promptreview "github.com/LingByte/ling-base/agentkit/plugin/guardrail/promptinjection/review"
)

// Plugin is the prompt injection guardrail implementation.
type Plugin struct {
	name         string
	reviewer     promptreview.Reviewer
	tokenCounter compat.TokenCounter
}

// New creates a new prompt injection plugin.
func New(options ...Option) (*Plugin, error) {
	opts := newOptions(options...)
	if opts.reviewer == nil {
		return nil, fmt.Errorf("newing prompt injection plugin: reviewer is nil")
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
			log.ErrorfContext(ctx, "Prompt injection review denied: %v", err)
			return &compat.BeforeModelResult{CustomResponse: p.blockedResponse("")}, nil
		}
		if decision == nil {
			err = fmt.Errorf("prompt injection reviewer returned nil decision")
			log.ErrorfContext(ctx, "Prompt injection review denied: %v", err)
			return &compat.BeforeModelResult{CustomResponse: p.blockedResponse("")}, nil
		}
		if !decision.Blocked {
			return nil, nil
		}
		denyMessage := promptInjectionDenyMessage(decision)
		log.WarnContext(ctx, denyMessage)
		return &compat.BeforeModelResult{CustomResponse: p.blockedResponse(denyMessage)}, nil
	}
}

func (p *Plugin) blockedResponse(content string) *compat.Response {
	if content == "" {
		content = "The input was blocked by the prompt injection guardrail."
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

func promptInjectionDenyMessage(decision *promptreview.Decision) string {
	return fmt.Sprintf(
		"Prompt injection detected (category: %s): %s",
		decision.Category,
		decision.Reason,
	)
}
