// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/limiter"
)

func setupMiniRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return client, mr
}

// ===== Sliding Window =====

func TestSlidingWindow_Basic(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewSlidingWindow(client, "test:sw", 5, time.Second)

	for i := 0; i < 5; i++ {
		assert.NoError(t, l.Acquire(ctx, nil))
	}
	err := l.Acquire(ctx, nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestSlidingWindow_Running(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewSlidingWindow(client, "test:sw:running", 10, time.Second)

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, 2, l.Running())
}

func TestSlidingWindow_WindowExpiry(t *testing.T) {
	client, mr := setupMiniRedis(t)
	ctx := context.Background()
	l := NewSlidingWindow(client, "test:sw:expiry", 2, 10*time.Second)

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, nil))

	// Fast-forward past the window.
	mr.FastForward(11 * time.Second)
	assert.NoError(t, l.Acquire(ctx, nil))
}

func TestSlidingWindow_ReleaseNoOp(t *testing.T) {
	client, _ := setupMiniRedis(t)
	l := NewSlidingWindow(client, "test:sw:release", 5, time.Second)
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

// ===== Token Bucket =====

func TestTokenBucket_Basic(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewTokenBucket(client, "test:tb", 10, 5)

	for i := 0; i < 5; i++ {
		assert.NoError(t, l.Acquire(ctx, nil))
	}
	err := l.Acquire(ctx, nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestTokenBucket_Refill(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewTokenBucket(client, "test:tb:refill", 10, 1) // 10/sec, burst 1

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, nil))

	// Wait for real-time refill (Lua script uses time.Now() for refill calc).
	// At 10 tokens/sec, need ~100ms for 1 token.
	time.Sleep(150 * time.Millisecond)
	assert.NoError(t, l.Acquire(ctx, nil))
}

func TestTokenBucket_Running(t *testing.T) {
	client, _ := setupMiniRedis(t)
	l := NewTokenBucket(client, "test:tb:running", 10, 5)
	assert.Equal(t, -1, l.Running())
}

// ===== Concurrency =====

func TestConcurrency_Basic(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewConcurrency(client, "test:conc", 3, 10*time.Second)

	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Acquire(ctx, nil))
	}
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, nil))
	assert.Equal(t, 3, l.Running())

	l.Release(nil)
	assert.Equal(t, 2, l.Running())
	assert.NoError(t, l.Acquire(ctx, nil))
}

func TestConcurrency_ReleaseBelowZero(t *testing.T) {
	client, _ := setupMiniRedis(t)
	l := NewConcurrency(client, "test:conc:zero", 5, 10*time.Second)

	// Release without Acquire — should not go negative.
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

func TestConcurrency_TTLExpiry(t *testing.T) {
	client, mr := setupMiniRedis(t)
	ctx := context.Background()
	l := NewConcurrency(client, "test:conc:ttl", 2, 10*time.Second)

	assert.NoError(t, l.Acquire(ctx, nil))
	assert.NoError(t, l.Acquire(ctx, nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, nil))

	// Fast-forward past TTL — counter should expire and reset.
	mr.FastForward(11 * time.Second)
	assert.NoError(t, l.Acquire(ctx, nil))
}

func TestConcurrency_Running(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewConcurrency(client, "test:conc:running", 10, 10*time.Second)

	l.Acquire(ctx, nil)
	l.Acquire(ctx, nil)
	assert.Equal(t, 2, l.Running())
}

// ===== Reset helper =====

func TestReset(t *testing.T) {
	client, _ := setupMiniRedis(t)
	ctx := context.Background()
	l := NewSlidingWindow(client, "test:reset", 5, time.Second)
	l.Acquire(ctx, nil)
	assert.Equal(t, 1, l.Running())

	assert.NoError(t, Reset(ctx, client, "test:reset"))
	// After delete, Get returns error → Running() returns -1.
	assert.Equal(t, -1, l.Running())
}
