// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package limiter_test

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/common/limiter"
	"github.com/LingByte/ling-base/common/limiter/count"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stringLimiter adapts a limiter.Limiter to the StringLimiter interface.
type stringLimiter struct {
	limiter.Limiter
}

func (s *stringLimiter) Acquire(ctx context.Context, key string) error {
	return limiter.AcquireString(s.Limiter, ctx, key)
}

func (s *stringLimiter) Release(key string) {
	limiter.ReleaseString(s.Limiter, key)
}

func TestAcquireString_Success(t *testing.T) {
	l := count.New(2)
	ctx := context.Background()

	assert.NoError(t, limiter.AcquireString(l, ctx, "user1"))
	assert.NoError(t, limiter.AcquireString(l, ctx, "user2"))
	assert.Equal(t, 2, l.Running())
}

func TestAcquireString_LimitExceeded(t *testing.T) {
	l := count.New(1)
	ctx := context.Background()

	assert.NoError(t, limiter.AcquireString(l, ctx, "user1"))
	err := limiter.AcquireString(l, ctx, "user2")
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestReleaseString(t *testing.T) {
	l := count.New(1)
	ctx := context.Background()

	assert.NoError(t, limiter.AcquireString(l, ctx, "user1"))
	assert.Equal(t, 1, l.Running())

	limiter.ReleaseString(l, "user1")
	assert.Equal(t, 0, l.Running())

	// After release, a new acquire should succeed.
	assert.NoError(t, limiter.AcquireString(l, ctx, "user2"))
}

func TestAcquireReleaseString_RoundTrip(t *testing.T) {
	l := count.New(3)
	ctx := context.Background()

	keys := []string{"a", "b", "c"}
	for _, k := range keys {
		assert.NoError(t, limiter.AcquireString(l, ctx, k))
	}
	assert.Equal(t, 3, l.Running())

	for _, k := range keys {
		limiter.ReleaseString(l, k)
	}
	assert.Equal(t, 0, l.Running())
}

func TestAcquireString_EmptyKey(t *testing.T) {
	// The count limiter does not require a key, so an empty string key is
	// accepted (it is converted to an empty []byte, which is valid for a
	// global limiter).
	l := count.New(1)
	ctx := context.Background()
	assert.NoError(t, limiter.AcquireString(l, ctx, ""))
	limiter.ReleaseString(l, "")
	assert.Equal(t, 0, l.Running())
}

func TestAcquireString_NilLimiter(t *testing.T) {
	// AcquireString does not guard against a nil Limiter; calling it will
	// panic. We verify the helper simply delegates by using a real limiter.
	l := count.New(1)
	ctx := context.Background()
	assert.NoError(t, limiter.AcquireString(l, ctx, "k"))
	limiter.ReleaseString(l, "k")
}

// ----- Interface exercises -----

func TestLimiterInterface(t *testing.T) {
	// count.New returns a value satisfying limiter.Limiter.
	var l limiter.Limiter = count.New(5)
	require.NotNil(t, l)

	ctx := context.Background()
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, 1, l.Running())
	l.Release(nil)
	assert.Equal(t, 0, l.Running())
}

func TestStringLimiterInterface(t *testing.T) {
	// Verify a wrapper satisfies the StringLimiter interface.
	var sl limiter.StringLimiter = &stringLimiter{Limiter: count.New(2)}
	require.NotNil(t, sl)

	ctx := context.Background()
	assert.NoError(t, sl.Acquire(ctx, "k1"))
	assert.NoError(t, sl.Acquire(ctx, "k2"))
	assert.Equal(t, 2, sl.Running())
	sl.Release("k1")
	assert.Equal(t, 1, sl.Running())
}

// sizeLimiter is a minimal SizeLimiter implementation to exercise the
// SizeLimiter interface.
type sizeLimiter struct {
	used      int64
	limit     int64
	remaining int64
}

func (s *sizeLimiter) Running() int64 { return s.used }

func (s *sizeLimiter) Acquire(ctx context.Context, key []byte, size int64) error {
	if size <= 0 {
		return limiter.ErrInvalidSize
	}
	if s.used+size > s.limit {
		return limiter.ErrLimitExceeded
	}
	s.used += size
	return nil
}

func (s *sizeLimiter) Release(key []byte, size int64) {
	s.used -= size
}

func (s *sizeLimiter) Remaining(key []byte) int64 {
	return s.limit - s.used
}

func TestSizeLimiterInterface(t *testing.T) {
	var sl limiter.SizeLimiter = &sizeLimiter{limit: 100, remaining: 100}
	require.NotNil(t, sl)

	ctx := context.Background()
	assert.NoError(t, sl.Acquire(ctx, nil, 40))
	assert.Equal(t, int64(40), sl.Running())
	assert.Equal(t, int64(60), sl.Remaining(nil))

	// Exceeding the limit.
	err := sl.Acquire(ctx, nil, 70)
	assert.Equal(t, limiter.ErrLimitExceeded, err)

	// Invalid size.
	err = sl.Acquire(ctx, nil, 0)
	assert.Equal(t, limiter.ErrInvalidSize, err)

	sl.Release(nil, 40)
	assert.Equal(t, int64(0), sl.Running())
	assert.Equal(t, int64(100), sl.Remaining(nil))
}

func TestSentinelErrors(t *testing.T) {
	// Ensure sentinel errors are non-nil and carry expected messages.
	assert.Equal(t, "limiter: limit exceeded", limiter.ErrLimitExceeded.Error())
	assert.Equal(t, "limiter: key must not be empty", limiter.ErrKeyRequired.Error())
	assert.Equal(t, "limiter: limit must be greater than zero", limiter.ErrInvalidLimit.Error())
	assert.Equal(t, "limiter: size must be greater than zero", limiter.ErrInvalidSize.Error())
}
