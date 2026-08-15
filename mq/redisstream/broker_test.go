// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redisstream

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Interface compliance checks (compile-time).
var (
	_ mq.Broker   = (*Broker)(nil)
	_ mq.Producer = (*Producer)(nil)
	_ mq.Consumer = (*Consumer)(nil)
	_ mq.Delivery = (*streamDelivery)(nil)
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.Addr)
	assert.True(t, cfg.BlockTime > 0)
	assert.True(t, cfg.Count > 0)
}

func TestNew_EmptyAddr(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "addr is required")
}

func TestNew_Defaults(t *testing.T) {
	b, err := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.True(t, b.cfg.BlockTime > 0)
	assert.True(t, b.cfg.Count > 0)
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		Addr:         "localhost:6379",
		Group:        "test-group",
		ConsumerName: "test-consumer",
		BlockTime:    3 * time.Second,
		Count:        20,
		MaxLen:       1000,
	}
	b, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "test-group", b.cfg.Group)
	assert.Equal(t, "test-consumer", b.cfg.ConsumerName)
	assert.Equal(t, int64(20), b.cfg.Count)
	assert.Equal(t, int64(1000), b.cfg.MaxLen)
}

func TestBroker_IsConnected_BeforeConnect(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.False(t, b.IsConnected())
}

func TestBroker_Close_NotConnected(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	// Close should be safe even if never connected.
	assert.NoError(t, b.Close())
}

func TestBroker_Close_Idempotent(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.Close())
	// Second close should not panic.
	assert.NoError(t, b.Close())
}

func TestBroker_Metrics_Initial(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	m := b.Metrics()
	assert.Equal(t, int64(0), m.Published)
	assert.Equal(t, int64(0), m.Consumed)
}

func TestBroker_DeclareExchange_NoOp(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.DeclareExchange("test", mq.ExchangeOptions{}))
}

func TestBroker_DeclareQueue_NoOp(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.DeclareQueue("test", mq.QueueOptions{}))
}

func TestBroker_Bind_NoOp(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.Bind("q", "ex", "rk"))
}

func TestBroker_Unbind_NoOp(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.Unbind("q", "ex", "rk"))
}

func TestBroker_DeleteExchange_NoOp(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	assert.NoError(t, b.DeleteExchange("test"))
}

func TestBroker_Consumer_NoHandler(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:6379"})
	_, err := b.Consumer("test", mq.ConsumeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
}

func TestBuildStreamValues(t *testing.T) {
	msg := &mq.Message{
		ID:              "msg-1",
		Body:            []byte("hello"),
		ContentType:     "application/json",
		CorrelationID:   "corr-1",
		ReplyTo:         "reply.q",
		Type:            "user.created",
		UserID:          "u1",
		AppID:           "app1",
		RoutingKey:      "rk",
		Priority:        5,
		DeliveryMode:    mq.Persistent,
		Timestamp:       time.Unix(0, 1700000000000000000),
		Headers:         map[string]any{"x-custom": "val"},
	}
	values := buildStreamValues(msg)
	assert.Equal(t, "hello", values["body"])
	assert.Equal(t, "msg-1", values["id"])
	assert.Equal(t, "application/json", values["content_type"])
	assert.Equal(t, "corr-1", values["correlation_id"])
	assert.Equal(t, "reply.q", values["reply_to"])
	assert.Equal(t, "user.created", values["type"])
	assert.Equal(t, "u1", values["user_id"])
	assert.Equal(t, "app1", values["app_id"])
	assert.Equal(t, "rk", values["routing_key"])
	assert.Equal(t, "val", values["h_x-custom"])
}

// ----------------------------------------------------------
// Integration tests (skip if REDIS_ADDR env not set)
// ----------------------------------------------------------

func redisAddr() string {
	return os.Getenv("REDIS_ADDR")
}

func newIntegrationBroker(t *testing.T) *Broker {
	t.Helper()
	addr := redisAddr()
	if addr == "" {
		t.Skip("REDIS_ADDR not set; skipping integration test")
	}
	b, err := New(Config{Addr: addr, Group: "test-group", Count: 10, BlockTime: time.Second})
	require.NoError(t, err)
	require.NoError(t, b.Connect())
	return b
}

func TestIntegration_Connect(t *testing.T) {
	b := newIntegrationBroker(t)
	defer b.Close()
	assert.True(t, b.IsConnected())
}

func TestIntegration_PublishConsume(t *testing.T) {
	b := newIntegrationBroker(t)
	defer b.Close()

	stream := "test-stream-pubcon"
	// Clean up any previous data.
	b.connMu.RLock()
	client := b.client
	b.connMu.RUnlock()
	_ = client.Del(context.Background(), stream).Err()

	producer, err := b.Producer(stream, mq.PublishOptions{})
	require.NoError(t, err)

	err = producer.Publish(context.Background(), &mq.Message{
		Body:        []byte(`{"event":"test"}`),
		ContentType: "application/json",
		Headers:     map[string]any{"x-key": "val"},
	})
	require.NoError(t, err)

	received := make(chan mq.Delivery, 1)
	consumer, err := b.Consumer(stream, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			received <- d
			return d.Ack()
		},
		Args: map[string]any{"group": "test-group-pubcon", "consumer": "c1"},
	})
	require.NoError(t, err)

	// Use a fresh group so we read from the beginning.
	// XGroupCreate with "0" start would read existing messages; but our
	// ensureGroup uses "$" (only new). So publish after starting.
	require.NoError(t, consumer.Start(context.Background()))
	defer consumer.Stop(5 * time.Second)

	// Publish another message after the consumer starts.
	err = producer.Publish(context.Background(), &mq.Message{
		Body: []byte(`{"event":"test2"}`),
	})
	require.NoError(t, err)

	select {
	case d := <-received:
		assert.NotEmpty(t, d.Body())
		assert.Equal(t, stream, d.RoutingKey())
		assert.Equal(t, stream, d.Exchange())
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestIntegration_DeleteQueue(t *testing.T) {
	b := newIntegrationBroker(t)
	defer b.Close()

	stream := "test-stream-delete"
	producer, err := b.Producer(stream, mq.PublishOptions{})
	require.NoError(t, err)
	err = producer.Publish(context.Background(), &mq.Message{Body: []byte("x")})
	require.NoError(t, err)

	err = b.DeleteQueue(stream)
	require.NoError(t, err)
}
