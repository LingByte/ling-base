// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLifecycle is a test Lifecycle implementation.
type mockLifecycle struct {
	started  atomic.Bool
	startErr error
	stopErr  error
	startCnt int32
	stopCnt  int32
}

func (m *mockLifecycle) Start(ctx context.Context) error {
	atomic.AddInt32(&m.startCnt, 1)
	if m.startErr != nil {
		return m.startErr
	}
	m.started.Store(true)
	return nil
}

func (m *mockLifecycle) Stop(ctx context.Context) error {
	atomic.AddInt32(&m.stopCnt, 1)
	m.started.Store(false)
	return m.stopErr
}

func (m *mockLifecycle) IsRunning() bool {
	return m.started.Load()
}

func TestLifecycleManager_Init(t *testing.T) {
	m := NewLifecycleManager()
	var called int32
	m.AddInitHook("hook1", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	m.AddInitHook("hook2", func(ctx context.Context) error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	err := m.Init(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(2), called)
}

func TestLifecycleManager_InitError(t *testing.T) {
	m := NewLifecycleManager()
	m.AddInitHook("bad", func(ctx context.Context) error {
		return assert.AnError
	})
	err := m.Init(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
}

func TestLifecycleManager_StartStop(t *testing.T) {
	m := NewLifecycleManager()
	lc1 := &mockLifecycle{}
	lc2 := &mockLifecycle{}
	m.AddLifecycle("c1", lc1)
	m.AddLifecycle("c2", lc2)

	err := m.Start(context.Background())
	assert.NoError(t, err)
	assert.True(t, lc1.IsRunning())
	assert.True(t, lc2.IsRunning())
	assert.True(t, m.IsRunning())

	err = m.Stop(context.Background())
	assert.NoError(t, err)
	assert.False(t, lc1.IsRunning())
	assert.False(t, lc2.IsRunning())
}

func TestLifecycleManager_StartTwice(t *testing.T) {
	m := NewLifecycleManager()
	m.AddLifecycle("c1", &mockLifecycle{})
	assert.NoError(t, m.Start(context.Background()))
	err := m.Start(context.Background())
	assert.Error(t, err)
}

func TestLifecycleManager_StartRollback(t *testing.T) {
	m := NewLifecycleManager()
	lc1 := &mockLifecycle{}
	lc2 := &mockLifecycle{startErr: assert.AnError}
	m.AddLifecycle("c1", lc1)
	m.AddLifecycle("c2", lc2)

	err := m.Start(context.Background())
	assert.Error(t, err)
	// lc1 was started, should be stopped during rollback.
	assert.False(t, lc1.IsRunning())
}

func TestLifecycleManager_ShutdownHooks(t *testing.T) {
	m := NewLifecycleManager()
	var order []string
	m.AddShutdownHook("h1", func(ctx context.Context) error {
		order = append(order, "h1")
		return nil
	})
	m.AddShutdownHook("h2", func(ctx context.Context) error {
		order = append(order, "h2")
		return nil
	})
	m.AddLifecycle("c1", &mockLifecycle{})

	m.Start(context.Background())
	m.Stop(context.Background())

	// Shutdown hooks run after lifecycle stop, in reverse order.
	assert.Equal(t, []string{"h2", "h1"}, order)
}

func TestLifecycleManager_StopError(t *testing.T) {
	m := NewLifecycleManager()
	lc := &mockLifecycle{stopErr: assert.AnError}
	m.AddLifecycle("c1", lc)
	m.Start(context.Background())
	err := m.Stop(context.Background())
	assert.Error(t, err)
}

func TestLifecycleManager_ComponentCount(t *testing.T) {
	m := NewLifecycleManager()
	m.AddLifecycle("c1", &mockLifecycle{})
	m.AddLifecycle("c2", &mockLifecycle{})
	assert.Equal(t, 2, m.ComponentCount())
}

func TestPhaseString(t *testing.T) {
	assert.Equal(t, "init", PhaseInit.String())
	assert.Equal(t, "start", PhaseStart.String())
	assert.Equal(t, "stop", PhaseStop.String())
	assert.Equal(t, "unknown", Phase(99).String())
}
