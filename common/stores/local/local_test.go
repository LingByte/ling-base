// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package local

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	s := New(Config{})
	if s.root != "uploads" {
		t.Errorf("root = %q, want uploads", s.root)
	}
	if s.newDirPerm != 0755 {
		t.Errorf("perm = %v, want 0755", s.newDirPerm)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	s := New(Config{Root: "/tmp/test-store", NewDirPerm: 0700})
	if s.root != "/tmp/test-store" {
		t.Errorf("root = %q", s.root)
	}
	if s.newDirPerm != 0700 {
		t.Errorf("perm = %v", s.newDirPerm)
	}
}

func TestStore_WriteReadDelete(t *testing.T) {
	dir := t.TempDir()
	s := New(Config{Root: dir})

	// Write
	content := []byte("hello world")
	if err := s.Write("test/file.txt", bytes.NewReader(content)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Exists
	ok, err := s.Exists("test/file.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("expected file to exist")
	}

	// Read
	r, size, err := s.Read("test/file.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer r.Close()
	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	buf := make([]byte, size)
	if _, err := r.Read(buf); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(buf) != string(content) {
		t.Errorf("content = %q, want %q", buf, content)
	}

	// Delete
	if err := s.Delete("test/file.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ok, _ = s.Exists("test/file.txt")
	if ok {
		t.Fatal("expected file to not exist after delete")
	}
}

func TestStore_Read_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, _, err := s.Read("nope.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestStore_Exists_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	ok, err := s.Exists("nope.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected false for non-existent file")
	}
}

func TestStore_PathTraversal(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.resolveKey("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestStore_PublicURL(t *testing.T) {
	s := New(Config{Root: "uploads"})
	got := s.PublicURL("images/photo.jpg")
	if !strings.HasSuffix(got, "uploads/images/photo.jpg") {
		t.Errorf("PublicURL = %q", got)
	}
}

func TestStore_Overwrite(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.Write("file.txt", bytes.NewReader([]byte("old")))
	s.Write("file.txt", bytes.NewReader([]byte("new")))
	r, _, _ := s.Read("file.txt")
	defer r.Close()
	buf := make([]byte, 3)
	r.Read(buf)
	if string(buf) != "new" {
		t.Errorf("content = %q, want new", buf)
	}
}

func TestStore_Delete_NonExist_NoError(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if err := s.Delete("nope.txt"); err != nil {
		t.Errorf("Delete non-existent should not error: %v", err)
	}
}

func TestStore_AbsoluteRoot(t *testing.T) {
	dir := t.TempDir()
	abs, _ := filepath.Abs(dir)
	s := New(Config{Root: abs})
	fname, err := s.resolveKey("test.txt")
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}
	if !filepath.IsAbs(fname) {
		t.Errorf("expected absolute path, got %q", fname)
	}
	// cleanup
	os.RemoveAll(abs)
}
