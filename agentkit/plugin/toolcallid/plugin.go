//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

// Package toolcallid provides a plugin that canonicalizes final tool call IDs.
package toolcallid

import (
	"context"
	"fmt"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
	pluginbase "github.com/LingByte/ling-base/agentkit/plugin"
)

const defaultPluginName = "tool_call_id"

type plugin struct {
	name string
}

// New creates a ToolCall ID plugin.
func New() pluginbase.Plugin {
	return newPlugin()
}

func newPlugin() *plugin {
	return &plugin{name: defaultPluginName}
}

// Name implements plugin.Plugin.
func (p *plugin) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Register implements plugin.Plugin.
func (p *plugin) Register(r *pluginbase.Registry) {
	if p == nil || r == nil {
		return
	}
	r.AfterModel(p.afterModel)
}

func (p *plugin) afterModel(
	ctx context.Context,
	args *compat.AfterModelArgs,
) (*compat.AfterModelResult, error) {
	if args == nil || args.Response == nil {
		return nil, nil
	}
	inv, _ := agent.InvocationFromContext(ctx)
	canonicalized, err := canonicalizeResponse(inv, args.Response)
	if err != nil {
		return nil, fmt.Errorf("canonicalize response: %w", err)
	}
	if canonicalized == nil {
		return nil, nil
	}
	*args.Response = *canonicalized
	return nil, nil
}
