// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/ling-base/agent"
	"context"
)

const (
	maxReadLines = 2000
	maxReadBytes = 50 * 1024 // 50 KB
)

// ReadTool reads file contents from disk.
type ReadTool struct {
	CWD string
}

// NewRead creates a read tool confined to cwd.
func NewRead(cwd string) *ReadTool {
	return &ReadTool{CWD: cwd}
}

type readArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

const readSchema = `{"type":"object","properties":{"path":{"type":"string","description":"File path relative to the working directory."},"offset":{"type":"integer","description":"Line number to start reading from (1-based). Default: 1."},"limit":{"type":"integer","description":"Maximum number of lines to read. Default: 2000."}},"required":["path"]}`

func (t *ReadTool) Name() string { return "read" }
func (t *ReadTool) Description() string {
	return "Read a file from the working directory. Returns content with line numbers."
}
func (t *ReadTool) Schema() json.RawMessage { return json.RawMessage(readSchema) }

func (t *ReadTool) Execute(ctx context.Context, raw json.RawMessage, progress func(string)) (agent.ToolResult, error) {
	var a readArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return agent.ToolResult{}, fmt.Errorf("invalid args: %w", err)
	}
	if a.Path == "" {
		return agent.ToolResult{}, fmt.Errorf("path is required")
	}

	path := resolvePath(t.CWD, a.Path)
	if !isInside(path, t.CWD) {
		return agent.ToolResult{Content: fmt.Sprintf("error: path %s is outside the working directory", a.Path), IsError: true}, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error: %v", err), IsError: true}, nil
	}
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return agent.ToolResult{Content: fmt.Sprintf("error reading dir: %v", err), IsError: true}, nil
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Directory: %s\n\n", a.Path))
		for _, e := range entries {
			if e.IsDir() {
				b.WriteString(e.Name() + "/\n")
			} else {
				b.WriteString(e.Name() + "\n")
			}
		}
		return agent.ToolResult{Content: b.String()}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error: %v", err), IsError: true}, nil
	}

	// Truncate if too large.
	if len(data) > maxReadBytes {
		data = data[:maxReadBytes]
	}

	lines := strings.Split(string(data), "\n")
	offset := a.Offset
	if offset < 1 {
		offset = 1
	}
	limit := a.Limit
	if limit <= 0 {
		limit = maxReadLines
	}

	var b strings.Builder
	end := offset + limit - 1
	if end > len(lines) {
		end = len(lines)
	}
	for i := offset - 1; i < end; i++ {
		fmt.Fprintf(&b, "%4d\t%s\n", i+1, lines[i])
	}

	if len(data) >= maxReadBytes {
		b.WriteString(fmt.Sprintf("\n... (truncated at %d bytes)\n", maxReadBytes))
	}

	return agent.ToolResult{Content: b.String()}, nil
}

// WriteTool writes content to a file, creating it if it doesn't exist.
type WriteTool struct {
	CWD string
}

// NewWrite creates a write tool confined to cwd.
func NewWrite(cwd string) *WriteTool {
	return &WriteTool{CWD: cwd}
}

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

const writeSchema = `{"type":"object","properties":{"path":{"type":"string","description":"File path relative to the working directory."},"content":{"type":"string","description":"The full content to write to the file."}},"required":["path","content"]}`

func (t *WriteTool) Name() string { return "write" }
func (t *WriteTool) Description() string {
	return "Write content to a file. Creates the file if it doesn't exist, overwrites if it does."
}
func (t *WriteTool) Schema() json.RawMessage { return json.RawMessage(writeSchema) }

func (t *WriteTool) Execute(ctx context.Context, raw json.RawMessage, progress func(string)) (agent.ToolResult, error) {
	var a writeArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return agent.ToolResult{}, fmt.Errorf("invalid args: %w", err)
	}
	if a.Path == "" {
		return agent.ToolResult{}, fmt.Errorf("path is required")
	}

	path := resolvePath(t.CWD, a.Path)
	if !isInside(path, t.CWD) {
		return agent.ToolResult{Content: fmt.Sprintf("error: path %s is outside the working directory", a.Path), IsError: true}, nil
	}

	// Create parent directories if needed.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error creating dirs: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(path, []byte(a.Content), 0o644); err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
	}

	lineCount := strings.Count(a.Content, "\n") + 1
	return agent.ToolResult{Content: fmt.Sprintf("Successfully wrote %d lines to %s", lineCount, a.Path)}, nil
}

// EditTool performs an exact string replacement in a file.
type EditTool struct {
	CWD string
}

// NewEdit creates an edit tool confined to cwd.
func NewEdit(cwd string) *EditTool {
	return &EditTool{CWD: cwd}
}

type editArgs struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

const editSchema = `{"type":"object","properties":{"path":{"type":"string","description":"File path relative to the working directory."},"old_string":{"type":"string","description":"The exact string to find in the file."},"new_string":{"type":"string","description":"The string to replace old_string with."}},"required":["path","old_string","new_string"]}`

func (t *EditTool) Name() string { return "edit" }
func (t *EditTool) Description() string {
	return "Edit a file by replacing an exact string. The old_string must be unique in the file."
}
func (t *EditTool) Schema() json.RawMessage { return json.RawMessage(editSchema) }

func (t *EditTool) Execute(ctx context.Context, raw json.RawMessage, progress func(string)) (agent.ToolResult, error) {
	var a editArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return agent.ToolResult{}, fmt.Errorf("invalid args: %w", err)
	}
	if a.Path == "" {
		return agent.ToolResult{}, fmt.Errorf("path is required")
	}
	if a.OldString == "" {
		return agent.ToolResult{}, fmt.Errorf("old_string is required")
	}
	if a.OldString == a.NewString {
		return agent.ToolResult{}, fmt.Errorf("old_string and new_string are identical")
	}

	path := resolvePath(t.CWD, a.Path)
	if !isInside(path, t.CWD) {
		return agent.ToolResult{Content: fmt.Sprintf("error: path %s is outside the working directory", a.Path), IsError: true}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error: %v", err), IsError: true}, nil
	}

	content := string(data)
	count := strings.Count(content, a.OldString)
	if count == 0 {
		return agent.ToolResult{Content: fmt.Sprintf("error: old_string not found in %s", a.Path), IsError: true}, nil
	}
	if count > 1 {
		return agent.ToolResult{Content: fmt.Sprintf("error: old_string appears %d times in %s; make it unique", count, a.Path), IsError: true}, nil
	}

	newContent := strings.Replace(content, a.OldString, a.NewString, 1)
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return agent.ToolResult{Content: fmt.Sprintf("error writing file: %v", err), IsError: true}, nil
	}

	return agent.ToolResult{Content: fmt.Sprintf("Successfully edited %s", a.Path)}, nil
}
