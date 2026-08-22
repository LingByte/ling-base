//
// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT
//

package relaymodel

import (
	"testing"

	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestNewModel(t *testing.T) {
	m := New("gpt-4o",
		WithAPIKey("test-key"),
		WithBaseURL("https://api.openai.com"),
		WithChannel(ChannelOpenAI),
		WithContextWindow(128000),
	)
	if m == nil {
		t.Fatal("New returned nil")
	}
	info := m.Info()
	if info.Name != "gpt-4o" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "gpt-4o")
	}
	if info.ContextWindow != 128000 {
		t.Errorf("Info().ContextWindow = %d, want 128000", info.ContextWindow)
	}
}

func TestNewModelDefaults(t *testing.T) {
	m := New("claude-sonnet-4-5-20250929")
	if m.name != "claude-sonnet-4-5-20250929" {
		t.Errorf("name = %q", m.name)
	}
	if m.channel != ChannelOpenAI {
		t.Errorf("default channel = %q, want %q", m.channel, ChannelOpenAI)
	}
}

func TestTranslateRequestSystemExtraction(t *testing.T) {
	m := New("gpt-4o", WithAPIKey("k"))
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage("You are helpful."),
			compat.NewUserMessage("Hello"),
		},
		GenerationConfig: compat.GenerationConfig{
			MaxTokens: ptrInt(1024),
			Stream:    true,
		},
	}
	richReq, err := m.translateRequest(req)
	if err != nil {
		t.Fatalf("translateRequest: %v", err)
	}
	if richReq.System != "You are helpful." {
		t.Errorf("System = %q", richReq.System)
	}
	if len(richReq.Messages) != 1 {
		t.Fatalf("Messages len = %d, want 1", len(richReq.Messages))
	}
	if richReq.Messages[0].Role != "user" {
		t.Errorf("Messages[0].Role = %q, want user", richReq.Messages[0].Role)
	}
	if richReq.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", richReq.MaxTokens)
	}
	if !richReq.Stream {
		t.Error("Stream should be true")
	}
}

func TestTranslateRequestToolMessage(t *testing.T) {
	m := New("gpt-4o", WithAPIKey("k"))
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("What's the weather?"),
			compat.NewAssistantMessage("Let me check."),
			compat.NewToolMessage("call_123", "get_weather", "Sunny, 22°C"),
		},
	}
	richReq, err := m.translateRequest(req)
	if err != nil {
		t.Fatalf("translateRequest: %v", err)
	}
	// 3 messages, but system is extracted, so 3 (user, assistant, tool→user)
	if len(richReq.Messages) != 3 {
		t.Fatalf("Messages len = %d, want 3", len(richReq.Messages))
	}
	// The tool message should be converted to a user message with tool_result block.
	toolMsg := richReq.Messages[2]
	if toolMsg.Role != "user" {
		t.Errorf("tool message role = %q, want user", toolMsg.Role)
	}
	if len(toolMsg.Content) != 1 || toolMsg.Content[0].Type != "tool_result" {
		t.Fatalf("expected tool_result block, got %+v", toolMsg.Content)
	}
	if toolMsg.Content[0].ToolUseID != "call_123" {
		t.Errorf("ToolUseID = %q", toolMsg.Content[0].ToolUseID)
	}
}

func TestTranslateRequestAssistantWithToolCalls(t *testing.T) {
	m := New("gpt-4o", WithAPIKey("k"))
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("What's the weather?"),
			{
				Role: compat.RoleAssistant,
				ToolCalls: []compat.ToolCall{
					{
						Type: "function",
						ID:   "call_456",
						Function: compat.FunctionDefinitionParam{
							Name:      "get_weather",
							Arguments: []byte(`{"city":"Paris"}`),
						},
					},
				},
			},
		},
	}
	richReq, err := m.translateRequest(req)
	if err != nil {
		t.Fatalf("translateRequest: %v", err)
	}
	assistantMsg := richReq.Messages[1]
	if assistantMsg.Role != "assistant" {
		t.Errorf("role = %q, want assistant", assistantMsg.Role)
	}
	// Should have a tool_use block.
	found := false
	for _, b := range assistantMsg.Content {
		if b.Type == "tool_use" && b.ID == "call_456" && b.Name == "get_weather" {
			found = true
			if string(b.Input) != `{"city":"Paris"}` {
				t.Errorf("Input = %q, want {\"city\":\"Paris\"}", string(b.Input))
			}
		}
	}
	if !found {
		t.Errorf("expected tool_use block in assistant message, got %+v", assistantMsg.Content)
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"end_turn", "stop"},
		{"stop", "stop"},
		{"tool_use", "tool_calls"},
		{"max_tokens", "length"},
		{"content_filter", "content_filter"},
		{"custom_reason", "custom_reason"},
		{"", ""},
	}
	for _, tt := range tests {
		got := mapStopReason(tt.in)
		if got != tt.want {
			t.Errorf("mapStopReason(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestClientOrBuildMissingAPIKey(t *testing.T) {
	m := New("gpt-4o") // no API key
	_, err := m.clientOrBuild()
	if err == nil {
		t.Error("expected error when API key is missing")
	}
}

func ptrInt(v int) *int { return &v }
