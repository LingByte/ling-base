package goagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/ling-base/agentkit/tool"
	compat "github.com/LingByte/ling-base/relay/compat"
	utcptools "github.com/universal-tool-calling-protocol/go-utcp/src/tools"
)

// File is an in-memory attachment passed to the model. It is an alias for
// compat.File so callers can construct attachments without importing the
// compat package directly.
type File = compat.File

// StreamChunk represents a single piece of a streaming LLM response.
// When Done is true, the stream is complete and FullText holds the aggregated
// output. When Err is non-nil, the stream encountered a fatal error.
type StreamChunk struct {
	Delta    string
	Done     bool
	FullText string
	Err      error
}

// ToolDefinition is the provider-neutral description of a callable tool used
// for native tool calling.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

// ToolCall is a provider-neutral tool invocation selected by a model.
type ToolCall struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolCallResponse contains either assistant text, one or more native tool
// calls, or both depending on the provider response.
type ToolCallResponse struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ErrToolCallingUnsupported is returned when the underlying model does not
// support native tool calling. With compat.Model this is rare but kept for
// backward-compatible error handling in the orchestrator.
var ErrToolCallingUnsupported = fmt.Errorf("native tool calling unsupported")

// ─── Internal helpers: compat.Model → simple prompt interface ─────────────

// generate calls the model with a simple text prompt and returns the full
// text response.
func (a *Agent) generate(ctx context.Context, prompt string) (string, error) {
	req := compat.NewRequest([]compat.Message{compat.NewUserMessage(prompt)})
	return a.collectText(ctx, req)
}

// generateWithFiles calls the model with a prompt and file attachments.
func (a *Agent) generateWithFiles(ctx context.Context, prompt string, files []File) (string, error) {
	msg := compat.NewUserMessage(prompt)
	for i := range files {
		f := files[i]
		msg.AddFileData(f.Name, f.Data, f.MimeType)
	}
	req := compat.NewRequest([]compat.Message{msg})
	return a.collectText(ctx, req)
}

// generateStream calls the model in streaming mode and returns a channel of
// incremental text chunks.
func (a *Agent) generateStream(ctx context.Context, prompt string) (<-chan StreamChunk, error) {
	req := compat.NewRequest([]compat.Message{compat.NewUserMessage(prompt)})
	req.GenerationConfig.Stream = true
	respCh, err := a.model.GenerateContent(ctx, req)
	if err != nil {
		return nil, err
	}
	outCh := make(chan StreamChunk, 16)
	go func() {
		defer close(outCh)
		var full strings.Builder
		for resp := range respCh {
			if resp.Error != nil {
				outCh <- StreamChunk{Err: resp.Error, Done: true}
				return
			}
			for _, choice := range resp.Choices {
				if choice.Delta.Content != "" {
					full.WriteString(choice.Delta.Content)
					outCh <- StreamChunk{Delta: choice.Delta.Content}
				}
			}
			if resp.Done {
				outCh <- StreamChunk{Done: true, FullText: full.String()}
				return
			}
		}
		outCh <- StreamChunk{Done: true, FullText: full.String()}
	}()
	return outCh, nil
}

// generateWithTools calls the model with native tool definitions and returns
// the text and/or tool calls from the response.
func (a *Agent) generateWithTools(ctx context.Context, prompt string, definitions []ToolDefinition) (ToolCallResponse, error) {
	toolMap := make(map[string]tool.Tool, len(definitions))
	for _, d := range definitions {
		d := d
		toolMap[d.Name] = &toolDefAdapter{decl: &tool.Declaration{
			Name:        d.Name,
			Description: d.Description,
			InputSchema: mapToSchema(d.InputSchema),
		}}
	}
	req := compat.NewRequest([]compat.Message{compat.NewUserMessage(prompt)})
	req.Tools = toolMap

	respCh, err := a.model.GenerateContent(ctx, req)
	if err != nil {
		return ToolCallResponse{}, err
	}

	var result ToolCallResponse
	for resp := range respCh {
		if resp.Error != nil {
			return ToolCallResponse{}, resp.Error
		}
		if resp.Done && len(resp.Choices) > 0 {
			msg := resp.Choices[0].Message
			result.Content = msg.Content
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				if len(tc.Function.Arguments) > 0 {
					_ = json.Unmarshal(tc.Function.Arguments, &args)
				}
				if args == nil {
					args = map[string]any{}
				}
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: args,
				})
			}
		}
	}
	return result, nil
}

// collectText calls GenerateContent and collects all text from the response
// channel into a single string.
func (a *Agent) collectText(ctx context.Context, req *compat.Request) (string, error) {
	respCh, err := a.model.GenerateContent(ctx, req)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for resp := range respCh {
		if resp.Error != nil {
			return "", resp.Error
		}
		for _, choice := range resp.Choices {
			if choice.Message.Content != "" {
				sb.WriteString(choice.Message.Content)
			}
			if choice.Delta.Content != "" {
				sb.WriteString(choice.Delta.Content)
			}
		}
	}
	return sb.String(), nil
}

// generateText is a package-level helper that calls a compat.Model with a
// simple prompt and returns the full text response. It is used by standalone
// policies (e.g. LLMEvaluatorInputPolicy) that hold a compat.Model directly.
func generateText(ctx context.Context, model compat.Model, prompt string) (string, error) {
	req := compat.NewRequest([]compat.Message{compat.NewUserMessage(prompt)})
	respCh, err := model.GenerateContent(ctx, req)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for resp := range respCh {
		if resp.Error != nil {
			return "", resp.Error
		}
		for _, choice := range resp.Choices {
			if choice.Message.Content != "" {
				sb.WriteString(choice.Message.Content)
			}
			if choice.Delta.Content != "" {
				sb.WriteString(choice.Delta.Content)
			}
		}
	}
	return sb.String(), nil
}

// ─── UTCP → agentkit/tool adapter ─────────────────────────────────────────

// toolDefAdapter wraps a ToolDefinition as an agentkit/tool.Tool so the
// relaymodel adapter can translate it into the provider's native tool format.
type toolDefAdapter struct {
	decl *tool.Declaration
}

func (a *toolDefAdapter) Declaration() *tool.Declaration {
	return a.decl
}

// utcpToolAdapter wraps a UTCP tool spec as an agentkit/tool.Tool so the
// relaymodel adapter can translate it into the provider's native tool format.
type utcpToolAdapter struct {
	spec utcptools.Tool
}

func (a *utcpToolAdapter) Declaration() *tool.Declaration {
	schemaMap := map[string]any{}
	if encoded, err := json.Marshal(a.spec.Inputs); err == nil {
		_ = json.Unmarshal(encoded, &schemaMap)
	}
	return &tool.Declaration{
		Name:        a.spec.Name,
		Description: a.spec.Description,
		InputSchema: mapToSchema(schemaMap),
	}
}

// utcpToolsToMap converts a slice of UTCP tool specs into the map format
// expected by relaymodel.
func utcpToolsToMap(specs []utcptools.Tool) map[string]tool.Tool {
	m := make(map[string]tool.Tool, len(specs))
	for _, spec := range specs {
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			continue
		}
		if _, exists := m[name]; exists {
			continue
		}
		m[name] = &utcpToolAdapter{spec: spec}
	}
	return m
}

// mapToSchema converts a map[string]any (from JSON marshalling of UTCP inputs)
// into a *tool.Schema.
func mapToSchema(m map[string]any) *tool.Schema {
	if m == nil {
		return &tool.Schema{Type: "object"}
	}
	s := &tool.Schema{}
	if t, ok := m["type"].(string); ok {
		s.Type = t
	} else {
		s.Type = "object"
	}
	if d, ok := m["description"].(string); ok {
		s.Description = d
	}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if rs, ok := r.(string); ok {
				s.Required = append(s.Required, rs)
			}
		}
	}
	if props, ok := m["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*tool.Schema, len(props))
		for k, v := range props {
			if vm, ok := v.(map[string]any); ok {
				s.Properties[k] = mapToSchema(vm)
			}
		}
	}
	return s
}

// CodeModeModelAdapter wraps a compat.Model to satisfy the interface expected
// by codemode.NewCodeModeUTCP:
//
//	Generate(ctx context.Context, prompt string) (any, error)
type CodeModeModelAdapter struct {
	Model compat.Model
}

// Generate calls the underlying compat.Model and returns the full text as any.
func (a *CodeModeModelAdapter) Generate(ctx context.Context, prompt string) (any, error) {
	return generateText(ctx, a.Model, prompt)
}

// singleTextResponse creates a channel that yields one completed Response
// containing the given text. Useful for test stubs implementing compat.Model.
func singleTextResponse(text string) <-chan *compat.Response {
	ch := make(chan *compat.Response, 1)
	finishReason := "stop"
	ch <- &compat.Response{
		Done:    true,
		Choices: []compat.Choice{{Message: compat.NewAssistantMessage(text), FinishReason: &finishReason}},
	}
	close(ch)
	return ch
}

// promptFromRequest extracts the first user message content from a Request.
func promptFromRequest(req *compat.Request) string {
	for _, msg := range req.Messages {
		if msg.Role == compat.RoleUser {
			return msg.Content
		}
	}
	return ""
}

// ─── DummyModel (for tests) ───────────────────────────────────────────────

// DummyModel is a test stub that returns a fixed prefix concatenated with the
// last user message. It implements compat.Model.
type DummyModel struct {
	Prefix string
}

// NewDummyModel creates a DummyModel that replies with prefix+prompt.
func NewDummyModel(prefix string) *DummyModel {
	return &DummyModel{Prefix: prefix}
}

// Info returns basic model metadata.
func (d *DummyModel) Info() compat.Info {
	return compat.Info{Name: "dummy", ContextWindow: 4096}
}

// GenerateContent returns a single response containing prefix+prompt.
func (d *DummyModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	var prompt string
	for _, msg := range req.Messages {
		if msg.Role == compat.RoleUser {
			prompt = msg.Content
			break
		}
	}
	text := d.Prefix + prompt
	finishReason := "stop"
	ch <- &compat.Response{
		Done:    true,
		Model:   "dummy",
		Choices: []compat.Choice{{Message: compat.NewAssistantMessage(text), FinishReason: &finishReason}},
	}
	close(ch)
	return ch, nil
}
