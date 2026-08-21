//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

// Package agent provides internal helpers for working with agent invocations.
package agent

import (
	publicagent "github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/model"
)

// ResolveInvokeAgentStream resolves the effective stream setting for an agent invocation.
func ResolveInvokeAgentStream(invocation *publicagent.Invocation, genCfg *model.GenerationConfig) bool {
	if invocation != nil && invocation.RunOptions.Stream != nil {
		return *invocation.RunOptions.Stream
	}
	if genCfg != nil {
		return genCfg.Stream
	}
	return false
}
