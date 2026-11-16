// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// CommandSource runs an external command and exposes its stdout as the
// backup stream. Stderr is captured and included in the error on failure.
type CommandSource struct {
	// Name is the executable (e.g. "mysqldump").
	Name string
	// Args are command arguments.
	Args []string
	// Env is appended to os.Environ() when non-nil (use KEY=value entries).
	Env []string
	// Dir is the working directory (optional).
	Dir string
}

// Open starts the command and returns a ReadCloser over stdout.
// Close waits for the process and returns a non-nil error if it failed.
func (s *CommandSource) Open(ctx context.Context) (io.ReadCloser, error) {
	if s == nil || s.Name == "" {
		return nil, fmt.Errorf("backup: empty command name")
	}
	cmd := exec.CommandContext(ctx, s.Name, s.Args...)
	if s.Dir != "" {
		cmd.Dir = s.Dir
	}
	if s.Env != nil {
		cmd.Env = append(os.Environ(), s.Env...)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("backup: stdout pipe: %w", err)
	}
	var stderrBuf limitedBuffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("backup: start %s: %w", s.Name, err)
	}
	return &cmdStdout{cmd: cmd, r: stdout, stderr: &stderrBuf}, nil
}

type cmdStdout struct {
	cmd     *exec.Cmd
	r       io.ReadCloser
	stderr  *limitedBuffer
	once    sync.Once
	waitErr error
}

func (c *cmdStdout) Read(p []byte) (int, error) { return c.r.Read(p) }

func (c *cmdStdout) Close() error {
	_ = c.r.Close()
	c.once.Do(func() {
		c.waitErr = c.cmd.Wait()
	})
	if c.waitErr != nil {
		msg := c.stderr.String()
		if msg != "" {
			return fmt.Errorf("backup: %s: %w: %s", c.cmd.Path, c.waitErr, msg)
		}
		return fmt.Errorf("backup: %s: %w", c.cmd.Path, c.waitErr)
	}
	return nil
}

// CommandSink feeds restored bytes to an external command's stdin
// (e.g. mysql, psql, mongorestore --archive).
type CommandSink struct {
	Name string
	Args []string
	Env  []string
	Dir  string
}

// Write starts the command, copies r to stdin, and waits for success.
func (s *CommandSink) Write(ctx context.Context, r io.Reader) error {
	if s == nil || s.Name == "" {
		return fmt.Errorf("backup: empty command name")
	}
	cmd := exec.CommandContext(ctx, s.Name, s.Args...)
	if s.Dir != "" {
		cmd.Dir = s.Dir
	}
	if s.Env != nil {
		cmd.Env = append(os.Environ(), s.Env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("backup: stdin pipe: %w", err)
	}
	var stderrBuf limitedBuffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("backup: start %s: %w", s.Name, err)
	}
	copyErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdin, r)
		_ = stdin.Close()
		copyErr <- err
	}()
	waitErr := cmd.Wait()
	cerr := <-copyErr
	if cerr != nil {
		return fmt.Errorf("backup: write stdin: %w", cerr)
	}
	if waitErr != nil {
		msg := stderrBuf.String()
		if msg != "" {
			return fmt.Errorf("backup: %s: %w: %s", s.Name, waitErr, msg)
		}
		return fmt.Errorf("backup: %s: %w", s.Name, waitErr)
	}
	return nil
}

// limitedBuffer keeps at most max bytes of stderr for error messages.
type limitedBuffer struct {
	mu  sync.Mutex
	b   []byte
	max int
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.max == 0 {
		l.max = 4096
	}
	remain := l.max - len(l.b)
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		l.b = append(l.b, p[:remain]...)
	} else {
		l.b = append(l.b, p...)
	}
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return string(l.b)
}
