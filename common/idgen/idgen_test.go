// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package idgen

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Snowflake =====

func TestSnowflake_NextID(t *testing.T) {
	s, err := NewSnowflake()
	require.NoError(t, err)

	ids := make(map[int64]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := s.NextID()
		assert.NotZero(t, id, "NextID returned 0")
		assert.False(t, ids[id], "duplicate id %d at iteration %d", id, i)
		ids[id] = true
	}
}

func TestSnowflake_NextID_Increasing(t *testing.T) {
	s, err := NewSnowflake()
	require.NoError(t, err)

	var prev int64
	for i := 0; i < 100; i++ {
		id := s.NextID()
		assert.Greater(t, id, prev, "IDs should be monotonically increasing")
		prev = id
	}
}

func TestSnowflake_NewWithID(t *testing.T) {
	s, err := NewSnowflakeWithID(42)
	require.NoError(t, err)
	assert.NotNil(t, s)

	id := s.NextID()
	assert.NotZero(t, id)
}

func TestSnowflake_NewWithID_Invalid(t *testing.T) {
	_, err := NewSnowflakeWithID(-1)
	assert.Error(t, err)

	_, err = NewSnowflakeWithID(maxMachineID + 1)
	assert.Error(t, err)
}

func TestSnowflake_InvalidMachineID(t *testing.T) {
	t.Setenv("MACHINE_ID", "99999")
	_, err := NewSnowflake()
	assert.Error(t, err)
}

func TestSnowflakeNext(t *testing.T) {
	id := SnowflakeNext()
	assert.NotZero(t, id)
}

func TestSnowflakeNextUint(t *testing.T) {
	id := SnowflakeNextUint()
	assert.NotZero(t, id)
	// Should fit in int63 (sign bit cleared).
	assert.LessOrEqual(t, id, uint(0x7FFFFFFFFFFFFFFF))
}

func TestClampSnowflakeUint(t *testing.T) {
	id := uint(0xFFFFFFFFFFFFFFFF)
	clamped := ClampSnowflakeUint(id)
	assert.Equal(t, uint(0x7FFFFFFFFFFFFFFF), clamped)
}

func TestSnowflake_Concurrent(t *testing.T) {
	s, err := NewSnowflake()
	require.NoError(t, err)

	ids := make(chan int64, 1000)
	done := make(chan struct{})
	seen := make(map[int64]bool)
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ids <- s.NextID()
			}
			done <- struct{}{}
		}()
	}

	go func() {
		for i := 0; i < 10; i++ {
			<-done
		}
		close(ids)
	}()

	for id := range ids {
		mu.Lock()
		assert.False(t, seen[id], "duplicate id %d", id)
		seen[id] = true
		mu.Unlock()
	}
	assert.Len(t, seen, 1000)
}

// ===== UUID v4 =====

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDv4_Format(t *testing.T) {
	u := UUIDv4()
	assert.Regexp(t, uuidRegex, u)
}

func TestUUIDv4_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		u := UUIDv4()
		assert.False(t, seen[u], "duplicate UUID: %s", u)
		seen[u] = true
	}
}

func TestUUIDv4Bytes(t *testing.T) {
	b := UUIDv4Bytes()
	assert.Equal(t, byte(0x40), b[6]&0xf0) // version 4
	assert.Equal(t, byte(0x80), b[8]&0xc0) // variant RFC 4122
}

// ===== UUID v7 =====

var uuidV7Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDv7_Format(t *testing.T) {
	u := UUIDv7()
	assert.Regexp(t, uuidV7Regex, u)
}

func TestUUIDv7_Sortable(t *testing.T) {
	u1 := UUIDv7()
	time.Sleep(2 * time.Millisecond)
	u2 := UUIDv7()
	assert.Less(t, u1, u2, "UUIDv7 should be lexicographically sortable by time")
}

func TestUUIDv7_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		u := UUIDv7()
		assert.False(t, seen[u], "duplicate UUIDv7: %s", u)
		seen[u] = true
	}
}

func TestUUIDv7Bytes(t *testing.T) {
	b := UUIDv7Bytes()
	assert.Equal(t, byte(0x70), b[6]&0xf0) // version 7
	assert.Equal(t, byte(0x80), b[8]&0xc0) // variant RFC 4122
}

// ===== Ordered UUID =====

func TestOrderedUUID_Length(t *testing.T) {
	u := OrderedUUID()
	assert.Len(t, u, 32, "OrderedUUID should be 32 hex chars")
}

func TestOrderedUUID_HexOnly(t *testing.T) {
	u := OrderedUUID()
	for _, c := range u {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"OrderedUUID should contain only hex chars, got %q", c)
	}
}

func TestOrderedUUID_Sortable(t *testing.T) {
	u1 := OrderedUUID()
	time.Sleep(2 * time.Millisecond)
	u2 := OrderedUUID()
	assert.Less(t, u1, u2, "OrderedUUID should be lexicographically sortable")
}

func TestOrderedUUID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		u := OrderedUUID()
		assert.False(t, seen[u], "duplicate OrderedUUID: %s", u)
		seen[u] = true
	}
}

// ===== Short ID =====

func TestShortID_NonEmpty(t *testing.T) {
	s := ShortID()
	assert.NotEmpty(t, s)
}

func TestShortID_Base62(t *testing.T) {
	s := ShortID()
	for _, c := range s {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'),
			"ShortID should be base62, got %q in %q", c, s)
	}
}

func TestShortID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		s := ShortID()
		assert.False(t, seen[s], "duplicate ShortID: %s", s)
		seen[s] = true
	}
}

func TestShortIDFromInt(t *testing.T) {
	s := ShortIDFromInt(123456789)
	assert.NotEmpty(t, s)
	decoded, err := ShortIDToInt(s)
	assert.NoError(t, err)
	assert.Equal(t, uint64(123456789), decoded)
}

func TestShortIDFromInt_Zero(t *testing.T) {
	assert.Equal(t, "0", ShortIDFromInt(0))
}

func TestShortID_RoundTrip(t *testing.T) {
	tests := []uint64{1, 62, 63, 100, 3844, 123456789, 18446744073709551615}
	for _, n := range tests {
		s := ShortIDFromInt(n)
		decoded, err := ShortIDToInt(s)
		assert.NoError(t, err)
		assert.Equal(t, n, decoded, "round-trip failed for %d → %s → %d", n, s, decoded)
	}
}

func TestRandomShortID(t *testing.T) {
	s := RandomShortID(16)
	assert.Len(t, s, 16)
	for _, c := range s {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'),
			"RandomShortID should be base62")
	}
}

func TestShortIDToInt_Invalid(t *testing.T) {
	_, err := ShortIDToInt("invalid!")
	assert.Error(t, err)
}

// ===== Random strings =====

func TestRandText(t *testing.T) {
	s := RandText(32)
	assert.Len(t, s, 32)
	for _, c := range s {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z'),
			"RandText should be lowercase alphanumeric, got %q", c)
	}
}

func TestRandNumberText(t *testing.T) {
	s := RandNumberText(8)
	assert.Len(t, s, 8)
	for _, c := range s {
		assert.True(t, c >= '0' && c <= '9', "RandNumberText should be numeric, got %q", c)
	}
}

func TestRandTextWithCharset(t *testing.T) {
	s := RandTextWithCharset(10, "ABC")
	assert.Len(t, s, 10)
	for _, c := range s {
		assert.True(t, strings.ContainsRune("ABC", c), "charset mismatch")
	}
}

func TestRandText_ZeroLength(t *testing.T) {
	assert.Equal(t, "", RandText(0))
}

func TestRandText_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		s := RandText(16)
		assert.False(t, seen[s], "duplicate random text: %s", s)
		seen[s] = true
	}
}
