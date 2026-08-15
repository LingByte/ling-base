// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redis

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/eventbus"
)

func setupMiniRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return client, mr
}

func TestNew_Basic(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:events", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	assert.NotNil(t, bus)
}

func TestNew_Publish(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:pub", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	e := eventbus.New("user.created", "user-123")
	err = bus.Publish(context.Background(), e)
	assert.NoError(t, err)

	// Verify the stream has 1 entry.
	len, err := bus.StreamLen(context.Background(), "user.created")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), len)
}

func TestNew_Publish_NilEvent(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:nil", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	err = bus.Publish(context.Background(), nil)
	assert.Error(t, err)
}

func TestNew_PublishAfterClose(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:close", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	bus.Close()

	err = bus.Publish(context.Background(), eventbus.New("test", nil))
	assert.Error(t, err)
	assert.True(t, eventbus.IsClosed(err))
}

func TestNew_SubscribeAndConsume(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:sub",
		WithConsumerGroup("worker-1"),
		WithBlockTime(100*time.Millisecond),
		WithWorkers(1),
	)
	require.NoError(t, err)
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe("user.created", func(ctx context.Context, e *eventbus.Event) error {
		received.Add(1)
		assert.Equal(t, "user.created", e.Name)
		return nil
	})

	bus.Start()

	// Publish an event.
	err = bus.Publish(context.Background(), eventbus.New("user.created", "user-123"))
	require.NoError(t, err)

	// Wait for consumption.
	require.Eventually(t, func() bool {
		return received.Load() >= 1
	}, 3*time.Second, 50*time.Millisecond)

	assert.Equal(t, int32(1), received.Load())
}

func TestNew_Metrics(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:metrics", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return nil
	})

	bus.Publish(context.Background(), eventbus.New("test", nil))

	m := bus.Metrics()
	assert.Equal(t, int64(1), m.Published)
}

func TestNew_Unsubscribe(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:unsub", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	sub := bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		return nil
	})

	err = bus.Unsubscribe(sub)
	assert.NoError(t, err)
}

func TestNew_PurgeStream(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:purge", WithConsumerGroup("worker-1"))
	require.NoError(t, err)
	defer bus.Close()

	bus.Publish(context.Background(), eventbus.New("test.event", nil))

	len, _ := bus.StreamLen(context.Background(), "test.event")
	assert.Equal(t, int64(1), len)

	err = bus.PurgeStream(context.Background(), "test.event")
	assert.NoError(t, err)

	len, _ = bus.StreamLen(context.Background(), "test.event")
	assert.Equal(t, int64(0), len)
}

func TestNew_Close_Idempotent(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:idem", WithConsumerGroup("worker-1"))
	require.NoError(t, err)

	assert.NoError(t, bus.Close())
	assert.NoError(t, bus.Close())
}

func TestNew_Middleware(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "test:mw",
		WithConsumerGroup("worker-1"),
		WithBlockTime(100*time.Millisecond),
		WithMiddleware(eventbus.RecoverMiddleware()),
	)
	require.NoError(t, err)
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe("test", func(ctx context.Context, e *eventbus.Event) error {
		received.Add(1)
		return nil
	})

	bus.Start()
	bus.Publish(context.Background(), eventbus.New("test", nil))

	require.Eventually(t, func() bool {
		return received.Load() >= 1
	}, 3*time.Second, 50*time.Millisecond)
}

func TestNew_DefaultPrefix(t *testing.T) {
	client, _ := setupMiniRedis(t)
	bus, err := New(client, "")
	require.NoError(t, err)
	defer bus.Close()
	assert.NotNil(t, bus)
}
