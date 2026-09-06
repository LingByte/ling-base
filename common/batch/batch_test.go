// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package batch_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/batch"
)

func TestNew_NilHandler(t *testing.T) {
	_, err := batch.New[int](nil)
	if err != batch.ErrHandlerNil {
		t.Errorf("New(nil): err = %v, want ErrHandlerNil", err)
	}
}

func TestNew_Defaults(t *testing.T) {
	b, err := batch.New[int](func(ctx context.Context, items []int) error { return nil })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()
}

func TestAdd_AndFlush(t *testing.T) {
	var flushed []int
	var mu sync.Mutex

	b, err := batch.New(func(ctx context.Context, items []int) error {
		mu.Lock()
		flushed = append(flushed, items...)
		mu.Unlock()
		return nil
	}, batch.WithSize[int](3))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for i := 1; i <= 3; i++ {
		if err := b.Add(i); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond)
	b.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(flushed) != 3 {
		t.Fatalf("flushed count = %d, want 3", len(flushed))
	}
}

func TestAdd_CloseReturnsErr(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error { return nil })
	b.Close()

	err := b.Add(1)
	if err != batch.ErrClosed {
		t.Errorf("Add after close: err = %v, want ErrClosed", err)
	}
}

func TestFlushAndWait(t *testing.T) {
	var flushed int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	}, batch.WithSize[int](100))

	for i := 0; i < 5; i++ {
		_ = b.Add(i)
	}

	if err := b.FlushAndWait(); err != nil {
		t.Fatalf("FlushAndWait: %v", err)
	}

	if atomic.LoadInt32(&flushed) != 5 {
		t.Errorf("flushed = %d, want 5", flushed)
	}

	b.Close()
}

func TestFlushAndWait_MultipleConcurrent(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	}, batch.WithSize[int](1000))

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Add(1)
			_ = b.FlushAndWait()
		}()
	}
	wg.Wait()
	b.Close()
}

func TestIntervalFlush(t *testing.T) {
	var flushed int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	}, batch.WithSize[int](1000), batch.WithInterval[int](100*time.Millisecond))

	_ = b.Add(1)
	_ = b.Add(2)

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&flushed) != 2 {
		t.Errorf("flushed = %d, want 2 (interval flush)", flushed)
	}

	b.Close()
}

func TestHandlerError(t *testing.T) {
	expectedErr := errors.New("handler failed")

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		return expectedErr
	}, batch.WithSize[int](1))

	_ = b.Add(1)
	time.Sleep(50 * time.Millisecond)

	if err := b.LastError(); err != expectedErr {
		t.Errorf("LastError = %v, want %v", err, expectedErr)
	}

	if b.ErrorCount() != 1 {
		t.Errorf("ErrorCount = %d, want 1", b.ErrorCount())
	}

	b.Close()
}

func TestErrorHandler(t *testing.T) {
	var called int32
	errHandler := func(err error) {
		atomic.AddInt32(&called, 1)
	}

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		return errors.New("fail")
	}, batch.WithSize[int](1), batch.WithErrorHandler[int](errHandler))

	_ = b.Add(1)
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("error handler called %d times, want 1", called)
	}

	b.Close()
}

func TestTryAdd(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error { return nil },
		batch.WithSize[int](1000),
		batch.WithBufferSize[int](1),
	)
	defer b.Close()

	if !b.TryAdd(1) {
		t.Error("TryAdd should succeed on first add")
	}
}

func TestTryAdd_Closed(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error { return nil })
	b.Close()

	if b.TryAdd(1) {
		t.Error("TryAdd should fail after close")
	}
}

func TestClose_Twice(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error { return nil })

	if err := b.Close(); err != nil {
		t.Errorf("First Close: %v", err)
	}
	_ = b.Close()
}

func TestStats(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error { return nil },
		batch.WithSize[int](2),
	)
	defer b.Close()

	_ = b.Add(1)
	_ = b.Add(2) // triggers flush
	_ = b.Add(3)
	_ = b.Add(4) // triggers flush

	time.Sleep(50 * time.Millisecond)

	stats := b.Stats()
	if stats.FlushCount < 2 {
		t.Errorf("FlushCount = %d, want >= 2", stats.FlushCount)
	}
	if stats.ItemCount < 4 {
		t.Errorf("ItemCount = %d, want >= 4", stats.ItemCount)
	}
}

func TestRunWithContext(t *testing.T) {
	var flushed int32

	ctx, cancel := context.WithCancel(context.Background())

	b, _ := batch.RunWithContext(ctx, func(ctx context.Context, items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	}, batch.WithSize[int](1000))

	_ = b.Add(1)
	_ = b.Add(2)

	cancel()
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&flushed) != 2 {
		t.Errorf("flushed = %d, want 2", flushed)
	}
}

func TestLargeBatch(t *testing.T) {
	var total int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		atomic.AddInt32(&total, int32(len(items)))
		return nil
	}, batch.WithSize[int](10))

	for i := 0; i < 100; i++ {
		_ = b.Add(i)
	}

	b.Close()

	if atomic.LoadInt32(&total) != 100 {
		t.Errorf("total = %d, want 100", total)
	}
}

func TestErrors(t *testing.T) {
	if batch.ErrClosed == nil {
		t.Error("ErrClosed should not be nil")
	}
	if batch.ErrHandlerNil == nil {
		t.Error("ErrHandlerNil should not be nil")
	}
}

func TestGenericString(t *testing.T) {
	var flushed []string
	var mu sync.Mutex

	b, _ := batch.New(func(ctx context.Context, items []string) error {
		mu.Lock()
		flushed = append(flushed, items...)
		mu.Unlock()
		return nil
	}, batch.WithSize[string](2))

	_ = b.Add("a")
	_ = b.Add("b")
	b.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(flushed) != 2 || flushed[0] != "a" || flushed[1] != "b" {
		t.Errorf("flushed = %v", flushed)
	}
}

// ──────────────────────────────────────────────
// New feature tests
// ──────────────────────────────────────────────

func TestPendingCount(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error {
		time.Sleep(50 * time.Millisecond) // slow handler
		return nil
	}, batch.WithSize[int](1000), batch.WithInterval[int](0))

	_ = b.Add(1)
	_ = b.Add(2)
	_ = b.Add(3)

	// Items should be pending (not yet flushed).
	if pc := b.PendingCount(); pc != 3 {
		t.Errorf("PendingCount = %d, want 3", pc)
	}

	b.Close()
}

func TestWithMaxBytes(t *testing.T) {
	var flushed int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	},
		batch.WithSize[int](1000),       // high count limit
		batch.WithMaxBytes[int](100),    // 100 byte limit
		batch.WithItemSize[int](func(i int) int { return i * 10 }), // each item = i*10 bytes
		batch.WithInterval[int](0),      // disable timer
	)

	// Add items: 1(10B), 2(20B), 3(30B), 4(40B), 5(50B)
	// After adding 5: total = 10+20+30+40+50 = 150 > 100 → flush
	for i := 1; i <= 5; i++ {
		_ = b.Add(i)
	}

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&flushed) == 0 {
		t.Error("WithMaxBytes should trigger flush when byte limit exceeded")
	}

	b.Close()
}

func TestWithRetry(t *testing.T) {
	var attempts int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			return errors.New("transient error")
		}
		return nil
	},
		batch.WithSize[int](1),
		batch.WithRetry[int](3),
		batch.WithRetryDelay[int](10*time.Millisecond),
	)

	_ = b.Add(1)
	time.Sleep(100 * time.Millisecond)

	// Should have retried and eventually succeeded.
	if atomic.LoadInt32(&attempts) < 3 {
		t.Errorf("attempts = %d, want >= 3", attempts)
	}

	b.Close()
}

func TestWithRetry_AllFail(t *testing.T) {
	var attempts int32

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("permanent error")
	},
		batch.WithSize[int](1),
		batch.WithRetry[int](2),
		batch.WithRetryDelay[int](10*time.Millisecond),
	)

	_ = b.Add(1)
	time.Sleep(100 * time.Millisecond)

	// Should have attempted 3 times (1 + 2 retries).
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}

	if b.ErrorCount() != 1 {
		t.Errorf("ErrorCount = %d, want 1", b.ErrorCount())
	}

	b.Close()
}

func TestWithShutdownTimeout(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	},
		batch.WithSize[int](1),
		batch.WithShutdownTimeout[int](50*time.Millisecond),
	)

	_ = b.Add(1)

	// Close should timeout and return relatively quickly.
	start := time.Now()
	_ = b.Close()
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("Close took %v, should have timed out quickly", elapsed)
	}
}

func TestFlushAndWait_Error(t *testing.T) {
	expectedErr := errors.New("flush error")

	b, _ := batch.New(func(ctx context.Context, items []int) error {
		return expectedErr
	}, batch.WithSize[int](1000))

	_ = b.Add(1)

	err := b.FlushAndWait()
	if err != expectedErr {
		t.Errorf("FlushAndWait: err = %v, want %v", err, expectedErr)
	}

	b.Close()
}

func TestPendingBytes(t *testing.T) {
	b, _ := batch.New(func(ctx context.Context, items []int) error {
		return nil
	},
		batch.WithSize[int](1000),
		batch.WithInterval[int](0),
		batch.WithMaxBytes[int](10000),
		batch.WithItemSize[int](func(i int) int { return i * 100 }),
	)

	_ = b.Add(1) // 100 bytes
	_ = b.Add(2) // 200 bytes
	_ = b.Add(3) // 300 bytes

	if pb := b.PendingBytes(); pb != 600 {
		t.Errorf("PendingBytes = %d, want 600", pb)
	}

	b.Close()
}
