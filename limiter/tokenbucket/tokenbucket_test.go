// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tokenbucket

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/limiter"
	"github.com/stretchr/testify/assert"
)

// ===== Non-blocking =====

func TestNew_BasicAcquire(t *testing.T) {
	l := New(10, 5)
	// Should allow burst=5 immediately.
	for i := 0; i < 5; i++ {
		assert.NoError(t, l.Acquire(context.Background(), nil))
	}
	// 6th should fail.
	err := l.Acquire(context.Background(), nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestNew_Refill(t *testing.T) {
	l := New(100, 1) // 100 tokens/sec, burst 1
	assert.NoError(t, l.Acquire(context.Background(), nil))
	// Should be empty now.
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(context.Background(), nil))
	// Wait for refill.
	time.Sleep(30 * time.Millisecond)
	// Should have refilled at least 1 token.
	assert.NoError(t, l.Acquire(context.Background(), nil))
}

func TestNew_Running(t *testing.T) {
	l := New(10, 5)
	assert.Equal(t, -1, l.Running())
}

func TestNew_ReleaseNoOp(t *testing.T) {
	l := New(10, 5)
	// Release should be a no-op for token bucket.
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

func TestNew_Concurrent(t *testing.T) {
	l := New(1000, 100)
	var acquired int32
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Acquire(context.Background(), nil) == nil {
				atomic.AddInt32(&acquired, 1)
			}
		}()
	}
	wg.Wait()
	// Should allow at most burst=100 immediately.
	assert.LessOrEqual(t, acquired, int32(100))
	assert.Greater(t, acquired, int32(0))
}

// ===== Blocking =====

func TestNewBlocking_BasicAcquire(t *testing.T) {
	l := NewBlocking(100, 3)
	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Acquire(context.Background(), nil))
	}
}

func TestNewBlocking_BlocksThenSucceeds(t *testing.T) {
	l := NewBlocking(1000, 1) // 1000/sec, burst 1
	assert.NoError(t, l.Acquire(context.Background(), nil))

	// Should block briefly then succeed after refill.
	start := time.Now()
	err := l.Acquire(context.Background(), nil)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	assert.Less(t, elapsed, time.Second) // should refill quickly at 1000/sec
}

func TestNewBlocking_ContextCancel(t *testing.T) {
	l := NewBlocking(1, 1) // 1/sec, burst 1
	assert.NoError(t, l.Acquire(context.Background(), nil))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := l.Acquire(ctx, nil)
	assert.Error(t, err)
}

// ===== Atomic =====

func TestNewAtomic_BasicAcquire(t *testing.T) {
	l := NewAtomic(10, 5)
	for i := 0; i < 5; i++ {
		assert.NoError(t, l.Acquire(context.Background(), nil))
	}
	err := l.Acquire(context.Background(), nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestNewAtomic_Refill(t *testing.T) {
	l := NewAtomic(1000, 1)
	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(context.Background(), nil))
	time.Sleep(20 * time.Millisecond)
	// Should have refilled.
	err := l.Acquire(context.Background(), nil)
	assert.NoError(t, err)
}

func TestNewAtomic_Concurrent(t *testing.T) {
	l := NewAtomic(10000, 100)
	var acquired int32
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Acquire(context.Background(), nil) == nil {
				atomic.AddInt32(&acquired, 1)
			}
		}()
	}
	wg.Wait()
	// The atomic variant has a small race window on refill, so it may
	// slightly exceed burst. Allow a small tolerance.
	assert.Greater(t, acquired, int32(0))
	assert.LessOrEqual(t, acquired, int32(110)) // burst + 10% tolerance
}

func TestNewAtomic_Running(t *testing.T) {
	l := NewAtomic(10, 5)
	assert.Equal(t, -1, l.Running())
}

// ===== Rate accuracy =====

func TestNew_RateAccuracy(t *testing.T) {
	l := New(100, 10) // 100/sec, burst 10
	// Consume burst.
	for i := 0; i < 10; i++ {
		l.Acquire(context.Background(), nil)
	}
	// Wait 100ms → should refill ~10 tokens.
	time.Sleep(100 * time.Millisecond)
	count := 0
	for i := 0; i < 20; i++ {
		if l.Acquire(context.Background(), nil) == nil {
			count++
		}
	}
	// Should have refilled roughly 10 tokens (±2 for timing).
	assert.InDelta(t, 10, count, 3)
}
