package sandbox

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// statFile is a test helper that checks if a subdirectory exists under root.
func statFile(root, sub string) (os.FileInfo, error) {
	return os.Stat(root + "/" + sub)
}

// writeFile is a test helper that writes content to a file.
func writeFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0o644)
}

// isSandboxUnavailable reports whether err indicates the OS sandbox backend
// is not present (e.g. sandbox-exec/bwrap missing or unsupported platform).
// Kind checks go through the error text because agentkit's sandboxError type
// is unexported across the module boundary.
func isSandboxUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UnsupportedBackend") || strings.Contains(msg, "SetupFailed")
}

// waitUntil polls fn until it returns true or the deadline passes.
func waitUntil(d time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fn()
}

func TestNewLocalShellRunner(t *testing.T) {
	runner := NewLocalShellRunner()
	if runner == nil {
		t.Fatal("NewLocalShellRunner returned nil")
	}
	if runner.Workspace().Path == "" {
		t.Error("workspace path is empty")
	}
	if runner.shellPath == "" {
		t.Error("shellPath is empty")
	}
}

func TestRuntimeShellRunnerRunCaptureOutput(t *testing.T) {
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "printf 'hello\nworld\n'",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") || !strings.Contains(result.Stdout, "world") {
		t.Errorf("stdout = %q, want hello+world", result.Stdout)
	}
}

func TestRuntimeShellRunnerRunNonZeroExit(t *testing.T) {
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "exit 42",
	})
	// Non-zero exit is not a Run-level error for the local (disabled) profile.
	if err != nil {
		t.Fatalf("unexpected error for non-zero exit: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("exit code = %d, want 42", result.ExitCode)
	}
}

func TestRuntimeShellRunnerRunStderr(t *testing.T) {
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "echo err-msg >&2",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(result.Stderr, "err-msg") {
		t.Errorf("stderr = %q, want err-msg", result.Stderr)
	}
}

func TestRuntimeShellRunnerRunEnvOverride(t *testing.T) {
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "printf '%s' \"$MY_VAR\"",
		Env:     []string{"MY_VAR=testvalue"},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Stdout != "testvalue" {
		t.Errorf("stdout = %q, want 'testvalue'", result.Stdout)
	}
}

func TestRuntimeShellRunnerRunTimeout(t *testing.T) {
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "sleep 10",
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
	if !result.TimedOut {
		t.Error("expected TimedOut=true")
	}
}

func TestRuntimeShellRunnerRunEmptyCommand(t *testing.T) {
	// An empty command is valid shell (bash -c "" exits 0). The runner
	// does not pre-validate the command; it delegates to the shell.
	runner := NewLocalShellRunner()
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0 for empty command", result.ExitCode)
	}
}

func TestRuntimeShellRunnerStartBackground(t *testing.T) {
	runner := NewLocalShellRunner()
	handle, err := runner.Start(context.Background(), ShellSpec{
		Command: "printf 'line1\nline2\n' && sleep 0.1",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !handle.Running() {
		t.Error("expected handle to be running immediately after Start")
	}

	// Wait for process to exit.
	if !waitUntil(3*time.Second, func() bool { return !handle.Running() }) {
		t.Fatal("background process did not exit in time")
	}

	// Read all output from offset 0.
	data, _, _, exitCode := handle.Read(0)
	if !strings.Contains(data, "line1") || !strings.Contains(data, "line2") {
		t.Errorf("output = %q, want line1+line2", data)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

func TestRuntimeShellRunnerStartKill(t *testing.T) {
	runner := NewLocalShellRunner()
	handle, err := runner.Start(context.Background(), ShellSpec{
		Command: "sleep 30",
	})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !handle.Running() {
		t.Fatal("expected running")
	}
	if err := handle.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	if !waitUntil(3*time.Second, func() bool { return !handle.Running() }) {
		t.Error("killed process still running")
	}
}

func TestShellHandleReadOffset(t *testing.T) {
	h := newShellHandle(nil)
	_, _ = h.Write([]byte("hello "))
	_, _ = h.Write([]byte("world"))

	data, offset, running, _ := h.Read(0)
	if data != "hello world" {
		t.Errorf("Read(0) = %q, want 'hello world'", data)
	}
	if offset != len("hello world") {
		t.Errorf("offset = %d, want %d", offset, len("hello world"))
	}
	// Before finish, the handle reports running=true.
	if !running {
		t.Error("expected running=true before finish")
	}

	// Read from offset 6 should return "world".
	data, _, _, _ = h.Read(6)
	if data != "world" {
		t.Errorf("Read(6) = %q, want 'world'", data)
	}
}

func TestShellHandleReadNegativeOffset(t *testing.T) {
	h := newShellHandle(nil)
	_, _ = h.Write([]byte("abc"))
	// Negative offset clamps to len(buf), so Read returns empty.
	data, offset, _, _ := h.Read(-1)
	if data != "" {
		t.Errorf("Read(-1) = %q, want empty (clamped to end)", data)
	}
	if offset != 3 {
		t.Errorf("offset = %d, want 3", offset)
	}
}

func TestShellHandleReadOverflowOffset(t *testing.T) {
	h := newShellHandle(nil)
	_, _ = h.Write([]byte("abc"))
	data, offset, _, _ := h.Read(100)
	if data != "" {
		t.Errorf("Read(100) = %q, want empty", data)
	}
	if offset != 3 {
		t.Errorf("offset = %d, want 3", offset)
	}
}

func TestShellHandleFinishWithExitError(t *testing.T) {
	h := newShellHandle(nil)
	// Simulate a non-zero exit via a fake command.
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "exit 7")
	_ = cmd.Start()
	err := cmd.Wait()
	h.finish(err)
	if h.Running() {
		t.Error("expected not running after finish")
	}
	_, _, _, code := h.Read(0)
	if code != 7 {
		t.Errorf("exit code = %d, want 7", code)
	}
}

func TestShellHandleFinishWithNonExitError(t *testing.T) {
	h := newShellHandle(nil)
	h.finish(context.Canceled)
	if h.Running() {
		t.Error("expected not running after finish")
	}
	_, _, _, code := h.Read(0)
	if code != -1 {
		t.Errorf("exit code = %d, want -1 for non-ExitError", code)
	}
}

func TestShellHandleKillNilFunc(t *testing.T) {
	h := newShellHandle(nil)
	if err := h.Kill(); err != nil {
		t.Errorf("Kill with nil killFn should be no-op, got %v", err)
	}
}

func TestShellHandleKillCallsFunc(t *testing.T) {
	called := false
	h := newShellHandle(func() error {
		called = true
		return nil
	})
	if err := h.Kill(); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	if !called {
		t.Error("killFn was not called")
	}
}

func TestNewOSShellRunnerEnsureLayout(t *testing.T) {
	tmp := t.TempDir()
	runner, err := NewOSShellRunner(tmp, nil, "")
	if err != nil {
		t.Fatalf("NewOSShellRunner failed: %v", err)
	}
	if runner.Workspace().Path != tmp {
		t.Errorf("workspace path = %q, want %q", runner.Workspace().Path, tmp)
	}
	// EnsureLayout creates skills/work/runs/out subdirs (not home/tmp).
	for _, dir := range []string{"work", "runs", "out", "skills"} {
		if _, err := statFile(tmp, dir); err != nil {
			t.Errorf("EnsureLayout did not create %s/: %v", dir, err)
		}
	}
}

func TestNewOSShellRunnerInvalidDir(t *testing.T) {
	// A path under a non-existent parent should fail EnsureLayout on some
	// systems; but MkdirAll typically handles it. Use a file path to force
	// an error.
	tmp := t.TempDir()
	filePath := tmp + "/notadir"
	if err := writeFile(filePath, []byte("x")); err != nil {
		t.Fatal(err)
	}
	_, err := NewOSShellRunner(filePath, nil, "")
	if err == nil {
		t.Error("expected error when workDir is a file, not a directory")
	}
}

func TestRuntimeShellRunnerRunWorkDirAbsolute(t *testing.T) {
	tmp := t.TempDir()
	runner, err := NewOSShellRunner(tmp, nil, "")
	if err != nil {
		t.Fatalf("NewOSShellRunner failed: %v", err)
	}
	// Pass an absolute WorkDir equal to the workspace root; it should be
	// resolved to "." and the command should run there.
	result, err := runner.Run(context.Background(), ShellSpec{
		Command: "pwd",
		WorkDir: tmp,
	})
	if err != nil {
		// OS sandbox may not be available in CI; skip if preflight fails.
		if isSandboxUnavailable(err) {
			t.Skipf("OS sandbox unavailable: %v", err)
		}
		t.Fatalf("Run failed: %v", err)
	}
	// The resolved cwd should be the workspace root (tmp), possibly with
	// symlinks resolved on macOS.
	if !strings.Contains(result.Stdout, tmp) && !strings.Contains(result.Stdout, "private/tmp") {
		t.Errorf("pwd output = %q, want it to contain %q", result.Stdout, tmp)
	}
}
