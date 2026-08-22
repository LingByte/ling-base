//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package processor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/graph"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	"github.com/stretchr/testify/require"
)

func TestProcessRequest_IgnoresRunOptionsMessages_UsesSessionOnly(t *testing.T) {
	// Even if RunOptions carries messages, content processor should only read from session.
	seed := []compat.Message{
		compat.NewSystemMessage("system guidance"),
		compat.NewUserMessage("hello"),
		compat.NewAssistantMessage("hi"),
	}

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("hi")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("latest from session")),
	)

	inv := &agent.Invocation{
		InvocationID: "inv-seed",
		AgentName:    "test-agent",
		Session:      sess,
		Message:      compat.NewUserMessage("hello"),
		RunOptions:   agent.RunOptions{Messages: seed},
	}

	req := &compat.Request{}
	ch := make(chan *event.Event, 2)
	p := NewContentRequestProcessor()

	p.ProcessRequest(context.Background(), inv, req, ch)

	// Expect only session-derived messages (3 entries), not the seed.
	require.Equal(t, 3, len(req.Messages))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("latest from session"), req.Messages[2]))
}

func TestProcessRequest_FiltersEmptyAssistantMessages(t *testing.T) {
	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		event.Event{
			Response: &compat.Response{
				Done: true,
				Choices: []compat.Choice{
					{Index: 0, Message: compat.Message{Role: compat.RoleAssistant}},
					{Index: 1, Message: compat.NewAssistantMessage("hi")},
				},
			},
			Author: "test-agent",
		},
	)

	inv := &agent.Invocation{
		InvocationID: "inv-empty-assistant",
		AgentName:    "test-agent",
		Session:      sess,
	}

	req := &compat.Request{}
	NewContentRequestProcessor().ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 2)
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[1]))
}

func TestProcessRequest_FiltersEmptyAssistantMessages_ToolCallResponse(t *testing.T) {
	toolCallMsg := compat.Message{
		Role: compat.RoleAssistant,
		ToolCalls: []compat.ToolCall{
			{
				Type: "function",
				ID:   "call_1",
				Function: compat.FunctionDefinitionParam{
					Name:      "get_user_phone",
					Arguments: []byte(`{"purpose":"test"}`),
				},
			},
		},
	}

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hi")),
		event.Event{
			Response: &compat.Response{
				Done: true,
				Choices: []compat.Choice{
					{Index: 0, Message: toolCallMsg},
					{Index: 1, Message: compat.Message{Role: compat.RoleAssistant}},
				},
			},
			Author: "test-agent",
		},
	)

	inv := &agent.Invocation{
		InvocationID: "inv-empty-assistant-toolcall",
		AgentName:    "test-agent",
		Session:      sess,
	}

	req := &compat.Request{}
	NewContentRequestProcessor().ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 2)
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hi"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(toolCallMsg, req.Messages[1]))
}

func TestProcessRequest_IncludeContentsNone_FiltersEmptyAssistantMessages(t *testing.T) {
	sess := &session.Session{}

	userEvt := newSessionEvent("user", compat.NewUserMessage("hello"))
	userEvt.InvocationID = "inv-include-none"
	assistantEvt := event.Event{
		InvocationID: "inv-include-none",
		Response: &compat.Response{
			Done: true,
			Choices: []compat.Choice{
				{Index: 0, Message: compat.Message{Role: compat.RoleAssistant}},
				{Index: 1, Message: compat.NewAssistantMessage("hi")},
			},
		},
		Author: "test-agent",
	}
	sess.Events = append(sess.Events, userEvt, assistantEvt)

	inv := &agent.Invocation{
		InvocationID: "inv-include-none",
		AgentName:    "test-agent",
		Session:      sess,
		RunOptions: agent.RunOptions{
			RuntimeState: map[string]any{
				graph.CfgKeyIncludeContents: "none",
			},
		},
	}

	req := &compat.Request{}
	NewContentRequestProcessor().ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 2)
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[1]))
}

func TestProcessRequest_IncludeInvocationMessage_WhenNoSession(t *testing.T) {
	// When no session or empty, include invocation.Message as the only message.
	inv := &agent.Invocation{
		InvocationID: "inv-empty",
		AgentName:    "test-agent",
		Session:      &session.Session{},
		Message:      compat.NewUserMessage("hi there"),
	}

	req := &compat.Request{}
	ch := make(chan *event.Event, 1)
	p := NewContentRequestProcessor()

	p.ProcessRequest(context.Background(), inv, req, ch)
	require.Equal(t, 1, len(req.Messages))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hi there"), req.Messages[0]))
}

func TestProcessRequest_InsertsInjectedContextMessages_AfterSystemMessages(t *testing.T) {
	requestContext := []compat.Message{
		compat.NewSystemMessage("ctx system"),
		compat.NewUserMessage("ctx user"),
		compat.NewAssistantMessage("ctx assistant"),
	}

	inv := &agent.Invocation{
		InvocationID: "inv-ctx",
		AgentName:    "test-agent",
		Session:      &session.Session{},
		Message:      compat.NewUserMessage("hi there"),
		RunOptions: agent.RunOptions{
			InjectedContextMessages: requestContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("agent system prompt"),
			compat.NewSystemMessage("session summary"),
		},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 2+len(requestContext)+1)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("session summary"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("ctx system"), req.Messages[2]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("ctx user"), req.Messages[3]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("ctx assistant"), req.Messages[4]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hi there"), req.Messages[5]))
}

func TestProcessRequest_InsertsInjectedContextMessages_WhenNoSystemMessages(t *testing.T) {
	requestContext := []compat.Message{
		compat.NewUserMessage("ctx user"),
		compat.NewAssistantMessage("ctx assistant"),
	}

	inv := &agent.Invocation{
		InvocationID: "inv-ctx-no-system",
		AgentName:    "test-agent",
		Message:      compat.NewUserMessage("hi there"),
		RunOptions: agent.RunOptions{
			InjectedContextMessages: requestContext,
		},
	}

	req := &compat.Request{}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, len(requestContext)+1)
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("ctx user"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("ctx assistant"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hi there"), req.Messages[2]))
}

func TestProcessRequest_InsertsInjectedContextMessages_BeforeSessionHistory(t *testing.T) {
	requestContext := []compat.Message{
		compat.NewSystemMessage("ctx system"),
		compat.NewUserMessage("ctx user"),
	}

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("hi")),
	)

	inv := &agent.Invocation{
		InvocationID: "inv-ctx-history",
		AgentName:    "test-agent",
		Session:      sess,
		Message:      compat.NewUserMessage("current"),
		RunOptions: agent.RunOptions{
			InjectedContextMessages: requestContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{compat.NewSystemMessage("agent system prompt")},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 1+len(requestContext)+3)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("ctx system"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("ctx user"), req.Messages[2]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[3]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[4]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("current"), req.Messages[5]))
}

func TestProcessRequest_InsertsLateContextMessages_BeforeLatestUserMessage(t *testing.T) {
	lateContext := []compat.Message{
		compat.NewUserMessage("late rules"),
		compat.NewUserMessage("late background"),
	}

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("hi")),
	)

	inv := &agent.Invocation{
		InvocationID: "inv-late-context",
		AgentName:    "test-agent",
		Session:      sess,
		Message:      compat.NewUserMessage("current"),
		RunOptions: agent.RunOptions{
			LateContextMessages: lateContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{compat.NewSystemMessage("agent system prompt")},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 1+2+len(lateContext)+1)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[2]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late rules"), req.Messages[3]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late background"), req.Messages[4]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("current"), req.Messages[5]))
}

func TestProcessRequest_InsertsLateContextMessages_AfterLeadingSystemWhenNoUser(t *testing.T) {
	lateContext := []compat.Message{
		compat.NewUserMessage("late rules"),
	}

	inv := &agent.Invocation{
		InvocationID: "inv-late-no-user",
		AgentName:    "test-agent",
		RunOptions: agent.RunOptions{
			LateContextMessages: lateContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("agent system prompt"),
			compat.NewAssistantMessage("assistant tail"),
		},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 3)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late rules"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("assistant tail"), req.Messages[2]))
}

func TestProcessRequest_InsertsLateContextMessages_KeepsToolTailAfterUserTurn(t *testing.T) {
	lateContext := []compat.Message{
		compat.NewUserMessage("late rules"),
		compat.NewUserMessage("late background"),
	}

	toolCallMsg := compat.Message{
		Role: compat.RoleAssistant,
		ToolCalls: []compat.ToolCall{
			{
				Type: "function",
				ID:   "call_1",
				Function: compat.FunctionDefinitionParam{
					Name:      "get_user_phone",
					Arguments: []byte(`{"purpose":"test"}`),
				},
			},
		},
	}
	toolResultMsg := compat.NewToolMessage("call_1", "get_user_phone", `{"status":"ok"}`)

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("hi")),
		newSessionEvent("user", compat.NewUserMessage("current")),
		newSessionEvent("test-agent", toolCallMsg),
		newSessionEvent("test-agent", toolResultMsg),
	)

	inv := &agent.Invocation{
		InvocationID: "inv-late-tool-tail",
		AgentName:    "test-agent",
		Session:      sess,
		Message:      compat.NewUserMessage("current"),
		RunOptions: agent.RunOptions{
			LateContextMessages: lateContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{compat.NewSystemMessage("agent system prompt")},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 8)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[2]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late rules"), req.Messages[3]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late background"), req.Messages[4]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("current"), req.Messages[5]))
	require.True(t, compat.MessagesEqual(toolCallMsg, req.Messages[6]))
	require.True(t, compat.MessagesEqual(toolResultMsg, req.Messages[7]))
}

func TestProcessRequest_InsertsInjectedAndLateContextMessages_InOrder(t *testing.T) {
	injectedContext := []compat.Message{
		compat.NewSystemMessage("ctx system"),
		compat.NewUserMessage("ctx user"),
	}
	lateContext := []compat.Message{
		compat.NewUserMessage("late rules"),
	}

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEvent("user", compat.NewUserMessage("hello")),
		newSessionEvent("test-agent", compat.NewAssistantMessage("hi")),
	)

	inv := &agent.Invocation{
		InvocationID: "inv-injected-and-late",
		AgentName:    "test-agent",
		Session:      sess,
		Message:      compat.NewUserMessage("current"),
		RunOptions: agent.RunOptions{
			InjectedContextMessages: injectedContext,
			LateContextMessages:     lateContext,
		},
	}

	req := &compat.Request{
		Messages: []compat.Message{compat.NewSystemMessage("agent system prompt")},
	}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 1+len(injectedContext)+2+len(lateContext)+1)
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("agent system prompt"), req.Messages[0]))

	// Injected context is inserted early, before session history.
	require.True(t, compat.MessagesEqual(compat.NewSystemMessage("ctx system"), req.Messages[1]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("ctx user"), req.Messages[2]))

	// Session history stays canonical.
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("hello"), req.Messages[3]))
	require.True(t, compat.MessagesEqual(compat.NewAssistantMessage("hi"), req.Messages[4]))

	// Late context is inserted right before the latest user message.
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("late rules"), req.Messages[5]))
	require.True(t, compat.MessagesEqual(compat.NewUserMessage("current"), req.Messages[6]))
}

func TestProcessRequest_NoDuplicateInvocationToolMessage(t *testing.T) {
	const (
		requestID   = "req-tool-message"
		toolCallID  = "call-1"
		toolName    = "external_tool"
		toolContent = `{"status":"ok"}`
	)

	msg := compat.NewToolMessage(toolCallID, toolName, toolContent)
	sess := &session.Session{
		Events: []event.Event{
			{
				RequestID: requestID,
				Response: &compat.Response{
					Done: true,
					Choices: []compat.Choice{
						{Index: 0, Message: msg},
					},
				},
				Author: "user",
			},
		},
	}

	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationMessage(msg),
		agent.WithInvocationRunOptions(
			agent.RunOptions{RequestID: requestID},
		),
	)
	inv.AgentName = "test-agent"

	req := &compat.Request{}
	p := NewContentRequestProcessor()
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 1)
	require.True(t, compat.MessagesEqual(msg, req.Messages[0]))
}

// When session exists but has no events for the current branch, the invocation
// message should still be included so sub agent gets the tool args.
func TestProcessRequest_IncludeInvocationMessage_WhenNoBranchEvents(t *testing.T) {
	// Session has events, but authored under a different filter key/branch.
	sess := &session.Session{}
	// Event authored by other-agent; with IncludeContentsFiltered and filterKey
	// set to current agent, this should be filtered out.
	sess.Events = append(sess.Events, event.Event{
		Response: &compat.Response{
			Done:    true,
			Choices: []compat.Choice{{Index: 0, Message: compat.NewAssistantMessage("context")}},
		},
		Author:    "other-agent",
		FilterKey: "other-agent",
		Version:   event.CurrentVersion,
	})

	// Build invocation explicitly with filter key set to sub-agent branch.
	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationMessage(compat.NewUserMessage("{\\\"target\\\":\\\"svc\\\"}")),
		agent.WithInvocationEventFilterKey("sub-agent"),
	)
	inv.AgentName = "sub-agent"

	req := &compat.Request{}
	ch := make(chan *event.Event, 1)
	p := NewContentRequestProcessor()

	p.ProcessRequest(context.Background(), inv, req, ch)

	// The other-agent event is filtered out; invocation message must be added.
	require.Equal(t, 1, len(req.Messages))
	require.True(t, compat.MessagesEqual(inv.Message, req.Messages[0]))
}

func TestProcessRequest_IncludeInvocationMessage_WhenNoBranchEvents_Multimodal(t *testing.T) {
	// Session has events, but authored under a different filter key/branch.
	sess := &session.Session{}
	sess.Events = append(sess.Events, event.Event{
		Response: &compat.Response{
			Done:    true,
			Choices: []compat.Choice{{Index: 0, Message: compat.NewAssistantMessage("context")}},
		},
		Author:    "other-agent",
		FilterKey: "other-agent",
		Version:   event.CurrentVersion,
	})

	msg := compat.NewUserMessage("")
	msg.AddImageURL("https://example.com/image.png", "auto")

	// Build invocation explicitly with filter key set to sub-agent branch.
	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationMessage(msg),
		agent.WithInvocationEventFilterKey("sub-agent"),
	)
	inv.AgentName = "sub-agent"

	req := &compat.Request{}
	ch := make(chan *event.Event, 1)
	p := NewContentRequestProcessor()

	p.ProcessRequest(context.Background(), inv, req, ch)

	// The other-agent event is filtered out; invocation message must be added.
	require.Equal(t, 1, len(req.Messages))
	require.True(t, compat.MessagesEqual(inv.Message, req.Messages[0]))
}

func TestProcessRequest_PreserveSameBranchKeepsRoles(t *testing.T) {
	makeInvocation := func(sess *session.Session) *agent.Invocation {
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationMessage(
				compat.NewUserMessage("latest request"),
			),
			agent.WithInvocationEventFilterKey("graph-agent"),
		)
		inv.AgentName = "graph-agent"
		inv.Branch = "graph-agent"
		return inv
	}

	assistantMsg := compat.NewAssistantMessage("node produced answer")
	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEventWithBranch("user", "graph-agent", "graph-agent", compat.NewUserMessage("hi")),
		newSessionEventWithBranch("graph-node", "graph-agent", "graph-agent/graph-node", assistantMsg),
	)

	// Default behavior now preserves same-branch assistant/tool roles.
	// Explicitly enabling preserve keeps assistant role.
	preserveReq := &compat.Request{}
	preserveProc := NewContentRequestProcessor(
		WithPreserveSameBranch(true),
	)
	preserveProc.ProcessRequest(
		context.Background(), makeInvocation(sess), preserveReq, nil,
	)
	require.Equal(t, 3, len(preserveReq.Messages))
	require.Equal(t, compat.RoleUser, preserveReq.Messages[0].Role)
	require.Equal(t, compat.RoleAssistant, preserveReq.Messages[1].Role)
	require.Equal(t, assistantMsg.Content, preserveReq.Messages[1].Content)

	// Disabling preserve rewrites same-branch events as user context.
	optOutReq := &compat.Request{}
	optOutProc := NewContentRequestProcessor(
		WithPreserveSameBranch(false),
	)
	optOutProc.ProcessRequest(
		context.Background(), makeInvocation(sess), optOutReq, nil,
	)
	require.Equal(t, 3, len(optOutReq.Messages))
	require.Equal(t, compat.RoleUser, optOutReq.Messages[0].Role)
	require.Equal(t, compat.RoleUser, optOutReq.Messages[1].Role)
	require.Contains(t, optOutReq.Messages[1].Content, "For context")
}

// When the historical event branch is an ancestor or descendant of the current
// branch, PreserveSameBranch=true should keep assistant roles.
func TestProcessRequest_PreserveSameBranch_AncestorDescendant(t *testing.T) {
	makeInvocation := func(sess *session.Session) *agent.Invocation {
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationMessage(
				compat.NewUserMessage("latest request"),
			),
			agent.WithInvocationEventFilterKey("graph-agent"),
		)
		inv.AgentName = "graph-agent"
		inv.Branch = "graph-agent/child"
		return inv
	}

	// ancestor: graph-agent
	// descendant: graph-agent/child/grandchild
	msgAncestor := compat.NewAssistantMessage("from ancestor")
	msgDesc := compat.NewAssistantMessage("from descendant")

	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEventWithBranch(
			"graph-root", "graph-agent", "graph-agent", msgAncestor,
		),
		newSessionEventWithBranch(
			"graph-leaf", "graph-agent",
			"graph-agent/child/grandchild", msgDesc,
		),
	)

	req := &compat.Request{}
	p := NewContentRequestProcessor(WithPreserveSameBranch(true))
	p.ProcessRequest(context.Background(), makeInvocation(sess), req, nil)

	require.Equal(t, 3, len(req.Messages))
	require.Equal(t, compat.RoleAssistant, req.Messages[0].Role)
	require.Equal(t, msgAncestor.Content, req.Messages[0].Content)
	require.Equal(t, compat.RoleAssistant, req.Messages[1].Role)
	require.Equal(t, msgDesc.Content, req.Messages[1].Content)
}

// When the historical event is on a different branch lineage, it should be
// converted to user context even when preserve is true (default).
func TestProcessRequest_CrossBranch_RewritesToUser(t *testing.T) {
	inv := agent.NewInvocation(
		agent.WithInvocationSession(&session.Session{}),
		agent.WithInvocationMessage(compat.NewUserMessage("ask")),
		agent.WithInvocationEventFilterKey("graph-agent"),
	)
	inv.AgentName = "graph-agent"
	inv.Branch = "graph-agent"

	// Cross-branch event (not same lineage). Use the same filter key so it is
	// included by IncludeContentsFiltered.
	msg := compat.NewAssistantMessage("foreign content")
	evt := newSessionEventWithBranch(
		"other-agent", "graph-agent", "other-root", msg,
	)

	sess := &session.Session{}
	sess.Events = append(sess.Events, evt)
	inv.Session = sess

	req := &compat.Request{}
	p := NewContentRequestProcessor(WithPreserveSameBranch(true))
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Equal(t, 2, len(req.Messages))
	require.Equal(t, compat.RoleUser, req.Messages[0].Role)
	require.Contains(t, req.Messages[0].Content, "For context")
}

func TestProcessRequest_PreserveForeignMessagesKeepsOriginalTranscript(t *testing.T) {
	inv := agent.NewInvocation(
		agent.WithInvocationSession(&session.Session{}),
		agent.WithInvocationMessage(compat.NewUserMessage("ask")),
		agent.WithInvocationEventFilterKey("graph-agent"),
	)
	inv.AgentName = "graph-agent"
	inv.Branch = "graph-agent"

	toolCallID := "call_select"
	sess := &session.Session{}
	sess.Events = append(sess.Events,
		newSessionEventWithBranch(
			"other-agent",
			"graph-agent",
			"other-root",
			compat.NewAssistantMessage("我来帮你调用上车点选择工具。"),
		),
		newSessionEventWithBranch(
			"other-agent",
			"graph-agent",
			"other-root",
			compat.Message{
				Role: compat.RoleAssistant,
				ToolCalls: []compat.ToolCall{{
					ID:   toolCallID,
					Type: "function",
					Function: compat.FunctionDefinitionParam{
						Name:      "select_get_on_address",
						Arguments: []byte(`{"title":"请问你要从哪里出发？"}`),
					},
				}},
			},
		),
		newSessionEventWithBranch(
			"other-agent",
			"graph-agent",
			"other-root",
			compat.Message{
				Role:     compat.RoleTool,
				ToolID:   toolCallID,
				ToolName: "select_get_on_address",
				Content:  `{"title":"凌波门"}`,
			},
		),
		newSessionEventWithBranch(
			"other-agent",
			"graph-agent",
			"other-root",
			compat.NewAssistantMessage("正在为你搜索武汉大学。"),
		),
	)
	inv.Session = sess

	req := &compat.Request{}
	p := NewContentRequestProcessor(
		WithPreserveSameBranch(true),
		WithPreserveForeignMessages(true),
	)
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Len(t, req.Messages, 5)
	require.Equal(t, compat.RoleAssistant, req.Messages[0].Role)
	require.Equal(t, "我来帮你调用上车点选择工具。", req.Messages[0].Content)
	require.Equal(t, compat.RoleAssistant, req.Messages[1].Role)
	require.Len(t, req.Messages[1].ToolCalls, 1)
	require.Equal(t, "select_get_on_address", req.Messages[1].ToolCalls[0].Function.Name)
	require.Equal(t, compat.RoleTool, req.Messages[2].Role)
	require.Equal(t, `{"title":"凌波门"}`, req.Messages[2].Content)
	require.Equal(t, compat.RoleAssistant, req.Messages[3].Role)
	require.Equal(t, "正在为你搜索武汉大学。", req.Messages[3].Content)
	require.Equal(t, compat.RoleUser, req.Messages[4].Role)
	require.Equal(t, "ask", req.Messages[4].Content)
}

func newSessionEvent(author string, msg compat.Message) event.Event {
	return event.Event{
		Response: &compat.Response{
			Done: true,
			Choices: []compat.Choice{
				{Index: 0, Message: msg},
			},
		},
		Author: author,
	}
}

// Test that session summary is merged into existing system message.
func TestProcessRequest_SessionSummary_MergesIntoSystemMessage(t *testing.T) {
	// Create session with summary
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "Session summary content",
				UpdatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	// Test case 1: Request has system message followed by user message
	req1 := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("existing system prompt"),
			compat.NewUserMessage("user question"),
		},
	}

	inv1 := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	inv1.AgentName = "test-agent"

	p1 := NewContentRequestProcessor(WithAddSessionSummary(true))
	p1.ProcessRequest(context.Background(), inv1, req1, nil)
	raw, ok := inv1.GetState(contentHasSessionSummaryStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	// Should have 3 messages: system (merged), user, current request
	require.Equal(t, 3, len(req1.Messages))
	require.Equal(t, compat.RoleSystem, req1.Messages[0].Role)
	require.Contains(t, req1.Messages[0].Content, "existing system prompt")
	require.Contains(t, req1.Messages[0].Content, "Session summary content")
	require.Equal(t, compat.RoleUser, req1.Messages[1].Role)
	require.Equal(t, "user question", req1.Messages[1].Content)
	require.Equal(t, compat.RoleUser, req1.Messages[2].Role)
	require.Equal(t, "current request", req1.Messages[2].Content)

	// Test case 2: Request has only user message (no system message)
	req2 := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("user question"),
		},
	}

	inv2 := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	inv2.AgentName = "test-agent"

	p2 := NewContentRequestProcessor(WithAddSessionSummary(true))
	p2.ProcessRequest(context.Background(), inv2, req2, nil)
	raw, ok = inv2.GetState(contentHasSessionSummaryStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	// Should have 3 messages: summary system, user, current request
	require.Equal(t, 3, len(req2.Messages))
	require.Equal(t, compat.RoleSystem, req2.Messages[0].Role)
	require.Equal(t, NewContentRequestProcessor().formatSummary("Session summary content"), req2.Messages[0].Content)
	require.Equal(t, compat.RoleUser, req2.Messages[1].Role)
	require.Equal(t, "user question", req2.Messages[1].Content)
	require.Equal(t, compat.RoleUser, req2.Messages[2].Role)
	require.Equal(t, "current request", req2.Messages[2].Content)

	// Test case 3: Request has multiple system messages (only first one gets merged)
	req3 := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system 1"),
			compat.NewSystemMessage("system 2"),
			compat.NewUserMessage("user question"),
		},
	}

	inv3 := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	inv3.AgentName = "test-agent"

	p3 := NewContentRequestProcessor(WithAddSessionSummary(true))
	p3.ProcessRequest(context.Background(), inv3, req3, nil)
	raw, ok = inv3.GetState(contentHasSessionSummaryStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	// Should have 4 messages: system1 (merged with summary), system2, user, current
	// request. Summary merges into first system message.
	require.Equal(t, 4, len(req3.Messages))
	require.Equal(t, compat.RoleSystem, req3.Messages[0].Role)
	require.Contains(t, req3.Messages[0].Content, "system 1")
	require.Contains(t, req3.Messages[0].Content, "Session summary content")
	require.Equal(t, compat.RoleSystem, req3.Messages[1].Role)
	require.Equal(t, "system 2", req3.Messages[1].Content)
	require.Equal(t, compat.RoleUser, req3.Messages[2].Role)
	require.Equal(t, "user question", req3.Messages[2].Content)
	require.Equal(t, compat.RoleUser, req3.Messages[3].Role)
	require.Equal(t, "current request", req3.Messages[3].Content)
}

func TestProcessRequest_SessionSummary_ResumesLatestCoveredToolRound(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	userMsg := compat.NewUserMessage("run the task")
	toolCallMsg := compat.Message{
		Role:    compat.RoleAssistant,
		Content: "Starting with step 1.",
		ToolCalls: []compat.ToolCall{{
			Type: "function",
			ID:   "call_1",
			Function: compat.FunctionDefinitionParam{
				Name:      "step_worker",
				Arguments: []byte(`{"step":1}`),
			},
		}},
	}
	toolResultMsg := compat.Message{
		Role:     compat.RoleTool,
		ToolID:   "call_1",
		ToolName: "step_worker",
		Content:  strings.Repeat("large-result;", 16),
	}
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "step 1 completed successfully",
				UpdatedAt: baseTime.Add(2 * time.Second),
				Boundary: session.NewSummaryBoundaryWithEventID(
					"test-agent",
					baseTime.Add(2*time.Second),
					"tool-result-1",
				),
			},
		},
		Events: []event.Event{
			{
				ID:           "user-1",
				Author:       "user",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime,
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Choices: []compat.Choice{{Index: 0, Message: userMsg}},
				},
			},
			{
				ID:           "tool-call-1",
				Author:       "test-agent",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime.Add(time.Second),
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Choices: []compat.Choice{{Index: 0, Message: toolCallMsg}},
				},
			},
			{
				ID:           "tool-result-1",
				Author:       "test-agent",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime.Add(2 * time.Second),
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Object:  compat.ObjectTypeToolResponse,
					Choices: []compat.Choice{{Index: 0, Message: toolResultMsg}},
				},
			},
		},
	}

	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationID("inv1"),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(userMsg),
		agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
	)
	inv.AgentName = "test-agent"

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system prompt"),
		},
	}
	p := NewContentRequestProcessor(
		WithAddSessionSummary(true),
		WithEnableContextCompaction(true),
		WithContextCompactionToolResultMaxTokens(10),
	)
	p.ProcessRequest(context.Background(), inv, req, nil)

	raw, ok := inv.GetState(contentHasCompactedToolResultsStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	require.Len(t, req.Messages, 4)
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.Contains(t, req.Messages[0].Content, "system prompt")
	require.Contains(t, req.Messages[0].Content,
		"step 1 completed successfully")
	require.True(t, compat.MessagesEqual(userMsg, req.Messages[1]))
	require.Equal(t, compat.RoleAssistant, req.Messages[2].Role)
	require.Equal(t, "Starting with step 1.", req.Messages[2].Content)
	require.Len(t, req.Messages[2].ToolCalls, 1)
	require.Equal(t, "call_1", req.Messages[2].ToolCalls[0].ID)
	require.JSONEq(t, `{"step":1}`, string(
		req.Messages[2].ToolCalls[0].Function.Arguments,
	))
	require.Equal(t, compat.RoleTool, req.Messages[3].Role)
	require.Equal(t, "call_1", req.Messages[3].ToolID)
	require.Equal(t, "step_worker", req.Messages[3].ToolName)
	require.Contains(t, req.Messages[3].Content, compactedToolResultPlaceholder)
	require.NotContains(t, req.Messages[3].Content, "large-result;")
	require.Equal(
		t,
		`{"step":1}`,
		string(sess.Events[1].Choices[0].Message.ToolCalls[0].Function.Arguments),
		"resume-tail compaction must not mutate session history",
	)
}

func TestProcessRequest_SessionSummary_PreservesSmallLatestToolRound(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	userMsg := compat.NewUserMessage("run the task")
	toolCallMsg := compat.Message{
		Role:    compat.RoleAssistant,
		Content: "Starting with step 1.",
		ToolCalls: []compat.ToolCall{{
			Type: "function",
			ID:   "call_1",
			Function: compat.FunctionDefinitionParam{
				Name:      "step_worker",
				Arguments: []byte(`{"step":1}`),
			},
		}},
	}
	toolResultMsg := compat.Message{
		Role:     compat.RoleTool,
		ToolID:   "call_1",
		ToolName: "step_worker",
		Content:  "small result",
	}
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "step 1 completed successfully",
				UpdatedAt: baseTime.Add(2 * time.Second),
				Boundary: session.NewSummaryBoundaryWithEventID(
					"test-agent",
					baseTime.Add(2*time.Second),
					"evt-tool-result",
				),
			},
		},
		Events: []event.Event{
			{
				Author:       "user",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime,
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Choices: []compat.Choice{{Index: 0, Message: userMsg}},
				},
			},
			{
				Author:       "test-agent",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime.Add(time.Second),
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Choices: []compat.Choice{{Index: 0, Message: toolCallMsg}},
				},
			},
			{
				ID:           "evt-tool-result",
				Author:       "test-agent",
				RequestID:    "req1",
				InvocationID: "inv1",
				Timestamp:    baseTime.Add(2 * time.Second),
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done:    true,
					Object:  compat.ObjectTypeToolResponse,
					Choices: []compat.Choice{{Index: 0, Message: toolResultMsg}},
				},
			},
		},
	}

	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationID("inv1"),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(userMsg),
		agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
	)
	inv.AgentName = "test-agent"

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system prompt"),
		},
	}
	p := NewContentRequestProcessor(WithAddSessionSummary(true))
	p.ProcessRequest(context.Background(), inv, req, nil)

	raw, ok := inv.GetState(contentHasCompactedToolResultsStateKey)
	require.False(t, ok)
	require.Nil(t, raw)

	require.Len(t, req.Messages, 4)
	require.True(t, compat.MessagesEqual(userMsg, req.Messages[1]))
	require.Equal(t, toolCallMsg.ToolCalls, req.Messages[2].ToolCalls)
	require.Equal(t, compat.RoleTool, req.Messages[3].Role)
	require.Equal(t, "call_1", req.Messages[3].ToolID)
	require.Equal(t, "step_worker", req.Messages[3].ToolName)
	require.Equal(t, "small result", req.Messages[3].Content)
}

func TestContentRequestProcessor_HasCompactedCurrentInvocationToolResults(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	since := baseTime.Add(2 * time.Second)

	t.Run("nil invocation", func(t *testing.T) {
		p := NewContentRequestProcessor()
		require.False(t, p.hasCompactedCurrentInvocationToolResults(nil, since))
	})

	t.Run("missing request metadata", func(t *testing.T) {
		p := NewContentRequestProcessor()
		inv := &agent.Invocation{
			Session: &session.Session{},
		}
		require.False(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("ignores non matching and non tool events", func(t *testing.T) {
		p := NewContentRequestProcessor()
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					{
						RequestID:    "req2",
						InvocationID: "inv1",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								"call_1",
								"worker",
								"result",
							)}},
						},
					},
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewAssistantMessage(
								"not a tool result",
							)}},
						},
					},
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    since.Add(time.Second),
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								"call_2",
								"worker",
								"after cutoff",
							)}},
						},
					},
				},
			}),
		)
		require.False(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("respects branch filter", func(t *testing.T) {
		p := NewContentRequestProcessor(WithBranchFilterMode(BranchFilterModeExact))
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationEventFilterKey("wanted"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						FilterKey:    "other",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								"call_1",
								"worker",
								"result",
							)}},
						},
					},
				},
			}),
		)
		require.False(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("detects compacted tool result", func(t *testing.T) {
		p := NewContentRequestProcessor(
			WithEnableContextCompaction(true),
			WithContextCompactionToolResultMaxTokens(1),
		)
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								"call_1",
								"worker",
								"result result",
							)}},
						},
					},
				},
			}),
		)
		require.True(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("does not infer missing tool name from branch history", func(t *testing.T) {
		reusedToolID := "call_reused"
		toolCallEvent := func(filterKey, toolName string, ts time.Time) event.Event {
			return event.Event{
				RequestID:    "req1",
				InvocationID: "inv1",
				FilterKey:    filterKey,
				Timestamp:    ts,
				Version:      event.CurrentVersion,
				Response: &compat.Response{
					Done: true,
					Choices: []compat.Choice{{Index: 0, Message: compat.Message{
						Role: compat.RoleAssistant,
						ToolCalls: []compat.ToolCall{{
							ID: reusedToolID,
							Function: compat.FunctionDefinitionParam{
								Name:      toolName,
								Arguments: []byte(`{}`),
							},
						}},
					}}},
				},
			}
		}
		p := NewContentRequestProcessor(
			WithBranchFilterMode(BranchFilterModeExact),
			WithEnableContextCompaction(true),
			WithContextCompactionKeepToolNames("session_load"),
			WithContextCompactionToolResultMaxTokens(1),
		)
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationEventFilterKey("wanted"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					toolCallEvent("other", "session_load", baseTime),
					toolCallEvent("wanted", "shell", baseTime.Add(time.Second)),
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						FilterKey:    "wanted",
						Timestamp:    baseTime.Add(1500 * time.Millisecond),
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								reusedToolID,
								"",
								"result result",
							)}},
						},
					},
				},
			}),
		)

		require.True(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("ignores kept tool result", func(t *testing.T) {
		p := NewContentRequestProcessor(
			WithEnableContextCompaction(true),
			WithContextCompactionKeepToolNames("session_load"),
		)
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.Message{
								Role: compat.RoleAssistant,
								ToolCalls: []compat.ToolCall{{
									ID: "call_1",
									Function: compat.FunctionDefinitionParam{
										Name:      "session_load",
										Arguments: []byte(`{}`),
									},
								}},
							}}},
						},
					},
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    baseTime.Add(time.Second),
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{{Index: 0, Message: compat.NewToolMessage(
								"call_1",
								"session_load",
								"kept result",
							)}},
						},
					},
				},
			}),
		)
		require.False(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})

	t.Run("detects compacted tool result in later choice", func(t *testing.T) {
		p := NewContentRequestProcessor(
			WithEnableContextCompaction(true),
			WithContextCompactionToolResultMaxTokens(1),
		)
		inv := agent.NewInvocation(
			agent.WithInvocationID("inv1"),
			agent.WithInvocationRunOptions(agent.RunOptions{RequestID: "req1"}),
			agent.WithInvocationSession(&session.Session{
				Events: []event.Event{
					{
						RequestID:    "req1",
						InvocationID: "inv1",
						Timestamp:    baseTime,
						Version:      event.CurrentVersion,
						Response: &compat.Response{
							Done: true,
							Choices: []compat.Choice{
								{Index: 0, Message: compat.NewAssistantMessage("progress")},
								{Index: 1, Message: compat.NewToolMessage(
									"call_1",
									"worker",
									"result result",
								)},
							},
						},
					},
				},
			}),
		)
		require.True(t, p.hasCompactedCurrentInvocationToolResults(inv, since))
	})
}

// Test additional edge cases for session summary insertion.
func TestProcessRequest_SessionSummary_EdgeCases(t *testing.T) {
	// Create session with summary
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "Session summary content",
				UpdatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	// Test case 1: Empty request messages
	req1 := &compat.Request{
		Messages: []compat.Message{},
	}

	inv1 := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	inv1.AgentName = "test-agent"

	p1 := NewContentRequestProcessor(WithAddSessionSummary(true))
	p1.ProcessRequest(context.Background(), inv1, req1, nil)
	raw, ok := inv1.GetState(contentHasSessionSummaryStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	// Should have 2 messages: summary system, current request
	require.Equal(t, 2, len(req1.Messages))
	require.Equal(t, compat.RoleSystem, req1.Messages[0].Role)
	require.Equal(t, NewContentRequestProcessor().formatSummary("Session summary content"), req1.Messages[0].Content)
	require.Equal(t, compat.RoleUser, req1.Messages[1].Role)
	require.Equal(t, "current request", req1.Messages[1].Content)

	// Test case 2: Only system messages
	req2 := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system prompt"),
		},
	}

	inv2 := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
	)
	inv2.AgentName = "test-agent"

	p2 := NewContentRequestProcessor(WithAddSessionSummary(true))
	p2.ProcessRequest(context.Background(), inv2, req2, nil)
	raw, ok = inv2.GetState(contentHasSessionSummaryStateKey)
	require.True(t, ok)
	require.Equal(t, true, raw)

	// Should have 2 messages: system (merged with summary), current request
	require.Equal(t, 2, len(req2.Messages))
	require.Equal(t, compat.RoleSystem, req2.Messages[0].Role)
	require.Contains(t, req2.Messages[0].Content, "system prompt")
	require.Contains(t, req2.Messages[0].Content, "Session summary content")
	require.Equal(t, compat.RoleUser, req2.Messages[1].Role)
	require.Equal(t, "current request", req2.Messages[1].Content)
}

func TestContentRequestProcessor_AggregatePrefixSummaries_Sorted(
	t *testing.T,
) {
	p := NewContentRequestProcessor()
	summaries := map[string]*session.Summary{
		"app/b": {
			Summary: "b",
			UpdatedAt: time.Date(
				2023, 1, 2, 12, 0, 0, 0, time.UTC,
			),
		},
		"app": {
			Summary: "root",
			UpdatedAt: time.Date(
				2023, 1, 1, 12, 0, 0, 0, time.UTC,
			),
		},
		"app/a": {
			Summary: "a",
			UpdatedAt: time.Date(
				2023, 1, 3, 12, 0, 0, 0, time.UTC,
			),
		},
		"other": {
			Summary: "ignored",
			UpdatedAt: time.Date(
				2023, 1, 4, 12, 0, 0, 0, time.UTC,
			),
		},
	}

	got, updatedAt := p.aggregatePrefixSummaries(summaries, "app")
	require.Equal(t, "root\n\na\n\nb", got)
	require.Equal(t,
		time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		updatedAt.CutoffTime(),
	)
}

func TestPromptCachePrefixStability_DynamicSystemTail(t *testing.T) {
	const (
		approxRunesPerToken = 4
		cachePrefixTokens   = 1024
		stableSysATokens    = 900
		stableSysBTokens    = 300

		stableSysAChar = "A"
		stableSysBChar = "B"

		summaryRun1 = "summary-run-1"
		summaryRun2 = "summary-run-2"
	)

	cachePrefixRunes := cachePrefixTokens * approxRunesPerToken

	sysA := strings.Repeat(
		stableSysAChar,
		stableSysATokens*approxRunesPerToken,
	)
	sysB := strings.Repeat(
		stableSysBChar,
		stableSysBTokens*approxRunesPerToken,
	)

	build := func(summaryText string) *compat.Request {
		sess := &session.Session{
			Summaries: map[string]*session.Summary{
				"test-agent": {
					Summary:   summaryText,
					UpdatedAt: time.Now(),
				},
			},
		}
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationEventFilterKey("test-agent"),
			agent.WithInvocationMessage(compat.NewUserMessage("hi")),
		)
		inv.AgentName = "test-agent"

		req := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage(sysA),
				compat.NewSystemMessage(sysB),
			},
		}

		p := NewContentRequestProcessor(WithAddSessionSummary(true))
		p.ProcessRequest(context.Background(), inv, req, nil)
		return req
	}

	render := func(messages []compat.Message) string {
		var b strings.Builder
		for _, msg := range messages {
			b.WriteString(msg.Role.String())
			b.WriteString(":")
			b.WriteString(msg.Content)
			b.WriteString("\n")
		}
		return b.String()
	}

	firstRunes := func(text string, maxRunes int) string {
		if maxRunes <= 0 {
			return ""
		}
		r := []rune(text)
		if len(r) <= maxRunes {
			return text
		}
		return string(r[:maxRunes])
	}

	reqRun1 := build(summaryRun1)
	reqRun2 := build(summaryRun2)

	prefixRun1 := firstRunes(render(reqRun1.Messages), cachePrefixRunes)
	prefixRun2 := firstRunes(render(reqRun2.Messages), cachePrefixRunes)

	// New behavior: summary is merged into the first system message.
	// This means the first system message's content changes across runs,
	// so the cacheable prefix does NOT stay stable.
	// The summary content is merged into sysA, changing its content.
	require.NotEqual(t, prefixRun1, prefixRun2)
	// The first message contains both sysA content and the summary.
	require.Contains(t, reqRun1.Messages[0].Content, summaryRun1)
	require.Contains(t, reqRun2.Messages[0].Content, summaryRun2)

	// Verify that sysB (second system message) stays stable.
	require.Equal(t, sysB, reqRun1.Messages[1].Content)
	require.Equal(t, sysB, reqRun2.Messages[1].Content)
}

func newSessionEventWithBranch(author, filterKey, branch string, msg compat.Message) event.Event {
	return event.Event{
		Response: &compat.Response{
			Done: true,
			Choices: []compat.Choice{
				{Index: 0, Message: msg},
			},
		},
		Author:    author,
		FilterKey: filterKey,
		Branch:    branch,
		Version:   event.CurrentVersion,
	}
}

// Test that session summary is injected as user message when SessionSummaryInjectionUser is used.
func TestProcessRequest_SessionSummary_UserInjectionMode(t *testing.T) {
	summaryTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "user mode summary content",
				UpdatedAt: summaryTime,
			},
		},
	}

	t.Run("no_existing_system_prompt", func(t *testing.T) {
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationEventFilterKey("test-agent"),
			agent.WithInvocationMessage(compat.NewUserMessage("current request")),
		)
		inv.AgentName = "test-agent"

		req := &compat.Request{}
		p := NewContentRequestProcessor(
			WithAddSessionSummary(true),
			WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
		)
		p.ProcessRequest(context.Background(), inv, req, nil)

		raw, ok := inv.GetState(contentHasSessionSummaryStateKey)
		require.True(t, ok)
		require.Equal(t, true, raw)

		// Should have at least 1 message. No system message should be present.
		require.GreaterOrEqual(t, len(req.Messages), 1)
		for _, msg := range req.Messages {
			require.NotEqual(t, compat.RoleSystem, msg.Role,
				"user injection mode should not produce system messages for summary")
		}
		// First message should contain the summary.
		require.Contains(t, req.Messages[0].Content, "user mode summary content")
		require.Equal(t, compat.RoleUser, req.Messages[0].Role)
	})

	t.Run("with_existing_system_prompt", func(t *testing.T) {
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationEventFilterKey("test-agent"),
			agent.WithInvocationMessage(compat.NewUserMessage("current request")),
		)
		inv.AgentName = "test-agent"

		req := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage("existing system prompt"),
			},
		}
		p := NewContentRequestProcessor(
			WithAddSessionSummary(true),
			WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
		)
		p.ProcessRequest(context.Background(), inv, req, nil)

		// System prompt should remain untouched (summary not merged into it).
		require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
		require.Equal(t, "existing system prompt", req.Messages[0].Content)
		// Summary should be a user message after system.
		found := false
		for _, msg := range req.Messages {
			if msg.Role == compat.RoleUser && strings.Contains(msg.Content, "user mode summary content") {
				found = true
				break
			}
		}
		require.True(t, found, "should find summary content in a user message")
	})

	t.Run("merges_with_first_user_history", func(t *testing.T) {
		historyTime := summaryTime.Add(time.Second)
		sessWithHistory := &session.Session{
			Summaries: map[string]*session.Summary{
				"test-agent": {
					Summary:   "summary to merge",
					UpdatedAt: summaryTime,
				},
			},
			Events: []event.Event{
				{
					Response: &compat.Response{
						Choices: []compat.Choice{{
							Message: compat.NewUserMessage("hello from history"),
						}},
					},
					Author:    "user",
					Timestamp: historyTime,
					FilterKey: "test-agent",
					Version:   event.CurrentVersion,
				},
			},
		}

		inv := agent.NewInvocation(
			agent.WithInvocationSession(sessWithHistory),
			agent.WithInvocationEventFilterKey("test-agent"),
			agent.WithInvocationMessage(compat.NewUserMessage("current request")),
		)
		inv.AgentName = "test-agent"

		req := &compat.Request{}
		p := NewContentRequestProcessor(
			WithAddSessionSummary(true),
			WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
		)
		p.ProcessRequest(context.Background(), inv, req, nil)

		// The first user message in history should have the summary merged into it.
		require.GreaterOrEqual(t, len(req.Messages), 1)
		require.Equal(t, compat.RoleUser, req.Messages[0].Role)
		require.Contains(t, req.Messages[0].Content, "summary to merge",
			"summary should be merged into first user history message")
		require.Contains(t, req.Messages[0].Content, "hello from history",
			"original history content should be preserved")
	})

	t.Run("system_mode_default_still_works", func(t *testing.T) {
		inv := agent.NewInvocation(
			agent.WithInvocationSession(sess),
			agent.WithInvocationEventFilterKey("test-agent"),
			agent.WithInvocationMessage(compat.NewUserMessage("current request")),
		)
		inv.AgentName = "test-agent"

		req := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage("existing system prompt"),
			},
		}
		// Default mode (system) should merge into system message.
		p := NewContentRequestProcessor(
			WithAddSessionSummary(true),
		)
		p.ProcessRequest(context.Background(), inv, req, nil)

		require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
		require.Contains(t, req.Messages[0].Content, "user mode summary content")
		require.Contains(t, req.Messages[0].Content, "existing system prompt")
	})
}

// Test formatSummaryForUser output.
func TestFormatSummaryForUser(t *testing.T) {
	p := NewContentRequestProcessor()
	result := p.formatSummaryForUser("test summary")
	require.Contains(t, result, "test summary")
	require.Contains(t, result, "Context from previous interactions")
	require.Contains(t, result, "summary_of_previous_interactions")
}

// Test prependSummaryUserMessage edge cases.
func TestPrependSummaryUserMessage(t *testing.T) {
	p := NewContentRequestProcessor()

	t.Run("empty_summary", func(t *testing.T) {
		msgs := []compat.Message{compat.NewUserMessage("hello")}
		result := p.prependSummaryUserMessage("", msgs, nil)
		require.Equal(t, msgs, result)
	})

	t.Run("empty_messages", func(t *testing.T) {
		result := p.prependSummaryUserMessage("some summary", nil, nil)
		require.Len(t, result, 1)
		require.Equal(t, compat.RoleUser, result[0].Role)
		require.Contains(t, result[0].Content, "some summary")
	})

	t.Run("later_user_message_merges", func(t *testing.T) {
		msgs := []compat.Message{
			compat.NewAssistantMessage("hi"),
			compat.NewUserMessage("hello"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, nil)
		require.Len(t, result, 2)
		require.Equal(t, compat.RoleAssistant, result[0].Role)
		require.Equal(t, compat.RoleUser, result[1].Role)
		require.Contains(t, result[1].Content, "some summary")
		require.Contains(t, result[1].Content, "hello")
	})

	t.Run("first_message_is_user_merges", func(t *testing.T) {
		msgs := []compat.Message{
			compat.NewUserMessage("hello"),
			compat.NewAssistantMessage("hi"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, nil)
		require.Len(t, result, 2)
		require.Equal(t, compat.RoleUser, result[0].Role)
		require.Contains(t, result[0].Content, "some summary")
		require.Contains(t, result[0].Content, "hello")
		require.Equal(t, compat.RoleAssistant, result[1].Role)
	})

	t.Run("zero_role_history_message_merges_as_user", func(t *testing.T) {
		msgs := []compat.Message{
			{Content: "zero role current request"},
			compat.NewAssistantMessage("hi"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, nil)
		require.Len(t, result, 2)
		require.Equal(t, compat.Role(""), result[0].Role)
		require.Contains(t, result[0].Content, "some summary")
		require.Contains(t, result[0].Content, "zero role current request")
		require.Equal(t, compat.RoleAssistant, result[1].Role)
	})

	t.Run("does_not_mutate_original", func(t *testing.T) {
		original := []compat.Message{
			compat.NewUserMessage("original content"),
		}
		originalContent := original[0].Content
		result := p.prependSummaryUserMessage("summary", original, nil)
		// Original should not be mutated.
		require.Equal(t, originalContent, original[0].Content)
		// Result should have merged content.
		require.Contains(t, result[0].Content, "summary")
		require.Contains(t, result[0].Content, "original content")
	})

	t.Run("first_history_user_preferred_over_req_prefix_user", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			compat.NewUserMessage("few-shot user example"),
		}
		msgs := []compat.Message{
			compat.NewUserMessage("history user"),
			compat.NewAssistantMessage("history assistant"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// Summary should stay attached to the live history user message.
		require.Equal(t, "few-shot user example", prefix[len(prefix)-1].Content)
		require.Len(t, result, 2)
		require.Contains(t, result[0].Content, "some summary")
		require.Contains(t, result[0].Content, "history user")
		require.Equal(t, compat.RoleAssistant, result[1].Role)
	})

	t.Run("later_history_user_preferred_over_req_prefix_user", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			compat.NewUserMessage("few-shot user example"),
		}
		msgs := []compat.Message{
			compat.NewAssistantMessage("history assistant"),
			compat.NewUserMessage("current user"),
			compat.NewToolMessage("tool-1", "search", "tool result"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// Even when history starts with assistant/tool output, the summary
		// should merge into the first available user message in history/current.
		require.Equal(t, "few-shot user example", prefix[len(prefix)-1].Content)
		require.Len(t, result, 3)
		require.Equal(t, compat.RoleAssistant, result[0].Role)
		require.Contains(t, result[1].Content, "some summary")
		require.Contains(t, result[1].Content, "current user")
		require.Equal(t, compat.RoleTool, result[2].Role)
	})

	t.Run("later_empty_history_user_preferred_over_req_prefix_user", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			compat.NewUserMessage("few-shot user example"),
		}
		msgs := []compat.Message{
			compat.NewAssistantMessage("history assistant"),
			{Role: compat.RoleUser, Content: ""},
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// An empty user message later in history/current should still win over
		// the trailing prefix user, and the summary should become its content.
		require.Equal(t, "few-shot user example", prefix[len(prefix)-1].Content)
		require.Len(t, result, 2)
		require.Equal(t, compat.RoleAssistant, result[0].Role)
		require.Equal(t, compat.RoleUser, result[1].Role)
		require.Contains(t, result[1].Content, "some summary")
	})

	t.Run("req_prefix_ends_with_user_merges_into_prefix_as_fallback", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			compat.NewUserMessage("few-shot user example"),
		}
		msgs := []compat.Message{
			compat.NewAssistantMessage("history assistant"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// With no user history/current message to attach to, fall back to the
		// trailing prefix user message.
		require.Contains(t, prefix[len(prefix)-1].Content, "some summary")
		require.Contains(t, prefix[len(prefix)-1].Content, "few-shot user example")
		require.Equal(t, msgs, result)
	})

	t.Run("req_prefix_ends_with_zero_role_user_merges_as_fallback", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			{Content: "zero role user example"},
		}
		msgs := []compat.Message{
			compat.NewAssistantMessage("history assistant"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		require.Equal(t, compat.Role(""), prefix[len(prefix)-1].Role)
		require.Contains(t, prefix[len(prefix)-1].Content, "some summary")
		require.Contains(t, prefix[len(prefix)-1].Content, "zero role user example")
		require.Equal(t, msgs, result)
	})

	t.Run("req_prefix_ends_with_system_no_merge_into_prefix", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
		}
		msgs := []compat.Message{
			compat.NewUserMessage("history user"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// Should merge into first history user message instead.
		require.Len(t, result, 1)
		require.Contains(t, result[0].Content, "some summary")
		require.Contains(t, result[0].Content, "history user")
		// Prefix should be untouched.
		require.Equal(t, "system prompt", prefix[0].Content)
	})

	t.Run("req_prefix_ends_with_empty_user_sets_content", func(t *testing.T) {
		prefix := []compat.Message{
			compat.NewSystemMessage("system prompt"),
			{Role: compat.RoleUser, Content: ""},
		}
		msgs := []compat.Message{
			compat.NewAssistantMessage("history assistant"),
		}
		result := p.prependSummaryUserMessage("some summary", msgs, prefix)
		// Summary should be set (not appended) on the empty user message.
		require.Contains(t, prefix[len(prefix)-1].Content, "some summary")
		require.Equal(t, msgs, result)
	})

	t.Run("custom_formatter_empty_return_skips", func(t *testing.T) {
		custom := NewContentRequestProcessor(
			WithSummaryFormatter(func(_ string) string { return "" }),
		)
		msgs := []compat.Message{compat.NewUserMessage("hello")}
		result := custom.prependSummaryUserMessage("some summary", msgs, nil)
		require.Equal(t, msgs, result)
	})
}

// Test formatSummaryForUser with custom SummaryFormatter.
func TestFormatSummaryForUser_CustomFormatter(t *testing.T) {
	p := NewContentRequestProcessor(
		WithSummaryFormatter(func(s string) string {
			return "CUSTOM: " + s
		}),
	)
	result := p.formatSummaryForUser("test summary")
	require.Equal(t, "CUSTOM: test summary", result)
}

// Test WithSessionSummaryInjectionMode option.
func TestWithSessionSummaryInjectionMode(t *testing.T) {
	p := NewContentRequestProcessor(
		WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
	)
	require.Equal(t, SessionSummaryInjectionUser, p.SessionSummaryInjectionMode)

	p2 := NewContentRequestProcessor(
		WithSessionSummaryInjectionMode(SessionSummaryInjectionSystem),
	)
	require.Equal(t, SessionSummaryInjectionSystem, p2.SessionSummaryInjectionMode)

	// Unknown value defaults to system.
	p3 := NewContentRequestProcessor(
		WithSessionSummaryInjectionMode("unknown"),
	)
	require.Equal(t, SessionSummaryInjectionSystem, p3.SessionSummaryInjectionMode)
}

// Test user injection mode with injected context ending in user message.
func TestProcessRequest_SessionSummary_UserMode_PrefersCurrentUserOverInjectedContext(t *testing.T) {
	summaryTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "injected context summary",
				UpdatedAt: summaryTime,
			},
		},
	}

	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewUserMessage("current request")),
		agent.WithInvocationRunOptions(agent.RunOptions{
			InjectedContextMessages: []compat.Message{
				compat.NewUserMessage("injected context user msg"),
			},
		}),
	)
	inv.AgentName = "test-agent"

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system prompt"),
		},
	}
	p := NewContentRequestProcessor(
		WithAddSessionSummary(true),
		WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
	)
	p.ProcessRequest(context.Background(), inv, req, nil)

	// The summary should stay attached to the live current user message instead
	// of being merged into the injected-context prefix user message.
	// System prompt should NOT contain the summary.
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.NotContains(t, req.Messages[0].Content, "injected context summary",
		"system prompt should not contain summary in user injection mode")

	// Injected-context user message should remain untouched.
	require.Equal(t, "injected context user msg", req.Messages[1].Content)

	// Find the current user message and verify summary was merged into it.
	foundMerged := false
	for _, msg := range req.Messages {
		if msg.Role == compat.RoleUser &&
			strings.Contains(msg.Content, "current request") &&
			strings.Contains(msg.Content, "injected context summary") {
			foundMerged = true
			break
		}
	}
	require.True(t, foundMerged,
		"summary should be merged into the current user message")
}

func TestProcessRequest_SessionSummary_UserMode_FallsBackToInjectedContextUser(t *testing.T) {
	summaryTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	sess := &session.Session{
		Summaries: map[string]*session.Summary{
			"test-agent": {
				Summary:   "fallback summary",
				UpdatedAt: summaryTime,
			},
		},
		Events: []event.Event{
			{
				Response: &compat.Response{
					Choices: []compat.Choice{{
						Message: compat.NewAssistantMessage("history assistant"),
					}},
				},
				Author:    "test-agent",
				Timestamp: summaryTime.Add(time.Second),
				FilterKey: "test-agent",
				Version:   event.CurrentVersion,
			},
		},
	}

	inv := agent.NewInvocation(
		agent.WithInvocationSession(sess),
		agent.WithInvocationEventFilterKey("test-agent"),
		agent.WithInvocationMessage(compat.NewToolMessage("tool-1", "search", "tool result")),
		agent.WithInvocationRunOptions(agent.RunOptions{
			InjectedContextMessages: []compat.Message{
				compat.NewUserMessage("injected context user msg"),
			},
		}),
	)
	inv.AgentName = "test-agent"

	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("system prompt"),
		},
	}
	p := NewContentRequestProcessor(
		WithAddSessionSummary(true),
		WithSessionSummaryInjectionMode(SessionSummaryInjectionUser),
	)
	p.ProcessRequest(context.Background(), inv, req, nil)

	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.NotContains(t, req.Messages[0].Content, "fallback summary")

	// No user history/current message is available, so fallback merges into the
	// injected-context user message instead of inserting a standalone user block.
	require.Contains(t, req.Messages[1].Content, "injected context user msg")
	require.Contains(t, req.Messages[1].Content, "fallback summary")
}
