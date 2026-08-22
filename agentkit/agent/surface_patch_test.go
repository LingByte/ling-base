//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package agent

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/agentkit/internal/surfacepatch"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/skill"
	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/require"
)

type surfacePatchTestTool struct {
	name string
}

func (t surfacePatchTestTool) Declaration() *tool.Declaration {
	return &tool.Declaration{Name: t.name}
}

func (surfacePatchTestTool) Call(context.Context, []byte) (any, error) {
	return nil, nil
}

type surfacePatchTestRepo struct{}

func (surfacePatchTestRepo) Summaries() []skill.Summary {
	return []skill.Summary{{Name: "demo"}}
}

func (surfacePatchTestRepo) Get(name string) (*skill.Skill, error) {
	return &skill.Skill{Summary: skill.Summary{Name: name}}, nil
}

func (surfacePatchTestRepo) Path(string) (string, error) {
	return "", nil
}

type surfacePatchTestModel struct {
	name string
}

func (m *surfacePatchTestModel) GenerateContent(
	context.Context,
	*compat.Request,
) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response)
	close(ch)
	return ch, nil
}

func (m *surfacePatchTestModel) Info() compat.Info {
	return compat.Info{Name: m.name}
}

func TestWithSurfacePatchForNode_MergesAndCopiesByValue(t *testing.T) {
	var first SurfacePatch
	first.SetInstruction("first instruction")

	var second SurfacePatch
	second.SetGlobalInstruction("global instruction")

	var third SurfacePatch
	third.SetInstruction("second instruction")

	opts := NewRunOptions(
		WithSurfacePatchForNode("root", first),
		WithSurfacePatchForNode("root", second),
		WithSurfacePatchForNode("root", third),
	)

	patch, ok := surfacepatch.PatchForNode(opts.CustomAgentConfigs, "root")
	require.True(t, ok)

	instruction, ok := patch.Instruction()
	require.True(t, ok)
	require.Equal(t, "second instruction", instruction)

	globalInstruction, ok := patch.GlobalInstruction()
	require.True(t, ok)
	require.Equal(t, "global instruction", globalInstruction)

	first.SetInstruction("mutated")
	patch, ok = surfacepatch.PatchForNode(opts.CustomAgentConfigs, "root")
	require.True(t, ok)

	instruction, ok = patch.Instruction()
	require.True(t, ok)
	require.Equal(t, "second instruction", instruction)
}

func TestSurfacePatch_Setters_ApplyAllSupportedSurfaces(t *testing.T) {
	modelValue := &surfacePatchTestModel{name: "patched"}
	repoValue := surfacePatchTestRepo{}
	var patch SurfacePatch
	patch.SetInstruction("instruction")
	patch.SetGlobalInstruction("global")
	patch.SetFewShot([][]compat.Message{{
		compat.NewUserMessage("few-shot user"),
		compat.NewAssistantMessage("few-shot assistant"),
	}})
	patch.SetModel(modelValue)
	patch.SetTools([]tool.Tool{surfacePatchTestTool{name: "tool_one"}})
	patch.AppendTools([]tool.Tool{surfacePatchTestTool{name: "tool_two"}})
	patch.SetSkillRepository(repoValue)
	patch.SetSuppressSubAgentTransfer()
	opts := NewRunOptions(WithSurfacePatchForNode("root", patch))
	stored, ok := surfacepatch.PatchForNode(opts.CustomAgentConfigs, "root")
	require.True(t, ok)
	instruction, ok := stored.Instruction()
	require.True(t, ok)
	require.Equal(t, "instruction", instruction)
	globalInstruction, ok := stored.GlobalInstruction()
	require.True(t, ok)
	require.Equal(t, "global", globalInstruction)
	fewShot, ok := stored.FewShot()
	require.True(t, ok)
	require.Len(t, fewShot, 1)
	require.Equal(t, "few-shot user", fewShot[0][0].Content)
	gotModel, ok := stored.Model()
	require.True(t, ok)
	require.Equal(t, modelValue, gotModel)
	gotTools, ok := stored.Tools()
	require.True(t, ok)
	require.Len(t, gotTools, 2)
	require.Equal(t, "tool_one", gotTools[0].Declaration().Name)
	require.Equal(t, "tool_two", gotTools[1].Declaration().Name)
	gotRepo, ok := stored.SkillRepository()
	require.True(t, ok)
	require.NotNil(t, gotRepo)
	require.True(t, stored.SuppressSubAgentTransfer())
}

func TestWithSurfacePatchForNode_IgnoresEmptyInputs(t *testing.T) {
	require.NotPanics(t, func() {
		WithSurfacePatchForNode("root", SurfacePatch{})(nil)
	})
	opts := NewRunOptions()
	WithSurfacePatchForNode("", SurfacePatch{})(&opts)
	require.Nil(t, opts.CustomAgentConfigs)
	var patch SurfacePatch
	patch.SetInstruction("instruction")
	WithSurfacePatchForNode("", patch)(&opts)
	require.Nil(t, opts.CustomAgentConfigs)
	WithSurfacePatchForNode("root", SurfacePatch{})(&opts)
	require.Nil(t, opts.CustomAgentConfigs)
}
