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
	compat "github.com/LingByte/ling-base/relay/compat"
)

// MessageOp interface defines operations that can be applied to message arrays.
// This provides atomic combination of multiple operations for state updates.
type MessageOp interface {
	Apply([]compat.Message) []compat.Message
}

// AppendMessages provides append capability for atomic combination.
// It can also be used for backward compatibility in unified expression.
type AppendMessages struct{ Items []compat.Message }

// Apply implements the MessageOp interface.
func (op AppendMessages) Apply(dst []compat.Message) []compat.Message {
	return append(dst, op.Items...)
}

// ReplaceLastUser replaces the last user message in the durable history.
// If no user message is found, it falls back to appending a new user message.
type ReplaceLastUser struct{ Content string }

// Apply implements the MessageOp interface.
func (op ReplaceLastUser) Apply(dst []compat.Message) []compat.Message {
	for i := len(dst) - 1; i >= 0; i-- {
		if dst[i].Role == compat.RoleUser {
			// Replace the content while preserving other fields.
			dst[i] = compat.Message{
				Role:             compat.RoleUser,
				Content:          op.Content,
				ContentParts:     dst[i].ContentParts,
				ToolID:           dst[i].ToolID,
				ToolName:         dst[i].ToolName,
				ToolCalls:        dst[i].ToolCalls,
				ReasoningContent: dst[i].ReasoningContent,
			}
			return dst
		}
	}
	// No user message at the end of history, append a new one.
	return append(dst, compat.NewUserMessage(op.Content))
}

// RemoveAllMessages clears all messages for full rebuild scenarios.
// Used sparingly: for reordering/trimming when starting fresh.
type RemoveAllMessages struct{}

// Apply implements the MessageOp interface.
func (RemoveAllMessages) Apply(_ []compat.Message) []compat.Message { return nil }
