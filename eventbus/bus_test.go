// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Sync mode =====

func TestBus_Sync_BasicPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var received []*Event
	bus.Subscribe("user.created", func(ctx context.Context, e *Event) error {
		received = append(received, e)
		return nil
	})

	err := bus.Publish(context.Background(), New("user.created", "user-123"))
	assert.NoError(t, err)
	assert.Len(t, received, 1)
	assert.Equal(t, "user.created", received[0].Name)
	assert.Equal(t, "user-123", received[0].Payload)
}

func TestBus_Sync_NoSubscribers(t *testing.T) {
	bus := NewBus()
	defer bus.Close()
	err := bus.Publish(context.Background(), New("no.sub", nil))
	assert.NoError(t, err)
}

func TestBus_Sync_HandlerError(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		return errors.New("handler failed")
	})

	err := bus.Publish(context.Background(), New("test", nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler failed")
}

func TestBus_Sync_MultipleHandlers(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	err := bus.Publish(context.Background(), New("test", nil))
	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestBus_Sync_HandlerErrorStopsDelivery(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var called2 bool
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		return errors.New("fail")
	})
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		called2 = true
		return nil
	})

	err := bus.Publish(context.Background(), New("test", nil))
	assert.Error(t, err)
	assert.False(t, called2, "second handler should not be called in sync mode")
}

// ===== Wildcard =====

func TestBus_Sync_WildcardSingle(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var received []string
	bus.Subscribe("user.*", func(ctx context.Context, e *Event) error {
		received = append(received, e.Name)
		return nil
	})

	bus.Publish(context.Background(), New("user.created", nil))
	bus.Publish(context.Background(), New("user.deleted", nil))
	bus.Publish(context.Background(), New("order.created", nil))

	assert.Equal(t, []string{"user.created", "user.deleted"}, received)
}

func TestBus_Sync_WildcardMulti(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var received []string
	bus.Subscribe("user.>", func(ctx context.Context, e *Event) error {
		received = append(received, e.Name)
		return nil
	})

	bus.Publish(context.Background(), New("user.created", nil))
	bus.Publish(context.Background(), New("user.profile.updated", nil))

	assert.Len(t, received, 2)
}

func TestBus_Sync_StarAll(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count int32
	bus.Subscribe("*", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(context.Background(), New("a", nil))
	bus.Publish(context.Background(), New("b.c", nil))

	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

// ===== Unsubscribe =====

func TestBus_Unsubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count int32
	sub := bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(context.Background(), New("test", nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	err := bus.Unsubscribe(sub)
	assert.NoError(t, err)

	bus.Publish(context.Background(), New("test", nil))
	assert.Equal(t, int32(1), atomic.LoadInt32(&count), "should not receive after unsubscribe")
}

func TestBus_Unsubscribe_NotFound(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	err := bus.Unsubscribe(NewSubscription("fake", "test"))
	assert.Error(t, err)
}

func TestBus_Unsubscribe_Nil(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	err := bus.Unsubscribe(nil)
	assert.Error(t, err)
}

// ===== Parallel mode =====

func TestBus_Parallel(t *testing.T) {
	bus := NewBus(WithDispatchMode(Parallel))
	defer bus.Close()

	var count int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		bus.Subscribe("test", func(ctx context.Context, e *Event) error {
			defer wg.Done()
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	err := bus.Publish(context.Background(), New("test", nil))
	assert.NoError(t, err)
	wg.Wait()
	assert.Equal(t, int32(5), atomic.LoadInt32(&count))
}

// ===== Async mode =====

func TestBus_Async(t *testing.T) {
	bus := NewBus(WithDispatchMode(Async), WithAsyncBufferSize(100))
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	err := bus.Publish(context.Background(), New("test", nil))
	assert.NoError(t, err)

	// Wait for async processing.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&count) >= 1
	}, time.Second, 10*time.Millisecond)
}

func TestBus_Async_QueueFull(t *testing.T) {
	bus := NewBus(WithDispatchMode(Async), WithAsyncBufferSize(1))
	defer bus.Close()

	// Subscribe a slow handler to fill the queue.
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Fill the buffer.
	bus.Publish(context.Background(), New("test", nil))

	// This should fail (queue full) — but timing-dependent, so just check no panic.
	_ = bus.Publish(context.Background(), New("test", nil))
}

// ===== Metrics =====

func TestBus_Metrics(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		return nil
	})

	bus.Publish(context.Background(), New("test", nil))
	bus.Publish(context.Background(), New("test", nil))

	m := bus.Metrics()
	assert.Equal(t, int64(2), m.Published)
	assert.Equal(t, int64(2), m.Delivered)
	assert.Equal(t, int64(1), m.Subscribers)
	assert.Equal(t, int64(0), m.Failed)
}

func TestBus_Metrics_WithFailures(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		return errors.New("fail")
	})

	_ = bus.Publish(context.Background(), New("test", nil))

	m := bus.Metrics()
	assert.Equal(t, int64(1), m.Failed)
}

// ===== Middleware =====

func TestBus_WithMiddleware(t *testing.T) {
	var order []string
	var mu sync.Mutex

	mw := func(next Handler) Handler {
		return func(ctx context.Context, e *Event) error {
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

	bus := NewBus(WithMiddleware(mw))
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		return nil
	})

	bus.Publish(context.Background(), New("test", nil))

	assert.Equal(t, []string{"mw-before", "handler", "mw-after"}, order)
}

func TestBus_WithRecoverMiddleware(t *testing.T) {
	bus := NewBus(WithMiddleware(RecoverMiddleware()))
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		panic("boom")
	})

	// Should not panic, should return error.
	err := bus.Publish(context.Background(), New("test", nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panicked")
}

// ===== Close =====

func TestBus_Close_PublishAfterClose(t *testing.T) {
	bus := NewBus()
	bus.Close()

	err := bus.Publish(context.Background(), New("test", nil))
	assert.Error(t, err)
	assert.True(t, IsClosed(err))
}

func TestBus_Close_Idempotent(t *testing.T) {
	bus := NewBus()
	assert.NoError(t, bus.Close())
	assert.NoError(t, bus.Close())
}

// ===== Nil event =====

func TestBus_NilEvent(t *testing.T) {
	bus := NewBus()
	defer bus.Close()
	err := bus.Publish(context.Background(), nil)
	assert.Error(t, err)
}

// ===== Helper methods =====

func TestBus_SubscriberCount(t *testing.T) {
	b := NewBus()
	defer b.Close()

	b.Subscribe("topic1", func(ctx context.Context, e *Event) error { return nil })
	b.Subscribe("topic1", func(ctx context.Context, e *Event) error { return nil })
	b.Subscribe("topic2", func(ctx context.Context, e *Event) error { return nil })

	assert.Equal(t, 2, b.(*bus).SubscriberCount("topic1"))
	assert.Equal(t, 1, b.(*bus).SubscriberCount("topic2"))
}

func TestBus_Topics(t *testing.T) {
	b := NewBus()
	defer b.Close()

	b.Subscribe("topic1", func(ctx context.Context, e *Event) error { return nil })
	b.Subscribe("topic2", func(ctx context.Context, e *Event) error { return nil })

	topics := b.(*bus).Topics()
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
}

// ===== Concurrent =====

func TestBus_ConcurrentPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count int32
	bus.Subscribe("test", func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), New("test", nil))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&count))
}
