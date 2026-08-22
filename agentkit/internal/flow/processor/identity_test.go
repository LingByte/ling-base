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

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestIdentityProc_Request(t *testing.T) {
	invocation := &agent.Invocation{
		AgentName:    "test-agent",
		InvocationID: "test-123",
	}

	tests := []struct {
		name         string
		agentName    string
		description  string
		messages     []compat.Message
		wantMessages int
		wantContent  string
	}{
		{
			name:         "adds identity with name and description",
			agentName:    "TestBot",
			description:  "A helpful testing assistant",
			messages:     []compat.Message{},
			wantMessages: 1,
			wantContent:  "You are TestBot. A helpful testing assistant",
		},
		{
			name:         "adds identity with name only",
			agentName:    "TestBot",
			description:  "",
			messages:     []compat.Message{},
			wantMessages: 1,
			wantContent:  "You are TestBot.",
		},
		{
			name:         "adds identity with description only",
			agentName:    "",
			description:  "A helpful assistant",
			messages:     []compat.Message{},
			wantMessages: 1,
			wantContent:  "A helpful assistant",
		},
		{
			name:         "no identity information",
			agentName:    "",
			description:  "",
			messages:     []compat.Message{},
			wantMessages: 0,
			wantContent:  "",
		},
		{
			name:         "prepends identity to existing system message",
			agentName:    "TestBot",
			description:  "A helpful assistant",
			messages:     []compat.Message{compat.NewSystemMessage("You have access to tools.")},
			wantMessages: 1,
			wantContent:  "You are TestBot. A helpful assistant",
		},
		{
			name:         "doesn't duplicate identity when already exists",
			agentName:    "TestBot",
			description:  "",
			messages:     []compat.Message{compat.NewSystemMessage("You are TestBot. You have access to tools.")},
			wantMessages: 1,
			wantContent:  "You are TestBot.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewIdentityRequestProcessor(
				tt.agentName,
				tt.description,
				WithAddNameToInstruction(true),
			)
			eventCh := make(chan *event.Event, 10)
			ctx := context.Background()

			request := &compat.Request{Messages: tt.messages}
			processor.ProcessRequest(ctx, invocation, request, eventCh)

			if len(request.Messages) != tt.wantMessages {
				t.Errorf("ProcessRequest() got %d messages, want %d", len(request.Messages), tt.wantMessages)
			}

			// Check if identity was added correctly
			if tt.wantContent != "" && tt.wantMessages > 0 {
				found := false
				for _, msg := range request.Messages {
					if msg.Role == compat.RoleSystem && strings.Contains(msg.Content, tt.wantContent) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ProcessRequest() identity content '%s' not found in system messages", tt.wantContent)
				}
			}
		})
	}
}
