// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShutdownManager_AddHook(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	m.AddHook("h1", func(ctx context.Context) error { return nil })
	m.AddHook("h2", func(ctx context.Context) error { return nil })
	assert.Equal(t, 2, m.HookCount())
}

func TestShutdownManager_Shutdown(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	var called int32
	m.AddHook("h1", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	m.AddHook("h2", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	m.Shutdown()
	assert.Equal(t, int32(2), atomic.LoadInt32(&called))
	assert.True(t, m.IsStopped())
}

func TestShutdownManager_Idempotent(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	var called int32
	m.AddHook("h1", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	m.Shutdown()
	m.Shutdown() // should not run hooks again
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestShutdownManager_PriorityOrder(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	var order []string
	m.AddPriorityHook("low", 1, func(ctx context.Context) error {
		order = append(order, "low")
		return nil
	})
	m.AddPriorityHook("high", 10, func(ctx context.Context) error {
		order = append(order, "high")
		return nil
	})
	m.AddPriorityHook("mid", 5, func(ctx context.Context) error {
		order = append(order, "mid")
		return nil
	})

	m.Shutdown()
	// Higher priority runs first.
	assert.Equal(t, []string{"high", "mid", "low"}, order)
}

func TestShutdownManager_HookError(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	var called int32
	m.AddHook("bad", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return assert.AnError
	})
	m.AddHook("good", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	m.Shutdown()
	// Both hooks should run despite error in first.
	assert.Equal(t, int32(2), atomic.LoadInt32(&called))
}

func TestShutdownManager_Stop(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	m.Stop()
	assert.True(t, m.IsStopped())
}

func TestShutdownManager_SetTimeout(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	m.SetTimeout(60 * time.Second)
	// No direct way to verify, but ensure no panic.
	assert.NotPanics(t, func() {
		m.SetTimeout(10 * time.Second)
	})
}

func TestShutdownManager_SetSignals(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	assert.NotPanics(t, func() {
		m.SetSignals()
	})
}

func TestShutdownManager_OnSignal(t *testing.T) {
	m := NewShutdownManager(5 * time.Second)
	m.OnSignal(func(sig os.Signal) {})
	m.AddHook("h", func(ctx context.Context) error { return nil })

	// Just verify it doesn't panic.
	assert.NotPanics(t, func() {
		m.Shutdown()
	})
}

func TestShutdownManager_DefaultTimeout(t *testing.T) {
	m := NewShutdownManager(0)
	// Should default to 30s.
	assert.NotPanics(t, func() {
		m.AddHook("h", func(ctx context.Context) error { return nil })
		m.Shutdown()
	})
}

func TestShutdownWithError(t *testing.T) {
	var called int32
	fn := ShutdownWithError("test", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return assert.AnError
	})
	err := fn(context.Background())
	assert.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestShutdownWithError_NoError(t *testing.T) {
	var called int32
	fn := ShutdownWithError("test", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	err := fn(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}
