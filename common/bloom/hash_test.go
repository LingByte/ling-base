// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bloom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoubleHash_NonNegative(t *testing.T) {
	h1, h2 := doubleHash("hello")
	assert.GreaterOrEqual(t, h1, uint64(0))
	assert.GreaterOrEqual(t, h2, uint64(0))
}

func TestDoubleHash_DifferentKeys(t *testing.T) {
	h1a, h2a := doubleHash("key-a")
	h1b, h2b := doubleHash("key-b")
	// Different keys should (almost) always produce different hashes.
	assert.NotEqual(t, h1a, h1b, "h1 should differ for different keys")
	assert.NotEqual(t, h2a, h2b, "h2 should differ for different keys")
}

func TestDoubleHash_Deterministic(t *testing.T) {
	h1a, h2a := doubleHash("same-key")
	h1b, h2b := doubleHash("same-key")
	assert.Equal(t, h1a, h1b)
	assert.Equal(t, h2a, h2b)
}

func TestDoubleHash_EmptyKey(t *testing.T) {
	// An empty key should still produce valid (non-negative) hashes, and h2
	// must never be zero (the function perturbs a zero h2).
	h1, h2 := doubleHash("")
	assert.GreaterOrEqual(t, h1, uint64(0))
	assert.NotZero(t, h2, "h2 must never be zero (would collapse indices)")
}

func TestDoubleHash_H2NeverZero(t *testing.T) {
	// Try many keys to ensure h2 is never zero (the perturbation guard).
	for i := 0; i < 1000; i++ {
		_, h2 := doubleHash(string(rune(i)))
		assert.NotZero(t, h2)
	}
}

func TestIndices_Count(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	indices := Indices("hello", m, k, nil)
	require.Len(t, indices, int(k))
}

func TestIndices_InRange(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	indices := Indices("hello", m, k, nil)
	for i, idx := range indices {
		assert.Less(t, idx, m, "index %d out of range", i)
	}
}

func TestIndices_Deterministic(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	a := Indices("deterministic-key", m, k, nil)
	b := Indices("deterministic-key", m, k, nil)
	assert.Equal(t, a, b)
}

func TestIndices_DifferentKeysDiffer(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	a := Indices("key-one", m, k, nil)
	b := Indices("key-two", m, k, nil)
	// The two index sets should not be identical.
	assert.NotEqual(t, a, b)
}

func TestIndices_ReusesBuffer(t *testing.T) {
	const m, k = uint64(1000), uint64(5)
	buf := make([]uint64, 0, k)
	out := Indices("buffer-test", m, k, buf)
	// The returned slice should reuse the buffer's backing array.
	require.Len(t, out, int(k))
	if cap(buf) >= int(k) {
		// Same backing array (first element pointer equal).
		assert.Equal(t, &buf[:1][0], &out[:1][0])
	}
}

func TestIndices_BufferTooSmall(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	// A buffer with insufficient capacity should trigger a new allocation.
	buf := make([]uint64, 0, 3)
	out := Indices("small-buf", m, k, buf)
	require.Len(t, out, int(k))
	// The returned slice has its own backing array (larger than buf).
	assert.GreaterOrEqual(t, cap(out), int(k))
}

func TestIndices_BufferLargerThanK(t *testing.T) {
	const m, k = uint64(1000), uint64(3)
	buf := make([]uint64, 0, 10)
	out := Indices("large-buf", m, k, buf)
	require.Len(t, out, int(k))
	// Should reuse the buffer and reslice to length k.
	assert.Equal(t, cap(buf), cap(out))
}

func TestIndices_EdgeCase_M1(t *testing.T) {
	// With m=1, every index must be 0.
	const m, k = uint64(1), uint64(5)
	indices := Indices("edge-m1", m, k, nil)
	require.Len(t, indices, int(k))
	for i, idx := range indices {
		assert.Equal(t, uint64(0), idx, "index %d should be 0 when m=1", i)
	}
}

func TestIndices_EdgeCase_K1(t *testing.T) {
	// With k=1, exactly one index is returned.
	const m, k = uint64(1000), uint64(1)
	indices := Indices("edge-k1", m, k, nil)
	require.Len(t, indices, 1)
	assert.Less(t, indices[0], m)
}

func TestIndices_EdgeCase_EmptyKey(t *testing.T) {
	const m, k = uint64(1000), uint64(7)
	indices := Indices("", m, k, nil)
	require.Len(t, indices, int(k))
	for i, idx := range indices {
		assert.Less(t, idx, m, "index %d out of range for empty key", i)
	}
}

func TestIndices_EdgeCase_M1K1(t *testing.T) {
	indices := Indices("edge", uint64(1), uint64(1), nil)
	require.Len(t, indices, 1)
	assert.Equal(t, uint64(0), indices[0])
}
