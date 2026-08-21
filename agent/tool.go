// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package agent implements a coding agent harness built on top of
// [github.com/LingByte/ling-base/relay]. It provides an agent loop
// that drives an LLM to use tools (read, write, edit, bash) for
// software engineering tasks.
//
// Quick start:
//
//	client := relay.New(relay.WithProvider(openai.NewProvider("sk-xxx")))
//	agent := agent.New(client, "gpt-4o", agent.WithTools(
//		tools.NewRead(cwd),
//		tools.NewWrite(cwd),
//		tools.NewEdit(cwd),
//		tools.NewBash(cwd),
//	))
//	err := agent.Prompt(ctx, "read main.go and explain it", func(ev agent.Event) {
//		switch e := ev.(type) {
//		case agent.EvAssistantText:
//			fmt.Print(e.Text)
//		}
//	})
package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool is a capability the agent can invoke. Implementations must be
// safe for concurrent use.
type Tool interface {
	// Name is the unique tool id shown to the LLM.
	Name() string
	// Description is a one-line summary shown to the LLM.
	Description() string
	// Schema is a JSON Schema object describing Execute's args.
	Schema() json.RawMessage
	// Execute runs the tool. progress may be called with partial
	// textual output for UIs; it is not sent to the LLM.
	Execute(ctx context.Context, args json.RawMessage, progress func(string)) (ToolResult, error)
}

// ToolResult is the outcome of Tool.Execute.
type ToolResult struct {
	// Content is sent back to the LLM as the tool's output.
	Content string
	// IsError marks this result as an error to the LLM.
	IsError bool
}

// Registry is a name→Tool map.
type Registry map[string]Tool

// NewRegistry builds a Registry from a list of tools.
func NewRegistry(tools ...Tool) Registry {
	r := Registry{}
	for _, t := range tools {
		r[t.Name()] = t
	}
	return r
}

// ToolSpec is the JSON definition sent to the LLM so it knows what
// tools are available and how to call them.
type ToolSpec struct {
	Type     string      `json:"type"` // always "function"
	Function ToolSpecFn  `json:"function"`
}

type ToolSpecFn struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Specs returns the tool definitions to advertise to the LLM,
// sorted by name for stable prompt caching.
func (r Registry) Specs() []ToolSpec {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	// sort for stable order (prompt caching)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	out := make([]ToolSpec, 0, len(r))
	for _, name := range names {
		t := r[name]
		out = append(out, ToolSpec{
			Type: "function",
			Function: ToolSpecFn{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return out
}

// Get returns the tool with the given name, or nil.
func (r Registry) Get(name string) Tool {
	return r[name]
}

// ToolCall is a parsed tool call from the LLM.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// String returns a human-readable summary for logging.
func (tc ToolCall) String() string {
	return fmt.Sprintf("%s(%s)", tc.Name, string(tc.Arguments))
}
