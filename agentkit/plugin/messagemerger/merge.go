//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package messagemerger

import compat "github.com/LingByte/ling-base/relay/compat"

func mergeConsecutiveMessages(
	messages []compat.Message,
	separator string,
) []compat.Message {
	if len(messages) == 0 {
		return messages
	}
	merged := make([]compat.Message, 0, len(messages))
	for _, msg := range messages {
		if len(merged) == 0 {
			merged = append(merged, cloneMessage(msg))
			continue
		}
		last := merged[len(merged)-1]
		if !canMergeConsecutiveMessage(last) ||
			!canMergeConsecutiveMessage(msg) ||
			last.Role != msg.Role {
			merged = append(merged, cloneMessage(msg))
			continue
		}
		merged[len(merged)-1] = mergeMessage(last, msg, separator)
	}
	return merged
}

func canMergeConsecutiveMessage(msg compat.Message) bool {
	if msg.ToolID != "" || msg.ToolName != "" {
		return false
	}
	switch msg.Role {
	case compat.RoleSystem, compat.RoleUser, compat.RoleAssistant:
		return true
	default:
		return false
	}
}

func mergeMessage(
	dst compat.Message,
	src compat.Message,
	separator string,
) compat.Message {
	dst.ReasoningContent = joinMessageText(
		dst.ReasoningContent,
		src.ReasoningContent,
		separator,
	)
	if len(dst.ContentParts) == 0 && len(src.ContentParts) == 0 {
		dst.Content = joinMessageText(dst.Content, src.Content, separator)
	} else {
		dst.ContentParts = mergeMessageContentParts(dst, src, separator)
		dst.Content = ""
	}
	if len(src.ToolCalls) > 0 {
		dst.ToolCalls = append(dst.ToolCalls, src.ToolCalls...)
	}
	return dst
}

func joinMessageText(first, second, separator string) string {
	if first == "" {
		return second
	}
	if second == "" {
		return first
	}
	return first + separator + second
}

func cloneMessage(msg compat.Message) compat.Message {
	cloned := msg
	if len(msg.ContentParts) > 0 {
		cloned.ContentParts = append(
			[]compat.ContentPart(nil),
			msg.ContentParts...,
		)
	}
	if len(msg.ToolCalls) > 0 {
		cloned.ToolCalls = append([]compat.ToolCall(nil), msg.ToolCalls...)
	}
	return cloned
}

func mergeMessageContentParts(
	dst compat.Message,
	src compat.Message,
	separator string,
) []compat.ContentPart {
	parts := orderedMessageContentParts(dst)
	if shouldInsertMessageSeparator(dst, src, separator) {
		parts = append(parts, textContentPart(separator))
	}
	return append(parts, orderedMessageContentParts(src)...)
}

func orderedMessageContentParts(msg compat.Message) []compat.ContentPart {
	parts := make([]compat.ContentPart, 0, len(msg.ContentParts)+1)
	if msg.Content != "" {
		parts = append(parts, textContentPart(msg.Content))
	}
	return append(parts, msg.ContentParts...)
}

func shouldInsertMessageSeparator(
	dst compat.Message,
	src compat.Message,
	separator string,
) bool {
	if separator == "" {
		return false
	}
	return messageEndsWithText(dst) && messageStartsWithText(src)
}

func messageStartsWithText(msg compat.Message) bool {
	if msg.Content != "" {
		return true
	}
	if len(msg.ContentParts) == 0 {
		return false
	}
	first := msg.ContentParts[0]
	return first.Type == compat.ContentTypeText &&
		first.Text != nil &&
		*first.Text != ""
}

func messageEndsWithText(msg compat.Message) bool {
	if len(msg.ContentParts) == 0 {
		return msg.Content != ""
	}
	last := msg.ContentParts[len(msg.ContentParts)-1]
	return last.Type == compat.ContentTypeText &&
		last.Text != nil &&
		*last.Text != ""
}

func textContentPart(text string) compat.ContentPart {
	return compat.ContentPart{
		Type: compat.ContentTypeText,
		Text: compat.StringPtr(text),
	}
}
