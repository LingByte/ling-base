//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package graph

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/stretchr/testify/require"
)

// Test that when a node returns State{messages: MessageOp}, execution still
// succeeds and the terminal completion snapshot carries the graph final state.
func TestMessagesReducerAppliesMessageOps(t *testing.T) {
	schema := MessagesStateSchema()
	sg := NewStateGraph(schema)

	sg.
		AddNode("seed", func(ctx context.Context, s State) (any, error) {
			return State{StateKeyMessages: []compat.Message{compat.NewUserMessage("u")}}, nil
		}).
		AddNode("op", func(ctx context.Context, s State) (any, error) {
			return State{StateKeyMessages: AppendMessages{Items: []compat.Message{compat.NewAssistantMessage("a")}}, StateKeyLastResponse: "a"}, nil
		}).
		SetEntryPoint("seed").
		AddEdge("seed", "op").
		SetFinishPoint("op")

	g, err := sg.Compile()
	require.NoError(t, err)
	exec, err := NewExecutor(g)
	require.NoError(t, err)

	inv := &agent.Invocation{InvocationID: "merge-msg-op"}
	ch, err := exec.Execute(context.Background(), State{}, inv)
	require.NoError(t, err)

	var lastResponse string
	var sawMessages bool
	for e := range ch {
		if e.Done && e.StateDelta != nil {
			if data, ok := e.StateDelta[StateKeyLastResponse]; ok {
				require.NoError(t, json.Unmarshal(data, &lastResponse))
			}
			_, sawMessages = e.StateDelta[StateKeyMessages]
		}
	}
	require.Equal(t, "a", lastResponse)
	require.True(t, sawMessages)
}

// Test that AddToolsConditionalEdges routes to the tools node when tool-calls
// are present in the assistant message.
func TestAddToolsConditionalEdgesRoutesToTools(t *testing.T) {
	schema := MessagesStateSchema()
	sg := NewStateGraph(schema)

	sg.
		AddNode("llm", func(ctx context.Context, s State) (any, error) {
			// Simulate an assistant message containing a tool call
			msgs := []compat.Message{
				compat.NewUserMessage("hi"),
				{
					Role: compat.RoleAssistant,
					ToolCalls: []compat.ToolCall{
						{
							Type:     "function",
							Function: compat.FunctionDefinitionParam{Name: "dummy"},
							ID:       "call-1",
						},
					},
				},
			}
			return State{StateKeyMessages: msgs}, nil
		}).
		AddNode("tools", func(ctx context.Context, s State) (any, error) {
			return State{"routed": "tools"}, nil
		}).
		AddNode("fallback", func(ctx context.Context, s State) (any, error) {
			return State{"routed": "fallback"}, nil
		}).
		SetEntryPoint("llm").
		SetFinishPoint("tools").
		SetFinishPoint("fallback").
		AddToolsConditionalEdges("llm", "tools", "fallback")

	g, err := sg.Compile()
	require.NoError(t, err)
	exec, err := NewExecutor(g)
	require.NoError(t, err)

	inv := &agent.Invocation{InvocationID: "tools-route"}
	ch, err := exec.Execute(context.Background(), State{}, inv)
	require.NoError(t, err)

	var routed string
	for e := range ch {
		if e.Done && e.StateDelta != nil {
			if b, ok := e.StateDelta["routed"]; ok {
				require.NoError(t, json.Unmarshal(b, &routed))
			}
		}
	}
	require.Equal(t, "tools", routed)
}
