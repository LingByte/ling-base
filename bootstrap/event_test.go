// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/LingByte/ling-base/eventbus"
)

// waitForBus drains pending async events by closing the bus (which waits
// for all async workers to finish) and then asserting.
func waitForBus(t *testing.T, bus eventbus.Bus) {
	t.Helper()
	// Close drains the async queue — workers process all pending events
	// before returning. We don't defer Close so we can reuse the pattern.
	_ = bus.Close()
}

func TestEvent_NewEvent(t *testing.T) {
	e := eventbus.New("test.event", "source")
	assert.Equal(t, "test.event", e.Name)
	assert.Equal(t, "source", e.Payload)
	assert.False(t, e.Time.IsZero())
	assert.NotEmpty(t, e.ID)
}

func TestEventBus_SubscribePublish(t *testing.T) {
	bus := newEventBus()

	var received *eventbus.Event
	bus.Subscribe("test.event", func(ctx context.Context, e *eventbus.Event) error {
		received = e
		return nil
	})

	e := eventbus.New("test.event", "src")
	err := bus.Publish(context.Background(), e)
	assert.NoError(t, err)
	waitForBus(t, bus)
	assert.Equal(t, "test.event", received.Name)
	assert.Equal(t, "src", received.Payload)
}

func TestEventBus_NoListeners(t *testing.T) {
	bus := newEventBus()
	err := bus.Publish(context.Background(), eventbus.New("no.listeners", nil))
	assert.NoError(t, err)
	waitForBus(t, bus)
}

func TestEventBus_MultipleListeners(t *testing.T) {
	bus := newEventBus()

	var order []string
	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error {
		order = append(order, "l1")
		return nil
	})
	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error {
		order = append(order, "l2")
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("ev", nil))
	waitForBus(t, bus)
	assert.ElementsMatch(t, []string{"l1", "l2"}, order)
}

func TestEventBus_ListenerError(t *testing.T) {
	bus := newEventBus()

	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error {
		return assert.AnError
	})
	// In async mode, Publish returns nil (enqueued) — the error is
	// recorded in metrics but not returned to the caller.
	err := bus.Publish(context.Background(), eventbus.New("ev", nil))
	assert.NoError(t, err)
	waitForBus(t, bus)
}

func TestEventBus_Wildcard(t *testing.T) {
	bus := newEventBus()

	var received []string
	bus.Subscribe("user.*", func(ctx context.Context, e *eventbus.Event) error {
		received = append(received, e.Name)
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("user.created", nil))
	bus.Publish(context.Background(), eventbus.New("user.deleted", nil))
	bus.Publish(context.Background(), eventbus.New("order.created", nil))

	waitForBus(t, bus)
	assert.ElementsMatch(t, []string{"user.created", "user.deleted"}, received)
}

func TestEventBus_Metrics(t *testing.T) {
	bus := newEventBus()

	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error { return nil })
	bus.Publish(context.Background(), eventbus.New("ev", nil))
	bus.Publish(context.Background(), eventbus.New("ev", nil))

	waitForBus(t, bus)
	m := bus.Metrics()
	assert.Equal(t, int64(2), m.Published)
	assert.Equal(t, int64(2), m.Delivered)
	assert.Equal(t, int64(1), m.Subscribers)
}

func TestEventBus_PublishWithTimeout(t *testing.T) {
	bus := newEventBus()

	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		bus.Publish(ctx, eventbus.New("ev", nil))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked too long")
	}
	waitForBus(t, bus)
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := newEventBus()

	var count int32
	bus.Subscribe("ev", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), eventbus.New("ev", nil))
		}()
	}
	wg.Wait()
	waitForBus(t, bus)
	assert.Equal(t, int32(100), atomic.LoadInt32(&count))
}

func TestEventBus_Middleware(t *testing.T) {
	bus := newEventBus()

	var order []string
	bus.Subscribe("ev", eventbus.ApplyMiddleware(
		func(ctx context.Context, e *eventbus.Event) error {
			order = append(order, "handler")
			return nil
		},
		eventbus.RecoverMiddleware(),
	))

	bus.Publish(context.Background(), eventbus.New("ev", nil))
	waitForBus(t, bus)
	assert.Contains(t, order, "handler")
}
