package swarm

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// newCommand creates an exec.Cmd for the given binary and args, with
// the working directory set to dir.
func newCommand(ctx context.Context, name string, args []string, dir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd
}

// Run executes the agent as a headless ling-agent subprocess.
func (r *execRunner) Run(ctx context.Context, sink Sink) error {
	args := r.buildArgs()
	cmd := newCommand(ctx, "ling-agent", args, r.agent.Dir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("swarm: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("swarm: start: %w", err)
	}

	sink.Activity("running")

	// Stream stdout to transcript.
	tmp := make([]byte, 4096)
	for {
		n, readErr := stdout.Read(tmp)
		if n > 0 {
			sink.Transcript(string(tmp[:n]))
		}
		if readErr != nil {
			if readErr != io.EOF {
				// non-EOF error; still wait for the process
			}
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
