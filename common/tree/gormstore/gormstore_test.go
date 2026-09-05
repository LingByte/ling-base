// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gormstore_test

import (
	"context"
	"errors"
	"testing"

	gormstore "github.com/LingByte/ling-base/common/tree/gormstore"
	"github.com/LingByte/ling-base/common/tree"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSQLiteStore(t *testing.T) *gormstore.Store {
	t.Helper()
	// Use a file-based temp DB instead of :memory: because the tree
	// package's Tx wraps each call in a transaction, and shared in-memory
	// SQLite has subtle cross-tx visibility quirks. A temp file is
	// simplest and cleaned up by t.TempDir().
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/tree.db"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := gormstore.New(db)
	if err != nil {
		t.Fatalf("gormstore.New: %v", err)
	}
	return s
}

func TestGormStore_AddAndQuery(t *testing.T) {
	store := newSQLiteStore(t)
	tr, err := tree.New(store)
	if err != nil {
		t.Fatalf("tree.New: %v", err)
	}
	ctx := context.Background()

	for _, p := range []string{"inf.monitor.prometheus", "inf.cicd.jenkins", "inf.monitor.thanos"} {
		if _, err := tr.Add(ctx, p); err != nil {
			t.Fatalf("Add %s: %v", p, err)
		}
	}

	// Exists
	ok, err := tr.Exists(ctx, "inf.monitor.prometheus")
	if err != nil || !ok {
		t.Errorf("Exists = %v %v", ok, err)
	}

	// Query children of group
	children, err := tr.Query(ctx, "inf", tree.QueryChildren)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("children = %v, want 2", children)
	}

	// Query leaves under group
	leaves, err := tr.Query(ctx, "inf", tree.QueryLeaves)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(leaves) != 3 {
		t.Errorf("leaves = %v, want 3", leaves)
	}

	// Delete cascade
	n, err := tr.Delete(ctx, "inf", true)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 6 {
		t.Errorf("deleted = %d, want 6", n)
	}
}

func TestGormStore_Duplicate(t *testing.T) {
	store := newSQLiteStore(t)
	tr, _ := tree.New(store)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	_, err := tr.Add(ctx, "inf.monitor.prometheus")
	if !errors.Is(err, tree.ErrNodeExists) {
		t.Errorf("second Add err = %v, want ErrNodeExists", err)
	}
}

func TestGormStore_DeleteNonForce(t *testing.T) {
	store := newSQLiteStore(t)
	tr, _ := tree.New(store)
	ctx := context.Background()

	if _, err := tr.Add(ctx, "inf.monitor.prometheus"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err := tr.Delete(ctx, "inf", false)
	if !errors.Is(err, tree.ErrHasChildren) {
		t.Errorf("Delete err = %v, want ErrHasChildren", err)
	}
}
