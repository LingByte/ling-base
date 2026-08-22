//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package agent

import (
	"github.com/LingByte/ling-base/agentkit/internal/surfacepatch"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/skill"
	"github.com/LingByte/ling-base/agentkit/tool"
)

// SurfacePatch represents one node's runtime surface overrides.
type SurfacePatch struct {
	patch surfacepatch.Patch
}

// SetInstruction sets the instruction surface override.
func (p *SurfacePatch) SetInstruction(text string) {
	p.patch.SetInstruction(text)
}

// SetGlobalInstruction sets the global instruction surface override.
func (p *SurfacePatch) SetGlobalInstruction(text string) {
	p.patch.SetGlobalInstruction(text)
}

// SetFewShot sets the few-shot surface override.
func (p *SurfacePatch) SetFewShot(examples [][]compat.Message) {
	p.patch.SetFewShot(examples)
}

// SetModel sets the model surface override.
func (p *SurfacePatch) SetModel(m compat.Model) {
	p.patch.SetModel(m)
}

// SetTools sets the tool surface override and clears appended tools.
func (p *SurfacePatch) SetTools(tools []tool.Tool) {
	p.patch.SetTools(tools)
}

// AppendTools appends tools to the node's runtime tool surface.
func (p *SurfacePatch) AppendTools(tools []tool.Tool) {
	p.patch.AppendTools(tools)
}

// SetSkillRepository sets the skill repository surface override.
func (p *SurfacePatch) SetSkillRepository(repo skill.Repository) {
	p.patch.SetSkillRepository(repo)
}

// SetSuppressSubAgentTransfer omits framework-managed sub-agent transfer
// (transfer_to_agent) from the node's tool surface even when the node's agent
// has sub-agents.
func (p *SurfacePatch) SetSuppressSubAgentTransfer() {
	p.patch.SetSuppressSubAgentTransfer()
}

// WithSurfacePatchForNode applies one node's runtime surface overrides to this run.
func WithSurfacePatchForNode(nodeID string, patch SurfacePatch) RunOption {
	return func(opts *RunOptions) {
		if opts == nil || nodeID == "" || patch.patch.IsEmpty() {
			return
		}
		opts.CustomAgentConfigs = surfacepatch.WithPatch(
			opts.CustomAgentConfigs,
			nodeID,
			patch.patch,
		)
	}
}
