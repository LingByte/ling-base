// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gormmigrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/LingByte/ling-base/common/migration"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func TestNew(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource()
	m, err := New(db, src)
	require.NoError(t, err)
	assert.NotNil(t, m)

	// Verify schema_migrations table was created.
	assert.True(t, db.Migrator().HasTable("schema_migrations"))
}

func TestNew_NilDB(t *testing.T) {
	src := migration.NewStaticSource()
	_, err := New(nil, src)
	require.Error(t, err)
}

func TestNew_NilSource(t *testing.T) {
	db := newTestDB(t)
	_, err := New(db, nil)
	require.Error(t, err)
}

func TestMustNew(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource()
	m := MustNew(db, src)
	assert.NotNil(t, m)
}

func TestUp(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{
			Version:     1,
			Description: "create_users",
			UpSQL:       "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);",
			DownSQL:     "DROP TABLE users;",
		},
		migration.Migration{
			Version:     2,
			Description: "add_email",
			UpSQL:       "ALTER TABLE users ADD COLUMN email TEXT;",
			DownSQL:     "", // SQLite doesn't support DROP COLUMN easily
		},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	err = m.Up(context.Background())
	require.NoError(t, err)

	// Verify tables exist.
	assert.True(t, db.Migrator().HasTable("users"))

	// Verify version.
	ver, applied, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, uint64(2), ver)
}

func TestUp_Idempotent(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{
			Version:     1,
			Description: "create_users",
			UpSQL:       "CREATE TABLE users (id INTEGER PRIMARY KEY);",
			DownSQL:     "DROP TABLE users;",
		},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// First up.
	err = m.Up(context.Background())
	require.NoError(t, err)

	// Second up should be no-op.
	err = m.Up(context.Background())
	require.NoError(t, err)

	ver, _, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), ver)
}

func TestUpTo(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "a", UpSQL: "CREATE TABLE a (id INTEGER);", DownSQL: "DROP TABLE a;"},
		migration.Migration{Version: 2, Description: "b", UpSQL: "CREATE TABLE b (id INTEGER);", DownSQL: "DROP TABLE b;"},
		migration.Migration{Version: 3, Description: "c", UpSQL: "CREATE TABLE c (id INTEGER);", DownSQL: "DROP TABLE c;"},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Apply only up to version 2.
	err = m.UpTo(context.Background(), 2)
	require.NoError(t, err)

	ver, _, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), ver)

	assert.True(t, db.Migrator().HasTable("a"))
	assert.True(t, db.Migrator().HasTable("b"))
	assert.False(t, db.Migrator().HasTable("c"))
}

func TestDown(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{
			Version:     1,
			Description: "create_users",
			UpSQL:       "CREATE TABLE users (id INTEGER PRIMARY KEY);",
			DownSQL:     "DROP TABLE users;",
		},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Apply.
	err = m.Up(context.Background())
	require.NoError(t, err)
	assert.True(t, db.Migrator().HasTable("users"))

	// Rollback.
	err = m.Down(context.Background())
	require.NoError(t, err)
	assert.False(t, db.Migrator().HasTable("users"))

	ver, applied, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, uint64(0), ver)
}

func TestDownTo(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "a", UpSQL: "CREATE TABLE a (id INTEGER);", DownSQL: "DROP TABLE a;"},
		migration.Migration{Version: 2, Description: "b", UpSQL: "CREATE TABLE b (id INTEGER);", DownSQL: "DROP TABLE b;"},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Apply all.
	err = m.Up(context.Background())
	require.NoError(t, err)

	// Rollback to version 1 (keep version 1, rollback version 2).
	err = m.DownTo(context.Background(), 1)
	require.NoError(t, err)

	ver, _, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), ver)

	assert.True(t, db.Migrator().HasTable("a"))
	assert.False(t, db.Migrator().HasTable("b"))
}

func TestSteps(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "a", UpSQL: "CREATE TABLE a (id INTEGER);", DownSQL: "DROP TABLE a;"},
		migration.Migration{Version: 2, Description: "b", UpSQL: "CREATE TABLE b (id INTEGER);", DownSQL: "DROP TABLE b;"},
		migration.Migration{Version: 3, Description: "c", UpSQL: "CREATE TABLE c (id INTEGER);", DownSQL: "DROP TABLE c;"},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Step forward 2.
	err = m.Steps(context.Background(), 2)
	require.NoError(t, err)

	ver, _, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), ver)

	// Step back 1.
	err = m.Steps(context.Background(), -1)
	require.NoError(t, err)

	ver, _, err = m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), ver)

	// Step forward 1 again.
	err = m.Steps(context.Background(), 1)
	require.NoError(t, err)

	ver, _, err = m.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(2), ver)
}

func TestSteps_Zero(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "a", UpSQL: "CREATE TABLE a (id INTEGER);", DownSQL: "DROP TABLE a;"},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	err = m.Steps(context.Background(), 0)
	require.NoError(t, err)
}

func TestStatus(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "a", UpSQL: "CREATE TABLE a (id INTEGER);", DownSQL: "DROP TABLE a;"},
		migration.Migration{Version: 2, Description: "b", UpSQL: "CREATE TABLE b (id INTEGER);", DownSQL: "DROP TABLE b;"},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Apply first migration only.
	err = m.UpTo(context.Background(), 1)
	require.NoError(t, err)

	statuses, err := m.Status(context.Background())
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	assert.True(t, statuses[0].Applied)
	assert.Equal(t, uint64(1), statuses[0].Version)
	assert.NotEmpty(t, statuses[0].AppliedAt)

	assert.False(t, statuses[1].Applied)
	assert.Equal(t, uint64(2), statuses[1].Version)
	assert.Empty(t, statuses[1].AppliedAt)
}

func TestVersion_NoMigrations(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource()
	m, err := New(db, src)
	require.NoError(t, err)

	ver, applied, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, uint64(0), ver)
}

func TestClose(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource()
	m, err := New(db, src)
	require.NoError(t, err)

	err = m.Close()
	require.NoError(t, err)
}

func TestUp_EmptyMigration_NoOp(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{Version: 1, Description: "noop", UpSQL: "", DownSQL: ""},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	// Should still record the version even with empty SQL.
	err = m.Up(context.Background())
	require.NoError(t, err)

	ver, applied, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, uint64(1), ver)
}

func TestUp_FailedMigration(t *testing.T) {
	db := newTestDB(t)
	src := migration.NewStaticSource(
		migration.Migration{
			Version:     1,
			Description: "bad",
			UpSQL:       "CREATE TABLE INVALID SYNTAX!!!;",
			DownSQL:     "",
		},
	)

	m, err := New(db, src)
	require.NoError(t, err)

	err = m.Up(context.Background())
	require.Error(t, err)

	// Version should not be recorded.
	ver, applied, err := m.Version(context.Background())
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, uint64(0), ver)
	_ = ver
}
