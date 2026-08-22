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
	"testing"

	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/stretchr/testify/require"
)

func TestAppendMessages(t *testing.T) {
	base := []compat.Message{compat.NewUserMessage("a")}
	op := AppendMessages{Items: []compat.Message{compat.NewAssistantMessage("b")}}
	out := op.Apply(base)
	require.Len(t, out, 2)
	require.Equal(t, compat.RoleUser, out[0].Role)
	require.Equal(t, compat.RoleAssistant, out[1].Role)
}

func TestReplaceLastUser(t *testing.T) {
	messages := []compat.Message{
		compat.NewUserMessage("u1"),
		compat.NewAssistantMessage("a1"),
		compat.NewUserMessage("u2"),
	}
	out := (ReplaceLastUser{Content: "u2-new"}).Apply(messages)
	require.Len(t, out, 3)
	require.Equal(t, compat.RoleUser, out[2].Role)
	require.Equal(t, "u2-new", out[2].Content)
}

func TestReplaceLastUserNoUserAppends(t *testing.T) {
	messages := []compat.Message{compat.NewAssistantMessage("a1")}
	out := (ReplaceLastUser{Content: "u-new"}).Apply(messages)
	require.Len(t, out, 2)
	require.Equal(t, compat.RoleUser, out[1].Role)
	require.Equal(t, "u-new", out[1].Content)
}

func TestRemoveAllMessages(t *testing.T) {
	base := []compat.Message{compat.NewUserMessage("x")}
	out := (RemoveAllMessages{}).Apply(base)
	require.Nil(t, out)
}
