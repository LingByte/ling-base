// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileSource reads a local file.
type FileSource struct {
	Path string
}

// Open opens the file for reading.
func (s *FileSource) Open(_ context.Context) (io.ReadCloser, error) {
	if s == nil || s.Path == "" {
		return nil, fmt.Errorf("backup: empty source path")
	}
	return os.Open(s.Path)
}

// FileSink overwrites a local file with restored content.
type FileSink struct {
	Path string
}

// Write replaces the file contents with r.
func (s *FileSink) Write(_ context.Context, r io.Reader) error {
	if s == nil || s.Path == "" {
		return fmt.Errorf("backup: empty sink path")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil && !os.IsExist(err) {
		// MkdirAll on "." or bare filename: Dir may be "." — still ok
		if filepath.Dir(s.Path) != "." {
			return fmt.Errorf("backup: mkdir sink: %w", err)
		}
	}
	f, err := os.Create(s.Path)
	if err != nil {
		return fmt.Errorf("backup: create sink file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("backup: write sink file: %w", err)
	}
	return nil
}

// FileDestination stores artifacts in a local directory.
type FileDestination struct {
	Dir string
}

// Write stores r under Dir/name (appending .gz when the stream is gzip).
func (d *FileDestination) Write(_ context.Context, name string, r io.Reader) (string, error) {
	if d == nil || d.Dir == "" {
		return "", fmt.Errorf("backup: empty destination dir")
	}
	if err := os.MkdirAll(d.Dir, 0o755); err != nil {
		return "", fmt.Errorf("backup: mkdir: %w", err)
	}
	br := bufio.NewReader(r)
	peek, _ := br.Peek(2)
	fname := name
	if len(peek) == 2 && peek[0] == 0x1f && peek[1] == 0x8b {
		if !strings.HasSuffix(fname, ".gz") {
			fname += ".gz"
		}
	}
	path := filepath.Join(d.Dir, fname)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("backup: create file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, br); err != nil {
		return "", fmt.Errorf("backup: write file: %w", err)
	}
	return path, nil
}

// Open opens a previously written artifact by key (absolute or relative path).
func (d *FileDestination) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if d == nil || d.Dir == "" {
		return nil, fmt.Errorf("backup: empty destination dir")
	}
	path := key
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.Dir, key)
	}
	return os.Open(path)
}

// Remove deletes an artifact by key.
func (d *FileDestination) Remove(_ context.Context, key string) error {
	if d == nil || d.Dir == "" {
		return fmt.Errorf("backup: empty destination dir")
	}
	path := key
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.Dir, key)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
