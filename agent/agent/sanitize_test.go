package agent

import (
	"encoding/json"
	"testing"
)

func TestSanitizeMessagesFillsEmptyContent(t *testing.T) {
	raw := []byte(`{"role":"assistant","content":null,"stop_reason":"end_turn"}`)
	var bad Message
	if err := json.Unmarshal(raw, &bad); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(bad.Content) != 0 {
		t.Fatalf("setup: expected empty content from null, got %d block(s)", len(bad.Content))
	}

	good := NewUserMessage("ok")
	in := []Message{good, bad, good}
	out := sanitizeMessages(in)

	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if len(out[1].Content) != 1 || out[1].Content[0].Type != BlockText {
		t.Errorf("empty-content message not repaired: %#v", out[1].Content)
	}
	if out[1].Content[0].Text != emptyContentPlaceholder {
		t.Errorf("placeholder text = %q, want %q", out[1].Content[0].Text, emptyContentPlaceholder)
	}
	if len(out[0].Content) != 1 || out[0].Content[0].Text != "ok" {
		t.Errorf("non-empty message was mutated: %#v", out[0].Content)
	}
}

// asstMessage builds an assistant Message from content blocks.
func asstMessage(blocks ...ContentBlock) Message {
	return Message{
		Role:    "assistant",
		Content: blocks,
	}
}

func TestSanitizeMessagesPassthrough(t *testing.T) {
	user := NewUserMessage("hi")
	asst := asstMessage(NewTextBlock("hello"))
	in := []Message{user, asst, user}
	out := sanitizeMessages(in)
	if &out[0] != &in[0] {
		t.Error("clean input should be returned without reallocation")
	}
	if got := sanitizeMessages(nil); got != nil {
		t.Errorf("nil input = %#v, want nil", got)
	}
}

func TestSanitizeMessagesRepairsOrphanToolUse(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]any{"cmd": "ls"})
	asstToolUse := asstMessage(
		NewToolUseBlock("chatcmpl-tool-90ed3808", "Bash", inputJSON),
	)
	userText := NewUserMessage("iteration prompt")
	out := sanitizeMessages([]Message{asstToolUse, userText})

	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if tr := out[1].Content[0]; !tr.IsToolResult() || tr.ToolUseID != "chatcmpl-tool-90ed3808" {
		t.Fatalf("missing synthetic tool_result; got %#v", out[1].Content)
	}
	if len(out[1].Content) != 2 || out[1].Content[1].Type != BlockText {
		t.Errorf("original content lost; got %#v", out[1].Content)
	}
}

func TestSanitizeMessagesOrphanToolUseAtEndInsertsUser(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]any{})
	asst := asstMessage(
		NewToolUseBlock("orphan-1", "Bash", inputJSON),
	)
	out := sanitizeMessages([]Message{asst})
	if len(out) != 2 || out[1].Role != "user" {
		t.Fatalf("expected an injected user message; got %d msgs, last role %v", len(out), out[len(out)-1].Role)
	}
	if tr := out[1].Content[0]; !tr.IsToolResult() || tr.ToolUseID != "orphan-1" {
		t.Errorf("injected message missing the synthetic result; got %#v", out[1].Content)
	}
}

func TestSanitizeMessagesMergesConsecutiveSameRole(t *testing.T) {
	mk := func(s string) Message {
		return NewUserMessage(s)
	}
	asst := asstMessage(NewTextBlock("ok"))
	out := sanitizeMessages([]Message{asst, mk("a"), mk("b"), mk("c")})
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if len(out[1].Content) != 3 {
		t.Errorf("merged user should hold 3 blocks; got %d", len(out[1].Content))
	}
}
