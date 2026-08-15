// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package activemq

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.Addr)
	assert.NotEmpty(t, cfg.Network)
	assert.True(t, cfg.Heartbeat > 0)
	assert.True(t, cfg.ConnectTimeout > 0)
}

func TestNew_EmptyAddr(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "addr is required")
}

func TestNew_Defaults(t *testing.T) {
	b, err := New(Config{Addr: "localhost:61613"})
	require.NoError(t, err)
	require.NotNil(t, b)
	assert.True(t, b.cfg.ConnectTimeout > 0)
	assert.Equal(t, "tcp", b.cfg.Network)
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		Addr:           "localhost:61613",
		Login:          "admin",
		Passcode:       "secret",
		Vhost:          "/",
		Heartbeat:      5 * time.Second,
		ConnectTimeout: 3 * time.Second,
	}
	b, err := New(cfg)
	require.NoError(t, err)
	assert.Equal(t, "admin", b.cfg.Login)
	assert.Equal(t, 5*time.Second, b.cfg.Heartbeat)
	assert.Equal(t, 3*time.Second, b.cfg.ConnectTimeout)
}

func TestBroker_IsConnected(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	assert.False(t, b.IsConnected())
}

func TestBroker_Close(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	assert.NoError(t, b.Close())
}

func TestBroker_Close_Idempotent(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	require.NoError(t, b.Close())
	assert.NoError(t, b.Close())
}

func TestBroker_Connect_InvalidAddr(t *testing.T) {
	b, _ := New(Config{Addr: "127.0.0.1:1", ConnectTimeout: 500 * time.Millisecond})
	err := b.Connect()
	assert.Error(t, err)
}

func TestBroker_Metrics_Initial(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	m := b.Metrics()
	assert.Equal(t, int64(0), m.Published)
	assert.Equal(t, int64(0), m.Consumed)
}

func TestBroker_Topology_NoOps(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	assert.NoError(t, b.DeclareExchange("ex", mq.DefaultExchangeOptions()))
	assert.NoError(t, b.DeclareQueue("q", mq.DefaultQueueOptions()))
	assert.NoError(t, b.Bind("q", "ex", "rk"))
	assert.NoError(t, b.Unbind("q", "ex", "rk"))
	assert.NoError(t, b.DeleteQueue("q"))
	assert.NoError(t, b.DeleteExchange("ex"))
}

func TestBroker_Producer_EmptyDestination(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	_, err := b.Producer("", mq.PublishOptions{})
	assert.Error(t, err)
}

func TestBroker_Consumer_NoHandler(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	_, err := b.Consumer("/queue/test", mq.ConsumeOptions{})
	assert.ErrorIs(t, err, mq.ErrNoHandler)
}

func TestBroker_Producer_AfterClose(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	require.NoError(t, b.Close())
	_, err := b.Producer("/queue/test", mq.PublishOptions{})
	assert.ErrorIs(t, err, mq.ErrClosed)
}

func TestBroker_Consumer_AfterClose(t *testing.T) {
	b, _ := New(Config{Addr: "localhost:61613"})
	require.NoError(t, b.Close())
	_, err := b.Consumer("/queue/test", mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error { return nil },
	})
	assert.ErrorIs(t, err, mq.ErrClosed)
}

func TestToHeaderValue(t *testing.T) {
	assert.Equal(t, "str", toHeaderValue("str"))
	assert.Equal(t, "bytes", toHeaderValue([]byte("bytes")))
	assert.Equal(t, "true", toHeaderValue(true))
	assert.Equal(t, "42", toHeaderValue(42))
	assert.Equal(t, "9223372036854775807", toHeaderValue(int64(9223372036854775807)))
	assert.Equal(t, "3.14", toHeaderValue(3.14))
	assert.Equal(t, "", toHeaderValue(struct{}{}))
}

// ------------------------------------------------------------
// Interface compliance
// ------------------------------------------------------------

func TestBroker_ImplementsBroker(t *testing.T) {
	var _ mq.Broker = (*Broker)(nil)
}

func TestProducer_ImplementsProducer(t *testing.T) {
	var _ mq.Producer = (*Producer)(nil)
}

func TestConsumer_ImplementsConsumer(t *testing.T) {
	var _ mq.Consumer = (*Consumer)(nil)
}

func TestDelivery_ImplementsDelivery(t *testing.T) {
	var _ mq.Delivery = (*stompDelivery)(nil)
}

// ------------------------------------------------------------
// Integration tests (skip if ACTIVEMQ_ADDR not set)
// ------------------------------------------------------------

func activemqAddr() string {
	return os.Getenv("ACTIVEMQ_ADDR")
}

func TestIntegration_ConnectPublishConsume(t *testing.T) {
	addr := activemqAddr()
	if addr == "" {
		t.Skip("ACTIVEMQ_ADDR not set; skipping integration test")
	}

	broker, err := New(Config{Addr: addr, ConnectTimeout: 5 * time.Second})
	require.NoError(t, err)
	require.NoError(t, broker.Connect())
	defer broker.Close()
	assert.True(t, broker.IsConnected())

	queue := "/queue/ling-base-itest"
	producer, err := broker.Producer(queue, mq.PublishOptions{})
	require.NoError(t, err)

	body := []byte("hello-activemq")
	require.NoError(t, producer.Publish(context.Background(), &mq.Message{
		Body:        body,
		ContentType: "text/plain",
	}))

	received := make(chan []byte, 1)
	consumer, err := broker.Consumer(queue, mq.ConsumeOptions{
		AutoAck: false,
		Handler: func(ctx context.Context, d mq.Delivery) error {
			received <- d.Body()
			return d.Ack()
		},
	})
	require.NoError(t, err)
	require.NoError(t, consumer.Start(context.Background()))
	defer consumer.Stop(5 * time.Second)

	select {
	case got := <-received:
		assert.Equal(t, body, got)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestIntegration_TopologyNoOps(t *testing.T) {
	addr := activemqAddr()
	if addr == "" {
		t.Skip("ACTIVEMQ_ADDR not set; skipping integration test")
	}
	broker, err := New(Config{Addr: addr, ConnectTimeout: 5 * time.Second})
	require.NoError(t, err)
	require.NoError(t, broker.Connect())
	defer broker.Close()

	assert.NoError(t, broker.DeclareExchange("ex", mq.DefaultExchangeOptions()))
	assert.NoError(t, broker.DeclareQueue("q", mq.DefaultQueueOptions()))
}
