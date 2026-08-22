//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package messagemerger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	compat "github.com/LingByte/ling-base/relay/compat"
	rootplugin "github.com/LingByte/ling-base/agentkit/plugin"
	"github.com/LingByte/ling-base/agentkit/plugin/messagemerger"
)

func TestPlugin_MergesSupportedRoles(t *testing.T) {
	p := messagemerger.New()
	m := rootplugin.MustNewManager(p)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("policy"),
			{
				Role: compat.RoleSystem,
				ContentParts: []compat.ContentPart{{
					Type: compat.ContentTypeText,
					Text: compat.StringPtr("extra system"),
				}},
			},
			compat.NewUserMessage("hello"),
			{
				Role: compat.RoleUser,
				ContentParts: []compat.ContentPart{{
					Type: compat.ContentTypeText,
					Text: compat.StringPtr("extra user"),
				}},
			},
			{
				Role:             compat.RoleAssistant,
				Content:          "first answer",
				ReasoningContent: "first reasoning",
				ToolCalls: []compat.ToolCall{{
					Type: "function",
					ID:   "call_1",
					Function: compat.FunctionDefinitionParam{
						Name:      "search",
						Arguments: []byte(`{"q":"weather"}`),
					},
				}},
			},
			{
				Role:             compat.RoleAssistant,
				Content:          "second answer",
				ReasoningContent: "second reasoning",
			},
			compat.NewToolMessage("call_1", "search", "search result"),
			compat.NewToolMessage("call_2", "lookup", "lookup result"),
			compat.NewUserMessage("tail"),
		},
	}
	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 6)
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.Empty(t, req.Messages[0].Content)
	require.Len(t, req.Messages[0].ContentParts, 3)
	require.NotNil(t, req.Messages[0].ContentParts[0].Text)
	require.Equal(t, "policy", *req.Messages[0].ContentParts[0].Text)
	require.NotNil(t, req.Messages[0].ContentParts[1].Text)
	require.Equal(t, "\n\n", *req.Messages[0].ContentParts[1].Text)
	require.NotNil(t, req.Messages[0].ContentParts[2].Text)
	require.Equal(t, "extra system", *req.Messages[0].ContentParts[2].Text)
	require.Equal(t, compat.RoleUser, req.Messages[1].Role)
	require.Empty(t, req.Messages[1].Content)
	require.Len(t, req.Messages[1].ContentParts, 3)
	require.NotNil(t, req.Messages[1].ContentParts[0].Text)
	require.Equal(t, "hello", *req.Messages[1].ContentParts[0].Text)
	require.NotNil(t, req.Messages[1].ContentParts[1].Text)
	require.Equal(t, "\n\n", *req.Messages[1].ContentParts[1].Text)
	require.NotNil(t, req.Messages[1].ContentParts[2].Text)
	require.Equal(t, "extra user", *req.Messages[1].ContentParts[2].Text)
	require.Equal(t, compat.RoleAssistant, req.Messages[2].Role)
	require.Equal(
		t,
		"first answer\n\nsecond answer",
		req.Messages[2].Content,
	)
	require.Equal(
		t,
		"first reasoning\n\nsecond reasoning",
		req.Messages[2].ReasoningContent,
	)
	require.Len(t, req.Messages[2].ToolCalls, 1)
	require.Equal(t, "call_1", req.Messages[2].ToolCalls[0].ID)
	require.Equal(t, compat.RoleTool, req.Messages[3].Role)
	require.Equal(t, "call_1", req.Messages[3].ToolID)
	require.Equal(t, compat.RoleTool, req.Messages[4].Role)
	require.Equal(t, "call_2", req.Messages[4].ToolID)
	require.Equal(t, compat.RoleUser, req.Messages[5].Role)
	require.Equal(t, "tail", req.Messages[5].Content)
}

func TestPlugin_DoesNotMergeToolMessages(t *testing.T) {
	p := messagemerger.New()
	m := rootplugin.MustNewManager(p)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("question"),
			compat.NewAssistantMessage("calling"),
			compat.NewToolMessage("call_1", "search", "result one"),
			compat.NewToolMessage("call_2", "lookup", "result two"),
		},
	}
	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 4)
	require.Equal(t, compat.RoleTool, req.Messages[2].Role)
	require.Equal(t, "call_1", req.Messages[2].ToolID)
	require.Equal(t, compat.RoleTool, req.Messages[3].Role)
	require.Equal(t, "call_2", req.Messages[3].ToolID)
}

func TestPlugin_NilRequestIsSafe(t *testing.T) {
	p := messagemerger.New()
	m := rootplugin.MustNewManager(p)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)
	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{},
	)
	require.NoError(t, err)
}

func TestNew_DefaultName(t *testing.T) {
	got := messagemerger.New()
	require.Equal(t, "consecutive_message_merger", got.Name())
}

func TestNew_WithName(t *testing.T) {
	got := messagemerger.New(messagemerger.WithName("custom_merger"))
	require.Equal(t, "custom_merger", got.Name())
}

func TestNew_EmptySeparatorOmitsJoinText(t *testing.T) {
	p := messagemerger.New(messagemerger.WithSeparator(""))
	m := rootplugin.MustNewManager(p)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("foo"),
			compat.NewUserMessage("bar"),
		},
	}
	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 1)
	require.Equal(t, "foobar", req.Messages[0].Content)
}
