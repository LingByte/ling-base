package agent

// Placeholders inserted when a transcript is structurally broken (typically
// because an earlier turn was interrupted, refused, or recorded incompletely).
// We never hide the repair — the model sees these strings and can adjust.
const (
	emptyContentPlaceholder  = "[Empty assistant turn — content was not recorded]"
	abandonedToolPlaceholder = "[Tool call abandoned — no result was recorded; treat the call as unanswered]"
)

// sanitizeMessages makes a message list satisfy the API's structural rules
// before sending. It repairs three classes of corruption — typically inherited
// from a resumed transcript — so a /goal run (or any resume) can't be poisoned
// by an earlier broken turn:
//
//  1. Empty content: nil or [] Content gets a placeholder text block (the API
//     rejects with "messages.<i>.content: Field required" otherwise).
//  2. Orphan tool_use: every tool_use block must be answered by a tool_result
//     in the IMMEDIATELY following message. Missing pairings get a synthetic
//     tool_result with is_error=true; if the next message isn't a user one
//     (or doesn't exist), a fresh user message is inserted.
//  3. Consecutive same-role messages: the API requires user/assistant
//     alternation, so runs of the same role are merged into one message
//     (their content blocks concatenated in order).
//
// Clean conversations return the input slice unchanged. Repaired ones get an
// independent slice; original messages are never mutated.
func sanitizeMessages(messages []Message) []Message {
	if !needsSanitize(messages) {
		return messages
	}
	out := append([]Message(nil), messages...)

	// (1) fill empty content
	for i, m := range out {
		if len(m.Content) == 0 {
			out[i].Content = []ContentBlock{NewTextBlock(emptyContentPlaceholder)}
		}
	}

	// (2) pair every tool_use with a tool_result in the next message
	for i := 0; i < len(out); i++ {
		if out[i].Role != "assistant" {
			continue
		}
		var missing []string
		for _, b := range out[i].Content {
			if b.IsToolUse() && !hasToolResultAt(out, i+1, b.ID) {
				missing = append(missing, b.ID)
			}
		}
		if len(missing) == 0 {
			continue
		}
		results := make([]ContentBlock, 0, len(missing))
		for _, id := range missing {
			results = append(results, NewToolResultBlock(id, abandonedToolPlaceholder, true))
		}
		if i+1 < len(out) && out[i+1].Role == "user" {
			// Prepend the synthetic results so they sit adjacent to the tool_use.
			merged := make([]ContentBlock, 0, len(results)+len(out[i+1].Content))
			merged = append(merged, results...)
			merged = append(merged, out[i+1].Content...)
			out[i+1].Content = merged
		} else {
			// Insert a fresh user message at i+1 carrying the synthetic results.
			out = append(out, Message{})
			copy(out[i+2:], out[i+1:])
			out[i+1] = NewUserMessageBlocks(results...)
		}
	}

	// (3) collapse consecutive same-role messages (alternation requirement)
	merged := make([]Message, 0, len(out))
	for _, m := range out {
		if n := len(merged); n > 0 && merged[n-1].Role == m.Role {
			combined := make([]ContentBlock, 0, len(merged[n-1].Content)+len(m.Content))
			combined = append(combined, merged[n-1].Content...)
			combined = append(combined, m.Content...)
			merged[n-1].Content = combined
			continue
		}
		merged = append(merged, m)
	}
	return merged
}

// hasToolResultAt reports whether messages[i] is a user message that carries a
// tool_result for toolUseID.
func hasToolResultAt(messages []Message, i int, toolUseID string) bool {
	if i < 0 || i >= len(messages) || messages[i].Role != "user" {
		return false
	}
	for _, b := range messages[i].Content {
		if b.IsToolResult() && b.ToolUseID == toolUseID {
			return true
		}
	}
	return false
}

// needsSanitize is the fast path — when it returns false, sanitizeMessages
// returns the input unchanged with no allocation.
func needsSanitize(messages []Message) bool {
	var prev string
	for i, m := range messages {
		if len(m.Content) == 0 {
			return true
		}
		if i > 0 && m.Role == prev {
			return true
		}
		prev = m.Role
		for _, b := range m.Content {
			if b.IsToolUse() && !hasToolResultAt(messages, i+1, b.ID) {
				return true
			}
		}
	}
	return false
}
