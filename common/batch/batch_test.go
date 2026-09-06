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
	b, err := batch.New[int](func(items []int) error { return nil })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()
}

func TestAdd_AndFlush(t *testing.T) {
	var flushed []int
	var mu sync.Mutex

	b, err := batch.New(func(items []int) error {
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

	// Size=3, so adding 3 items should trigger a flush.
	// Give it a moment.
	time.Sleep(50 * time.Millisecond)

	b.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(flushed) != 3 {
		t.Fatalf("flushed count = %d, want 3", len(flushed))
	}
}

func TestAdd_CloseReturnsErr(t *testing.T) {
	b, _ := batch.New(func(items []int) error { return nil })
	b.Close()

	err := b.Add(1)
	if err != batch.ErrClosed {
		t.Errorf("Add after close: err = %v, want ErrClosed", err)
	}
}

func TestFlushAndWait(t *testing.T) {
	var flushed int32

	b, _ := batch.New(func(items []int) error {
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

func TestIntervalFlush(t *testing.T) {
	var flushed int32

	b, _ := batch.New(func(items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	}, batch.WithSize[int](1000), batch.WithInterval[int](100*time.Millisecond))

	_ = b.Add(1)
	_ = b.Add(2)

	// Wait for interval flush.
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&flushed) != 2 {
		t.Errorf("flushed = %d, want 2 (interval flush)", flushed)
	}

	b.Close()
}

func TestHandlerError(t *testing.T) {
	expectedErr := errors.New("handler failed")

	b, _ := batch.New(func(items []int) error {
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

	b, _ := batch.New(func(items []int) error {
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
	b, _ := batch.New(func(items []int) error { return nil },
		batch.WithSize[int](1000),
		batch.WithBufferSize[int](1),
	)
	defer b.Close()

	if !b.TryAdd(1) {
		t.Error("TryAdd should succeed on first add")
	}
}

func TestTryAdd_Closed(t *testing.T) {
	b, _ := batch.New(func(items []int) error { return nil })
	b.Close()

	if b.TryAdd(1) {
		t.Error("TryAdd should fail after close")
	}
}

func TestClose_Twice(t *testing.T) {
	b, _ := batch.New(func(items []int) error { return nil })

	if err := b.Close(); err != nil {
		t.Errorf("First Close: %v", err)
	}
	// Second close should not panic.
	_ = b.Close()
}

func TestStats(t *testing.T) {
	b, _ := batch.New(func(items []int) error { return nil },
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

	b, _ := batch.RunWithContext(ctx, func(items []int) error {
		atomic.AddInt32(&flushed, int32(len(items)))
		return nil
	}, batch.WithSize[int](1000))

	_ = b.Add(1)
	_ = b.Add(2)

	// Cancel context — should trigger close and flush.
	cancel()

	// Wait for close to complete.
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&flushed) != 2 {
		t.Errorf("flushed = %d, want 2", flushed)
	}
}

func TestLargeBatch(t *testing.T) {
	var total int32

	b, _ := batch.New(func(items []int) error {
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

// Test with string type to verify generics work.
func TestGenericString(t *testing.T) {
	var flushed []string
	var mu sync.Mutex

	b, _ := batch.New(func(items []string) error {
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
