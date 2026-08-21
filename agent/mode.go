// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// RunPrintMode runs the agent in print mode: assistant text goes to
// stdout, tool activity goes to stderr.
func RunPrintMode(ctx context.Context, a *Agent, prompt string) error {
	return a.Prompt(ctx, prompt, func(ev Event) {
		switch e := ev.(type) {
		case EvAssistantText:
			fmt.Fprint(os.Stdout, e.Text)
		case EvToolCallStart:
			fmt.Fprintf(os.Stderr, "\n[tool] %s %s\n", e.Name, truncate(e.Arguments, 200))
		case EvToolCallEnd:
			if e.Err != nil {
				fmt.Fprintf(os.Stderr, "[tool] %s error: %v\n", e.Name, e.Err)
			} else if e.Result.IsError {
				fmt.Fprintf(os.Stderr, "[tool] %s failed: %s\n", e.Name, truncate(e.Result.Content, 200))
			} else {
				fmt.Fprintf(os.Stderr, "[tool] %s done: %s\n", e.Name, truncate(e.Result.Content, 200))
			}
		case EvTurnStart:
			if e.Step > 1 {
				fmt.Fprintf(os.Stderr, "\n--- step %d ---\n", e.Step)
			}
		case EvUsage:
			fmt.Fprintf(os.Stderr, "[usage] in=%d out=%d total=%d\n", e.InputTokens, e.OutputTokens, e.TotalTokens)
		case EvDone:
			fmt.Fprintln(os.Stdout) // newline after final text
		case EvError:
			fmt.Fprintf(os.Stderr, "[error] %v\n", e.Err)
		}
	})
}

// RunJSONMode runs the agent in JSON mode: every event is emitted as
// a newline-delimited JSON object to stdout.
func RunJSONMode(ctx context.Context, a *Agent, prompt string) error {
	return a.Prompt(ctx, prompt, func(ev Event) {
		obj := eventToJSON(ev)
		data, _ := json.Marshal(obj)
		fmt.Fprintln(os.Stdout, string(data))
	})
}

// eventToJSON converts an Event to a map for JSON serialization.
func eventToJSON(ev Event) map[string]any {
	m := map[string]any{}
	switch e := ev.(type) {
	case EvUserMessage:
		m["type"] = "user"
		m["text"] = e.Text
	case EvTurnStart:
		m["type"] = "turn_start"
		m["step"] = e.Step
	case EvTurnEnd:
		m["type"] = "turn_end"
		m["stop_reason"] = e.StopReason
	case EvAssistantText:
		m["type"] = "assistant_text"
		m["text"] = e.Text
	case EvToolCallStart:
		m["type"] = "tool_call_start"
		m["name"] = e.Name
		m["arguments"] = e.Arguments
	case EvToolCallEnd:
		m["type"] = "tool_call_end"
		m["name"] = e.Name
		m["content"] = e.Result.Content
		m["is_error"] = e.Result.IsError
		if e.Err != nil {
			m["error"] = e.Err.Error()
		}
	case EvUsage:
		m["type"] = "usage"
		m["input_tokens"] = e.InputTokens
		m["output_tokens"] = e.OutputTokens
		m["total_tokens"] = e.TotalTokens
	case EvDone:
		m["type"] = "done"
	case EvError:
		m["type"] = "error"
		m["error"] = e.Err.Error()
	}
	return m
}

// RunFromStdin reads a prompt from stdin (piped mode).
func RunFromStdin(ctx context.Context, a *Agent, jsonMode bool) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	prompt := string(data)
	if jsonMode {
		return RunJSONMode(ctx, a, prompt)
	}
	return RunPrintMode(ctx, a, prompt)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
