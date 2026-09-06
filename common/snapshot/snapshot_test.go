// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package snapshot

import (
	"errors"
	"strings"
	"testing"
)

// ─── Basic CRUD tests ───

func TestStore_Create(t *testing.T) {
	store := NewStore[string]()
	snap, err := store.Create("doc-1", "Hello", "alice")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if snap.Version != 1 {
		t.Errorf("expected version 1, got %d", snap.Version)
	}
	if snap.Content != "Hello" {
		t.Errorf("expected 'Hello', got %q", snap.Content)
	}
	if snap.Contributor != "alice" {
		t.Errorf("expected alice, got %q", snap.Contributor)
	}
}

func TestStore_Create_Duplicate(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "Hello", "alice")
	_, err := store.Create("doc-1", "World", "bob")
	if err == nil {
		t.Error("expected error for duplicate subject")
	}
}

func TestStore_Update(t *testing.T) {
	store := NewStore[string]()
	snap1, _ := store.Create("doc-1", "Hello", "alice")
	snap2, err := store.Update("doc-1", snap1.Version, "Hello World", "bob")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if snap2.Version != 2 {
		t.Errorf("expected version 2, got %d", snap2.Version)
	}
	if snap2.Content != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", snap2.Content)
	}
}

func TestStore_Update_VersionConflict(t *testing.T) {
	store := NewStore[string]()
	snap1, _ := store.Create("doc-1", "Hello", "alice")
	// Try to update with wrong expected version.
	_, err := store.Update("doc-1", snap1.Version+10, "World", "bob")
	if !errors.Is(err, ErrVersionConflict) {
		t.Errorf("expected ErrVersionConflict, got %v", err)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	store := NewStore[string]()
	_, err := store.Update("missing", 1, "content", "alice")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ─── Get tests ───

func TestStore_GetLatest(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")
	store.Update("doc-1", 2, "v3", "carol")

	snap, err := store.GetLatest("doc-1")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if snap.Version != 3 {
		t.Errorf("expected version 3, got %d", snap.Version)
	}
	if snap.Content != "v3" {
		t.Errorf("expected 'v3', got %q", snap.Content)
	}
}

func TestStore_GetLatest_NotFound(t *testing.T) {
	store := NewStore[string]()
	_, err := store.GetLatest("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_GetByVersion(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")

	snap, err := store.GetByVersion("doc-1", 1)
	if err != nil {
		t.Fatalf("get by version: %v", err)
	}
	if snap.Content != "v1" {
		t.Errorf("expected 'v1', got %q", snap.Content)
	}
}

func TestStore_GetByVersion_NotFound(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	_, err := store.GetByVersion("doc-1", 999)
	if !errors.Is(err, ErrVersionNotFound) {
		t.Errorf("expected ErrVersionNotFound, got %v", err)
	}
}

func TestStore_GetContent(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "Hello World", "alice")

	content, err := store.GetContent("doc-1")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if content != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", content)
	}
}

func TestStore_GetContentAtVersion(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")

	content, err := store.GetContentAtVersion("doc-1", 1)
	if err != nil {
		t.Fatalf("get content at version: %v", err)
	}
	if content != "v1" {
		t.Errorf("expected 'v1', got %q", content)
	}
}

// ─── Rollback tests ───

func TestStore_Rollback(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")
	store.Update("doc-1", 2, "v3", "carol")

	// Rollback to v1.
	snap, err := store.Rollback("doc-1", 1, "alice")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if snap.Version != 4 {
		t.Errorf("expected version 4, got %d", snap.Version)
	}
	if snap.Content != "v1" {
		t.Errorf("expected 'v1' after rollback, got %q", snap.Content)
	}

	// Latest content should be v1.
	content, _ := store.GetContent("doc-1")
	if content != "v1" {
		t.Errorf("expected 'v1' as current, got %q", content)
	}
}

func TestStore_Rollback_NotFound(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	_, err := store.Rollback("doc-1", 999, "alice")
	if !errors.Is(err, ErrVersionNotFound) {
		t.Errorf("expected ErrVersionNotFound, got %v", err)
	}
}

// ─── History tests ───

func TestStore_History(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")
	store.Update("doc-1", 2, "v3", "carol")

	history, err := store.History("doc-1")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(history))
	}
	// Verify versions are in order.
	for i, snap := range history {
		if snap.Version != int64(i+1) {
			t.Errorf("expected version %d at index %d, got %d", i+1, i, snap.Version)
		}
	}
}

func TestStore_History_NotFound(t *testing.T) {
	store := NewStore[string]()
	_, err := store.History("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ─── Contributors tests ───

func TestStore_Contributors(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")
	store.Update("doc-1", 2, "v3", "alice") // alice again

	contributors, err := store.Contributors("doc-1")
	if err != nil {
		t.Fatalf("contributors: %v", err)
	}
	if len(contributors) != 2 {
		t.Fatalf("expected 2 unique contributors, got %d", len(contributors))
	}
}

// ─── Delete tests ───

func TestStore_Delete(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")

	err := store.Delete("doc-1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = store.GetLatest("doc-1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	store := NewStore[string]()
	err := store.Delete("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ─── Count and ListSubjects tests ───

func TestStore_Count(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v1", "alice")
	store.Update("doc-1", 1, "v2", "bob")

	count, err := store.Count("doc-1")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestStore_ListSubjects(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-a", "v1", "alice")
	store.Create("doc-b", "v1", "bob")
	store.Create("doc-c", "v1", "carol")

	subjects := store.ListSubjects()
	if len(subjects) != 3 {
		t.Fatalf("expected 3 subjects, got %d", len(subjects))
	}
	// Should be sorted.
	if subjects[0] != "doc-a" || subjects[1] != "doc-b" || subjects[2] != "doc-c" {
		t.Errorf("unexpected order: %v", subjects)
	}
}

// ─── Diff-based snapshot tests ───

// simpleDiff creates a simple "old→new" patch string.
func simpleDiff(old, new string) (string, error) {
	return old + "\x00" + new, nil
}

// simplePatch applies a "old→new" patch.
func simplePatch(content, patch string) (string, error) {
	parts := strings.SplitN(patch, "\x00", 2)
	if len(parts) == 2 {
		return parts[1], nil // return the "new" part
	}
	return patch, nil
}

func TestStore_WithDiffFunc(t *testing.T) {
	store := NewStore[string](WithDiffFunc[string](
		simpleDiff, simplePatch,
		func(s string) string { return s },
		func(s string) (string, error) { return s, nil },
	))

	snap1, _ := store.Create("doc-1", "Hello", "alice")
	if snap1.IsPatch {
		t.Error("first snapshot should not be a patch")
	}

	snap2, _ := store.Update("doc-1", 1, "Hello World", "bob")
	if !snap2.IsPatch {
		t.Error("subsequent snapshot should be a patch")
	}
	if snap2.Patch == "" {
		t.Error("expected non-empty patch")
	}

	// Content should still be reconstructable.
	content, _ := store.GetContent("doc-1")
	if content != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", content)
	}
}

// ─── Concurrency tests ───

func TestStore_ConcurrentUpdates(t *testing.T) {
	store := NewStore[string]()
	store.Create("doc-1", "v0", "init")

	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for {
				latest, err := store.GetLatest("doc-1")
				if err != nil {
					done <- err
					return
				}
				_, err = store.Update("doc-1", latest.Version,
					"v"+latest.Content[1:], "g"+string(rune('0'+n)))
				if err == nil {
					done <- nil
					return
				}
				if !errors.Is(err, ErrVersionConflict) {
					done <- err
					return
				}
				// Retry on conflict.
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("goroutine %d error: %v", i, err)
		}
	}

	count, _ := store.Count("doc-1")
	if count != 11 { // 1 initial + 10 updates
		t.Errorf("expected 11 snapshots, got %d", count)
	}
}

// ─── Struct content tests ───

type Article struct {
	Title string
	Body  string
}

func TestStore_StructContent(t *testing.T) {
	store := NewStore[Article]()
	store.Create("art-1", Article{Title: "T1", Body: "B1"}, "alice")
	store.Update("art-1", 1, Article{Title: "T2", Body: "B2"}, "bob")

	content, _ := store.GetContent("art-1")
	if content.Title != "T2" || content.Body != "B2" {
		t.Errorf("unexpected content: %+v", content)
	}
}
