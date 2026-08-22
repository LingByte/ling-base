//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package runner

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/stretchr/testify/require"
)

// capturingRunner captures the arguments passed to Run for assertions.
type capturingRunner struct {
	lastUserID    string
	lastSessionID string
	lastMessage   compat.Message
	lastRunOpts   agent.RunOptions
}

func (c *capturingRunner) Run(ctx context.Context, userID, sessionID string, message compat.Message, runOpts ...agent.RunOption) (<-chan *event.Event, error) {
	c.lastUserID = userID
	c.lastSessionID = sessionID
	c.lastMessage = message

	var ro agent.RunOptions
	for _, opt := range runOpts {
		opt(&ro)
	}
	c.lastRunOpts = ro

	ch := make(chan *event.Event)
	close(ch)
	return ch, nil
}

func (c *capturingRunner) Close() error {
	return nil
}

func TestRunWithMessages_PassesHistoryAndLatestUser(t *testing.T) {
	r := &capturingRunner{}
	msgs := []compat.Message{
		compat.NewSystemMessage("sys"),
		compat.NewAssistantMessage("a1"),
		compat.NewUserMessage("u1"),
		compat.NewAssistantMessage("a2"),
		compat.NewUserMessage("latest-user"),
	}

	_, err := RunWithMessages(context.Background(), r, "u", "s", msgs)
	require.NoError(t, err)

	// Latest user message should be passed as the invocation message.
	require.Equal(t, compat.RoleUser, r.lastMessage.Role)
	require.Equal(t, "latest-user", r.lastMessage.Content)

	// The run option should carry the full message history.
	require.Equal(t, len(msgs), len(r.lastRunOpts.Messages))
	for i := range msgs {
		require.Equal(t, msgs[i].Role, r.lastRunOpts.Messages[i].Role)
		require.Equal(t, msgs[i].Content, r.lastRunOpts.Messages[i].Content)
	}
}

func TestRunWithMessages_NoUserFallback(t *testing.T) {
	r := &capturingRunner{}
	msgs := []compat.Message{
		compat.NewSystemMessage("sys"),
		compat.NewAssistantMessage("only-assistant"),
	}

	_, err := RunWithMessages(context.Background(), r, "u", "s", msgs)
	require.NoError(t, err)

	// No user message found -> zero-value message is passed.
	require.Equal(t, "", r.lastMessage.Content)
	require.Equal(t, compat.Role(""), r.lastMessage.Role)

	// Still carries the full history in run options.
	require.Equal(t, len(msgs), len(r.lastRunOpts.Messages))
}
