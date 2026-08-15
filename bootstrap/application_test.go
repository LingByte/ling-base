// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/LingByte/ling-base/eventbus"
)

// testAppComponent is a Lifecycle component for testing.
type testAppComponent struct {
	started atomic.Bool
}

func (c *testAppComponent) Start(ctx context.Context) error {
	c.started.Store(true)
	return nil
}

func (c *testAppComponent) Stop(ctx context.Context) error {
	c.started.Store(false)
	return nil
}

func (c *testAppComponent) IsRunning() bool {
	return c.started.Load()
}

func TestApplication_New(t *testing.T) {
	app := New("testapp")
	assert.Equal(t, "testapp", app.Name())
	assert.Equal(t, StatusCreated, app.Status())
	assert.NotNil(t, app.Registry())
	assert.NotNil(t, app.Lifecycle())
	assert.NotNil(t, app.Profiles())
	assert.NotNil(t, app.Properties())
	assert.NotNil(t, app.Events())
}

func TestApplication_WithOptions(t *testing.T) {
	app := New("testapp",
		WithProfile(ProfileProd),
		WithBannerText("CustomBanner"),
		WithShutdownTimeout(60*time.Second),
		WithProperties(map[string]string{"key": "value"}),
	)
	assert.True(t, app.Profiles().IsProd())
	assert.Equal(t, "CustomBanner", app.bannerText)
	assert.Equal(t, "value", app.Properties().Get("key"))
}

func TestApplication_Register(t *testing.T) {
	app := New("testapp")
	comp := &testAppComponent{}
	err := app.Register("comp", comp)
	assert.NoError(t, err)
	assert.True(t, app.Registry().Contains("comp"))
	assert.Equal(t, 1, app.Lifecycle().ComponentCount())
}

func TestApplication_RegisterDuplicate(t *testing.T) {
	app := New("testapp")
	app.Register("comp", &testAppComponent{})
	err := app.Register("comp", &testAppComponent{})
	assert.Error(t, err)
}

func TestApplication_MustRegister(t *testing.T) {
	app := New("testapp")
	assert.NotPanics(t, func() {
		app.MustRegister("comp", &testAppComponent{})
	})
	assert.Panics(t, func() {
		app.MustRegister("comp", &testAppComponent{})
	})
}

func TestApplication_RunAsync(t *testing.T) {
	app := New("testapp",
		WithProfile(ProfileTest),
		WithShutdownTimeout(5*time.Second),
	)

	comp := &testAppComponent{}
	app.Register("comp", comp)

	errCh := app.RunAsync()

	// Wait for start.
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("app didn't start in time")
	}

	assert.Equal(t, StatusRunning, app.Status())
	assert.True(t, comp.IsRunning())
	assert.True(t, app.Uptime() > 0)

	// Stop the app.
	app.Stop()

	// Give it a moment to shut down.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, StatusStopped, app.Status())
	assert.False(t, comp.IsRunning())
}

func TestApplication_RunAsyncInitError(t *testing.T) {
	app := New("testapp")
	app.AddInitHook("bad", func(ctx context.Context) error {
		return assert.AnError
	})

	errCh := app.RunAsync()
	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Equal(t, StatusFailed, app.Status())
	case <-time.After(2 * time.Second):
		t.Fatal("app didn't fail in time")
	}
}

func TestApplication_RunAsyncStartError(t *testing.T) {
	app := New("testapp")
	// Register a component that fails to start.
	app.Register("bad", &failingComponent{})

	errCh := app.RunAsync()
	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Equal(t, StatusFailed, app.Status())
	case <-time.After(2 * time.Second):
		t.Fatal("app didn't fail in time")
	}
}

type failingComponent struct{}

func (c *failingComponent) Start(ctx context.Context) error { return assert.AnError }
func (c *failingComponent) Stop(ctx context.Context) error  { return nil }
func (c *failingComponent) IsRunning() bool                 { return false }

func TestApplication_Events(t *testing.T) {
	app := New("testapp", WithShutdownTimeout(2*time.Second))

	var eventsReceived []string
	app.OnEvent(EventAppStarting, func(ctx context.Context, e *eventbus.Event) error {
		eventsReceived = append(eventsReceived, e.Name)
		return nil
	})
	app.OnEvent(EventAppStarted, func(ctx context.Context, e *eventbus.Event) error {
		eventsReceived = append(eventsReceived, e.Name)
		return nil
	})
	app.OnEvent(EventAppReady, func(ctx context.Context, e *eventbus.Event) error {
		eventsReceived = append(eventsReceived, e.Name)
		return nil
	})

	errCh := app.RunAsync()
	<-errCh

	time.Sleep(50 * time.Millisecond) // let events propagate
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	assert.Contains(t, eventsReceived, EventAppStarting)
	assert.Contains(t, eventsReceived, EventAppStarted)
	assert.Contains(t, eventsReceived, EventAppReady)
}

func TestApplication_AddShutdownHook(t *testing.T) {
	app := New("testapp", WithShutdownTimeout(2*time.Second))
	var hookCalled int32
	app.AddShutdownHook("custom", func(ctx context.Context) error {
		atomic.AddInt32(&hookCalled, 1)
		return nil
	})

	errCh := app.RunAsync()
	<-errCh
	app.Stop()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&hookCalled))
}

func TestApplication_StatusString(t *testing.T) {
	assert.Equal(t, "created", StatusCreated.String())
	assert.Equal(t, "running", StatusRunning.String())
	assert.Equal(t, "stopped", StatusStopped.String())
	assert.Equal(t, "failed", StatusFailed.String())
	assert.Equal(t, "unknown", AppStatus(99).String())
}

func TestApplication_UptimeNotStarted(t *testing.T) {
	app := New("testapp")
	assert.Equal(t, time.Duration(0), app.Uptime())
}
