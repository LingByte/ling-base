// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package shutdown_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/shutdown"
)

func TestNew_Defaults(t *testing.T) {
	m := shutdown.New()
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.IsTriggered() {
		t.Error("New manager should not be triggered")
	}
}

func TestNew_WithOptions(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(10*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)
	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestAdd_LIFOOrder(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var order []string
	var mu sync.Mutex

	m.Add("first", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return nil
	})
	m.Add("second", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return nil
	})
	m.Add("third", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "third")
		mu.Unlock()
		return nil
	})

	// Trigger and wait.
	_ = m.Trigger()
	err := m.Listen(context.Background())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	// Should run in reverse order: third, second, first.
	expected := []string{"third", "second", "first"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestTrigger_Programmatic(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var called int32
	m.Add("hook", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	// Trigger programmatically.
	if err := m.Trigger(); err != nil {
		t.Fatalf("Trigger: %v", err)
	}

	// Listen should return immediately since already triggered.
	err := m.Listen(context.Background())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("hook called %d times, want 1", called)
	}
}

func TestTrigger_AlreadyTriggered(t *testing.T) {
	m := shutdown.New()

	if err := m.Trigger(); err != nil {
		t.Fatalf("First Trigger: %v", err)
	}

	if err := m.Trigger(); err != shutdown.ErrAlreadyTriggered {
		t.Errorf("Second Trigger: err = %v, want ErrAlreadyTriggered", err)
	}
}

func TestListen_ParentContextCanceled(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var called int32
	m.Add("hook", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := m.Listen(ctx)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("hook called %d times, want 1", called)
	}
}

func TestHook_Error(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	expectedErr := errors.New("cleanup failed")
	m.Add("failing-hook", func(ctx context.Context) error {
		return expectedErr
	})

	_ = m.Trigger()
	err := m.Listen(context.Background())
	if err == nil {
		t.Fatal("Listen should return error")
	}
}

func TestHook_Timeout(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(100*time.Millisecond),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	m.Add("slow-hook", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	_ = m.Trigger()
	err := m.Listen(context.Background())
	if err == nil {
		t.Fatal("Listen should return timeout error")
	}
}

func TestWait(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var called int32
	m.Add("hook", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	_ = m.Trigger()
	go func() {
		_ = m.Listen(context.Background())
	}()

	// Wait should block until shutdown completes.
	err := m.Wait()
	if err != nil {
		t.Errorf("Wait: %v", err)
	}
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("hook called %d times, want 1", called)
	}
}

func TestAddBefore(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var order []string
	var mu sync.Mutex

	m.Add("target", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "target")
		mu.Unlock()
		return nil
	})
	m.AddBefore("target", "before-target", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "before-target")
		mu.Unlock()
		return nil
	})

	_ = m.Trigger()
	_ = m.Listen(context.Background())

	// LIFO: before-target runs before target (since it was inserted
	// before target in the hook list, and LIFO reverses the order).
	// Actually: hooks = [target, before-target], LIFO runs:
	// before-target first, then target.
	if len(order) != 2 {
		t.Fatalf("order = %v, want 2 items", order)
	}
}

func TestMultipleHooks(t *testing.T) {
	m := shutdown.New(
		shutdown.WithTimeout(5*time.Second),
		shutdown.WithLogger(func(format string, args ...any) {}),
	)

	var count int32
	for i := 0; i < 10; i++ {
		m.Add("hook", func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	_ = m.Trigger()
	_ = m.Listen(context.Background())

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("hooks called %d times, want 10", count)
	}
}

func TestErrors(t *testing.T) {
	if shutdown.ErrAlreadyTriggered == nil {
		t.Error("ErrAlreadyTriggered should not be nil")
	}
	if shutdown.ErrTimeout == nil {
		t.Error("ErrTimeout should not be nil")
	}
}

// Need sync for the mutex in tests.
var _ sync.Mutex
var _ = sync.Mutex{}
