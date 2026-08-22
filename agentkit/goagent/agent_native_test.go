package goagent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LingByte/ling-base/agentkit/memory/gomemory"
	compat "github.com/LingByte/ling-base/relay/compat"
	utcpTools "github.com/universal-tool-calling-protocol/go-utcp/src/tools"
)

type nativeToolModel struct {
	responses []ToolCallResponse
	calls     int
	prompts   []string
	tools     []ToolDefinition
}

func (m *nativeToolModel) Info() compat.Info { return compat.Info{Name: "native-tool"} }

func (m *nativeToolModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	prompt := promptFromRequest(req)
	m.prompts = append(m.prompts, prompt)

	// Check if tools are set on the request (native tool calling path).
	hasTools := req.ToolsLen() > 0
	if hasTools {
		m.calls++
		// Capture tool definitions for assertions.
		if len(m.tools) == 0 {
			if toolMap, ok := req.Tools.(map[string]interface{}); ok {
				for name := range toolMap {
					m.tools = append(m.tools, ToolDefinition{Name: name})
				}
			}
		}
		if len(m.responses) == 0 {
			return singleTextResponse("fallback"), nil
		}
		response := m.responses[0]
		m.responses = m.responses[1:]

		// Build compat.Response with tool calls or text.
		ch := make(chan *compat.Response, 1)
		finishReason := "tool_calls"
		if len(response.ToolCalls) == 0 {
			finishReason = "stop"
		}
		msg := compat.NewAssistantMessage(response.Content)
		for _, tc := range response.ToolCalls {
			argsBytes, _ := json.Marshal(tc.Arguments)
			msg.ToolCalls = append(msg.ToolCalls, compat.ToolCall{
				Type: "function",
				Function: compat.FunctionDefinitionParam{
					Name:      tc.Name,
					Arguments: argsBytes,
				},
				ID: tc.ID,
			})
		}
		ch <- &compat.Response{
			Done:    true,
			Choices: []compat.Choice{{Message: msg, FinishReason: &finishReason}},
		}
		close(ch)
		return ch, nil
	}

	// No tools → text response.
	return singleTextResponse("fallback"), nil
}

func TestAgentUsesNativeToolCallsWhenModelSupportsThem(t *testing.T) {
	model := &nativeToolModel{
		responses: []ToolCallResponse{
			{ToolCalls: []ToolCall{{Name: "echo", Arguments: map[string]any{"input": "hello"}}}},
			{Content: "finished"},
		},
	}
	localTool := &stubTool{spec: ToolSpec{
		Name:        "echo",
		Description: "Echoes input",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"input": map[string]any{"type": "string"}},
			"required":   []string{"input"},
		},
	}}

	a, err := New(Options{
		Model:  model,
		Memory: gomemory.NewSessionMemory(&gomemory.MemoryBank{}, 4),
		Tools:  []Tool{localTool},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	out, err := a.Generate(context.Background(), "session", "run echo")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if out != "finished" {
		t.Fatalf("Generate returned %q, want %q", out, "finished")
	}
	if model.calls != 2 {
		t.Fatalf("native model called %d times, want 2", model.calls)
	}
}

func TestNativeToolDefinitionsDefaultSchemaType(t *testing.T) {
	definitions := nativeToolDefinitions([]utcpTools.Tool{{Name: "empty"}})
	if len(definitions) != 1 {
		t.Fatalf("got %d definitions, want 1", len(definitions))
	}
	if got := definitions[0].InputSchema["type"]; got != "object" {
		t.Fatalf("schema type = %v, want object", got)
	}
}
