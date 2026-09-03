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

// ===== ULID =====

var ulidRegex = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

func TestULID_Length(t *testing.T) {
	u := ULID()
	assert.Len(t, u, 26, "ULID should be 26 characters")
}

func TestULID_Format(t *testing.T) {
	u := ULID()
	assert.Regexp(t, ulidRegex, u, "ULID should match Crockford Base32")
}

func TestULID_NoLowercase(t *testing.T) {
	u := ULID()
	for _, c := range u {
		assert.False(t, c >= 'a' && c <= 'z', "ULID should not contain lowercase, got %q", c)
	}
}

func TestULID_Sortable(t *testing.T) {
	u1 := ULID()
	time.Sleep(2 * time.Millisecond)
	u2 := ULID()
	assert.Less(t, u1, u2, "ULID should be lexicographically sortable by time")
}

func TestULID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		u := ULID()
		assert.False(t, seen[u], "duplicate ULID: %s", u)
		seen[u] = true
	}
}

func TestULID_CrockfordAlphabet(t *testing.T) {
	// Crockford Base32 excludes I, L, O, U.
	u := ULID()
	for _, c := range u {
		assert.NotContains(t, "ILOU", string(c), "ULID should not contain excluded chars I/L/O/U")
	}
}

// ===== NanoID =====

func TestNanoID_DefaultSize(t *testing.T) {
	id := NanoID(0)
	assert.Len(t, id, 21, "NanoID with size 0 should use default 21")
}

func TestNanoID_CustomSize(t *testing.T) {
	id := NanoID(10)
	assert.Len(t, id, 10)
}

func TestNanoID_NegativeSize(t *testing.T) {
	id := NanoID(-5)
	assert.Len(t, id, 21, "NanoID with negative size should use default 21")
}

func TestNanoID_URLSafeAlphabet(t *testing.T) {
	id := NanoID(100)
	for _, c := range id {
		assert.True(t,
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-',
			"NanoID should be URL-safe, got %q", c)
	}
}

func TestNanoID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := NanoID(21)
		assert.False(t, seen[id], "duplicate NanoID: %s", id)
		seen[id] = true
	}
}

func TestNanoIDWithAlphabet_Custom(t *testing.T) {
	id := NanoIDWithAlphabet(20, "abcdef")
	assert.Len(t, id, 20)
	for _, c := range id {
		assert.True(t, strings.ContainsRune("abcdef", c), "NanoID should only use custom alphabet, got %q", c)
	}
}

func TestNanoIDWithAlphabet_EmptyAlphabet(t *testing.T) {
	// Empty alphabet falls back to default.
	id := NanoIDWithAlphabet(10, "")
	assert.Len(t, id, 10)
}

func TestNanoIDWithAlphabet_SingleChar(t *testing.T) {
	id := NanoIDWithAlphabet(10, "X")
	assert.Len(t, id, 10)
	for _, c := range id {
		assert.Equal(t, 'X', c)
	}
}

func TestNanoIDWithAlphabet_Unique(t *testing.T) {
	seen := make(map[string]bool, 500)
	for i := 0; i < 500; i++ {
		id := NanoIDWithAlphabet(16, "0123456789")
		assert.False(t, seen[id], "duplicate NanoID with custom alphabet: %s", id)
		seen[id] = true
	}
}

// ===== Additional coverage tests =====

func TestSnowflake_NextID_ClockRollback(t *testing.T) {
	s, err := NewSnowflakeWithID(1)
	require.NoError(t, err)

	// Generate one ID to set lastStamp.
	id1 := s.NextID()
	require.NotZero(t, id1)

	// Manually advance lastStamp into the future to simulate clock rollback.
	s.mu.Lock()
	s.lastStamp = s.lastStamp + 10_000_000 // 10 seconds in the future
	s.mu.Unlock()

	// NextID should return 0 due to clock rollback.
	id2 := s.NextID()
	assert.Zero(t, id2, "NextID should return 0 on clock rollback")
}

func TestSnowflakeNext_NilDefault(t *testing.T) {
	orig := defaultSnowflake
	defaultSnowflake = nil
	defer func() { defaultSnowflake = orig }()

	assert.Zero(t, SnowflakeNext(), "SnowflakeNext should return 0 when defaultSnowflake is nil")
}

func TestSnowflakeNextUint_NilDefault(t *testing.T) {
	orig := defaultSnowflake
	defaultSnowflake = nil
	defer func() { defaultSnowflake = orig }()

	assert.Zero(t, SnowflakeNextUint(), "SnowflakeNextUint should return 0 when defaultSnowflake is nil")
}

func TestShortID_FallbackToRandom(t *testing.T) {
	orig := defaultSnowflake
	defaultSnowflake = nil
	defer func() { defaultSnowflake = orig }()

	// When defaultSnowflake is nil, SnowflakeNext returns 0, so ShortID
	// falls back to RandomShortID(12).
	s := ShortID()
	assert.Len(t, s, 12, "ShortID fallback should produce 12-char string")
	for _, c := range s {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'),
			"ShortID fallback should be base62, got %q", c)
	}
}

func TestULIDFromTime_ClockRollback(t *testing.T) {
	ulidMu.Lock()
	ulidLastMs = 0
	ulidMu.Unlock()

	// Generate a ULID with a large timestamp.
	large := ulIDFromTime(1_700_000_000_000) // year 2023+

	// Now generate with a smaller timestamp — should be clamped to ulidLastMs.
	small := ulIDFromTime(1_000_000)

	// The timestamp portion should be the same (clamped). The first 9 chars
	// (45 bits) are purely timestamp; char 9 includes 2 random bits so we
	// only compare the first 9.
	assert.Equal(t, large[:9], small[:9], "ULID timestamp should be clamped on rollback")

	// Reset for other tests.
	ulidMu.Lock()
	ulidLastMs = 0
	ulidMu.Unlock()
}

func TestNanoIDWithAlphabet_ZeroSize(t *testing.T) {
	id := NanoIDWithAlphabet(0, "X")
	assert.Len(t, id, nanoIDDefaultSize, "size 0 should default to %d", nanoIDDefaultSize)
}

func TestNanoIDWithAlphabet_NegativeSize(t *testing.T) {
	id := NanoIDWithAlphabet(-5, "X")
	assert.Len(t, id, nanoIDDefaultSize, "negative size should default to %d", nanoIDDefaultSize)
}

func TestNanoIDWithAlphabet_EmptyAlphabetZeroSize(t *testing.T) {
	id := NanoIDWithAlphabet(0, "")
	assert.Len(t, id, nanoIDDefaultSize)
}

func TestOrderedUUID_Format(t *testing.T) {
	u := OrderedUUID()
	assert.Len(t, u, 32)
	// First 12 hex chars encode timestamp — should be non-zero for current time.
	assert.NotEqual(t, "000000000000", u[:12], "timestamp prefix should be non-zero")
}

func TestRandomShortID_ZeroLength(t *testing.T) {
	s := RandomShortID(0)
	assert.Empty(t, s)
}

func TestRandomShortID_SingleChar(t *testing.T) {
	s := RandomShortID(50)
	assert.Len(t, s, 50)
}

func TestSnowflake_NextID_SequenceOverflow(t *testing.T) {
	s, err := NewSnowflakeWithID(1)
	require.NoError(t, err)

	// Force sequence to max and timestamp to current, so the next call
	// triggers the sequence-overflow spin-wait loop.
	s.mu.Lock()
	s.lastStamp = currentMicro()
	s.sequence = maxSequence
	s.mu.Unlock()

	id := s.NextID()
	assert.NotZero(t, id, "NextID should return valid ID after sequence overflow spin-wait")
}
