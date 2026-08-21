package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/ling-base/agent/sandbox"
)

func newTestBash(t *testing.T) *Bash {
	t.Helper()
	runner := sandbox.NewLocalShellRunner()
	b, err := NewBash(runner)
	if err != nil {
		t.Fatalf("NewBash: %v", err)
	}
	return b
}

func TestBashNameAndDescription(t *testing.T) {
	b := newTestBash(t)
	if b.Name() != "Bash" {
		t.Errorf("Name() = %q, want 'Bash'", b.Name())
	}
	desc, err := b.Description(context.Background())
	if err != nil {
		t.Fatalf("Description: %v", err)
	}
	if !strings.Contains(desc, "shell") {
		t.Errorf("Description = %q, want 'shell'", desc)
	}
}

func TestBashValidateInputEmptyCommand(t *testing.T) {
	b := newTestBash(t)
	err := b.ValidateInput(json.RawMessage(`{"command":""}`))
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestBashValidateInputValid(t *testing.T) {
	b := newTestBash(t)
	err := b.ValidateInput(json.RawMessage(`{"command":"echo hi"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBashExecuteCaptureOutput(t *testing.T) {
	b := newTestBash(t)
	results, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"printf 'hello\n'"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].IsError {
		t.Errorf("IsError = true, want false")
	}
	if !strings.Contains(results[0].Content, "hello") {
		t.Errorf("Content = %q, want 'hello'", results[0].Content)
	}
}

func TestBashExecuteNonZeroExit(t *testing.T) {
	b := newTestBash(t)
	results, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"exit 3"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !results[0].IsError {
		t.Error("expected IsError for non-zero exit")
	}
	if !strings.Contains(results[0].Content, "exit code 3") {
		t.Errorf("Content = %q, want 'exit code 3'", results[0].Content)
	}
}

func TestBashExecuteStderr(t *testing.T) {
	b := newTestBash(t)
	results, _ := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"echo err-msg >&2"}`))
	if !strings.Contains(results[0].Content, "err-msg") {
		t.Errorf("Content = %q, want 'err-msg'", results[0].Content)
	}
}

func TestBashExecuteTimeout(t *testing.T) {
	b := newTestBash(t)
	results, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"sleep 10","timeout":100}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !results[0].IsError {
		t.Error("expected IsError for timeout")
	}
	if !strings.Contains(results[0].Content, "timed out") {
		t.Errorf("Content = %q, want 'timed out'", results[0].Content)
	}
}

func TestBashExecuteNoOutput(t *testing.T) {
	b := newTestBash(t)
	results, _ := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"true"}`))
	if !strings.Contains(results[0].Content, "no output") {
		t.Errorf("Content = %q, want '[no output]'", results[0].Content)
	}
}

func TestBashExecuteBackgroundWithoutStore(t *testing.T) {
	b := newTestBash(t) // no ShellStore passed
	results, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"sleep 1","run_in_background":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !results[0].IsError {
		t.Error("expected IsError when no ShellStore configured")
	}
	if !strings.Contains(results[0].Content, "not available") {
		t.Errorf("Content = %q, want 'not available'", results[0].Content)
	}
}

func TestBashExecuteBackgroundWithStore(t *testing.T) {
	runner := sandbox.NewLocalShellRunner()
	store := NewShellStore(context.Background())
	b, err := NewBash(runner, store)
	if err != nil {
		t.Fatalf("NewBash: %v", err)
	}
	results, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{"command":"printf 'bg-output\\n'","run_in_background":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if results[0].IsError {
		t.Errorf("unexpected error: %s", results[0].Content)
	}
	if !strings.Contains(results[0].Content, "Started background shell") {
		t.Errorf("Content = %q, want 'Started background shell'", results[0].Content)
	}
	// Clean up.
	store.KillAll()
}

func TestBashExecuteInvalidJSON(t *testing.T) {
	b := newTestBash(t)
	_, err := b.Execute(context.Background(), Context{}, json.RawMessage(`{bad json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFormatBashOutput(t *testing.T) {
	tests := []struct {
		name   string
		result sandbox.ShellResult
		want   string
	}{
		{"stdout only", sandbox.ShellResult{Stdout: "hello"}, "hello"},
		{"stdout+stderr", sandbox.ShellResult{Stdout: "out", Stderr: "err"}, "out\nerr"},
		{"nonzero exit", sandbox.ShellResult{ExitCode: 1}, "[exit code 1]"},
		{"timeout", sandbox.ShellResult{TimedOut: true, ExitCode: 124}, "[command timed out, exit code 124]"},
		{"empty", sandbox.ShellResult{}, "[no output]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBashOutput(tt.result)
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatBashOutput() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

func TestFormatBashOutputTruncation(t *testing.T) {
	long := strings.Repeat("x", bashMaxOutput+1000)
	got := formatBashOutput(sandbox.ShellResult{Stdout: long})
	if !strings.Contains(got, "output truncated") {
		t.Error("expected truncation marker")
	}
	if len(got) > bashMaxOutput+100 {
		t.Errorf("output not truncated: len=%d", len(got))
	}
}

func TestBashPermissionRequest(t *testing.T) {
	b := newTestBash(t)
	req := b.PermissionRequest(json.RawMessage(`{"command":"git status"}`))
	// bashparser should extract "git status" as the prefix.
	if req.Specifier == "" {
		t.Error("expected non-empty specifier")
	}
	if !strings.Contains(req.Specifier, "git") {
		t.Errorf("specifier = %q, want 'git'", req.Specifier)
	}
}

// Ensure the test doesn't hang forever if something goes wrong.
func init() {
	// No-op; just ensures the package compiles with time imported.
	_ = time.Second
}
