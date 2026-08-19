// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMigrationRunner is a test MigrationRunner.
type mockMigrationRunner struct {
	called int32
	err    error
}

func (m *mockMigrationRunner) RunMigrations(ctx context.Context) error {
	atomic.AddInt32(&m.called, 1)
	return m.err
}

// mockAutoMigrator is a test AutoMigrator.
type mockAutoMigrator struct {
	called int32
	models []any
	err    error
}

func (m *mockAutoMigrator) AutoMigrate(dst ...any) error {
	atomic.AddInt32(&m.called, 1)
	m.models = dst
	return m.err
}

func TestWithMigration(t *testing.T) {
	runner := &mockMigrationRunner{}
	app := New("test", WithMigration(runner))

	assert.NotNil(t, app.migrationRunner)
	assert.Nil(t, app.autoMigrator)
}

func TestWithAutoMigrate(t *testing.T) {
	am := &mockAutoMigrator{}
	type User struct{}
	type Order struct{}
	app := New("test", WithAutoMigrate(am, &User{}, &Order{}))

	assert.Nil(t, app.migrationRunner)
	assert.NotNil(t, app.autoMigrator)
	assert.Len(t, app.autoMigrateModels, 2)
}

func TestRunMigrations_MigrationRunner(t *testing.T) {
	runner := &mockMigrationRunner{}
	app := New("test", WithMigration(runner))

	err := app.runMigrations(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&runner.called))
}

func TestRunMigrations_AutoMigrate(t *testing.T) {
	am := &mockAutoMigrator{}
	type User struct{}
	app := New("test", WithAutoMigrate(am, &User{}))

	err := app.runMigrations(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&am.called))
	assert.Len(t, am.models, 1)
}

func TestRunMigrations_None(t *testing.T) {
	app := New("test")
	err := app.runMigrations(context.Background())
	require.NoError(t, err)
}

func TestRunMigrations_Error(t *testing.T) {
	runner := &mockMigrationRunner{err: context.DeadlineExceeded}
	app := New("test", WithMigration(runner))

	err := app.runMigrations(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration failed")
}

func TestRunMigrations_AutoMigrateError(t *testing.T) {
	am := &mockAutoMigrator{err: context.DeadlineExceeded}
	type User struct{}
	app := New("test", WithAutoMigrate(am, &User{}))

	err := app.runMigrations(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auto-migrate failed")
}

func TestMigrationRunnerInterface(t *testing.T) {
	var _ MigrationRunner = (*mockMigrationRunner)(nil)
	var _ AutoMigrator = (*mockAutoMigrator)(nil)
}
