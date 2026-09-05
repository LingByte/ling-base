// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/constants"
)

func TestNewDataSourceManager_SingleWrite(t *testing.T) {
	mgr, err := NewDataSourceManager(
		DataSourceConfig{Name: "master", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared"},
	)
	if err != nil {
		t.Fatalf("NewDataSourceManager: %v", err)
	}
	defer mgr.Close()

	if mgr.Write() == nil {
		t.Error("Write() returned nil")
	}
	if mgr.Read() == nil {
		t.Error("Read() returned nil")
	}
	// No read replicas → Read falls back to Write.
	if mgr.Read() != mgr.Write() {
		t.Error("Read should fall back to Write when no read replicas")
	}
}

func TestNewDataSourceManager_ReadWriteSplit(t *testing.T) {
	mgr, err := NewDataSourceManager(
		DataSourceConfig{Name: "master", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared&w=1"},
		DataSourceConfig{Name: "replica-1", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared&r=1"},
		DataSourceConfig{Name: "replica-2", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared&r=2"},
	)
	if err != nil {
		t.Fatalf("NewDataSourceManager: %v", err)
	}
	defer mgr.Close()

	if len(mgr.Writes()) != 1 {
		t.Errorf("writes = %d, want 1", len(mgr.Writes()))
	}
	if len(mgr.Reads()) != 2 {
		t.Errorf("reads = %d, want 2", len(mgr.Reads()))
	}

	// Read should NOT equal Write when read replicas exist.
	if mgr.Read() == mgr.Write() {
		t.Error("Read should differ from Write when read replicas exist")
	}

	// Round-robin: two consecutive Read calls should hit different replicas.
	first := mgr.Read()
	second := mgr.Read()
	if first == second {
		t.Error("round-robin: expected two different read instances")
	}
}

func TestNewDataSourceManager_NoConfigs(t *testing.T) {
	_, err := NewDataSourceManager()
	if err == nil {
		t.Error("expected error for zero configs")
	}
}

func TestNewDataSourceManager_NoWrite(t *testing.T) {
	_, err := NewDataSourceManager(
		DataSourceConfig{Name: "replica", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
	)
	if err == nil {
		t.Error("expected error when no write datasource configured")
	}
}

func TestNewDataSourceManager_DuplicateName(t *testing.T) {
	_, err := NewDataSourceManager(
		DataSourceConfig{Name: "db", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
		DataSourceConfig{Name: "db", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
	)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestDataSourceManager_GetByName(t *testing.T) {
	mgr, err := NewDataSourceManager(
		DataSourceConfig{Name: "master", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
		DataSourceConfig{Name: "replica", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
	)
	if err != nil {
		t.Fatalf("NewDataSourceManager: %v", err)
	}
	defer mgr.Close()

	if mgr.Get("master") == nil {
		t.Error("Get(master) returned nil")
	}
	if mgr.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) should return nil")
	}
}

func TestDataSourceManager_ContextSelection(t *testing.T) {
	mgr, err := NewDataSourceManager(
		DataSourceConfig{Name: "master", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared&ctx=w"},
		DataSourceConfig{Name: "replica", Role: RoleRead, Driver: constants.DBDriverSQLite, DSN: "file::memory:?cache=shared&ctx=r"},
	)
	if err != nil {
		t.Fatalf("NewDataSourceManager: %v", err)
	}
	defer mgr.Close()

	// Use RoleRead in context.
	ctx := mgr.Use(context.Background(), RoleRead)
	db := FromContext(ctx, nil)
	if db == nil {
		t.Fatal("FromContext returned nil for RoleRead")
	}
	if db != mgr.Read() {
		// Read() round-robins; with one replica it's deterministic.
		t.Error("FromContext(RoleRead) should return the read instance")
	}

	// Use RoleWrite in context.
	ctx = mgr.Use(context.Background(), RoleWrite)
	db = FromContext(ctx, nil)
	if db != mgr.Write() {
		t.Error("FromContext(RoleWrite) should return the write instance")
	}

	// UseName pins to a specific datasource.
	ctx = mgr.UseName(context.Background(), "master")
	db = FromContext(ctx, nil)
	if db != mgr.Get("master") {
		t.Error("FromContext(UseName master) should return the master instance")
	}

	// No selection → fallback.
	db = FromContext(context.Background(), mgr.Write())
	if db != mgr.Write() {
		t.Error("FromContext with no selection should return fallback")
	}
}

func TestDataSourceManager_PingAll(t *testing.T) {
	mgr, err := NewDataSourceManager(
		DataSourceConfig{Name: "master", Role: RoleWrite, Driver: constants.DBDriverSQLite, DSN: "file::memory:"},
	)
	if err != nil {
		t.Fatalf("NewDataSourceManager: %v", err)
	}
	defer mgr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := mgr.PingAll(ctx); err != nil {
		t.Errorf("PingAll: %v", err)
	}
}
