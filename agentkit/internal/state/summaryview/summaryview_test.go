//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package summaryview

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestFinalizeBindsModelVisiblePrefix(t *testing.T) {
	timestamp := time.Now()
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		SessionID:            "session",
		ContentRequestLength: 3,
		Items: []Item{
			{
				Message:      compat.NewUserMessage("old"),
				Boundary:     Boundary{EventID: "event-1", Timestamp: timestamp},
				RequestIndex: 1,
			},
			{
				Message:      compat.NewAssistantMessage("answer"),
				Boundary:     Boundary{EventID: "event-2", Timestamp: timestamp.Add(time.Second)},
				RequestIndex: 2,
			},
		},
	})
	request := &compat.Request{Messages: []compat.Message{
		compat.NewSystemMessage("inserted"),
		compat.NewSystemMessage("stable"),
		compat.NewUserMessage("old"),
		compat.NewAssistantMessage("answer"),
	}}

	Finalize(invocation, request, 41_959)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Equal(t, 41_959, view.RequestTokens)
	require.Equal(t, 2, view.Items[0].RequestIndex)
	require.Equal(t, 3, view.Items[1].RequestIndex)

	prefix, ok := view.PrefixMessages(request.Messages, 1)
	require.True(t, ok)
	require.Equal(t, request.Messages[:3], prefix)
	boundary, ok := view.PrefixBoundary(1)
	require.True(t, ok)
	require.Equal(t, "event-1", boundary.EventID)

}

func TestRebaseAfterTransformTracksSafeCompletedPrefix(t *testing.T) {
	now := time.Now()
	assistant := compat.Message{
		Role: compat.RoleAssistant,
		ToolCalls: []compat.ToolCall{
			{ID: "call_keep"},
			{ID: "call_orphan"},
		},
	}
	toolResult := compat.NewToolMessage("call_keep", "lookup", "ok")
	before := []compat.Message{
		compat.NewSystemMessage("fixed"),
		assistant,
		toolResult,
	}
	filteredAssistant := assistant
	filteredAssistant.ToolCalls = filteredAssistant.ToolCalls[:1]
	after := []compat.Message{
		before[0],
		filteredAssistant,
		toolResult,
		compat.NewUserMessage("downgraded orphan call"),
	}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: len(before),
		Items: []Item{
			{
				Message: assistant,
				EffectiveEvent: event.Event{
					ID: "event-1",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: assistant,
					}}},
				},
				Boundary:     Boundary{EventID: "event-1", Timestamp: now},
				RequestIndex: 1,
			},
			{
				Message: toolResult,
				EffectiveEvent: event.Event{
					ID: "event-2",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: toolResult,
					}}},
				},
				Boundary: Boundary{
					EventID:   "event-2",
					Timestamp: now.Add(time.Second),
				},
				RequestIndex: 2,
			},
		},
	})

	rebased := RebaseAfterTransform(
		invocation,
		before,
		after,
		[]int{0, 1, 2, 1},
	)

	require.True(t, rebased)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Equal(t, len(after), view.ContentRequestLength)
	require.Len(t, view.Items, 3)
	require.Equal(t, []int{1, 2, 3}, []int{
		view.Items[0].RequestIndex,
		view.Items[1].RequestIndex,
		view.Items[2].RequestIndex,
	})
	require.True(t, view.Items[0].Boundary.IsZero())
	require.True(t, view.Items[1].Boundary.IsZero())
	require.Equal(t, "event-2", view.Items[2].Boundary.EventID)
	require.Equal(
		t,
		"downgraded orphan call",
		view.Items[2].EffectiveEvent.Response.Choices[0].Message.Content,
	)
	_, ok = view.PrefixBoundary(2)
	require.False(t, ok)
	boundary, ok := view.PrefixBoundary(3)
	require.True(t, ok)
	require.Equal(t, "event-2", boundary.EventID)

	Finalize(invocation, &compat.Request{Messages: after}, 100)
	view, ok = Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Equal(t, 100, view.RequestTokens)
}

func TestRebaseAfterTransformAcceptsImplicitIdentitySources(t *testing.T) {
	message := compat.NewUserMessage("history")
	messages := []compat.Message{
		compat.NewSystemMessage("fixed"),
		message,
	}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: len(messages),
		Items: []Item{{
			Message:      message,
			RequestIndex: 1,
		}},
	})

	rebased := RebaseAfterTransform(
		invocation,
		messages,
		messages,
		nil,
	)

	require.True(t, rebased)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Equal(t, 1, view.Items[0].RequestIndex)
}

func TestRebaseAfterTransformRejectsInvalidatedBinding(t *testing.T) {
	message := compat.NewUserMessage("history")
	messages := []compat.Message{message}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: len(messages),
		Items: []Item{{
			Message:      message,
			RequestIndex: 0,
		}},
	})
	InvalidateBinding(invocation)

	rebased := RebaseAfterTransform(
		invocation,
		messages,
		messages,
		nil,
	)

	require.False(t, rebased)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.False(t, view.Bound)
}

func TestRebaseAfterTransformRejectsEmptyExplicitSources(t *testing.T) {
	message := compat.NewUserMessage("history")
	messages := []compat.Message{message}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: len(messages),
		Items: []Item{{
			Message:      message,
			RequestIndex: 0,
		}},
	})

	rebased := RebaseAfterTransform(
		invocation,
		messages,
		messages,
		[]int{},
	)

	require.False(t, rebased)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.False(t, view.Bound)
}

func TestRebaseAfterTransformUsesOriginalProjectionLength(t *testing.T) {
	duplicate := compat.NewUserMessage("duplicate")
	current := compat.NewUserMessage("current")
	before := []compat.Message{duplicate, duplicate, current}
	after := []compat.Message{
		before[0],
		compat.NewUserMessage("split one"),
		compat.NewUserMessage("split two"),
		current,
	}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: 2,
		Items: []Item{{
			Message: duplicate,
			Boundary: Boundary{
				EventID:   "event-1",
				Timestamp: time.Now(),
			},
			RequestIndex: 0,
		}},
	})

	require.True(t, RebaseAfterTransform(
		invocation,
		before,
		after,
		[]int{0, 1, 1, 2},
	))
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.True(t, view.Bound)
	require.Len(t, view.Items, 2)
	require.Equal(t, 1, view.Items[0].RequestIndex)
	require.Equal(t, 2, view.Items[1].RequestIndex)
	require.True(t, view.Items[0].Boundary.IsZero())
	require.Equal(t, "event-1", view.Items[1].Boundary.EventID)
}

func TestRebaseAfterTransformFailsClosedWithoutCompleteProvenance(t *testing.T) {
	invocation := agent.NewInvocation()
	before := []compat.Message{compat.NewUserMessage("visible")}
	AttachProjection(invocation, &View{
		ContentRequestLength: len(before),
		Items: []Item{{
			Message:      before[0],
			RequestIndex: 0,
		}},
	})

	require.False(t, RebaseAfterTransform(
		invocation,
		before,
		[]compat.Message{
			compat.NewUserMessage("rewritten one"),
			compat.NewUserMessage("rewritten two"),
		},
		nil,
	))
	Finalize(invocation, &compat.Request{Messages: before}, 100)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.False(t, view.Bound)
	require.Equal(t, 100, view.RequestTokens)
}

func TestSnapshotIsIsolated(t *testing.T) {
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		Items: []Item{{
			Message: compat.NewUserMessage("visible"),
			EffectiveEvent: event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.NewUserMessage("visible"),
				}}},
			},
		}},
	})

	first, ok := Snapshot(invocation)
	require.True(t, ok)
	first.Items[0].Message.Content = "mutated"
	first.Items[0].EffectiveEvent.Response.Choices[0].Message.Content = "mutated"

	second, ok := Snapshot(invocation)
	require.True(t, ok)
	require.Equal(t, "visible", second.Items[0].Message.Content)
	require.Equal(
		t,
		"visible",
		second.Items[0].EffectiveEvent.Response.Choices[0].Message.Content,
	)
}

func TestInvocationViewFinalizationIsIsolated(t *testing.T) {
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: 1,
		Items: []Item{{
			Message:      compat.NewUserMessage("visible"),
			RequestIndex: 0,
		}},
	})

	view := invocation.View()
	Finalize(view, &compat.Request{Messages: []compat.Message{
		compat.NewUserMessage("visible"),
	}}, 42)

	viewSnapshot, ok := Snapshot(view)
	require.True(t, ok)
	require.True(t, viewSnapshot.Bound)
	require.Equal(t, 42, viewSnapshot.RequestTokens)

	originalSnapshot, ok := Snapshot(invocation)
	require.True(t, ok)
	require.False(t, originalSnapshot.Bound)
	require.Zero(t, originalSnapshot.RequestTokens)
}

func TestInvocationViewSnapshotNestedStateIsIsolated(t *testing.T) {
	finishReason := "stop"
	errorParam := "param"
	errorCode := "code"
	text := "text"
	toolCallIndex := 1
	message := compat.Message{
		Role: compat.RoleAssistant,
		ContentParts: []compat.ContentPart{
			{Text: &text},
			{
				Image: &compat.Image{Data: []byte("image")},
				ContentRef: &compat.ContentRef{
					ArtifactName: "original",
				},
			},
			{Audio: &compat.Audio{Data: []byte("audio")}},
			{Video: &compat.Video{Data: []byte("video")}},
			{File: &compat.File{Data: []byte("file")}},
		},
		ToolCalls: []compat.ToolCall{{
			Index: &toolCallIndex,
			Function: compat.FunctionDefinitionParam{
				Arguments: []byte("original"),
			},
			ExtraFields: map[string]any{
				"nested": map[string]any{"value": "original"},
			},
		}},
	}
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{Items: []Item{{
		Message: message,
		EffectiveEvent: event.Event{Response: &compat.Response{
			Choices: []compat.Choice{{
				Message:      message,
				Delta:        message,
				FinishReason: &finishReason,
			}},
			Error: &compat.ResponseError{
				Param: &errorParam,
				Code:  &errorCode,
			},
		}, ParentMetadata: &event.ParentInvocationMetadata{TriggerID: "original"}},
	}}})

	viewSnapshot, ok := Snapshot(invocation.View())
	require.True(t, ok)
	viewMessage := &viewSnapshot.Items[0].Message
	viewMessage.ToolCalls[0].Function.Arguments[0] = 'X'
	*viewMessage.ToolCalls[0].Index = 2
	viewMessage.ToolCalls[0].ExtraFields["nested"].(map[string]any)["value"] = "mutated"
	*viewMessage.ContentParts[0].Text = "mutated"
	viewMessage.ContentParts[1].Image.Data[0] = 'X'
	viewMessage.ContentParts[1].ContentRef.ArtifactName = "mutated"
	viewMessage.ContentParts[2].Audio.Data[0] = 'X'
	viewMessage.ContentParts[3].Video.Data[0] = 'X'
	viewMessage.ContentParts[4].File.Data[0] = 'X'
	choice := &viewSnapshot.Items[0].EffectiveEvent.Response.Choices[0]
	choice.Message.ToolCalls[0].Function.Arguments[0] = 'X'
	choice.Delta.ContentParts[1].Image.Data[0] = 'X'
	*choice.FinishReason = "mutated"
	*viewSnapshot.Items[0].EffectiveEvent.Response.Error.Param = "mutated"
	*viewSnapshot.Items[0].EffectiveEvent.Response.Error.Code = "mutated"
	viewSnapshot.Items[0].EffectiveEvent.ParentMetadata.TriggerID = "mutated"

	originalSnapshot, ok := Snapshot(invocation)
	require.True(t, ok)
	originalMessage := originalSnapshot.Items[0].Message
	require.Equal(t, "original", string(
		originalMessage.ToolCalls[0].Function.Arguments,
	))
	require.Equal(t, 1, *originalMessage.ToolCalls[0].Index)
	require.Equal(t, "original",
		originalMessage.ToolCalls[0].ExtraFields["nested"].(map[string]any)["value"],
	)
	require.Equal(t, "text", *originalMessage.ContentParts[0].Text)
	require.Equal(t, "image", string(
		originalMessage.ContentParts[1].Image.Data,
	))
	require.Equal(t, "original",
		originalMessage.ContentParts[1].ContentRef.ArtifactName,
	)
	require.Equal(t, "audio", string(originalMessage.ContentParts[2].Audio.Data))
	require.Equal(t, "video", string(originalMessage.ContentParts[3].Video.Data))
	require.Equal(t, "file", string(originalMessage.ContentParts[4].File.Data))
	originalChoice := originalSnapshot.Items[0].EffectiveEvent.Response.Choices[0]
	require.Equal(t, "original", string(
		originalChoice.Message.ToolCalls[0].Function.Arguments,
	))
	require.Equal(t, "image", string(
		originalChoice.Delta.ContentParts[1].Image.Data,
	))
	require.Equal(t, "stop", *originalChoice.FinishReason)
	require.Equal(t, "param",
		*originalSnapshot.Items[0].EffectiveEvent.Response.Error.Param,
	)
	require.Equal(t, "code",
		*originalSnapshot.Items[0].EffectiveEvent.Response.Error.Code,
	)
	require.Equal(t, "original",
		originalSnapshot.Items[0].EffectiveEvent.ParentMetadata.TriggerID,
	)
}

func TestContextAndInvocationLifecycle(t *testing.T) {
	invocation := agent.NewInvocation()
	_, ok := Snapshot(invocation)
	require.False(t, ok)
	Finalize(invocation, &compat.Request{}, 1)
	Clear(nil)
	AttachProjection(nil, &View{})
	AttachProjection(invocation, nil)

	view := &View{
		SessionID: "session",
		Items: []Item{{
			Message: compat.NewUserMessage("visible"),
		}},
	}
	AttachProjection(invocation, view)
	view.Items[0].Message.Content = "mutated source"
	stored, ok := Snapshot(invocation)
	require.True(t, ok)
	require.Equal(t, "visible", stored.Items[0].Message.Content)

	ctx := ContextWithView(nil, stored)
	fromContext, ok := FromContext(ctx)
	require.True(t, ok)
	fromContext.Items[0].Message.Content = "mutated context copy"
	again, ok := FromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "visible", again.Items[0].Message.Content)
	require.Same(t, ctx, ContextWithView(ctx, nil))
	_, ok = FromContext(nil)
	require.False(t, ok)
	_, ok = FromContext(context.Background())
	require.False(t, ok)

	Clear(invocation)
	_, ok = Snapshot(invocation)
	require.False(t, ok)
}

func TestViewSelectsNonContiguousItems(t *testing.T) {
	now := time.Now()
	view := &View{
		Bound: true,
		Items: []Item{
			{
				RequestIndex: 1,
				Boundary:     Boundary{EventID: "first", Timestamp: now},
			},
			{
				RequestIndex: 2,
				Boundary:     Boundary{},
			},
			{
				RequestIndex: 3,
				Boundary: Boundary{
					EventID:   "third",
					Timestamp: now.Add(time.Second),
				},
			},
		},
	}
	parent := []compat.Message{
		compat.NewSystemMessage("fixed"),
		compat.NewUserMessage("first"),
		compat.NewAssistantMessage("excluded"),
		compat.NewUserMessage("third"),
	}

	messages, ok := view.MessagesForItems(parent, []int{0, 2})
	require.True(t, ok)
	require.Equal(t, []compat.Message{parent[0], parent[1], parent[3]}, messages)
	boundary, ok := view.BoundaryForItems([]int{0, 1, 2})
	require.True(t, ok)
	require.Equal(t, "third", boundary.EventID)
	boundary, ok = view.BoundaryForItems([]int{0, 1})
	require.True(t, ok)
	require.Equal(t, "first", boundary.EventID)

	_, ok = view.MessagesForItems(parent, nil)
	require.False(t, ok)
	_, ok = view.MessagesForItems(parent, []int{2, 1})
	require.False(t, ok)
	_, ok = view.MessagesForItems(parent, []int{3})
	require.False(t, ok)
	_, ok = (&View{}).MessagesForItems(parent, []int{0})
	require.False(t, ok)
	_, ok = view.BoundaryForItems([]int{3})
	require.False(t, ok)
	_, ok = (*View)(nil).BoundaryForItems([]int{0})
	require.False(t, ok)
	_, ok = view.PrefixMessages(parent, 0)
	require.False(t, ok)
	_, ok = view.PrefixMessages(parent, 4)
	require.False(t, ok)
	_, ok = view.PrefixBoundary(0)
	require.False(t, ok)
	require.Nil(t, (*View)(nil).Events())
}

func TestSnapshotClonesEffectiveEventMetadata(t *testing.T) {
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{Items: []Item{{
		EffectiveEvent: event.Event{
			Response: &compat.Response{Choices: []compat.Choice{{
				Message: compat.NewAssistantMessage("answer"),
			}}},
			LongRunningToolIDs: map[string]struct{}{"call": {}},
			StateDelta:         map[string][]byte{"key": []byte("value")},
			Extensions: map[string]json.RawMessage{
				"metadata": json.RawMessage(`{"value":"original"}`),
			},
			Actions: &event.EventActions{SkipSummarization: true},
		},
	}}})

	first, ok := Snapshot(invocation)
	require.True(t, ok)
	first.Items[0].EffectiveEvent.LongRunningToolIDs["other"] = struct{}{}
	first.Items[0].EffectiveEvent.StateDelta["key"][0] = 'x'
	first.Items[0].EffectiveEvent.Extensions["metadata"][10] = 'x'
	first.Items[0].EffectiveEvent.Actions.SkipSummarization = false

	second, ok := Snapshot(invocation)
	require.True(t, ok)
	require.NotContains(t, second.Items[0].EffectiveEvent.LongRunningToolIDs, "other")
	require.Equal(t, "value", string(second.Items[0].EffectiveEvent.StateDelta["key"]))
	require.JSONEq(
		t,
		`{"value":"original"}`,
		string(second.Items[0].EffectiveEvent.Extensions["metadata"]),
	)
	require.True(t, second.Items[0].EffectiveEvent.Actions.SkipSummarization)
	require.Len(t, second.Events(), 1)
}

func TestFinalizeLeavesUnmatchedProjectionUnbound(t *testing.T) {
	invocation := agent.NewInvocation()
	AttachProjection(invocation, &View{
		ContentRequestLength: 1,
		Items: []Item{{
			Message:      compat.NewUserMessage("expected"),
			RequestIndex: 0,
		}},
	})

	Finalize(invocation, &compat.Request{Messages: []compat.Message{
		compat.NewAssistantMessage("different"),
	}}, 10)
	view, ok := Snapshot(invocation)
	require.True(t, ok)
	require.False(t, view.Bound)
	require.Equal(t, 10, view.RequestTokens)

	Finalize(nil, &compat.Request{}, 1)
	Finalize(invocation, nil, 1)
}

func TestViewBoundaryAndBindingEdgeCases(t *testing.T) {
	view := &View{Items: []Item{{Boundary: Boundary{}}}}
	_, ok := view.PrefixBoundary(2)
	require.False(t, ok)

	parent := []compat.Message{compat.NewUserMessage("visible")}
	invalidFirst := &View{
		Bound: true,
		Items: []Item{{
			RequestIndex: -1,
		}},
	}
	_, ok = invalidFirst.MessagesForItems(parent, []int{0})
	require.False(t, ok)
	invalidFirst.Items[0].RequestIndex = len(parent) + 1
	_, ok = invalidFirst.MessagesForItems(parent, []int{0})
	require.False(t, ok)

	invalidItem := &View{
		Bound: true,
		Items: []Item{
			{RequestIndex: 0},
			{RequestIndex: len(parent)},
		},
	}
	_, ok = invalidItem.MessagesForItems(parent, []int{1})
	require.False(t, ok)

	require.False(t, bindItems(nil, parent))
	require.False(t, bindItems(&View{}, parent))
	require.False(t, bindItems(view, nil))

	messages := []compat.Message{
		compat.NewUserMessage("different"),
		compat.NewUserMessage("visible"),
	}
	require.Equal(t, 1, findItem(
		messages,
		compat.NewUserMessage("visible"),
		0,
		0,
		0,
	))
	require.Equal(t, -1, findItem(
		messages,
		compat.NewAssistantMessage("missing"),
		0,
		0,
		0,
	))

	require.Nil(t, cloneView(nil))
	setEffectiveMessage(nil, compat.NewUserMessage("ignored"))
	effective := event.Event{Response: &compat.Response{
		Choices: []compat.Choice{{Message: compat.NewAssistantMessage("old")}},
	}}
	setEffectiveMessage(&effective, compat.NewAssistantMessage("new"))
	require.Equal(
		t,
		"new",
		effective.Response.Choices[0].Message.Content,
	)
}

func TestMessageIdentityMatchesStableFields(t *testing.T) {
	require.False(t, messageIdentityMatches(
		compat.NewAssistantMessage("same"),
		compat.NewUserMessage("same"),
	))
	require.True(t, messageIdentityMatches(
		compat.NewToolMessage("call", "renamed", "new payload"),
		compat.NewToolMessage("call", "lookup", "old payload"),
	))
	require.False(t, messageIdentityMatches(
		compat.NewToolMessage("other", "lookup", "payload"),
		compat.NewToolMessage("call", "lookup", "payload"),
	))

	wantToolCalls := compat.NewAssistantMessage("old")
	wantToolCalls.ToolCalls = []compat.ToolCall{{ID: "call"}}
	gotToolCalls := compat.NewAssistantMessage("new")
	gotToolCalls.ToolCalls = []compat.ToolCall{{
		ID: "call",
		Function: compat.FunctionDefinitionParam{
			Name:      "lookup",
			Arguments: []byte(`{"query":"new"}`),
		},
	}}
	require.True(t, messageIdentityMatches(gotToolCalls, wantToolCalls))
	gotToolCalls.ToolCalls[0].ID = "other"
	require.False(t, messageIdentityMatches(gotToolCalls, wantToolCalls))

	wantContent := compat.NewUserMessage("same")
	wantContent.ReasoningContent = "reasoning"
	gotContent := wantContent
	gotContent.ReasoningSignature = "provider-specific-signature"
	require.True(t, messageIdentityMatches(gotContent, wantContent))
	gotContent.Content = "different"
	require.False(t, messageIdentityMatches(gotContent, wantContent))

	require.Nil(t, toolCallIDs(nil))
	require.Equal(
		t,
		[]string{"first", "second"},
		toolCallIDs([]compat.ToolCall{{ID: "first"}, {ID: "second"}}),
	)
}
