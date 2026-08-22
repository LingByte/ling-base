//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package processor

import (
	"context"
	"testing"
	"time"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	sessioninmemory "github.com/LingByte/ling-base/agentkit/session/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onDemandSessionService struct {
	session.Service
}

type onDemandSearchOnlyService struct {
	session.Service
}

func (s *onDemandSessionService) SearchEvents(
	context.Context,
	session.EventSearchRequest,
) ([]session.EventSearchResult, error) {
	return nil, nil
}

func (s *onDemandSessionService) GetEventWindow(
	context.Context,
	session.EventWindowRequest,
) (*session.EventWindow, error) {
	return &session.EventWindow{}, nil
}

func (s *onDemandSearchOnlyService) SearchEvents(
	context.Context,
	session.EventSearchRequest,
) ([]session.EventSearchResult, error) {
	return nil, nil
}

func TestOnDemandSessionRequestProcessor_ProcessRequest(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("base instruction"),
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: &onDemandSessionService{Service: sessioninmemory.NewSessionService()},
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 2)
	assert.Contains(t, req.Messages[0].Content, "Progressive disclosure for session history is available.")
	assert.Contains(t, req.Messages[0].Content, "base instruction")

	first := req.Messages[0].Content
	p.ProcessRequest(context.Background(), inv, req, nil)
	assert.Equal(t, first, req.Messages[0].Content)
}

func TestOnDemandSessionRequestProcessor_SkipsWithoutSupport(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session: session.NewSession("app", "user", "sess"),
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 1)
	assert.Equal(t, compat.RoleUser, req.Messages[0].Role)
}

func TestOnDemandSessionRequestProcessor_LoadOnlyOverview(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: sessioninmemory.NewSessionService(),
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	assert.Contains(t, req.Messages[0].Content, "Exact session history loading is available.")
	assert.NotContains(t, req.Messages[0].Content, "Use session_search before session_load")
}

func TestOnDemandSessionRequestProcessor_SearchOnlyOverview(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session: session.NewSession("app", "user", "sess"),
		SessionService: &onDemandSearchOnlyService{
			Service: sessioninmemory.NewSessionService(),
		},
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	assert.Contains(t, req.Messages[0].Content, "Use session_search")
	assert.NotContains(t, req.Messages[0].Content, "Exact session history loading is available.")
}

func TestOnDemandSessionRequestProcessor_EmptySystemMessage(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage(""),
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: sessioninmemory.NewSessionService(),
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	assert.Equal(t, onDemandSessionLoadOverview, req.Messages[0].Content)

	p.ProcessRequest(context.Background(), nil, nil, nil)
}

func TestOnDemandSessionRequestProcessor_InsertsSystemMessage(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: &onDemandSessionService{Service: sessioninmemory.NewSessionService()},
	}

	p.ProcessRequest(context.Background(), inv, req, nil)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	assert.Contains(t, req.Messages[0].Content, "scope=current_session")
	assert.Equal(t, compat.RoleUser, req.Messages[1].Role)
}

func TestOnDemandSessionRequestProcessor_EmitsInstructionEvent(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		InvocationID:   "invocation",
		AgentName:      "agent",
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: sessioninmemory.NewSessionService(),
	}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)
	require.Len(t, req.Messages, 2)
	select {
	case got := <-ch:
		require.NotNil(t, got)
		require.Equal(t, compat.ObjectTypePreprocessingInstruction, got.Object)
	case <-time.After(time.Second):
		t.Fatal("expected preprocessing instruction event")
	}
}

func TestOnDemandSessionRequestProcessor_RebuildForContextCompaction(t *testing.T) {
	p := NewOnDemandSessionRequestProcessor()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("base instruction"),
			compat.NewUserMessage("hello"),
		},
	}
	inv := &agent.Invocation{
		Session:        session.NewSession("app", "user", "sess"),
		SessionService: &onDemandSessionService{Service: sessioninmemory.NewSessionService()},
	}

	require.True(t, p.SupportsContextCompactionRebuild(inv))
	p.RebuildRequestForContextCompaction(context.Background(), inv, req)
	require.Len(t, req.Messages, 2)
	assert.Contains(t, req.Messages[0].Content, "Progressive disclosure for session history is available.")
}
