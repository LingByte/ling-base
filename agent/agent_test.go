// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

// mockProvider implements relay.Provider for testing.
type mockProvider struct {
	name     string
	apiType  int
	adaptor  mockAdaptor
	baseURL  string
	apiKey   string
}

func (m *mockProvider) Name() string    { return m.name }
func (m *mockProvider) ApiType() int    { return m.apiType }
func (m *mockProvider) Adaptor() interface{} { return m.adaptor }

// mockAdaptor is a minimal adaptor that returns canned responses.
type mockAdaptor struct{}

// Since we can't easily mock the full relay pipeline, we test the
// tool registry and event types instead.

func TestRegistrySpecs(t *testing.T) {
	r := NewRegistry(
		&fakeTool{name: "read", desc: "Read a file", schema: json.RawMessage(`{"type":"object"}`)},
		&fakeTool{name: "write", desc: "Write a file", schema: json.RawMessage(`{"type":"object"}`)},
	)
	specs := r.Specs()
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	// Specs should be sorted by name.
	if specs[0].Function.Name != "read" {
		t.Errorf("expected first spec to be 'read', got %s", specs[0].Function.Name)
	}
	if specs[1].Function.Name != "write" {
		t.Errorf("expected second spec to be 'write', got %s", specs[1].Function.Name)
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry(
		&fakeTool{name: "read", desc: "Read", schema: json.RawMessage(`{}`)},
	)
	if r.Get("read") == nil {
		t.Error("expected to find 'read' tool")
	}
	if r.Get("nonexistent") != nil {
		t.Error("expected nil for unknown tool")
	}
}

func TestToolCallString(t *testing.T) {
	tc := ToolCall{
		ID:        "call_123",
		Name:      "read",
		Arguments: json.RawMessage(`{"path":"main.go"}`),
	}
	s := tc.String()
	if s != `read({"path":"main.go"})` {
		t.Errorf("unexpected string: %s", s)
	}
}

func TestParseArgs(t *testing.T) {
	cfg := ParseArgs([]string{"--provider", "claude", "--model", "claude-sonnet-4", "hello world"})
	if cfg.Provider != "claude" {
		t.Errorf("expected provider=claude, got %s", cfg.Provider)
	}
	if cfg.Model != "claude-sonnet-4" {
		t.Errorf("expected model=claude-sonnet-4, got %s", cfg.Model)
	}
	if cfg.Prompt != "hello world" {
		t.Errorf("expected prompt='hello world', got %s", cfg.Prompt)
	}
}

func TestParseArgsJSON(t *testing.T) {
	cfg := ParseArgs([]string{"--json", "test prompt"})
	if !cfg.JSON {
		t.Error("expected JSON=true")
	}
	if cfg.Prompt != "test prompt" {
		t.Errorf("expected prompt='test prompt', got %s", cfg.Prompt)
	}
}

func TestDefaultModel(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "gpt-4o"},
		{"claude", "claude-sonnet-4-20250514"},
		{"deepseek", "deepseek-chat"},
		{"unknown", "gpt-4o"},
	}
	for _, tt := range tests {
		got := DefaultModel(tt.provider)
		if got != tt.expected {
			t.Errorf("DefaultModel(%q) = %q, want %q", tt.provider, got, tt.expected)
		}
	}
}

func TestEventToJSON(t *testing.T) {
	ev := EvAssistantText{Text: "hello"}
	m := eventToJSON(ev)
	if m["type"] != "assistant_text" {
		t.Errorf("expected type=assistant_text, got %v", m["type"])
	}
	if m["text"] != "hello" {
		t.Errorf("expected text=hello, got %v", m["text"])
	}
}

// fakeTool is a minimal Tool implementation for testing.
type fakeTool struct {
	name   string
	desc   string
	schema json.RawMessage
}

func (f *fakeTool) Name() string                       { return f.name }
func (f *fakeTool) Description() string                 { return f.desc }
func (f *fakeTool) Schema() json.RawMessage             { return f.schema }
func (f *fakeTool) Execute(ctx context.Context, args json.RawMessage, progress func(string)) (ToolResult, error) {
	return ToolResult{Content: "ok"}, nil
}

// Ensure relay types are referenced (avoids unused import in test).
var _ relay.Message = dto.Message{}
var _ = context.Background
