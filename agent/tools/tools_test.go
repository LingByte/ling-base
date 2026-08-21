// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("line1\nline2\nline3\n"), 0o644)

	tool := NewRead(dir)
	args, _ := json.Marshal(map[string]any{"path": "test.txt"})
	result, err := tool.Execute(context.Background(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !contains(result.Content, "line1") || !contains(result.Content, "line2") {
		t.Errorf("expected content to contain lines, got: %s", result.Content)
	}
}

func TestReadToolOutsideCWD(t *testing.T) {
	dir := t.TempDir()
	tool := NewRead(dir)
	args, _ := json.Marshal(map[string]any{"path": "../../etc/passwd"})
	result, _ := tool.Execute(context.Background(), args, nil)
	if !result.IsError {
		t.Error("expected error for path outside CWD")
	}
}

func TestWriteTool(t *testing.T) {
	dir := t.TempDir()
	tool := NewWrite(dir)
	args, _ := json.Marshal(map[string]any{
		"path":    "new.txt",
		"content": "hello world",
	})
	result, err := tool.Execute(context.Background(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(data))
	}
}

func TestWriteToolCreatesDirs(t *testing.T) {
	dir := t.TempDir()
	tool := NewWrite(dir)
	args, _ := json.Marshal(map[string]any{
		"path":    "sub/dir/file.txt",
		"content": "nested",
	})
	result, err := tool.Execute(context.Background(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "dir", "file.txt")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestEditTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("foo bar baz"), 0o644)

	tool := NewEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"path":       "edit.txt",
		"old_string": "bar",
		"new_string": "qux",
	})
	result, err := tool.Execute(context.Background(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "edit.txt"))
	if string(data) != "foo qux baz" {
		t.Errorf("expected 'foo qux baz', got %s", string(data))
	}
}

func TestEditToolNotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("hello"), 0o644)

	tool := NewEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"path":       "edit.txt",
		"old_string": "nonexistent",
		"new_string": "replacement",
	})
	result, _ := tool.Execute(context.Background(), args, nil)
	if !result.IsError {
		t.Error("expected error for not found")
	}
}

func TestEditToolMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "dup.txt"), []byte("foo foo foo"), 0o644)

	tool := NewEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"path":       "dup.txt",
		"old_string": "foo",
		"new_string": "bar",
	})
	result, _ := tool.Execute(context.Background(), args, nil)
	if !result.IsError {
		t.Error("expected error for multiple matches")
	}
}

func TestBashTool(t *testing.T) {
	dir := t.TempDir()
	tool := NewBash(dir)
	args, _ := json.Marshal(map[string]any{"command": "echo hello"})
	result, err := tool.Execute(context.Background(), args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !contains(result.Content, "hello") {
		t.Errorf("expected output to contain 'hello', got: %s", result.Content)
	}
}

func TestBashToolTimeout(t *testing.T) {
	dir := t.TempDir()
	tool := NewBash(dir)
	args, _ := json.Marshal(map[string]any{"command": "sleep 10", "timeout": 1})
	result, _ := tool.Execute(context.Background(), args, nil)
	if !result.IsError {
		t.Error("expected timeout error")
	}
	if !contains(result.Content, "timed out") {
		t.Errorf("expected 'timed out' in message, got: %s", result.Content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
