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
