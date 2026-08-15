// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package inbox

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestGormStore(t *testing.T) *GormStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&GormMessage{}))
	return NewGormStore(db)
}

func TestGormStore_CreateAndGetByID(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "Hello", Content: "World"}))

	var row GormMessage
	require.NoError(t, s.db.Where("user_id = ?", "u1").First(&row).Error)
	assert.Equal(t, "Hello", row.Title)

	got, err := s.GetByID("u1", "1")
	require.NoError(t, err)
	assert.Equal(t, "Hello", got.Title)
	assert.Equal(t, "World", got.Content)
	assert.False(t, got.Read)
}

func TestGormStore_Create_RequiresUserID(t *testing.T) {
	s := newTestGormStore(t)
	err := s.Create(Message{Title: "no user"})
	require.Error(t, err)
}

func TestGormStore_List_PaginationAndFilter(t *testing.T) {
	s := newTestGormStore(t)
	now := time.Now()
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A", Content: "x", Read: false, CreatedAt: now}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "B", Content: "y", Read: true, CreatedAt: now.Add(time.Second)}))
	require.NoError(t, s.Create(Message{UserID: "u2", Title: "C", Content: "z", Read: false}))

	// All
	res, err := s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Total)
	assert.EqualValues(t, 1, res.TotalUnread)
	assert.EqualValues(t, 1, res.TotalRead)
	assert.Len(t, res.List, 2)
	// Newest first
	assert.Equal(t, "B", res.List[0].Title)

	// Unread only
	res, err = s.List("u1", 1, 10, FilterUnread, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, res.Total)
	assert.Len(t, res.List, 1)
	assert.Equal(t, "A", res.List[0].Title)

	// Keyword
	res, err = s.List("u1", 1, 10, FilterAll, "B", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.EqualValues(t, 1, res.Total)
	assert.Equal(t, "B", res.List[0].Title)
}

func TestGormStore_UnreadCount(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "B"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "C", Read: true}))

	count, err := s.UnreadCount("u1")
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)
}

func TestGormStore_MarkRead(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))
	require.NoError(t, s.MarkRead("u1", "1"))
	got, err := s.GetByID("u1", "1")
	require.NoError(t, err)
	assert.True(t, got.Read)
}

func TestGormStore_MarkAllRead(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "B"}))
	require.NoError(t, s.MarkAllRead("u1"))
	count, err := s.UnreadCount("u1")
	require.NoError(t, err)
	assert.EqualValues(t, 0, count)
}

func TestGormStore_Delete(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))
	require.NoError(t, s.Delete("u1", "1"))
	_, err := s.GetByID("u1", "1")
	require.Error(t, err)
}

func TestGormStore_BatchDelete(t *testing.T) {
	s := newTestGormStore(t)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "B"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "C"}))

	removed, err := s.BatchDelete("u1", []string{"1", "2"})
	require.NoError(t, err)
	assert.EqualValues(t, 2, removed)

	removed, err = s.BatchDelete("u1", nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, removed)
}

func TestGormStore_CleanOldUnread(t *testing.T) {
	s := newTestGormStore(t)
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "old", CreatedAt: old}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "new"}))

	removed, err := s.CleanOldUnread(time.Now().Add(-1 * time.Hour))
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)

	count, err := s.UnreadCount("u1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}
