// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package count

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/limiter"
	"github.com/stretchr/testify/assert"
)

func TestNew_InvalidLimit(t *testing.T) {
	// n=0 means no permits can be acquired.
	l := New(0)
	err := l.Acquire(context.Background(), nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestNew_BasicAcquireRelease(t *testing.T) {
	l := New(3)
	assert.Equal(t, 0, l.Running())

	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.Equal(t, 2, l.Running())

	l.Release(nil)
	assert.Equal(t, 1, l.Running())

	l.Release(nil)
	assert.Equal(t, 0, l.Running())
}

func TestNew_LimitExceeded(t *testing.T) {
	l := New(2)
	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.NoError(t, l.Acquire(context.Background(), nil))
	err := l.Acquire(context.Background(), nil)
	assert.Equal(t, limiter.ErrLimitExceeded, err)
	assert.Equal(t, 2, l.Running())
}

func TestNew_ReleaseRestores(t *testing.T) {
	l := New(1)
	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(context.Background(), nil))
	l.Release(nil)
	assert.NoError(t, l.Acquire(context.Background(), nil))
}

func TestNew_Concurrent(t *testing.T) {
	l := New(10)
	var wg sync.WaitGroup
	var acquired int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Acquire(context.Background(), nil) == nil {
				atomic.AddInt32(&acquired, 1)
				time.Sleep(time.Millisecond)
				l.Release(nil)
			}
		}()
	}
	wg.Wait()
	// With limit=10, at most 10 can acquire simultaneously. Some may
	// release fast enough for others to acquire, but we can't guarantee
	// all 100 succeed. Just verify some acquired and none panicked.
	assert.Greater(t, atomic.LoadInt32(&acquired), int32(0))
	assert.LessOrEqual(t, atomic.LoadInt32(&acquired), int32(100))
}

// ===== Blocking =====

func TestNewBlocking_Basic(t *testing.T) {
	l := NewBlocking(2)
	assert.Equal(t, 0, l.Running())

	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.NoError(t, l.Acquire(context.Background(), nil))
	assert.Equal(t, 2, l.Running())
}

func TestNewBlocking_BlocksUntilRelease(t *testing.T) {
	l := NewBlocking(1)
	assert.NoError(t, l.Acquire(context.Background(), nil))

	done := make(chan error, 1)
	go func() {
		done <- l.Acquire(context.Background(), nil)
	}()

	select {
	case err := <-done:
		t.Fatalf("Acquire should have blocked: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	l.Release(nil)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Acquire should have unblocked after Release")
	}
}

func TestNewBlocking_ContextCancel(t *testing.T) {
	l := NewBlocking(1)
	assert.NoError(t, l.Acquire(context.Background(), nil))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := l.Acquire(ctx, nil)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestNewBlocking_Running(t *testing.T) {
	l := NewBlocking(5)
	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Acquire(context.Background(), nil))
	}
	assert.Equal(t, 3, l.Running())
}
