// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPoolSubmitComplete(t *testing.T) {
	p := NewWorkerPool(4, 16)
	p.Start()

	var executed atomic.Int64
	const n = 100
	for i := 0; i < n; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}

	p.Stop()

	assert.Equal(t, int64(n), executed.Load())
	assert.Equal(t, int64(n), p.Stats().Completed)
	assert.Equal(t, int64(0), p.Stats().Failed)
	assert.Equal(t, 0, p.Stats().Pending)
}

func TestWorkerPoolGracefulStop(t *testing.T) {
	p := NewWorkerPool(2, 8)
	p.Start()

	var executed atomic.Int64
	const n = 50
	for i := 0; i < n; i++ {
		require.NoError(t, p.Submit(func() {
			time.Sleep(2 * time.Millisecond)
			executed.Add(1)
		}))
	}

	p.Stop()

	assert.Equal(t, int64(n), executed.Load(), "all queued tasks should complete on graceful stop")
	stats := p.Stats()
	assert.Equal(t, int64(n), stats.Completed)
	assert.Equal(t, 0, stats.Pending)
}

func TestWorkerPoolPanicRecovery(t *testing.T) {
	p := NewWorkerPool(2, 8)
	p.Start()

	var executed atomic.Int64
	require.NoError(t, p.Submit(func() {
		panic("boom")
	}))
	for i := 0; i < 5; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}

	p.Stop()

	assert.Equal(t, int64(5), executed.Load(), "pool should keep running after a panic")
	stats := p.Stats()
	assert.Equal(t, int64(5), stats.Completed)
	assert.Equal(t, int64(1), stats.Failed)
}

func TestWorkerPoolSubmitAfterStop(t *testing.T) {
	p := NewWorkerPool(2, 4)
	p.Start()
	p.Stop()

	err := p.Submit(func() {})
	require.ErrorIs(t, err, ErrPoolClosed)
}

func TestWorkerPoolConcurrentSubmit(t *testing.T) {
	p := NewWorkerPool(8, 64)
	p.Start()

	var executed atomic.Int64
	const goroutines = 20
	const perG = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				_ = p.Submit(func() {
					executed.Add(1)
				})
			}
		}()
	}
	wg.Wait()
	p.Stop()

	assert.Equal(t, int64(goroutines*perG), executed.Load())
	stats := p.Stats()
	assert.Equal(t, int64(goroutines*perG), stats.Submitted)
	assert.Equal(t, int64(goroutines*perG), stats.Completed)
}

func TestWorkerPoolMetrics(t *testing.T) {
	p := NewWorkerPool(3, 16)
	p.Start()

	stats := p.Stats()
	assert.Equal(t, 3, stats.Workers)
	assert.Equal(t, int64(0), stats.Submitted)

	var executed atomic.Int64
	for i := 0; i < 10; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}
	p.Stop()

	stats = p.Stats()
	assert.Equal(t, int64(10), stats.Submitted)
	assert.Equal(t, int64(10), stats.Completed)
	assert.Equal(t, int64(0), stats.Failed)
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 3, stats.Workers)
}

func TestWorkerPoolDoubleStop(t *testing.T) {
	p := NewWorkerPool(2, 4)
	p.Start()

	require.NotPanics(t, func() {
		p.Stop()
		p.Stop()
	})
}

func TestWorkerPoolSubmitBlocksWhenQueueFull(t *testing.T) {
	// Create a pool with queueSize=1 and don't start workers,
	// so the queue won't be drained.
	p := NewWorkerPool(1, 1)

	// Fill the queue with one task (no workers running to drain it).
	require.NoError(t, p.Submit(func() {}))

	// Second Submit should block because the queue is full.
	done := make(chan error, 1)
	go func() {
		done <- p.Submit(func() {})
	}()

	select {
	case err := <-done:
		t.Fatalf("Submit should block when queue is full, got err: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	// Stop the pool — the blocked Submit should return ErrPoolClosed.
	p.Stop()

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrPoolClosed)
	case <-time.After(time.Second):
		t.Fatal("blocked Submit did not return after Stop")
	}
}

func TestWorkerPoolStatsAccuracy(t *testing.T) {
	p := NewWorkerPool(2, 8)
	p.Start()

	// Initial stats.
	stats := p.Stats()
	assert.Equal(t, 2, stats.Workers)
	assert.Equal(t, int64(0), stats.Submitted)
	assert.Equal(t, int64(0), stats.Completed)
	assert.Equal(t, int64(0), stats.Failed)
	assert.Equal(t, 0, stats.Pending)

	var executed atomic.Int64
	for i := 0; i < 5; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}

	p.Stop()

	stats = p.Stats()
	assert.Equal(t, int64(5), stats.Submitted)
	assert.Equal(t, int64(5), stats.Completed)
	assert.Equal(t, int64(0), stats.Failed)
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 2, stats.Workers)
}

func TestWorkerPoolStartAlreadyStarted(t *testing.T) {
	p := NewWorkerPool(2, 8)

	// Start twice — should not panic, workers should still function.
	require.NotPanics(t, func() {
		p.Start()
		p.Start()
	})

	var executed atomic.Int64
	for i := 0; i < 10; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}

	p.Stop()

	assert.Equal(t, int64(10), executed.Load(), "all tasks should complete even after double Start")
	stats := p.Stats()
	assert.Equal(t, int64(10), stats.Completed)
}

func TestWorkerPoolSubmitAfterStopReturnsError(t *testing.T) {
	p := NewWorkerPool(2, 4)
	p.Start()
	p.Stop()

	// Multiple submits after stop should all return ErrPoolClosed.
	for i := 0; i < 3; i++ {
		err := p.Submit(func() {})
		require.ErrorIs(t, err, ErrPoolClosed)
	}
}

func TestWorkerPoolPendingCount(t *testing.T) {
	// Use a blocking task to keep workers busy, then verify pending count.
	p := NewWorkerPool(1, 4)
	p.Start()

	block := make(chan struct{})

	// Submit one blocking task — occupies the single worker.
	require.NoError(t, p.Submit(func() {
		<-block
	}))

	// Submit 3 more tasks — they should be pending in the queue.
	for i := 0; i < 3; i++ {
		require.NoError(t, p.Submit(func() {}))
	}

	// Give a moment for the first task to start.
	time.Sleep(20 * time.Millisecond)

	stats := p.Stats()
	assert.Equal(t, int64(4), stats.Submitted)
	assert.Equal(t, 3, stats.Pending, "3 tasks should be pending in the queue")

	// Unblock the worker so Stop can proceed.
	close(block)
	p.Stop()
}

func TestWorkerPoolFailedCountWithMultiplePanics(t *testing.T) {
	p := NewWorkerPool(2, 8)
	p.Start()

	// Submit several panicking tasks.
	for i := 0; i < 3; i++ {
		require.NoError(t, p.Submit(func() {
			panic("boom")
		}))
	}

	// Submit some normal tasks.
	var executed atomic.Int64
	for i := 0; i < 5; i++ {
		require.NoError(t, p.Submit(func() {
			executed.Add(1)
		}))
	}

	p.Stop()

	stats := p.Stats()
	assert.Equal(t, int64(5), executed.Load())
	assert.Equal(t, int64(5), stats.Completed)
	assert.Equal(t, int64(3), stats.Failed, "3 panicking tasks should be counted as failed")
}
