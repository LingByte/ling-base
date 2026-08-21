// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

// Agent is a stateful conversation bound to a relay client, a model,
// a system prompt, and a set of tools.
type Agent struct {
	Client    *relay.Client
	Model     string
	System    string
	Tools     Registry
	MaxSteps  int // 0 = unlimited
	AutoApprove bool // if true, skip confirmation for all tools

	// BeforeToolExecute, if set, is called before each tool runs.
	// Returning (allowed=false, reason) short-circuits the call with
	// an error result containing reason.
	BeforeToolExecute func(call ToolCall) (allowed bool, reason string)

	// OnEvent, if set, mirrors every Event to this callback in
	// addition to the per-Prompt sink.
	OnEvent func(Event)

	mu       sync.Mutex
	messages []relay.Message
}

// Option configures an Agent.
type Option func(*Agent)

// WithTools sets the tool registry.
func WithTools(tools ...Tool) Option {
	return func(a *Agent) { a.Tools = NewRegistry(tools...) }
}

// WithSystem sets the system prompt.
func WithSystem(s string) Option {
	return func(a *Agent) { a.System = s }
}

// WithMaxSteps sets the maximum loop iterations (0 = unlimited).
func WithMaxSteps(n int) Option {
	return func(a *Agent) { a.MaxSteps = n }
}

// WithAutoApprove skips tool confirmation.
func WithAutoApprove(b bool) Option {
	return func(a *Agent) { a.AutoApprove = b }
}

// New creates an Agent with sensible defaults.
func New(client *relay.Client, model string, opts ...Option) *Agent {
	a := &Agent{
		Client:   client,
		Model:    model,
		Tools:    Registry{},
		MaxSteps: 0,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Messages returns a copy of the current transcript.
func (a *Agent) Messages() []relay.Message {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]relay.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// SetMessages replaces the transcript (used for session resume).
func (a *Agent) SetMessages(msgs []relay.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = make([]relay.Message, len(msgs))
	copy(a.messages, msgs)
}

// Prompt sends user text to the agent and runs the loop to completion.
// sink receives every Event emitted during the run.
func (a *Agent) Prompt(ctx context.Context, text string, sink func(Event)) error {
	if sink == nil {
		sink = func(Event) {}
	}
	sink = a.wrapSink(sink)

	// Add user message.
	user := relay.Message{
		Role:    "user",
		Content: text,
	}
	a.mu.Lock()
	a.messages = append(a.messages, user)
	a.mu.Unlock()

	sink(EvUserMessage{Text: text})

	return a.runLoop(ctx, sink)
}

// Continue runs the loop against the existing transcript without
// adding a new user message. Used after manual message injection.
func (a *Agent) Continue(ctx context.Context, sink func(Event)) error {
	if sink == nil {
		sink = func(Event) {}
	}
	sink = a.wrapSink(sink)
	return a.runLoop(ctx, sink)
}

func (a *Agent) wrapSink(sink func(Event)) func(Event) {
	if a.OnEvent == nil {
		return sink
	}
	obs := a.OnEvent
	return func(ev Event) {
		obs(ev)
		sink(ev)
	}
}

// runLoop is the core agent loop: call LLM → check for tool calls →
// execute tools → feed results back → repeat until no tool calls.
func (a *Agent) runLoop(ctx context.Context, sink func(Event)) error {
	for step := 1; a.MaxSteps <= 0 || step <= a.MaxSteps; step++ {
		sink(EvTurnStart{Step: step})

		resp, err := a.callLLM(ctx)
		if err != nil {
			sink(EvError{Err: err})
			return fmt.Errorf("agent: LLM call failed at step %d: %w", step, err)
		}

		if len(resp.Choices) == 0 {
			sink(EvError{Err: fmt.Errorf("no choices in response")})
			return fmt.Errorf("agent: no choices in response at step %d", step)
		}

		choice := resp.Choices[0]
		sink(EvUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		})

		// Emit assistant text if any.
		text := choice.Message.StringContent()
		if text != "" {
			sink(EvAssistantText{Text: text})
		}

		// Add assistant message to transcript.
		a.mu.Lock()
		a.messages = append(a.messages, choice.Message)
		a.mu.Unlock()

		sink(EvTurnEnd{StopReason: choice.FinishReason})

		// If the model wants to call tools, execute them and loop.
		if choice.FinishReason == "tool_calls" {
			toolCalls := choice.Message.ParseToolCalls()
			toolMsg, err := a.executeTools(ctx, toolCalls, sink)
			if err != nil {
				sink(EvError{Err: err})
				return err
			}
			a.mu.Lock()
			a.messages = append(a.messages, toolMsg)
			a.mu.Unlock()
			continue
		}

		// No tool calls — we're done.
		sink(EvDone{})
		return nil
	}

	sink(EvError{Err: fmt.Errorf("max steps (%d) exceeded", a.MaxSteps)})
	return fmt.Errorf("agent: max steps (%d) exceeded", a.MaxSteps)
}

// callLLM sends the current transcript to the LLM and returns the response.
func (a *Agent) callLLM(ctx context.Context) (*relay.ChatResponse, error) {
	a.mu.Lock()
	msgs := make([]relay.Message, 0, len(a.messages)+1)
	if a.System != "" {
		msgs = append(msgs, relay.Message{
			Role:    "system",
			Content: a.System,
		})
	}
	msgs = append(msgs, a.messages...)
	a.mu.Unlock()

	req := &relay.ChatRequest{
		Model:    a.Model,
		Messages: msgs,
	}

	// Attach tool definitions if any.
	if len(a.Tools) > 0 {
		specs := a.Tools.Specs()
		tools := make([]relay.Tool, 0, len(specs))
		for _, s := range specs {
			specBytes, _ := json.Marshal(s)
			var tc relay.Tool
			json.Unmarshal(specBytes, &tc)
			tools = append(tools, tc)
		}
		req.Tools = tools
	}

	return a.Client.Chat(ctx, req)
}

// executeTools runs all tool calls in a single assistant message and
// returns a single tool-results message.
func (a *Agent) executeTools(ctx context.Context, calls []dto.ToolCallRequest, sink func(Event)) (relay.Message, error) {
	// Build a tool-results message. In OpenAI format, each tool result
	// is a separate message with role="tool" and tool_call_id.
	// However, relay's Message struct carries a single ToolCallId, so
	// we return the last result's metadata. For multiple calls, we
	// append individual messages to the transcript instead.
	//
	// In practice, most providers handle multiple tool results as
	// multiple "tool" messages. We'll build them as separate messages
	// and let the caller append them all.
	//
	// But since our return type is a single Message, we handle the
	// common case (single tool call) directly and fall back to
	// concatenation for multiple.

	results := make([]string, 0, len(calls))
	var lastID string

	for _, call := range calls {
		tc := ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: json.RawMessage(call.Function.Arguments),
		}

		sink(EvToolCallStart{Name: tc.Name, Arguments: string(tc.Arguments)})

		// Permission check.
		if a.BeforeToolExecute != nil {
			allowed, reason := a.BeforeToolExecute(tc)
			if !allowed {
				errMsg := fmt.Sprintf("tool execution denied: %s", reason)
				sink(EvToolCallEnd{Name: tc.Name, Result: ToolResult{Content: errMsg, IsError: true}})
				results = append(results, errMsg)
				lastID = call.ID
				continue
			}
		}

		tool := a.Tools.Get(tc.Name)
		if tool == nil {
			errMsg := fmt.Sprintf("unknown tool: %s", tc.Name)
			sink(EvToolCallEnd{Name: tc.Name, Result: ToolResult{Content: errMsg, IsError: true}})
			results = append(results, errMsg)
			lastID = call.ID
			continue
		}

		result, err := tool.Execute(ctx, tc.Arguments, func(progress string) {
			// Progress is not sent to LLM; could be used for UI.
		})
		if err != nil {
			errMsg := fmt.Sprintf("tool %s failed: %v", tc.Name, err)
			result = ToolResult{Content: errMsg, IsError: true}
			sink(EvToolCallEnd{Name: tc.Name, Result: result, Err: err})
		} else {
			sink(EvToolCallEnd{Name: tc.Name, Result: result})
		}

		results = append(results, result.Content)
		lastID = call.ID
	}

	// Combine all results into a single message.
	// For OpenAI format: role="tool", tool_call_id=<id>, content=<result>
	// When there are multiple calls, we join results with separators.
	// This is a simplification — ideally each result should be its own
	// message, but relay's Message struct is designed for single tool
	// results. A future version can append multiple messages.
	combined := ""
	for i, r := range results {
		if i > 0 {
			combined += "\n---\n"
		}
		combined += r
	}

	return relay.Message{
		Role:       "tool",
		Content:    combined,
		ToolCallId: lastID,
	}, nil
}
