//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package message

import (
	"testing"

	"github.com/stretchr/testify/assert"

	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestIsEmptyAssistantMessage(t *testing.T) {
	assert.True(t, IsEmptyAssistantMessage(compat.Message{
		Role: compat.RoleAssistant,
	}))
	assert.True(t, IsEmptyAssistantMessage(compat.Message{
		Role:             compat.RoleAssistant,
		ReasoningContent: "reasoning without visible payload",
	}))
	assert.False(t, IsEmptyAssistantMessage(compat.Message{
		Role: compat.RoleUser,
	}))
	assert.False(t, IsEmptyAssistantMessage(compat.Message{
		Role:    compat.RoleAssistant,
		Content: "visible content",
	}))
	assert.False(t, IsEmptyAssistantMessage(compat.Message{
		Role: compat.RoleAssistant,
		ContentParts: []compat.ContentPart{
			{Type: compat.ContentTypeText},
		},
	}))
	assert.False(t, IsEmptyAssistantMessage(compat.Message{
		Role: compat.RoleAssistant,
		ToolCalls: []compat.ToolCall{
			{ID: "call_1"},
		},
	}))
}
