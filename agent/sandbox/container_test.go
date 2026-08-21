package sandbox

import (
	"errors"
	"testing"
)

func TestNewContainerDefaultsRuntime(t *testing.T) {
	c := NewContainer("", "python:3.12-slim", true, false, "none")
	if c.Runtime != "docker" {
		t.Errorf("Runtime = %q, want 'docker' when empty", c.Runtime)
	}
	if c.Image != "python:3.12-slim" {
		t.Errorf("Image = %q", c.Image)
	}
	if !c.MountCWD {
		t.Error("MountCWD should be true")
	}
	if c.ReadOnly {
		t.Error("ReadOnly should be false")
	}
	if c.Network != "none" {
		t.Errorf("Network = %q, want 'none'", c.Network)
	}
}

func TestNewContainerPodman(t *testing.T) {
	c := NewContainer("podman", "alpine:latest", false, true, "bridge")
	if c.Runtime != "podman" {
		t.Errorf("Runtime = %q, want 'podman'", c.Runtime)
	}
	if c.MountCWD {
		t.Error("MountCWD should be false")
	}
	if !c.ReadOnly {
		t.Error("ReadOnly should be true")
	}
}

func TestContainerName(t *testing.T) {
	c := NewContainer("docker", "python:3.12", true, false, "")
	if got := c.Name(); got != "docker:python:3.12" {
		t.Errorf("Name() = %q, want 'docker:python:3.12'", got)
	}
}

func TestContainerBuildArgsBasic(t *testing.T) {
	c := NewContainer("docker", "python:3.12-slim", false, false, "")
	args := c.buildArgs(Request{Command: "echo hello"})
	// Expected: run --rm -i <image> sh -c "echo hello"
	if len(args) < 5 {
		t.Fatalf("args too short: %v", args)
	}
	if args[0] != "run" || args[1] != "--rm" || args[2] != "-i" {
		t.Errorf("prefix args = %v, want [run --rm -i]", args[:3])
	}
	if args[len(args)-3] != "sh" || args[len(args)-2] != "-c" {
		t.Errorf("suffix = %v, want [... sh -c <cmd>]", args[len(args)-3:])
	}
	// The command should be present.
	if !containsString(args, "echo hello") {
		t.Errorf("args = %v, want 'echo hello' present", args)
	}
}

func TestContainerBuildArgsWithNetwork(t *testing.T) {
	c := NewContainer("docker", "alpine", false, false, "none")
	args := c.buildArgs(Request{Command: "true"})
	if !containsPair(args, "--network", "none") {
		t.Errorf("args = %v, want --network none", args)
	}
}

func TestContainerBuildArgsWithMountCWD(t *testing.T) {
	c := NewContainer("docker", "alpine", true, false, "")
	args := c.buildArgs(Request{Command: "pwd", WorkingDir: "/project"})
	// Should contain -v /project:/project and -w /project.
	if !containsPair(args, "-v", "/project:/project") {
		t.Errorf("args = %v, want -v /project:/project", args)
	}
	if !containsPair(args, "-w", "/project") {
		t.Errorf("args = %v, want -w /project", args)
	}
}

func TestContainerBuildArgsWithReadOnlyMount(t *testing.T) {
	c := NewContainer("docker", "alpine", true, true, "")
	args := c.buildArgs(Request{Command: "ls", WorkingDir: "/repo"})
	if !containsPair(args, "-v", "/repo:/repo:ro") {
		t.Errorf("args = %v, want -v /repo:/repo:ro", args)
	}
}

func TestContainerBuildArgsWithEnv(t *testing.T) {
	c := NewContainer("docker", "alpine", false, false, "")
	args := c.buildArgs(Request{Command: "env", Env: []string{"FOO=bar", "BAZ=qux"}})
	if !containsPair(args, "-e", "FOO=bar") {
		t.Errorf("args = %v, want -e FOO=bar", args)
	}
	if !containsPair(args, "-e", "BAZ=qux") {
		t.Errorf("args = %v, want -e BAZ=qux", args)
	}
}

func TestContainerBuildArgsNoMountWhenDisabled(t *testing.T) {
	c := NewContainer("docker", "alpine", false, false, "")
	args := c.buildArgs(Request{Command: "pwd", WorkingDir: "/project"})
	for i, a := range args {
		if a == "-v" || a == "-w" {
			t.Errorf("unexpected mount arg %s at index %d: %v", a, i, args)
		}
	}
}

func TestRuntimeAvailableDocker(t *testing.T) {
	// Docker may or may not be installed; just verify the function doesn't
	// panic and returns a bool.
	got := RuntimeAvailable("docker")
	_ = got
}

func TestRuntimeAvailableEmptyDefaultsDocker(t *testing.T) {
	// Empty runtime should default to "docker" lookup.
	orig := execLookPath
	defer func() { execLookPath = orig }()
	execLookPath = func(file string) (string, error) {
		if file != "docker" {
			t.Errorf("looked up %q, want 'docker'", file)
		}
		return "/usr/local/bin/docker", nil
	}
	if !RuntimeAvailable("") {
		t.Error("RuntimeAvailable('') should return true when docker is on PATH")
	}
}

func TestRuntimeAvailableNotFound(t *testing.T) {
	orig := execLookPath
	defer func() { execLookPath = orig }()
	execLookPath = func(file string) (string, error) {
		return "", errNotFound
	}
	if RuntimeAvailable("nonexistent-runtime") {
		t.Error("expected false for non-existent runtime")
	}
}

// --- helpers ---

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func containsPair(slice []string, flag, value string) bool {
	for i := 0; i < len(slice)-1; i++ {
		if slice[i] == flag && slice[i+1] == value {
			return true
		}
	}
	return false
}

// errNotFound is a sentinel for execLookPath test override.
var errNotFound = errors.New("executable not found")
