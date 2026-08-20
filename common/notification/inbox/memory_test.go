// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package inbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────

// seed creates `n` messages for the user with deterministic titles and
// read state alternating starting from `startRead`.
func seed(t *testing.T, s *MemoryStore, userID string, n int, startRead bool) []Message {
	t.Helper()
	out := make([]Message, 0, n)
	read := startRead
	for i := 0; i < n; i++ {
		msg := Message{
			UserID:  userID,
			Title:   "Title " + userID + "-" + itoa(i),
			Content: "Content " + itoa(i),
			Read:    read,
		}
		require.NoError(t, s.Create(msg))
		// fetch back to get the assigned ID/timestamps
		got, err := s.GetByID(userID, lastID(s, userID))
		require.NoError(t, err)
		out = append(out, *got)
		read = !read
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// lastID returns the ID of the most recently appended message for the
// user (messages are appended in creation order).
func lastID(s *MemoryStore, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.messages[userID]
	if len(list) == 0 {
		return ""
	}
	return list[len(list)-1].ID
}

// ──────────────────────────────────────────────
// MemoryStore.Create
// ──────────────────────────────────────────────

func TestMemoryStore_Create(t *testing.T) {
	s := NewMemoryStore()

	// happy path: ID and timestamps generated
	before := time.Now()
	msg := Message{UserID: "u1", Title: "Hello", Content: "World"}
	require.NoError(t, s.Create(msg))

	got, err := s.GetByID("u1", "1")
	require.NoError(t, err)
	assert.Equal(t, "1", got.ID)
	assert.Equal(t, "u1", got.UserID)
	assert.Equal(t, "Hello", got.Title)
	assert.False(t, got.Read)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
	assert.True(t, got.CreatedAt.After(before.Add(-time.Second)))

	// second message gets incremented ID
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "Second"}))
	got2, err := s.GetByID("u1", "2")
	require.NoError(t, err)
	assert.Equal(t, "2", got2.ID)

	// missing userID
	err = s.Create(Message{Title: "no user"})
	assert.Error(t, err)

	// pre-set ID and timestamps are preserved
	fixed := time.Unix(1000, 0)
	require.NoError(t, s.Create(Message{ID: "custom", UserID: "u2", Title: "x", CreatedAt: fixed, UpdatedAt: fixed}))
	got3, err := s.GetByID("u2", "custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", got3.ID)
	assert.Equal(t, fixed, got3.CreatedAt)
}

// ──────────────────────────────────────────────
// MemoryStore.GetByID
// ──────────────────────────────────────────────

func TestMemoryStore_GetByID(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "A"}))

	got, err := s.GetByID("u1", "1")
	require.NoError(t, err)
	assert.Equal(t, "A", got.Title)

	// not found: wrong id
	_, err = s.GetByID("u1", "999")
	assert.Error(t, err)

	// not found: wrong user
	_, err = s.GetByID("nobody", "1")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// MemoryStore.List
// ──────────────────────────────────────────────

func TestMemoryStore_List_Pagination(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 5, false)

	// page 1 size 2 -> 2 items, total 5
	res, err := s.List("u1", 1, 2, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)
	assert.Equal(t, int64(5), res.Total)
	assert.Equal(t, int64(3), res.TotalUnread) // indices 0,2,4 unread
	assert.Equal(t, int64(2), res.TotalRead)

	// page 3 size 2 -> 1 item
	res, err = s.List("u1", 3, 2, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 1)

	// page beyond range -> empty list, total still 5
	res, err = s.List("u1", 10, 2, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, res.List)
	assert.Equal(t, int64(5), res.Total)

	// default page/size when zero/negative
	res, err = s.List("u1", 0, 0, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 5) // default size 10 -> all 5 fit
	assert.Equal(t, int64(5), res.Total)
}

func TestMemoryStore_List_Filters(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 4, false) // unread, read, unread, read

	// all
	res, err := s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 4)

	// unread
	res, err = s.List("u1", 1, 10, FilterUnread, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)
	for _, m := range res.List {
		assert.False(t, m.Read)
	}

	// read
	res, err = s.List("u1", 1, 10, FilterRead, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)
	for _, m := range res.List {
		assert.True(t, m.Read)
	}

	// invalid filter falls back to all
	res, err = s.List("u1", 1, 10, "bogus", "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 4)
}

func TestMemoryStore_List_Keywords(t *testing.T) {
	s := NewMemoryStore()
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "Welcome Aboard", Content: "Please confirm your email"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "Maintenance", Content: "System downtime soon"}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "welcome back", Content: "another welcome"}))

	// title keyword (case-insensitive)
	res, err := s.List("u1", 1, 10, FilterAll, "welcome", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)

	// content keyword
	res, err = s.List("u1", 1, 10, FilterAll, "", "confirm", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 1)
	assert.Equal(t, "Welcome Aboard", res.List[0].Title)

	// keyword that matches nothing
	res, err = s.List("u1", 1, 10, FilterAll, "nonexistent", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, res.List)
	assert.Equal(t, int64(0), res.Total)
}

func TestMemoryStore_List_TimeRange(t *testing.T) {
	s := NewMemoryStore()
	old := time.Unix(1000, 0)
	mid := time.Unix(2000, 0)
	recent := time.Unix(3000, 0)

	require.NoError(t, s.Create(Message{UserID: "u1", Title: "old", Content: "x", CreatedAt: old}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "mid", Content: "x", CreatedAt: mid}))
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "new", Content: "x", CreatedAt: recent}))

	// startTime=mid excludes older (old)
	res, err := s.List("u1", 1, 10, FilterAll, "", "", mid, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2) // mid + new

	// endTime=mid excludes newer (new)
	res, err = s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, mid)
	require.NoError(t, err)
	assert.Len(t, res.List, 2) // old + mid

	// both bounds == mid -> only mid
	res, err = s.List("u1", 1, 10, FilterAll, "", "", mid, mid)
	require.NoError(t, err)
	assert.Len(t, res.List, 1)
	assert.Equal(t, "mid", res.List[0].Title)
}

func TestMemoryStore_List_EmptyUser(t *testing.T) {
	s := NewMemoryStore()
	res, err := s.List("ghost", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, res.List)
	assert.Equal(t, int64(0), res.Total)
	assert.Equal(t, int64(0), res.TotalUnread)
	assert.Equal(t, int64(0), res.TotalRead)
}

// ──────────────────────────────────────────────
// MemoryStore.UnreadCount
// ──────────────────────────────────────────────

func TestMemoryStore_UnreadCount(t *testing.T) {
	s := NewMemoryStore()

	// no messages -> 0
	count, err := s.UnreadCount("u1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	seed(t, s, "u1", 4, false) // unread, read, unread, read
	count, err = s.UnreadCount("u1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// other user unaffected
	count, err = s.UnreadCount("u2")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// ──────────────────────────────────────────────
// MemoryStore.MarkRead
// ──────────────────────────────────────────────

func TestMemoryStore_MarkRead(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 2, false)

	require.NoError(t, s.MarkRead("u1", "1"))
	got, err := s.GetByID("u1", "1")
	require.NoError(t, err)
	assert.True(t, got.Read)

	// idempotent: marking an already-read message is a no-op
	require.NoError(t, s.MarkRead("u1", "1"))

	// not found
	err = s.MarkRead("u1", "999")
	assert.Error(t, err)

	// wrong user
	err = s.MarkRead("nobody", "1")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// MemoryStore.MarkAllRead
// ──────────────────────────────────────────────

func TestMemoryStore_MarkAllRead(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 4, false) // 2 unread, 2 read

	require.NoError(t, s.MarkAllRead("u1"))

	count, err := s.UnreadCount("u1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// all messages now read
	res, err := s.List("u1", 1, 10, FilterRead, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 4)

	// mark all on a user with no messages is fine
	require.NoError(t, s.MarkAllRead("ghost"))
}

// ──────────────────────────────────────────────
// MemoryStore.Delete
// ──────────────────────────────────────────────

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 3, false)

	require.NoError(t, s.Delete("u1", "2"))
	_, err := s.GetByID("u1", "2")
	assert.Error(t, err)

	// remaining
	res, err := s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)

	// not found
	err = s.Delete("u1", "999")
	assert.Error(t, err)

	// wrong user
	err = s.Delete("nobody", "1")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// MemoryStore.BatchDelete
// ──────────────────────────────────────────────

func TestMemoryStore_BatchDelete(t *testing.T) {
	s := NewMemoryStore()
	seed(t, s, "u1", 5, false)

	// delete 2 existing + 1 non-existent -> removed count 2
	removed, err := s.BatchDelete("u1", []string{"1", "3", "999"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed)

	res, err := s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 3)

	// empty ids -> 0 removed
	removed, err = s.BatchDelete("u1", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), removed)

	// user with no messages
	removed, err = s.BatchDelete("ghost", []string{"1"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), removed)
}

// ──────────────────────────────────────────────
// MemoryStore.CleanOldUnread
// ──────────────────────────────────────────────

func TestMemoryStore_CleanOldUnread(t *testing.T) {
	s := NewMemoryStore()
	old := time.Unix(1000, 0)
	recent := time.Unix(2000, 0)

	// old unread (should be cleaned)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "old-unread", Read: false, CreatedAt: old}))
	// old read (should be preserved)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "old-read", Read: true, CreatedAt: old}))
	// recent unread (should be preserved)
	require.NoError(t, s.Create(Message{UserID: "u1", Title: "recent-unread", Read: false, CreatedAt: recent}))
	// another user's old unread (should be cleaned)
	require.NoError(t, s.Create(Message{UserID: "u2", Title: "old-unread-u2", Read: false, CreatedAt: old}))

	threshold := time.Unix(1500, 0)
	removed, err := s.CleanOldUnread(threshold)
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed)

	// u1 has 2 left
	res, err := s.List("u1", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Len(t, res.List, 2)

	// u2 has 0 left
	res, err = s.List("u2", 1, 10, FilterAll, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, res.List)

	// clean again -> nothing to remove
	removed, err = s.CleanOldUnread(threshold)
	require.NoError(t, err)
	assert.Equal(t, int64(0), removed)
}

// ──────────────────────────────────────────────
// sortByCreatedDesc
// ──────────────────────────────────────────────

func TestSortByCreatedDesc(t *testing.T) {
	t1 := time.Unix(100, 0)
	t2 := time.Unix(200, 0)
	t3 := time.Unix(300, 0)
	msgs := []Message{
		{ID: "a", CreatedAt: t2},
		{ID: "b", CreatedAt: t1},
		{ID: "c", CreatedAt: t3},
	}
	sortByCreatedDesc(msgs)
	assert.Equal(t, "c", msgs[0].ID) // newest first
	assert.Equal(t, "a", msgs[1].ID)
	assert.Equal(t, "b", msgs[2].ID)
}

func TestSortByCreatedDesc_Empty(t *testing.T) {
	sortByCreatedDesc(nil) // should not panic
	sortByCreatedDesc([]Message{})
}

func TestSortByCreatedDesc_Single(t *testing.T) {
	msgs := []Message{{ID: "x", CreatedAt: time.Now()}}
	sortByCreatedDesc(msgs)
	assert.Len(t, msgs, 1)
}
