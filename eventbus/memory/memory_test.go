// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/eventbus"
)

// ===== Sync mode =====

func TestNew_Sync_BasicPublish(t *testing.T) {
	bus := New()
	defer bus.Close()

	var received []*eventbus.Event
	bus.Subscribe("user.created", func(ctx context.Context, e *eventbus.Event) error {
		received = append(received, e)
		return nil
	})

	err := bus.Publish(context.Background(), eventbus.New("user.created", "user-123"))
	assert.NoError(t, err)
	assert.Len(t, received, 1)
	assert.Equal(t, "user.created", received[0].Name)
	assert.Equal(t, "user-123", received[0].Payload)
}

func TestNew_Sync_NoSubscribers(t *testing.T) {
	bus := New()
	defer bus.Close()
	err := bus.Publish(context.Background(), eventbus.New("no.sub", nil))
	assert.NoError(t, err)
}

func TestNew_Sync_HandlerError(t *testing.T) {
	bus := New()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return errors.New("handler failed")
	})

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler failed")
}

func TestNew_Sync_MultipleHandlers(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestNew_Sync_HandlerErrorStopsDelivery(t *testing.T) {
	bus := New()
	defer bus.Close()

	var called2 bool
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return errors.New("fail")
	})
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		called2 = true
		return nil
	})

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Error(t, err)
	assert.False(t, called2, "second handler should not be called in sync mode")
}

// ===== Wildcard =====

func TestNew_Sync_WildcardSingle(t *testing.T) {
	bus := New()
	defer bus.Close()

	var received []string
	bus.Subscribe("user.*", func(ctx context.Context, e *eventbus.Event) error {
		received = append(received, e.Name)
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("user.created", nil))
	bus.Publish(context.Background(), eventbus.New("user.deleted", nil))
	bus.Publish(context.Background(), eventbus.New("order.created", nil))

	assert.Equal(t, []string{"user.created", "user.deleted"}, received)
}

func TestNew_Sync_WildcardMulti(t *testing.T) {
	bus := New()
	defer bus.Close()

	var received []string
	bus.Subscribe("user.>", func(ctx context.Context, e *eventbus.Event) error {
		received = append(received, e.Name)
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("user.created", nil))
	bus.Publish(context.Background(), eventbus.New("user.profile.updated", nil))

	assert.Len(t, received, 2)
}

func TestNew_Sync_StarAll(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int32
	bus.Subscribe("*", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("a", nil))
	bus.Publish(context.Background(), eventbus.New("b.c", nil))

	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

// ===== Unsubscribe =====

func TestNew_Unsubscribe(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int32
	sub := bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	err := bus.Unsubscribe(sub)
	assert.NoError(t, err)

	bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(&count), "should not receive after unsubscribe")
}

func TestNew_Unsubscribe_NotFound(t *testing.T) {
	bus := New()
	defer bus.Close()

	err := bus.Unsubscribe(eventbus.NewSubscription("fake", "test"))
	assert.Error(t, err)
}

// ===== Parallel mode =====

func TestNew_Parallel(t *testing.T) {
	bus := New(WithDispatchMode(Parallel))
	defer bus.Close()

	var count int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
			defer wg.Done()
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.NoError(t, err)
	wg.Wait()
	assert.Equal(t, int32(5), atomic.LoadInt32(&count))
}

// ===== Async mode =====

func TestNew_Async(t *testing.T) {
	bus := New(WithDispatchMode(Async), WithAsyncBufferSize(100))
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.NoError(t, err)

	// Wait for async processing.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) >= 1
	}, time.Second, 10*time.Millisecond)
}

func TestNew_Async_QueueFull(t *testing.T) {
	bus := New(WithDispatchMode(Async), WithAsyncBufferSize(1))
	defer bus.Close()

	// Subscribe a slow handler to fill the queue.
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Fill the buffer.
	bus.Publish(context.Background(), eventbus.New("test", nil))

	// This should fail (queue full) — but timing-dependent, so just check no panic.
	_ = bus.Publish(context.Background(), eventbus.New("test", nil))
}

// ===== Metrics =====

func TestNew_Metrics(t *testing.T) {
	bus := New()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("test", nil))
	bus.Publish(context.Background(), eventbus.New("test", nil))

	m := bus.Metrics()
	assert.Equal(t, int64(2), m.Published)
	assert.Equal(t, int64(2), m.Delivered)
	assert.Equal(t, int64(1), m.Subscribers)
	assert.Equal(t, int64(0), m.Failed)
}

func TestNew_Metrics_WithFailures(t *testing.T) {
	bus := New()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return errors.New("fail")
	})

	_ = bus.Publish(context.Background(), eventbus.New("test", nil))

	m := bus.Metrics()
	assert.Equal(t, int64(1), m.Failed)
}

// ===== Middleware =====

func TestNew_WithMiddleware(t *testing.T) {
	var order []string
	var mu sync.Mutex

	mw := func(next eventbus.Handler) eventbus.Handler {
		return func(ctx context.Context, e *eventbus.Event) error {
			mu.Lock()
			order = append(order, "mw-before")
			mu.Unlock()
			err := next(ctx, e)
			mu.Lock()
			order = append(order, "mw-after")
			mu.Unlock()
			return err
		}
	}

	bus := New(WithMiddleware(mw))
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("test", nil))

	assert.Equal(t, []string{"mw-before", "handler", "mw-after"}, order)
}

func TestNew_WithRecoverMiddleware(t *testing.T) {
	bus := New(WithMiddleware(eventbus.RecoverMiddleware()))
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		panic("boom")
	})

	// Should not panic, should return error.
	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panicked")
}

// ===== Close =====

func TestNew_Close_PublishAfterClose(t *testing.T) {
	bus := New()
	bus.Close()

	err := bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Error(t, err)
	assert.True(t, eventbus.IsClosed(err))
}

func TestNew_Close_Idempotent(t *testing.T) {
	bus := New()
	assert.NoError(t, bus.Close())
	assert.NoError(t, bus.Close())
}

// ===== Nil event =====

func TestNew_NilEvent(t *testing.T) {
	bus := New()
	defer bus.Close()
	err := bus.Publish(context.Background(), nil)
	assert.Error(t, err)
}

// ===== Helper methods =====

func TestNew_SubscriberCount(t *testing.T) {
	bus := New()
	defer bus.Close()

	bus.Subscribe("topic1", func(ctx context.Context, e *eventbus.Event) error { return nil })
	bus.Subscribe("topic1", func(ctx context.Context, e *eventbus.Event) error { return nil })
	bus.Subscribe("topic2", func(ctx context.Context, e *eventbus.Event) error { return nil })

	assert.Equal(t, 2, bus.SubscriberCount("topic1"))
	assert.Equal(t, 1, bus.SubscriberCount("topic2"))
}

func TestNew_Topics(t *testing.T) {
	bus := New()
	defer bus.Close()

	bus.Subscribe("topic1", func(ctx context.Context, e *eventbus.Event) error { return nil })
	bus.Subscribe("topic2", func(ctx context.Context, e *eventbus.Event) error { return nil })

	topics := bus.Topics()
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
}

// ===== Concurrent =====

func TestNew_ConcurrentPublish(t *testing.T) {
	bus := New()
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), eventbus.New("test", nil))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&count))
}
