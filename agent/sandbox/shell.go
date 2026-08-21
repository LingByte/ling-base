// Package sandbox shell execution entry point.
//
// ShellRunner is the primary API for tools that execute shell commands through
// the sandbox. It abstracts over Runtime (OS sandbox via seatbelt/bwrap) and
// Container (docker/podman CLI). Unlike the old Executor shim, ShellRunner is
// backed by the full sandbox runtime: PermissionProfile enforcement, denial
// diagnostics, workspace isolation, and seccomp filters.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/agentkit/codeexecutor"
)

// resolveShell returns bash if available, else sh.
func resolveShell() string {
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	return "/bin/sh"
}

// ShellSpec describes a shell command to execute.
type ShellSpec struct {
	Command    string        // the shell command line (interpreted by bash -c)
	WorkDir    string        // working directory; empty means workspace root
	Timeout    time.Duration // 0 means no explicit timeout
	Env        []string      // extra KEY=VALUE environment entries
	Background bool          // start detached and return a handle immediately
}

// ShellResult is the captured output of a foreground shell command.
type ShellResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

// ShellHandle is a background shell process whose combined stdout+stderr can be
// read incrementally. Safe for concurrent use.
type ShellHandle struct {
	mu       sync.Mutex
	buf      []byte
	done     bool
	exitCode int
	killFn   func() error
	waitCh   chan struct{}
}

func newShellHandle(kill func() error) *ShellHandle {
	return &ShellHandle{killFn: kill, waitCh: make(chan struct{})}
}

func (h *ShellHandle) Write(b []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf = append(h.buf, b...)
	return len(b), nil
}

// Read returns buffered output from offset onward, the new offset, whether the
// process has exited, and its exit code.
func (h *ShellHandle) Read(offset int) (data string, newOffset int, running bool, exitCode int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if offset < 0 || offset > len(h.buf) {
		offset = len(h.buf)
	}
	return string(h.buf[offset:]), len(h.buf), !h.done, h.exitCode
}

// Running reports whether the process is still executing.
func (h *ShellHandle) Running() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !h.done
}

// Kill terminates the process (no-op if already exited).
func (h *ShellHandle) Kill() error {
	if h.killFn != nil {
		return h.killFn()
	}
	return nil
}

func (h *ShellHandle) finish(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.done = true
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			h.exitCode = exitErr.ExitCode()
		} else {
			h.exitCode = -1
		}
	}
	close(h.waitCh)
}

// ShellRunner executes shell commands through the sandbox. Implementations:
//   - RuntimeShellRunner: OS sandbox (seatbelt/bwrap) or unconfined, via Runtime
//   - ContainerShellRunner: docker/podman CLI isolation
type ShellRunner interface {
	// Run executes a foreground shell command and returns captured output.
	Run(ctx context.Context, spec ShellSpec) (ShellResult, error)
	// Start launches a background shell command and returns a handle for
	// incremental output reading.
	Start(ctx context.Context, spec ShellSpec) (*ShellHandle, error)
}

// RuntimeShellRunner runs shell commands through a sandbox Runtime. The
// workspace is typically the project directory, with OS sandbox restricting
// writes to the workspace and configured write roots.
type RuntimeShellRunner struct {
	runtime   *Runtime
	workspace codeexecutor.Workspace
	shellPath string
}

// NewRuntimeShellRunner creates a ShellRunner backed by the given Runtime and
// Workspace. The workspace path determines where commands execute; OS sandbox
// policy (PermissionProfile) restricts what the command can access.
func NewRuntimeShellRunner(rt *Runtime, ws codeexecutor.Workspace) *RuntimeShellRunner {
	return &RuntimeShellRunner{
		runtime:   rt,
		workspace: ws,
		shellPath: resolveShell(),
	}
}

// NewLocalShellRunner creates an unconfined ShellRunner (no OS sandbox) with a
// temporary workspace. This is the default when no sandbox mode is configured.
func NewLocalShellRunner() *RuntimeShellRunner {
	rt := NewRuntime(
		WithPermissionProfile(DangerFullAccessProfile()),
		WithSessionPolicy(SessionPolicy{Persistence: SessionPersistencePerSession}),
	)
	ws, _ := rt.CreateWorkspace(context.Background(), "local", codeexecutor.WorkspacePolicy{})
	return NewRuntimeShellRunner(rt, ws)
}

// NewOSShellRunner creates an OS-sandboxed ShellRunner (seatbelt on macOS,
// bwrap on Linux) with the given directory as the workspace root. The workspace
// path determines where commands execute; the OS sandbox restricts writes to
// the workspace and configured write roots.
//
// EnsureLayout is called on the workDir, creating work/home/tmp/runs/out/skills
// subdirectories inside it (these are harmless and can be gitignored).
func NewOSShellRunner(workDir string, writeRoots []string, network string) (*RuntimeShellRunner, error) {
	profile := WorkspaceWriteProfile()
	for _, root := range writeRoots {
		if root != "" {
			profile = profile.WithWritePaths(root)
		}
	}
	if network == "none" {
		profile = profile.WithNetworkPolicy(NetworkPolicy{Mode: NetworkRestricted})
	} else {
		profile = profile.WithNetworkPolicy(NetworkPolicy{Mode: NetworkEnabled})
	}
	rt := NewRuntime(
		WithBackend(BackendAuto),
		WithPermissionProfile(profile),
		WithSessionPolicy(SessionPolicy{Persistence: SessionPersistencePerSession}),
	)
	// Build the workspace directly at the project directory instead of using
	// CreateWorkspace (which creates a temp dir). EnsureLayout sets up the
	// standard subdirectories inside the project dir.
	if _, err := codeexecutor.EnsureLayout(workDir); err != nil {
		return nil, fmt.Errorf("sandbox: ensure workspace layout: %w", err)
	}
	ws := codeexecutor.Workspace{ID: "os", Path: workDir}
	return NewRuntimeShellRunner(rt, ws), nil
}

// Workspace returns the workspace commands execute in.
func (r *RuntimeShellRunner) Workspace() codeexecutor.Workspace { return r.workspace }

func (r *RuntimeShellRunner) Run(ctx context.Context, spec ShellSpec) (ShellResult, error) {
	cwd := spec.WorkDir
	if cwd == "" {
		cwd = "."
	}
	// Resolve cwd relative to workspace root. If WorkDir is an absolute host
	// path outside the workspace, resolveWorkspacePath will reject it. For the
	// common case (WorkDir = project dir = workspace root), cwd = "." works.
	if filepath.IsAbs(cwd) {
		rel, err := filepath.Rel(r.workspace.Path, cwd)
		if err == nil && !strings.HasPrefix(rel, "..") {
			cwd = rel
		}
	}

	env := make(map[string]string, len(spec.Env))
	for _, kv := range spec.Env {
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			env[kv[:idx]] = kv[idx+1:]
		}
	}

	result, err := r.runtime.RunProgram(ctx, r.workspace, codeexecutor.RunProgramSpec{
		Cmd:     r.shellPath,
		Args:    []string{"-c", spec.Command},
		Cwd:     cwd,
		Env:     env,
		Timeout: spec.Timeout,
	})
	if err != nil {
		// RunProgram returns a sandboxError on timeout with the result populated.
		return ShellResult{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
			TimedOut: result.TimedOut,
		}, err
	}
	return ShellResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		TimedOut: result.TimedOut,
	}, nil
}

func (r *RuntimeShellRunner) Start(ctx context.Context, spec ShellSpec) (*ShellHandle, error) {
	cwd := spec.WorkDir
	if cwd == "" {
		cwd = "."
	}
	if filepath.IsAbs(cwd) {
		rel, err := filepath.Rel(r.workspace.Path, cwd)
		if err == nil && !strings.HasPrefix(rel, "..") {
			cwd = rel
		}
	}

	env := make(map[string]string, len(spec.Env))
	for _, kv := range spec.Env {
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			env[kv[:idx]] = kv[idx+1:]
		}
	}

	proc, err := r.runtime.StartProcess(ctx, r.workspace, ProcessSpec{
		Cmd:     r.shellPath,
		Args:    []string{"-c", spec.Command},
		Cwd:     cwd,
		Env:     env,
		Timeout: 0, // background: run until killed or exited
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox: start background process: %w", err)
	}

	handle := newShellHandle(proc.Kill)

	// Drain stdout and stderr into the handle's combined buffer. These
	// goroutines complete when the child exits (pipes return EOF).
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(handle, proc.Stdout())
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(handle, proc.Stderr())
	}()

	// Wait for output to be fully drained, then reap the process and finish.
	go func() {
		wg.Wait()
		err := proc.Wait()
		handle.finish(err)
	}()

	return handle, nil
}

// --- Container shell runner ---

// ContainerShellRunner runs shell commands inside a docker/podman container.
// The container CLI provides isolation; the sandbox Runtime is not involved.
type ContainerShellRunner struct {
	container *Container
}

// NewContainerShellRunner wraps a Container as a ShellRunner.
func NewContainerShellRunner(c *Container) *ContainerShellRunner {
	return &ContainerShellRunner{container: c}
}

func (c *ContainerShellRunner) Run(ctx context.Context, spec ShellSpec) (ShellResult, error) {
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, c.container.Runtime, c.container.buildArgs(containerRequest(spec))...)
	cmd.WaitDelay = postCancelWait
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	resp := ShellResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		resp.TimedOut = true
		resp.ExitCode = 124
		return resp, nil
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			resp.ExitCode = exitErr.ExitCode()
			return resp, nil
		}
		return resp, fmt.Errorf("%s run failed: %w (%s)", c.container.Runtime, err, stderr.String())
	}
	return resp, nil
}

func (c *ContainerShellRunner) Start(ctx context.Context, spec ShellSpec) (*ShellHandle, error) {
	cmdCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cmdCtx, c.container.Runtime, c.container.buildArgs(containerRequest(spec))...)
	cmd.WaitDelay = postCancelWait
	handle := newShellHandle(func() error { cancel(); return nil })
	cmd.Stdout = handle
	cmd.Stderr = handle
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	go func() {
		err := cmd.Wait()
		cancel()
		handle.finish(err)
	}()
	return handle, nil
}

func containerRequest(spec ShellSpec) Request {
	return Request{
		Command:    spec.Command,
		WorkingDir: spec.WorkDir,
		Env:        spec.Env,
	}
}

// Request is used internally by ContainerShellRunner to build docker args.
// It mirrors the subset of ShellSpec the container CLI needs.
type Request struct {
	Command    string
	WorkingDir string
	Env        []string
}
