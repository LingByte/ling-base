package sandbox

import (
	"os/exec"
	"time"
)

// execLookPath is exec.LookPath, aliased so tests can override it.
var execLookPath = exec.LookPath

// postCancelWait bounds how long a container process waits for stdout/stderr
// pipes after the process exits or ctx is cancelled.
const postCancelWait = 5 * time.Second

// Container describes a docker/podman container used by ContainerShellRunner.
// The OS sandbox runtime is OS-level only (seatbelt/bwrap), so container
// execution keeps the direct CLI path.
type Container struct {
	Runtime  string // "docker" or "podman"
	Image    string
	MountCWD bool
	ReadOnly bool
	Network  string
}

// NewContainer builds a container descriptor. runtime defaults to "docker".
func NewContainer(runtime, image string, mountCWD, readOnly bool, network string) *Container {
	if runtime == "" {
		runtime = "docker"
	}
	return &Container{Runtime: runtime, Image: image, MountCWD: mountCWD, ReadOnly: readOnly, Network: network}
}

func (c *Container) Name() string { return c.Runtime + ":" + c.Image }

// buildArgs assembles the `docker run` arguments for a request.
func (c *Container) buildArgs(req Request) []string {
	args := []string{"run", "--rm", "-i"}
	if c.Network != "" {
		args = append(args, "--network", c.Network)
	}
	if c.MountCWD && req.WorkingDir != "" {
		mount := req.WorkingDir + ":" + req.WorkingDir
		if c.ReadOnly {
			mount += ":ro"
		}
		args = append(args, "-v", mount, "-w", req.WorkingDir)
	}
	for _, e := range req.Env {
		args = append(args, "-e", e)
	}
	args = append(args, c.Image, "sh", "-c", req.Command)
	return args
}

// RuntimeAvailable reports whether the named container runtime is on PATH.
func RuntimeAvailable(runtime string) bool {
	if runtime == "" {
		runtime = "docker"
	}
	_, err := execLookPath(runtime)
	return err == nil
}
