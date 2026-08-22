//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package processor

import (
	"context"
	"strings"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	"github.com/LingByte/ling-base/agentkit/tool"
)

type instructionTestAgent struct {
	tools []tool.Tool
}

func (a *instructionTestAgent) Run(
	context.Context, *agent.Invocation,
) (<-chan *event.Event, error) {
	return nil, nil
}

func (a *instructionTestAgent) Tools() []tool.Tool { return a.tools }

func (a *instructionTestAgent) Info() agent.Info { return agent.Info{} }

func (a *instructionTestAgent) SubAgents() []agent.Agent { return nil }

func (a *instructionTestAgent) FindSubAgent(string) agent.Agent { return nil }

type instructionTestTool struct{ decl tool.Declaration }

func (t instructionTestTool) Declaration() *tool.Declaration { return &t.decl }

func (instructionTestTool) Call(context.Context, []byte) (any, error) {
	return nil, nil
}

func TestInstructionProc_JSONInjection_StructuredOutput(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
	}

	p := NewInstructionRequestProcessor(
		"base instruction",
		"base system",
		WithStructuredOutputSchema(schema),
	)

	req := &compat.Request{Messages: []compat.Message{compat.NewUserMessage("hi")}}
	inv := &agent.Invocation{AgentName: "a", InvocationID: "id-1"}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)

	if len(req.Messages) == 0 || req.Messages[0].Role != compat.RoleSystem {
		t.Fatalf("expected a system message to be created")
	}
	content := req.Messages[0].Content
	if !strings.Contains(content, "IMPORTANT: Return ONLY a JSON object") {
		t.Errorf("expected JSON instructions to be injected")
	}
	if !strings.Contains(content, `"type": "object"`) {
		t.Errorf("expected schema content to be present in instructions")
	}
}

func TestInstructionProc_JSONInjection_StructuredOutput_AllowsTools(
	t *testing.T,
) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
	}

	p := NewInstructionRequestProcessor(
		"base instruction",
		"base system",
		WithStructuredOutputSchema(schema),
	)

	req := &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("hi")},
	}
	inv := &agent.Invocation{
		AgentName:    "a",
		InvocationID: "id-1",
		Message:      compat.NewUserMessage("hi"),
		Agent: &instructionTestAgent{
			tools: []tool.Tool{
				instructionTestTool{
					decl: tool.Declaration{Name: "test"},
				},
			},
		},
	}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)

	if len(req.Messages) == 0 || req.Messages[0].Role != compat.RoleSystem {
		t.Fatalf("expected a system message to be created")
	}
	content := req.Messages[0].Content
	if !strings.Contains(content, "You MAY call tools") {
		t.Errorf("expected tools to be permitted in JSON instructions")
	}
	if strings.Contains(content, "IMPORTANT: Return ONLY a JSON object") {
		t.Errorf("expected tools-aware JSON instructions")
	}
	if !strings.Contains(content, "return ONLY a JSON object") {
		t.Errorf("expected final JSON-only rule to be present")
	}
}

func TestInstructionProc_JSONInjection_OutputSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"y": map[string]any{"type": "number"},
		},
	}

	p := NewInstructionRequestProcessor(
		"",
		"sys",
		WithOutputSchema(schema),
	)

	req := &compat.Request{Messages: []compat.Message{}}
	inv := &agent.Invocation{AgentName: "a", InvocationID: "id-2"}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)

	if len(req.Messages) == 0 || req.Messages[0].Role != compat.RoleSystem {
		t.Fatalf("expected a system message to be created")
	}
	content := req.Messages[0].Content
	if !strings.Contains(content, "IMPORTANT: Return ONLY a JSON object") {
		t.Errorf("expected JSON instructions to be injected for output_schema")
	}
	if !strings.Contains(content, `"y"`) {
		t.Errorf("expected schema properties to be present in instructions")
	}
}

func TestInstructionProc_JSONInjection_UsesInvocationStructuredOutput(t *testing.T) {
	staticSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
	}
	runSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"z": map[string]any{"type": "integer"},
		},
	}

	p := NewInstructionRequestProcessor(
		"",
		"",
		WithStructuredOutputSchema(staticSchema),
	)

	req := &compat.Request{Messages: []compat.Message{compat.NewUserMessage("hi")}}
	inv := &agent.Invocation{
		AgentName:    "a",
		InvocationID: "id-3",
		StructuredOutput: &compat.StructuredOutput{
			Type: compat.StructuredOutputJSONSchema,
			JSONSchema: &compat.JSONSchemaConfig{
				Name:   "run_output",
				Schema: runSchema,
			},
		},
	}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)

	if len(req.Messages) == 0 || req.Messages[0].Role != compat.RoleSystem {
		t.Fatalf("expected a system message to be created")
	}
	content := req.Messages[0].Content
	if !strings.Contains(content, `"z"`) {
		t.Errorf("expected invocation schema to be present in instructions")
	}
	if strings.Contains(content, `"x"`) {
		t.Errorf("did not expect static schema to be used when invocation schema is present")
	}
}

func TestInstructionProc_JSONInjection_AppendsSchemaAfterPlaceholderRendering(
	t *testing.T,
) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{
				"type":        "string",
				"description": "literal {research_topics}",
			},
		},
	}

	p := NewInstructionRequestProcessor(
		"topic {research_topics}",
		"",
		WithOutputSchema(schema),
	)

	req := &compat.Request{Messages: []compat.Message{compat.NewUserMessage("hi")}}
	inv := &agent.Invocation{
		AgentName:    "a",
		InvocationID: "id-4",
		Session: &session.Session{State: session.StateMap{
			"research_topics": []byte(`"AI"`),
		}},
	}
	ch := make(chan *event.Event, 1)

	p.ProcessRequest(context.Background(), inv, req, ch)

	if len(req.Messages) == 0 || req.Messages[0].Role != compat.RoleSystem {
		t.Fatalf("expected a system message to be created")
	}
	content := req.Messages[0].Content
	if !strings.Contains(content, "topic AI") {
		t.Fatalf("expected rendered instruction content, got %q", content)
	}
	if !strings.Contains(content, `"description": "literal {research_topics}"`) {
		t.Fatalf("expected schema text to keep literal braces, got %q", content)
	}
}
