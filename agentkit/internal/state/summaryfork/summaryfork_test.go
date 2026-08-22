//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package summaryfork

import (
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/require"
)

type stubTool struct {
	name string
}

func (t stubTool) Declaration() *tool.Declaration {
	return &tool.Declaration{Name: t.name}
}

func TestAttachSnapshotsRequest(t *testing.T) {
	text := "part"
	index := 1
	maxTokens := 10
	req := &compat.Request{
		Messages: []compat.Message{{
			Role: compat.RoleAssistant,
			ContentParts: []compat.ContentPart{{
				Type: compat.ContentTypeText,
				Text: &text,
			}, {
				Type: compat.ContentTypeVideo,
				Video: &compat.Video{
					Data:   []byte("video"),
					Format: "mp4",
				},
			}},
			ToolCalls: []compat.ToolCall{{
				Index: &index,
				Function: compat.FunctionDefinitionParam{
					Arguments: []byte(`{"q":"original"}`),
				},
				ExtraFields: map[string]any{
					"nested": map[string]any{"k": "v"},
				},
			}},
		}},
		GenerationConfig: compat.GenerationConfig{
			MaxTokens: &maxTokens,
			Stop:      []string{"END"},
		},
		StructuredOutput: &compat.StructuredOutput{
			Type: compat.StructuredOutputJSONSchema,
			JSONSchema: &compat.JSONSchemaConfig{
				Schema: map[string]any{"type": "object"},
			},
		},
		ExtraFields: map[string]any{
			"metadata": map[string]any{"id": "one"},
		},
		Headers: map[string]string{"X-Trace": "one"},
	}

	inv := agent.NewInvocation()
	Attach(inv, req)

	*req.Messages[0].ContentParts[0].Text = "mutated"
	req.Messages[0].ContentParts[1].Video.Data[0] = 'V'
	req.Messages[0].ToolCalls[0].Function.Arguments[0] = '['
	req.Messages[0].ToolCalls[0].ExtraFields["nested"].(map[string]any)["k"] = "changed"
	*req.GenerationConfig.MaxTokens = 20
	req.GenerationConfig.Stop[0] = "STOP"
	req.StructuredOutput.JSONSchema.Schema["type"] = "array"
	req.ExtraFields["metadata"].(map[string]any)["id"] = "two"
	req.Headers["X-Trace"] = "two"

	got, ok := Request(inv)
	require.True(t, ok)
	require.Equal(t, "part", *got.Messages[0].ContentParts[0].Text)
	require.Equal(t, []byte("video"), got.Messages[0].ContentParts[1].Video.Data)
	require.Equal(t, byte('{'), got.Messages[0].ToolCalls[0].Function.Arguments[0])
	require.Equal(t, "v", got.Messages[0].ToolCalls[0].ExtraFields["nested"].(map[string]any)["k"])
	require.Equal(t, 10, *got.GenerationConfig.MaxTokens)
	require.Equal(t, "END", got.GenerationConfig.Stop[0])
	require.Equal(t, "object", got.StructuredOutput.JSONSchema.Schema["type"])
	require.Equal(t, "one", got.ExtraFields["metadata"].(map[string]any)["id"])
	require.Equal(t, "one", got.Headers["X-Trace"])

	got.GenerationConfig.Stop[0] = "again"
	*got.GenerationConfig.MaxTokens = 30
	again, ok := Request(inv)
	require.True(t, ok)
	require.Equal(t, 10, *again.GenerationConfig.MaxTokens)
	require.Equal(t, "END", again.GenerationConfig.Stop[0])
}

func TestAttachHandlesNilAndZeroValueRequest(t *testing.T) {
	Attach(nil, &compat.Request{})
	Attach(agent.NewInvocation(), nil)

	inv := agent.NewInvocation()
	_, ok := Request(inv)
	require.False(t, ok)

	inv.SetState(stateKey, (*compat.Request)(nil))
	_, ok = Request(inv)
	require.False(t, ok)

	Attach(inv, &compat.Request{})
	got, ok := Request(inv)
	require.True(t, ok)
	require.NotNil(t, got)
	require.Nil(t, got.Messages)
	require.Nil(t, got.StructuredOutput)
	require.Nil(t, got.Headers)
	require.Nil(t, got.Tools)
}

func TestAttachSnapshotsMultimodalPartsAndTools(t *testing.T) {
	text := "text"
	req := &compat.Request{
		Messages: []compat.Message{{
			Role: compat.RoleUser,
			ContentParts: []compat.ContentPart{
				{
					Type: compat.ContentTypeText,
					Text: &text,
				},
				{
					Type: compat.ContentTypeImage,
					Image: &compat.Image{
						Data: []byte{1, 2, 3},
					},
				},
				{
					Type: compat.ContentTypeAudio,
					Audio: &compat.Audio{
						Data: []byte{4, 5, 6},
					},
				},
				{
					Type: compat.ContentTypeFile,
					File: &compat.File{
						Data: []byte{7, 8, 9},
					},
				},
			},
		}},
		Tools: map[string]tool.Tool{
			"lookup": stubTool{name: "lookup"},
		},
	}

	inv := agent.NewInvocation()
	Attach(inv, req)

	*req.Messages[0].ContentParts[0].Text = "changed"
	req.Messages[0].ContentParts[1].Image.Data[0] = 9
	req.Messages[0].ContentParts[2].Audio.Data[0] = 9
	req.Messages[0].ContentParts[3].File.Data[0] = 9
	req.Tools.(map[string]tool.Tool)["lookup"] = stubTool{name: "changed"}
	req.Tools.(map[string]tool.Tool)["extra"] = stubTool{name: "extra"}

	got, ok := Request(inv)
	require.True(t, ok)
	parts := got.Messages[0].ContentParts
	require.Equal(t, "text", *parts[0].Text)
	require.Equal(t, []byte{1, 2, 3}, parts[1].Image.Data)
	require.Equal(t, []byte{4, 5, 6}, parts[2].Audio.Data)
	require.Equal(t, []byte{7, 8, 9}, parts[3].File.Data)
	require.Len(t, got.Tools, 1)
	require.Equal(t, "lookup", got.Tools.(map[string]tool.Tool)["lookup"].Declaration().Name)
}

func TestAppendResponseExtendsSnapshot(t *testing.T) {
	inv := agent.NewInvocation()
	Attach(inv, &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("stable system"),
			compat.NewUserMessage("question"),
		},
	})

	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{{
		Message: compat.Message{
			Role: compat.RoleAssistant,
			ToolCalls: []compat.ToolCall{{
				ID: "call_1",
				Function: compat.FunctionDefinitionParam{
					Name: "lookup",
				},
			}},
		},
	}}})
	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{{
		Message: compat.NewToolMessage("call_1", "lookup", "result"),
	}}})

	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 4)
	require.Equal(t, compat.RoleAssistant, got.Messages[2].Role)
	require.Len(t, got.Messages[2].ToolCalls, 1)
	require.Equal(t, compat.RoleTool, got.Messages[3].Role)
	require.Equal(t, "result", got.Messages[3].Content)
}

func TestInvocationViewAppendIsIsolated(t *testing.T) {
	invocation := agent.NewInvocation()
	Attach(invocation, &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	})

	view := invocation.View()
	AppendResponse(view, &compat.Response{Choices: []compat.Choice{{
		Message: compat.NewAssistantMessage("answer"),
	}}})

	viewRequest, ok := Request(view)
	require.True(t, ok)
	require.Len(t, viewRequest.Messages, 2)

	originalRequest, ok := Request(invocation)
	require.True(t, ok)
	require.Len(t, originalRequest.Messages, 1)
	require.Equal(t, "question", originalRequest.Messages[0].Content)
}

func TestInvocationViewRequestContentRefIsIsolated(t *testing.T) {
	invocation := agent.NewInvocation()
	Attach(invocation, &compat.Request{Messages: []compat.Message{{
		Role: compat.RoleUser,
		ContentParts: []compat.ContentPart{{
			ContentRef: &compat.ContentRef{ArtifactName: "original"},
		}},
	}}})

	viewRequest, ok := Request(invocation.View())
	require.True(t, ok)
	viewRequest.Messages[0].ContentParts[0].ContentRef.ArtifactName = "mutated"

	originalRequest, ok := Request(invocation)
	require.True(t, ok)
	require.Equal(t, "original",
		originalRequest.Messages[0].ContentParts[0].ContentRef.ArtifactName,
	)
}

func TestInvalidateClearsSnapshotUntilNextAttach(t *testing.T) {
	inv := agent.NewInvocation()
	req := &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	}
	Attach(inv, req)
	_, ok := Request(inv)
	require.True(t, ok)

	Invalidate(inv)
	_, ok = Request(inv)
	require.False(t, ok)

	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{{
		Message: compat.NewAssistantMessage("ignored"),
	}}})
	_, ok = Request(inv)
	require.False(t, ok)

	Attach(inv, req)
	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 1)
	require.Equal(t, "question", got.Messages[0].Content)
}

func TestAppendResponseNoopsWithoutPayload(t *testing.T) {
	AppendResponse(nil, &compat.Response{Choices: []compat.Choice{{
		Message: compat.NewAssistantMessage("ignored"),
	}}})
	AppendResponse(agent.NewInvocation(), &compat.Response{Choices: []compat.Choice{{
		Message: compat.NewAssistantMessage("ignored"),
	}}})

	inv := agent.NewInvocation()
	Attach(inv, &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	})

	AppendResponse(inv, nil)
	AppendResponse(inv, &compat.Response{})
	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{{}}})

	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 1)
}

func TestAppendResponseUsesDeltaFallback(t *testing.T) {
	inv := agent.NewInvocation()
	Attach(inv, &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	})

	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{{
		Delta: compat.NewAssistantMessage("streamed"),
	}}})

	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 2)
	require.Equal(t, "streamed", got.Messages[1].Content)
}

func TestAppendResponseKeepsPrimaryChoiceOnly(t *testing.T) {
	inv := agent.NewInvocation()
	Attach(inv, &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	})

	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{
		{
			Index:   1,
			Message: compat.NewAssistantMessage("alternative"),
		},
		{
			Index:   0,
			Message: compat.NewAssistantMessage("primary"),
		},
	}})

	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 2)
	require.Equal(t, "primary", got.Messages[1].Content)
}

func TestAppendResponseFallsBackToFirstChoice(t *testing.T) {
	inv := agent.NewInvocation()
	Attach(inv, &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("question")},
	})

	AppendResponse(inv, &compat.Response{Choices: []compat.Choice{
		{
			Index:   2,
			Message: compat.NewAssistantMessage("first"),
		},
		{
			Index:   3,
			Message: compat.NewAssistantMessage("second"),
		},
	}})

	got, ok := Request(inv)
	require.True(t, ok)
	require.Len(t, got.Messages, 2)
	require.Equal(t, "first", got.Messages[1].Content)
}
