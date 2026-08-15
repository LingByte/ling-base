// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package keycount

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/limiter"
	"github.com/stretchr/testify/assert"
)

// ===== Mutex-based =====

func TestNew_BasicAcquireRelease(t *testing.T) {
	l := New(3)
	assert.Equal(t, 0, l.Running())

	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.NoError(t, l.Acquire(context.Background(), []byte("b")))
	assert.Equal(t, 3, l.Running())
}

func TestNew_PerKeyLimit(t *testing.T) {
	l := New(2)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	err := l.Acquire(context.Background(), []byte("a"))
	assert.Equal(t, limiter.ErrLimitExceeded, err)

	// Different key should still work.
	assert.NoError(t, l.Acquire(context.Background(), []byte("b")))
}

func TestNew_ReleaseRestores(t *testing.T) {
	l := New(1)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(context.Background(), []byte("a")))
	l.Release([]byte("a"))
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
}

func TestNew_ReleaseCleansUp(t *testing.T) {
	l := New(1)
	l.Acquire(context.Background(), []byte("a"))
	l.Release([]byte("a"))
	// Internal map should be empty.
	ml := l.(*mutexLimit)
	ml.mu.Lock()
	assert.Len(t, ml.current, 0)
	ml.mu.Unlock()
}

func TestNew_ReleaseUnknownKey(t *testing.T) {
	l := New(1)
	assert.NotPanics(t, func() {
		l.Release([]byte("nonexistent"))
	})
}

func TestNew_Concurrent(t *testing.T) {
	l := New(5)
	var wg sync.WaitGroup
	var acquired int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := []byte("user")
			if l.Acquire(context.Background(), key) == nil {
				atomic.AddInt32(&acquired, 1)
				time.Sleep(time.Millisecond)
				l.Release(key)
			}
		}(i)
	}
	wg.Wait()
	assert.Greater(t, atomic.LoadInt32(&acquired), int32(0))
	assert.LessOrEqual(t, atomic.LoadInt32(&acquired), int32(50))
}

// ===== Sync (atomic) =====

func TestNewSync_Basic(t *testing.T) {
	l := NewSync(2)
	assert.Equal(t, 0, l.Running())

	assert.NoError(t, l.Acquire(context.Background(), "a"))
	assert.NoError(t, l.Acquire(context.Background(), "a"))
	err := l.Acquire(context.Background(), "a")
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestNewSync_PerKey(t *testing.T) {
	l := NewSync(1)
	assert.NoError(t, l.Acquire(context.Background(), "a"))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(context.Background(), "a"))
	assert.NoError(t, l.Acquire(context.Background(), "b"))
}

func TestNewSync_Release(t *testing.T) {
	l := NewSync(1)
	l.Acquire(context.Background(), "a")
	l.Release("a")
	assert.NoError(t, l.Acquire(context.Background(), "a"))
}

func TestNewSync_Concurrent(t *testing.T) {
	l := NewSync(3)
	var wg sync.WaitGroup
	var acquired int32
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Acquire(context.Background(), "k") == nil {
				atomic.AddInt32(&acquired, 1)
				time.Sleep(time.Millisecond)
				l.Release("k")
			}
		}()
	}
	wg.Wait()
	assert.Greater(t, atomic.LoadInt32(&acquired), int32(0))
	assert.LessOrEqual(t, atomic.LoadInt32(&acquired), int32(100))
}

// ===== Blocking =====

func TestNewBlocking_Basic(t *testing.T) {
	l := NewBlocking(2)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	assert.Equal(t, 2, l.Running())
}

func TestNewBlocking_BlocksPerKey(t *testing.T) {
	l := NewBlocking(1)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))

	done := make(chan error, 1)
	go func() {
		done <- l.Acquire(context.Background(), []byte("a"))
	}()

	select {
	case <-done:
		t.Fatal("should have blocked")
	case <-time.After(50 * time.Millisecond):
	}

	l.Release([]byte("a"))
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("should have unblocked")
	}
}

func TestNewBlocking_DifferentKeyNotBlocked(t *testing.T) {
	l := NewBlocking(1)
	assert.NoError(t, l.Acquire(context.Background(), []byte("a")))
	// Different key should not block.
	assert.NoError(t, l.Acquire(context.Background(), []byte("b")))
}

func TestNewBlocking_ReleaseCleansUp(t *testing.T) {
	l := NewBlocking(1)
	l.Acquire(context.Background(), []byte("a"))
	l.Release([]byte("a"))
	bl := l.(*blockingLimit)
	bl.mu.Lock()
	assert.Len(t, bl.current, 0)
	bl.mu.Unlock()
}

func TestNewBlocking_ReleaseUnknownKey(t *testing.T) {
	l := NewBlocking(1)
	assert.NotPanics(t, func() {
		l.Release([]byte("nonexistent"))
	})
}
