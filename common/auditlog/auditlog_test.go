// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package auditlog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_LogAndQuery(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	err := logger.LogAction(ctx, "user-1", "login", "session", "sess-1",
		WithIP("10.0.0.1"),
		WithUserAgent("curl/8"),
		WithStatus("success"),
		WithDetail(map[string]any{"method": "password"}),
		WithRequestID("req-1"),
	)
	require.NoError(t, err)

	entries, err := store.Query(ctx, &Filter{UserID: "user-1"})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.NotEmpty(t, e.ID)
	assert.False(t, e.Timestamp.IsZero())
	assert.Equal(t, "user-1", e.UserID)
	assert.Equal(t, "login", e.Action)
	assert.Equal(t, "session", e.Resource)
	assert.Equal(t, "sess-1", e.ResourceID)
	assert.Equal(t, "10.0.0.1", e.IP)
	assert.Equal(t, "curl/8", e.UserAgent)
	assert.Equal(t, "success", e.Status)
	assert.Equal(t, "password", e.Detail["method"])
	assert.Equal(t, "req-1", e.RequestID)
}

func TestLogger_LogDirectEntry(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	// Pre-filled ID and timestamp should be preserved.
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	err := logger.Log(ctx, &Entry{
		ID:        "custom-id",
		Timestamp: ts,
		UserID:    "u",
		Action:    "a",
	})
	require.NoError(t, err)

	entries, err := store.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "custom-id", entries[0].ID)
	assert.Equal(t, ts, entries[0].Timestamp)
}

func TestLogger_NilStorage(t *testing.T) {
	logger := NewLogger(nil)
	err := logger.Log(context.Background(), &Entry{UserID: "u"})
	assert.Error(t, err)
}

func TestLogger_NilEntry(t *testing.T) {
	logger := NewLogger(NewMemoryStorage())
	err := logger.Log(context.Background(), nil)
	assert.Error(t, err)
}

func TestMemoryStorage_Filter(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	base := time.Now()
	entries := []*Entry{
		{UserID: "u1", Action: "create", Resource: "order", Timestamp: base.Add(-3 * time.Hour)},
		{UserID: "u1", Action: "update", Resource: "order", Timestamp: base.Add(-2 * time.Hour)},
		{UserID: "u2", Action: "create", Resource: "order", Timestamp: base.Add(-1 * time.Hour)},
		{UserID: "u1", Action: "delete", Resource: "user", Timestamp: base},
	}
	for _, e := range entries {
		// Pre-set ID so Log doesn't overwrite; but we want auto ID too,
		// so just log directly with the timestamp preserved.
		err := logger.Log(ctx, e)
		require.NoError(t, err)
	}

	// Filter by user.
	res, err := store.Query(ctx, &Filter{UserID: "u1"})
	require.NoError(t, err)
	assert.Len(t, res, 3)

	// Filter by action.
	res, err = store.Query(ctx, &Filter{Action: "create"})
	require.NoError(t, err)
	assert.Len(t, res, 2)

	// Filter by resource.
	res, err = store.Query(ctx, &Filter{Resource: "order"})
	require.NoError(t, err)
	assert.Len(t, res, 3)

	// Filter by time range.
	start := base.Add(-90 * time.Minute)
	end := base.Add(1 * time.Minute)
	res, err = store.Query(ctx, &Filter{StartTime: &start, EndTime: &end})
	require.NoError(t, err)
	assert.Len(t, res, 2) // the -1h and base entries

	// Combined filter + limit.
	res, err = store.Query(ctx, &Filter{UserID: "u1", Limit: 1})
	require.NoError(t, err)
	require.Len(t, res, 1)
	// Newest-first ordering: the latest u1 entry is the "delete" one.
	assert.Equal(t, "delete", res[0].Action)

	// No matches.
	res, err = store.Query(ctx, &Filter{UserID: "nobody"})
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestMemoryStorage_QueryNilFilter(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()
	_ = store.Save(ctx, &Entry{UserID: "u", Action: "a"})
	_ = store.Save(ctx, &Entry{UserID: "u2", Action: "b"})

	res, err := store.Query(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestNewID_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id, err := newID()
		require.NoError(t, err)
		assert.Len(t, id, 32) // 16 bytes hex
		_, dup := seen[id]
		assert.False(t, dup, "duplicate id generated")
		seen[id] = struct{}{}
	}
}

func TestLogger_NilLogger(t *testing.T) {
	// Calling Log on a nil *Logger must return an error, not panic.
	var l *Logger
	err := l.Log(context.Background(), &Entry{UserID: "u"})
	assert.Error(t, err)
}

func TestLogger_LogAction_NoOptions(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	// LogAction with no options still records a valid entry.
	err := logger.LogAction(ctx, "u", "a", "r", "rid")
	require.NoError(t, err)

	entries, err := store.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "u", e.UserID)
	assert.Equal(t, "a", e.Action)
	assert.Equal(t, "r", e.Resource)
	assert.Equal(t, "rid", e.ResourceID)
	assert.Empty(t, e.IP)
	assert.Empty(t, e.Status)
	assert.NotEmpty(t, e.ID)
	assert.False(t, e.Timestamp.IsZero())
}

func TestLogger_LogAction_AllOptions(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	err := logger.LogAction(ctx, "u", "a", "r", "rid",
		WithIP("1.2.3.4"),
		WithUserAgent("test-agent"),
		WithStatus("ok"),
		WithDetail(map[string]any{"k": "v"}),
		WithRequestID("req-x"),
	)
	require.NoError(t, err)

	entries, err := store.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "1.2.3.4", e.IP)
	assert.Equal(t, "test-agent", e.UserAgent)
	assert.Equal(t, "ok", e.Status)
	assert.Equal(t, "v", e.Detail["k"])
	assert.Equal(t, "req-x", e.RequestID)
}

func TestMemoryStorage_EmptyFilter(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Querying an empty store with an empty filter returns no entries.
	res, err := store.Query(ctx, &Filter{})
	require.NoError(t, err)
	assert.Empty(t, res)

	// Querying an empty store with nil filter also returns no entries.
	res, err = store.Query(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestMemoryStorage_FilterEndTimeExcludes(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	base := time.Now()
	entries := []*Entry{
		{UserID: "u", Action: "old", Timestamp: base.Add(-2 * time.Hour)},
		{UserID: "u", Action: "new", Timestamp: base},
	}
	for _, e := range entries {
		require.NoError(t, logger.Log(ctx, e))
	}

	// EndTime that excludes the newest entry.
	end := base.Add(-1 * time.Hour)
	res, err := store.Query(ctx, &Filter{EndTime: &end})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "old", res[0].Action)
}

func TestMemoryStorage_FilterStartTimeExcludes(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	base := time.Now()
	entries := []*Entry{
		{UserID: "u", Action: "old", Timestamp: base.Add(-2 * time.Hour)},
		{UserID: "u", Action: "new", Timestamp: base},
	}
	for _, e := range entries {
		require.NoError(t, logger.Log(ctx, e))
	}

	// StartTime that excludes the oldest entry.
	start := base.Add(-1 * time.Hour)
	res, err := store.Query(ctx, &Filter{StartTime: &start})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "new", res[0].Action)
}

func TestMemoryStorage_FilterLimit(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, logger.Log(ctx, &Entry{UserID: "u", Action: "a"}))
	}

	// Limit caps the result count.
	res, err := store.Query(ctx, &Filter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestMemoryStorage_FilterActionAndResourceMismatch(t *testing.T) {
	store := NewMemoryStorage()
	logger := NewLogger(store)
	ctx := context.Background()

	require.NoError(t, logger.Log(ctx, &Entry{UserID: "u", Action: "create", Resource: "order"}))
	require.NoError(t, logger.Log(ctx, &Entry{UserID: "u", Action: "update", Resource: "user"}))

	// Action mismatch.
	res, err := store.Query(ctx, &Filter{Action: "delete"})
	require.NoError(t, err)
	assert.Empty(t, res)

	// Resource mismatch.
	res, err = store.Query(ctx, &Filter{Resource: "product"})
	require.NoError(t, err)
	assert.Empty(t, res)
}
