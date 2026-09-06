// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package reconciler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ─── Rate limiter tests ───

func TestDefaultRateLimiter_Delay(t *testing.T) {
	rl := NewDefaultRateLimiter()
	if d := rl.Delay("k", 0); d != 0 {
		t.Errorf("failures=0 should return 0, got %v", d)
	}
	if d := rl.Delay("k", 1); d < 5*time.Millisecond {
		t.Errorf("failures=1 should return >= base, got %v", d)
	}
	if d := rl.Delay("k", 10); d > rl.Max {
		t.Errorf("delay should be capped at max, got %v", d)
	}
}

func TestFixedIntervalRateLimiter(t *testing.T) {
	rl := &FixedIntervalRateLimiter{Interval: 100 * time.Millisecond}
	if d := rl.Delay("k", 5); d != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", d)
	}
}

func TestNoRateLimiter(t *testing.T) {
	rl := NoRateLimiter{}
	if d := rl.Delay("k", 100); d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

// ─── Work queue tests ───

func TestWorkQueue_AddGet(t *testing.T) {
	q := newWorkQueue()
	q.Add("a")
	q.Add("b")
	q.Add("a") // dedup

	// Map iteration order is non-deterministic, so collect both keys.
	keys := make(map[string]bool)
	for i := 0; i < 2; i++ {
		key, _, ok := q.Get()
		if !ok {
			t.Fatalf("expected ok=true, got false")
		}
		keys[key] = true
	}
	if !keys["a"] || !keys["b"] {
		t.Fatalf("expected keys {a, b}, got %v", keys)
	}
	// Queue should be empty now — Get blocks, so use shutdown.
	go func() {
		time.Sleep(10 * time.Millisecond)
		q.Shutdown()
	}()
	_, _, ok := q.Get()
	if ok {
		t.Error("expected ok=false after shutdown")
	}
}

func TestWorkQueue_AddAfter(t *testing.T) {
	q := newWorkQueue()
	q.AddAfter("delayed", 50*time.Millisecond)

	start := time.Now()
	key, _, ok := q.Get()
	elapsed := time.Since(start)

	if !ok || key != "delayed" {
		t.Fatalf("expected 'delayed', got %q ok=%v", key, ok)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected ~50ms delay, got %v", elapsed)
	}
}

func TestWorkQueue_Shutdown(t *testing.T) {
	q := newWorkQueue()
	q.Shutdown()
	_, _, ok := q.Get()
	if ok {
		t.Error("expected ok=false after shutdown")
	}
}

func TestWorkQueue_DoneRequeue(t *testing.T) {
	q := newWorkQueue()
	q.Add("k")
	key, _, _ := q.Get()
	if key != "k" {
		t.Fatalf("expected 'k', got %q", key)
	}
	// Requeue immediately
	q.Done("k", 0, true, 0)
	key, _, ok := q.Get()
	if !ok || key != "k" {
		t.Fatalf("expected requeued 'k', got %q ok=%v", key, ok)
	}
}

func TestWorkQueue_DoneNoRequeue(t *testing.T) {
	q := newWorkQueue()
	q.Add("k")
	k, _, _ := q.Get()
	if k != "k" {
		t.Fatalf("expected 'k', got %q", k)
	}
	q.Done("k", 0, false, 0)
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got len=%d", q.Len())
	}
}

func TestWorkQueue_Dedup(t *testing.T) {
	q := newWorkQueue()
	q.Add("x")
	q.Add("x")
	q.Add("x")
	if q.Len() != 1 {
		t.Errorf("expected 1 deduped item, got %d", q.Len())
	}
}

// ─── Controller tests ───

func TestController_BasicReconcile(t *testing.T) {
	var processed atomic.Int64
	ctrl := NewController("test").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			processed.Add(1)
			return Result{}, nil
		}).
		WithWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)

	// Wait for startup
	time.Sleep(20 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "a"})
	ctrl.Enqueue(Request{Key: "b"})

	// Wait for processing
	requireCondition(t, func() bool { return processed.Load() >= 2 }, 1*time.Second,
		"expected 2 processed, got %d", processed.Load())

	cancel()
}

func TestController_RequeueAfter(t *testing.T) {
	var processed atomic.Int64
	ctrl := NewController("test-requeue").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			count := processed.Add(1)
			if count < 3 {
				return Result{RequeueAfter: 20 * time.Millisecond}, nil
			}
			return Result{}, nil
		}).
		WithWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "x"})

	requireCondition(t, func() bool { return processed.Load() >= 3 }, 1*time.Second,
		"expected 3 processed, got %d", processed.Load())

	if ctrl.RequeueCount() < 2 {
		t.Errorf("expected 2 requeues, got %d", ctrl.RequeueCount())
	}
}

func TestController_ErrorRetry(t *testing.T) {
	var attempts atomic.Int64
	ctrl := NewController("test-error").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			a := attempts.Add(1)
			if a < 3 {
				return Result{}, errors.New("transient error")
			}
			return Result{}, nil
		}).
		WithWorkers(1).
		WithRateLimiter(&FixedIntervalRateLimiter{Interval: 10 * time.Millisecond}).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "err"})

	requireCondition(t, func() bool { return attempts.Load() >= 3 }, 1*time.Second,
		"expected 3 attempts, got %d", attempts.Load())

	if ctrl.ErrorCount() < 2 {
		t.Errorf("expected 2 errors, got %d", ctrl.ErrorCount())
	}
}

func TestController_ErrorHandler(t *testing.T) {
	var handlerCalled atomic.Bool
	ctrl := NewController("test-errhandler").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			return Result{}, errors.New("fail")
		}).
		WithWorkers(1).
		WithRateLimiter(NoRateLimiter{}).
		WithErrorHandler(func(key string, err error) {
			handlerCalled.Store(true)
		}).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "k"})

	requireCondition(t, func() bool { return handlerCalled.Load() }, 500*time.Millisecond,
		"error handler not called")

	cancel()
}

func TestController_MultipleWorkers(t *testing.T) {
	var processed atomic.Int64
	ctrl := NewController("test-multi").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			time.Sleep(10 * time.Millisecond)
			processed.Add(1)
			return Result{}, nil
		}).
		WithWorkers(4).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	// Enqueue 8 items
	for i := 0; i < 8; i++ {
		ctrl.Enqueue(Request{Key: fmt.Sprintf("item-%d", i)})
	}

	// With 4 workers and 10ms per item, 8 items should take ~20ms (2 batches)
	requireCondition(t, func() bool { return processed.Load() >= 8 }, 500*time.Millisecond,
		"expected 8 processed, got %d", processed.Load())

	cancel()
}

func TestController_Stop(t *testing.T) {
	ctrl := NewController("test-stop").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			return Result{}, nil
		}).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	if !ctrl.IsRunning() {
		t.Fatal("expected running")
	}

	ctrl.Stop()
	requireCondition(t, func() bool { return !ctrl.IsRunning() }, 500*time.Millisecond,
		"expected stopped")
}

func TestController_DoubleStart(t *testing.T) {
	ctrl := NewController("test-double").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			return Result{}, nil
		}).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	// Second start should return error
	err := ctrl.Start(ctx)
	if err == nil {
		t.Error("expected error on double start")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' error, got %v", err)
	}

	cancel()
}

func TestController_NoReconciler(t *testing.T) {
	ctrl := NewController("test-noreconciler").Build()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "k"})

	requireCondition(t, func() bool { return ctrl.ErrorCount() >= 1 }, 500*time.Millisecond,
		"expected error from nil reconciler")

	cancel()
}

func TestController_QueueDepth(t *testing.T) {
	ctrl := NewController("test-depth").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			time.Sleep(100 * time.Millisecond) // slow to keep queue populated
			return Result{}, nil
		}).
		WithWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(20 * time.Millisecond)

	for i := 0; i < 10; i++ {
		ctrl.Enqueue(Request{Key: fmt.Sprintf("k-%d", i)})
	}

	// Some items should be queued (not all processed yet)
	requireCondition(t, func() bool { return ctrl.QueueDepth() > 0 || ctrl.ProcessedCount() > 0 }, 200*time.Millisecond,
		"queue should have items or be processing")

	cancel()
}

func TestController_RequeuePullsForward(t *testing.T) {
	var processed atomic.Int64
	ctrl := NewController("test-pullforward").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			c := processed.Add(1)
			if c == 1 {
				// First time: schedule requeue after 10s
				return Result{RequeueAfter: 10 * time.Second}, nil
			}
			return Result{}, nil
		}).
		WithWorkers(1).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	ctrl.Enqueue(Request{Key: "k"})

	// Wait for first pass
	requireCondition(t, func() bool { return processed.Load() >= 1 }, 1*time.Second,
		"first pass not done")

	// Wait a bit more to ensure Done has been called (RequeueAfter=10s)
	time.Sleep(100 * time.Millisecond)

	// Now the item is scheduled for 10s later. Enqueue immediately should
	// pull it forward.
	ctrl.Enqueue(Request{Key: "k"})

	requireCondition(t, func() bool { return processed.Load() >= 2 }, 2*time.Second,
		"requeue should be pulled forward by immediate enqueue, got %d", processed.Load())
}

func TestController_GracefulShutdown(t *testing.T) {
	var completed atomic.Int64
	var started atomic.Int64
	ctrl := NewController("test-graceful").
		WithReconcilerFunc(func(ctx context.Context, req Request) (Result, error) {
			started.Add(1)
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
			return Result{}, nil
		}).
		WithWorkers(2).
		Build()

	ctx, cancel := context.WithCancel(context.Background())

	go ctrl.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Enqueue 2 items
	ctrl.Enqueue(Request{Key: "a"})
	ctrl.Enqueue(Request{Key: "b"})

	// Wait for both to start
	requireCondition(t, func() bool { return started.Load() >= 2 }, 500*time.Millisecond,
		"expected 2 started, got %d", started.Load())

	// Cancel while processing — should wait for in-flight to complete.
	cancel()

	// Wait for controller to finish
	requireCondition(t, func() bool { return !ctrl.IsRunning() }, 1*time.Second,
		"controller should stop")

	if completed.Load() < 2 {
		t.Errorf("expected 2 completed after graceful shutdown, got %d", completed.Load())
	}
}

// ─── Helpers ───

func requireCondition(t *testing.T, cond func() bool, timeout time.Duration, msg string, args ...interface{}) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf(msg, args...)
}
