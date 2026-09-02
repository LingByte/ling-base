// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package trie

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Insert / Search
// ──────────────────────────────────────────────

func TestNew_Empty(t *testing.T) {
	tr := New()
	assert.Equal(t, 0, tr.Size())
	assert.Empty(t, tr.Keys())
}

func TestInsert_Search(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)

	v, ok := tr.Search("apple")
	require.True(t, ok)
	assert.Equal(t, 1, v)
	assert.Equal(t, 1, tr.Size())
}

func TestInsert_Overwrite(t *testing.T) {
	tr := New()
	tr.Insert("key", "old")
	tr.Insert("key", "new")

	v, ok := tr.Search("key")
	require.True(t, ok)
	assert.Equal(t, "new", v)
	assert.Equal(t, 1, tr.Size(), "overwrite should not increase size")
}

func TestInsert_EmptyKey_Ignored(t *testing.T) {
	tr := New()
	tr.Insert("", 1)
	assert.Equal(t, 0, tr.Size())

	_, ok := tr.Search("")
	assert.False(t, ok)
}

func TestSearch_Missing(t *testing.T) {
	tr := New()
	tr.Insert("hello", 1)

	_, ok := tr.Search("hell")
	assert.False(t, ok, "prefix that is not a full key should not match")
	_, ok = tr.Search("hellox")
	assert.False(t, ok)
	_, ok = tr.Search("xyz")
	assert.False(t, ok)
}

func TestInsertMany(t *testing.T) {
	tr := New()
	tr.InsertMany(map[string]any{
		"a":   1,
		"ab":  2,
		"abc": 3,
		"b":   4,
	})

	assert.Equal(t, 4, tr.Size())

	v, ok := tr.Search("abc")
	require.True(t, ok)
	assert.Equal(t, 3, v)
}

func TestInsertMany_Empty(t *testing.T) {
	tr := New()
	tr.InsertMany(nil)
	assert.Equal(t, 0, tr.Size())
}

// ──────────────────────────────────────────────
// Delete
// ──────────────────────────────────────────────

func TestDelete_Existing(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)
	tr.Insert("app", 2)

	assert.True(t, tr.Delete("apple"))
	assert.Equal(t, 1, tr.Size())
	_, ok := tr.Search("apple")
	assert.False(t, ok)

	// "app" still present
	v, ok := tr.Search("app")
	require.True(t, ok)
	assert.Equal(t, 2, v)
}

func TestDelete_PrunesLeafNodes(t *testing.T) {
	tr := New()
	tr.Insert("abc", 1)

	assert.True(t, tr.Delete("abc"))
	assert.Equal(t, 0, tr.Size())
	assert.False(t, tr.HasPrefix("a"))
	assert.False(t, tr.HasPrefix("ab"))
}

func TestDelete_KeepsSharedBranch(t *testing.T) {
	tr := New()
	tr.Insert("abc", 1)
	tr.Insert("abd", 2)

	assert.True(t, tr.Delete("abc"))
	// "ab" branch must remain because "abd" still exists
	assert.True(t, tr.HasPrefix("ab"))
	v, ok := tr.Search("abd")
	require.True(t, ok)
	assert.Equal(t, 2, v)
}

func TestDelete_Missing(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)

	assert.False(t, tr.Delete("app"))
	assert.False(t, tr.Delete("xyz"))
	assert.False(t, tr.Delete(""))
}

func TestDelete_ThenReinsert(t *testing.T) {
	tr := New()
	tr.Insert("key", 1)
	require.True(t, tr.Delete("key"))
	tr.Insert("key", 2)

	v, ok := tr.Search("key")
	require.True(t, ok)
	assert.Equal(t, 2, v)
	assert.Equal(t, 1, tr.Size())
}

// ──────────────────────────────────────────────
// HasPrefix / PrefixMatch / PrefixMatchWithValues
// ──────────────────────────────────────────────

func TestHasPrefix(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)
	tr.Insert("app", 2)
	tr.Insert("banana", 3)

	assert.True(t, tr.HasPrefix("app"))
	assert.True(t, tr.HasPrefix("apple"))
	assert.True(t, tr.HasPrefix("ban"))
	assert.False(t, tr.HasPrefix("cat"))
	assert.False(t, tr.HasPrefix("apx"))
	assert.True(t, tr.HasPrefix(""), "empty prefix matches everything")
}

func TestPrefixMatch(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)
	tr.Insert("app", 2)
	tr.Insert("application", 3)
	tr.Insert("banana", 4)

	got := tr.PrefixMatch("app")
	assert.Equal(t, []string{"app", "apple", "application"}, got)
}

func TestPrefixMatch_EmptyPrefix_ReturnsAll(t *testing.T) {
	tr := New()
	tr.Insert("b", 1)
	tr.Insert("a", 2)
	tr.Insert("c", 3)

	got := tr.PrefixMatch("")
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestPrefixMatch_NoMatch(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)

	assert.Empty(t, tr.PrefixMatch("xyz"))
}

func TestPrefixMatchWithValues(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)
	tr.Insert("app", 2)
	tr.Insert("banana", 3)

	got := tr.PrefixMatchWithValues("app")
	assert.Equal(t, map[string]any{"app": 2, "apple": 1}, got)
}

func TestPrefixMatchWithValues_NoMatch(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)
	assert.Nil(t, tr.PrefixMatchWithValues("xyz"))
}

// ──────────────────────────────────────────────
// Keys / Clear
// ──────────────────────────────────────────────

func TestKeys_Sorted(t *testing.T) {
	tr := New()
	tr.Insert("banana", 1)
	tr.Insert("apple", 2)
	tr.Insert("cherry", 3)

	assert.Equal(t, []string{"apple", "banana", "cherry"}, tr.Keys())
}

func TestKeys_Empty(t *testing.T) {
	tr := New()
	assert.Empty(t, tr.Keys())
}

func TestClear(t *testing.T) {
	tr := New()
	tr.Insert("a", 1)
	tr.Insert("b", 2)

	tr.Clear()
	assert.Equal(t, 0, tr.Size())
	assert.Empty(t, tr.Keys())
	assert.False(t, tr.HasPrefix("a"))

	// usable after clear
	tr.Insert("c", 3)
	v, ok := tr.Search("c")
	require.True(t, ok)
	assert.Equal(t, 3, v)
}

// ──────────────────────────────────────────────
// LongestPrefix
// ──────────────────────────────────────────────

func TestLongestPrefix(t *testing.T) {
	tr := New()
	tr.Insert("/api", 1)
	tr.Insert("/api/v1", 2)
	tr.Insert("/api/v1/users", 3)

	got, ok := tr.LongestPrefix("/api/v1/users/123")
	require.True(t, ok)
	assert.Equal(t, "/api/v1/users", got)
}

func TestLongestPrefix_ShorterMatch(t *testing.T) {
	tr := New()
	tr.Insert("/api", 1)
	tr.Insert("/api/v1", 2)

	got, ok := tr.LongestPrefix("/api/v2")
	require.True(t, ok)
	assert.Equal(t, "/api", got)
}

func TestLongestPrefix_ExactMatch(t *testing.T) {
	tr := New()
	tr.Insert("hello", 1)

	got, ok := tr.LongestPrefix("hello")
	require.True(t, ok)
	assert.Equal(t, "hello", got)
}

func TestLongestPrefix_NoMatch(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)

	_, ok := tr.LongestPrefix("banana")
	assert.False(t, ok)
}

func TestLongestPrefix_EmptyInput(t *testing.T) {
	tr := New()
	tr.Insert("a", 1)

	_, ok := tr.LongestPrefix("")
	assert.False(t, ok)
}

func TestLongestPrefix_EmptyTrie(t *testing.T) {
	tr := New()
	_, ok := tr.LongestPrefix("anything")
	assert.False(t, ok)
}

// ──────────────────────────────────────────────
// Autocomplete
// ──────────────────────────────────────────────

func TestAutocomplete(t *testing.T) {
	tr := New()
	for _, w := range []string{"app", "apple", "application", "apply", "banana"} {
		tr.Insert(w, w)
	}

	got := tr.Autocomplete("app", 3)
	assert.Equal(t, []string{"app", "apple", "application"}, got)
}

func TestAutocomplete_NoLimit(t *testing.T) {
	tr := New()
	for _, w := range []string{"app", "apple", "application", "apply"} {
		tr.Insert(w, w)
	}

	got := tr.Autocomplete("app", 0)
	assert.Len(t, got, 4)
}

func TestAutocomplete_NoMatch(t *testing.T) {
	tr := New()
	tr.Insert("apple", 1)

	assert.Empty(t, tr.Autocomplete("xyz", 10))
}

func TestAutocomplete_LimitLargerThanResults(t *testing.T) {
	tr := New()
	tr.Insert("app", 1)
	tr.Insert("apple", 2)

	got := tr.Autocomplete("app", 100)
	assert.Len(t, got, 2)
}

// ──────────────────────────────────────────────
// Walk
// ──────────────────────────────────────────────

func TestWalk_All(t *testing.T) {
	tr := New()
	tr.Insert("b", 2)
	tr.Insert("a", 1)
	tr.Insert("c", 3)

	var keys []string
	var vals []any
	tr.Walk(func(k string, v any) bool {
		keys = append(keys, k)
		vals = append(vals, v)
		return true
	})

	assert.Equal(t, []string{"a", "b", "c"}, keys)
	assert.Equal(t, []any{1, 2, 3}, vals)
}

func TestWalk_StopEarly(t *testing.T) {
	tr := New()
	tr.Insert("a", 1)
	tr.Insert("b", 2)
	tr.Insert("c", 3)

	var keys []string
	tr.Walk(func(k string, v any) bool {
		keys = append(keys, k)
		return k != "b"
	})

	assert.Equal(t, []string{"a", "b"}, keys)
}

func TestWalk_Empty(t *testing.T) {
	tr := New()
	called := false
	tr.Walk(func(k string, v any) bool {
		called = true
		return true
	})
	assert.False(t, called)
}

// ──────────────────────────────────────────────
// Unicode
// ──────────────────────────────────────────────

func TestUnicode_Keys(t *testing.T) {
	tr := New()
	tr.Insert("你好", 1)
	tr.Insert("你好世界", 2)
	tr.Insert("你好吗", 3)

	v, ok := tr.Search("你好世界")
	require.True(t, ok)
	assert.Equal(t, 2, v)

	got := tr.PrefixMatch("你好")
	assert.Equal(t, []string{"你好", "你好世界", "你好吗"}, got)
}

func TestUnicode_Emoji(t *testing.T) {
	tr := New()
	tr.Insert("😀😂", 1)
	tr.Insert("😀😍", 2)

	v, ok := tr.Search("😀😂")
	require.True(t, ok)
	assert.Equal(t, 1, v)

	got := tr.PrefixMatch("😀")
	assert.Equal(t, []string{"😀😂", "😀😍"}, got)
}

func TestUnicode_LongestPrefix(t *testing.T) {
	tr := New()
	tr.Insert("你好", 1)
	tr.Insert("你好世界", 2)

	got, ok := tr.LongestPrefix("你好世界和平")
	require.True(t, ok)
	assert.Equal(t, "你好世界", got)
}

func TestUnicode_Delete(t *testing.T) {
	tr := New()
	tr.Insert("你好", 1)
	tr.Insert("你好世界", 2)

	assert.True(t, tr.Delete("你好世界"))
	v, ok := tr.Search("你好")
	require.True(t, ok)
	assert.Equal(t, 1, v)
}

// ──────────────────────────────────────────────
// SafeTrie
// ──────────────────────────────────────────────

func TestSafeTrie_Basic(t *testing.T) {
	s := NewSafe()
	s.Insert("key", 1)

	v, ok := s.Search("key")
	require.True(t, ok)
	assert.Equal(t, 1, v)

	assert.True(t, s.Delete("key"))
	_, ok = s.Search("key")
	assert.False(t, ok)
}

func TestSafeTrie_AllMethods(t *testing.T) {
	s := NewSafe()
	s.InsertMany(map[string]any{"a": 1, "ab": 2, "abc": 3})

	assert.True(t, s.HasPrefix("ab"))
	assert.Equal(t, []string{"a", "ab", "abc"}, s.PrefixMatch("a"))
	assert.Equal(t, map[string]any{"a": 1, "ab": 2, "abc": 3}, s.PrefixMatchWithValues(""))
	assert.Equal(t, 3, s.Size())
	assert.Equal(t, []string{"a", "ab", "abc"}, s.Keys())
	assert.Equal(t, []string{"a", "ab"}, s.Autocomplete("a", 2))

	got, ok := s.LongestPrefix("abcd")
	require.True(t, ok)
	assert.Equal(t, "abc", got)

	var walked []string
	s.Walk(func(k string, v any) bool {
		walked = append(walked, k)
		return true
	})
	assert.Equal(t, []string{"a", "ab", "abc"}, walked)

	s.Clear()
	assert.Equal(t, 0, s.Size())
}

func TestSafeTrie_Concurrent(t *testing.T) {
	s := NewSafe()
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Insert(fmt.Sprintf("key-%d", i), i)
		}(i)
	}
	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = s.Search(fmt.Sprintf("key-%d", i))
			_ = s.HasPrefix("key-")
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 50, s.Size())
}
