// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package idempotency_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/idempotency"
)

func TestHandler_Begin_Claimed(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	result, err := h.Begin(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if result != idempotency.BeginClaimed {
		t.Errorf("result = %d, want BeginClaimed", result)
	}
}

func TestHandler_Begin_AlreadyInFlight(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-2"); err != nil {
		t.Fatalf("first Begin: %v", err)
	}
	result, err := h.Begin(ctx, "op-2")
	if err != nil {
		t.Fatalf("second Begin: %v", err)
	}
	if result != idempotency.BeginAlreadyInFlight {
		t.Errorf("result = %d, want BeginAlreadyInFlight", result)
	}
}

func TestHandler_Begin_AlreadyAccomplished(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-3"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := h.Complete(ctx, "op-3"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	result, err := h.Begin(ctx, "op-3")
	if err != nil {
		t.Fatalf("second Begin: %v", err)
	}
	if result != idempotency.BeginAlreadyAccomplished {
		t.Errorf("result = %d, want BeginAlreadyAccomplished", result)
	}
}

func TestHandler_Complete(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-4"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := h.Complete(ctx, "op-4"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	ok, err := h.IsAccomplished(ctx, "op-4")
	if err != nil {
		t.Fatalf("IsAccomplished: %v", err)
	}
	if !ok {
		t.Error("IsAccomplished = false, want true")
	}
}

func TestHandler_Del_AllowsRetry(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-5"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := h.Del(ctx, "op-5"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	// Should be claimable again.
	result, err := h.Begin(ctx, "op-5")
	if err != nil {
		t.Fatalf("second Begin: %v", err)
	}
	if result != idempotency.BeginClaimed {
		t.Errorf("result = %d, want BeginClaimed after Del", result)
	}
}

func TestHandler_Execute_Success(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	var calls int32
	fn := func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}
	ctx := context.Background()
	if err := h.Execute(ctx, "op-6", fn); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if err := h.Execute(ctx, "op-6", fn); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestHandler_Execute_ErrorAllowsRetry(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	fn := func(ctx context.Context) error {
		return errors.New("boom")
	}
	ctx := context.Background()
	if err := h.Execute(ctx, "op-7", fn); err == nil {
		t.Error("Execute should return error")
	}
	// Should be claimable again.
	result, err := h.Begin(ctx, "op-7")
	if err != nil {
		t.Fatalf("Begin after error: %v", err)
	}
	if result != idempotency.BeginClaimed {
		t.Errorf("result = %d, want BeginClaimed after error", result)
	}
}

func TestHandler_Execute_AlreadyInFlight(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-8"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	err := h.Execute(ctx, "op-8", func(ctx context.Context) error { return nil })
	if !errors.Is(err, idempotency.ErrAlreadyInFlight) {
		t.Errorf("Execute error = %v, want ErrAlreadyInFlight", err)
	}
}

func TestHandler_EmptyKey(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	if _, err := h.Begin(context.Background(), ""); !errors.Is(err, idempotency.ErrEmptyKey) {
		t.Errorf("Begin empty key error = %v, want ErrEmptyKey", err)
	}
	if err := h.Complete(context.Background(), ""); !errors.Is(err, idempotency.ErrEmptyKey) {
		t.Errorf("Complete empty key error = %v, want ErrEmptyKey", err)
	}
}

func TestHandler_TTL_Expiry(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage(), idempotency.WithTTL(50*time.Millisecond))
	defer h.Close()

	ctx := context.Background()
	if _, err := h.Begin(ctx, "op-9"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	// After TTL, the entry should have expired and be claimable again.
	result, err := h.Begin(ctx, "op-9")
	if err != nil {
		t.Fatalf("Begin after TTL: %v", err)
	}
	if result != idempotency.BeginClaimed {
		t.Errorf("result = %d, want BeginClaimed after TTL expiry", result)
	}
}

func TestHandler_ConcurrentBegin(t *testing.T) {
	h := idempotency.New(idempotency.NewMemoryStorage())
	defer h.Close()

	var claimed int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := h.Begin(context.Background(), "concurrent-op")
			if err != nil {
				return
			}
			if result == idempotency.BeginClaimed {
				atomic.AddInt32(&claimed, 1)
			}
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&claimed) != 1 {
		t.Errorf("claimed = %d, want exactly 1", claimed)
	}
}

func TestState_String(t *testing.T) {
	cases := []struct {
		state idempotency.State
		want  string
	}{
		{idempotency.StateAbsent, "absent"},
		{idempotency.StateInFlight, "in-flight"},
		{idempotency.StateAccomplished, "accomplished"},
	}
	for _, c := range cases {
		if got := c.state.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", c.state, got, c.want)
		}
	}
}
