// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tree_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/tree"
	"github.com/LingByte/ling-base/common/tree/memory"
)

// fixedClock returns a deterministic time for stable CreatedAt/UpdatedAt.
func fixedClock() func() time.Time {
	t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func newTestTree(t *testing.T) tree.Tree {
	t.Helper()
	tr, err := tree.New(memory.New(), tree.WithClock(fixedClock()))
	if err != nil {
		t.Fatalf("tree.New: %v", err)
	}
	return tr
}

func TestAdd_FullPath(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	leaf, err := tr.Add(ctx, "inf.monitor.prometheus")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if leaf.Level != tree.LevelApp {
		t.Errorf("leaf level = %d, want %d", leaf.Level, tree.LevelApp)
	}
	if leaf.Name != "prometheus" {
		t.Errorf("leaf name = %q, want %q", leaf.Name, "prometheus")
	}
}

func TestAdd_Duplicate(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	_, err := tr.Add(ctx, "inf.monitor.prometheus")
	if !errors.Is(err, tree.ErrNodeExists) {
		t.Fatalf("second Add err = %v, want ErrNodeExists", err)
	}
}

func TestAdd_AddAppToExistingGroupProject(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	// Create g.p first.
	if _, err := tr.Add(ctx, "inf.monitor"); err != nil {
		t.Fatalf("Add g.p: %v", err)
	}
	// Now add a second app under the same g.p.
	if _, err := tr.Add(ctx, "inf.monitor.thanos"); err != nil {
		t.Fatalf("Add second app: %v", err)
	}
	// Both apps should exist.
	ok1, _ := tr.Exists(ctx, "inf.monitor.prometheus")
	// prometheus wasn't added yet; add it.
	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add prometheus: %v", err)
	}
	ok1, _ = tr.Exists(ctx, "inf.monitor.prometheus")
	ok2, _ := tr.Exists(ctx, "inf.monitor.thanos")
	if !ok1 || !ok2 {
		t.Errorf("expected both apps to exist: prometheus=%v thanos=%v", ok1, ok2)
	}
}

func TestGet(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	n, err := tr.Get(ctx, "inf.monitor.prometheus")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if n.Name != "prometheus" {
		t.Errorf("name = %q", n.Name)
	}

	_, err = tr.Get(ctx, "inf.nonexistent")
	if !errors.Is(err, tree.ErrNodeNotFound) {
		t.Errorf("Get nonexistent err = %v, want ErrNodeNotFound", err)
	}
}

func TestExists(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	ok, err := tr.Exists(ctx, "inf.monitor.prometheus")
	if err != nil || !ok {
		t.Errorf("Exists = %v %v, want true nil", ok, err)
	}
	ok, err = tr.Exists(ctx, "inf.monitor.nonexistent")
	if err != nil || ok {
		t.Errorf("Exists = %v %v, want false nil", ok, err)
	}
	// Exists requires full g.p.a.
	_, err = tr.Exists(ctx, "inf.monitor")
	if !errors.Is(err, tree.ErrInvalidPath) {
		t.Errorf("Exists g.p err = %v, want ErrInvalidPath", err)
	}
}

func TestQuery_ChildrenOfGroup(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.cicd.jenkins", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	children, err := tr.Query(ctx, "inf", tree.QueryChildren)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := map[string]bool{"monitor": true, "cicd": true}
	if len(children) != 2 {
		t.Fatalf("children = %v, want 2", children)
	}
	for _, c := range children {
		if !want[c] {
			t.Errorf("unexpected child %q", c)
		}
	}
}

func TestQuery_LeavesUnderGroup(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.cicd.jenkins", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	leaves, err := tr.Query(ctx, "inf", tree.QueryLeaves)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := map[string]bool{
		"inf.monitor.prometheus": true,
		"inf.cicd.jenkins":       true,
		"inf.monitor.thanos":     true,
	}
	if len(leaves) != 3 {
		t.Fatalf("leaves = %v, want 3", leaves)
	}
	for _, l := range leaves {
		if !want[l] {
			t.Errorf("unexpected leaf %q", l)
		}
	}
}

func TestQuery_LeavesUnderProject(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	leaves, err := tr.Query(ctx, "inf.monitor", tree.QueryLeaves)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := map[string]bool{
		"inf.monitor.prometheus": true,
		"inf.monitor.thanos":     true,
	}
	if len(leaves) != 2 {
		t.Fatalf("leaves = %v, want 2", leaves)
	}
	for _, l := range leaves {
		if !want[l] {
			t.Errorf("unexpected leaf %q", l)
		}
	}
}

func TestQuery_Exists(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	res, err := tr.Query(ctx, "inf.monitor.prometheus", tree.QueryExists)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res) != 1 || res[0] != "inf.monitor.prometheus" {
		t.Errorf("res = %v", res)
	}

	res, err = tr.Query(ctx, "inf.monitor.nonexistent", tree.QueryExists)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("res = %v, want empty", res)
	}
}

func TestDelete_NonForceWithChildren(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err := tr.Delete(ctx, "inf", false)
	if !errors.Is(err, tree.ErrHasChildren) {
		t.Errorf("Delete err = %v, want ErrHasChildren", err)
	}
}

func TestDelete_ForceCascade(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.cicd.jenkins", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	// Delete inf with force: should remove 1 group + 2 projects + 3 apps = 6.
	n, err := tr.Delete(ctx, "inf", true)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 6 {
		t.Errorf("deleted = %d, want 6", n)
	}
	// inf should no longer exist.
	ok, _ := tr.Exists(ctx, "inf.monitor.prometheus")
	if ok {
		t.Error("inf.monitor.prometheus still exists after cascade delete")
	}
}

func TestDelete_Leaf(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	n, err := tr.Delete(ctx, "inf.monitor.prometheus", false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted = %d, want 1", n)
	}
	// Parent should still exist.
	_, err = tr.Get(ctx, "inf.monitor")
	if err != nil {
		t.Errorf("parent deleted: %v", err)
	}
}

func TestListChildren(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	children, err := tr.ListChildren(ctx, "inf.monitor")
	if err != nil {
		t.Fatalf("ListChildren: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("children = %d, want 2", len(children))
	}
}

func TestSubTree(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.cicd.jenkins"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}
	sub, err := tr.SubTree(ctx, "inf")
	if err != nil {
		t.Fatalf("SubTree: %v", err)
	}
	// 1 group + 2 projects + 2 apps = 5
	if len(sub) != 5 {
		t.Errorf("subtree size = %d, want 5", len(sub))
	}
}

func TestInvalidPath(t *testing.T) {
	tr := newTestTree(t)
	ctx := context.Background()

	cases := []string{"", "inf..monitor", "a.b.c.d"}
	for _, p := range cases {
		_, err := tr.Add(ctx, p)
		if err == nil {
			t.Errorf("Add(%q) expected error, got nil", p)
		}
	}
}
