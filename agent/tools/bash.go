// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/LingByte/ling-base/agent"
)

const defaultBashTimeout = 30 * time.Second

// BashTool executes shell commands.
type BashTool struct {
	CWD     string
	Timeout time.Duration
}

// NewBash creates a bash tool that runs commands in cwd.
func NewBash(cwd string) *BashTool {
	return &BashTool{CWD: cwd, Timeout: defaultBashTimeout}
}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // seconds, 0 = default
}

const bashSchema = `{"type":"object","properties":{"command":{"type":"string","description":"The shell command to execute."},"timeout":{"type":"integer","description":"Timeout in seconds. Default: 30."}},"required":["command"]}`

func (t *BashTool) Name() string { return "bash" }
func (t *BashTool) Description() string {
	return "Execute a shell command in the working directory and return stdout/stderr."
}
func (t *BashTool) Schema() json.RawMessage { return json.RawMessage(bashSchema) }

func (t *BashTool) Execute(ctx context.Context, raw json.RawMessage, progress func(string)) (agent.ToolResult, error) {
	var a bashArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return agent.ToolResult{}, fmt.Errorf("invalid args: %w", err)
	}
	if a.Command == "" {
		return agent.ToolResult{}, fmt.Errorf("command is required")
	}

	timeout := t.Timeout
	if a.Timeout > 0 {
		timeout = time.Duration(a.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", a.Command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", a.Command)
	}
	cmd.Dir = t.CWD

	output, err := cmd.CombinedOutput()
	result := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return agent.ToolResult{
			Content: fmt.Sprintf("Command timed out after %s.\nOutput so far:\n%s", timeout, result),
			IsError: true,
		}, nil
	}

	if err != nil {
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return agent.ToolResult{
			Content: fmt.Sprintf("Command failed (exit %d):\n%s", exitCode, result),
			IsError: true,
		}, nil
	}

	// Trim trailing whitespace for cleaner output.
	result = strings.TrimRight(result, "\n\r ")
	if result == "" {
		result = "(command produced no output)"
	}
	return agent.ToolResult{Content: result}, nil
}
