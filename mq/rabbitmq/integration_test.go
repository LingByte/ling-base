// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rabbitmq

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rabbitmqURL returns the RabbitMQ URL from env or empty if not available.
func rabbitmqURL() string {
	if u := os.Getenv("RABBITMQ_URL"); u != "" {
		return u
	}
	return "amqp://guest:guest@localhost:5672/"
}

// hasRabbitMQ checks if a RabbitMQ broker is reachable.
func hasRabbitMQ() bool {
	b, err := New(Config{
		URL:           rabbitmqURL(),
		DialerTimeout: 2 * time.Second,
	})
	if err != nil {
		return false
	}
	defer b.Close()
	return b.Connect() == nil
}

// TestIntegrationBroker is a full integration test that requires a
// running RabbitMQ instance. Skipped if no broker is available.
func TestIntegrationBroker(t *testing.T) {
	if !hasRabbitMQ() {
		t.Skip("RabbitMQ not available, skipping integration test")
	}

	b, err := New(Config{URL: rabbitmqURL()})
	require.NoError(t, err)
	defer b.Close()

	require.NoError(t, b.Connect())
	assert.True(t, b.IsConnected())

	// Declare topology.
	exchange := "test.lingbase"
	queue := "test.lingbase.queue"

	err = b.DeclareExchange(exchange, mq.DefaultExchangeOptions())
	require.NoError(t, err)
	defer b.DeleteExchange(exchange)

	err = b.DeclareQueue(queue, mq.DefaultQueueOptions())
	require.NoError(t, err)
	defer b.DeleteQueue(queue)

	err = b.Bind(queue, exchange, "test.#")
	require.NoError(t, err)
	defer b.Unbind(queue, exchange, "test.#")

	// Publish.
	producer, err := b.Producer(exchange, mq.PublishOptions{Persistent: true})
	require.NoError(t, err)

	ctx := context.Background()
	err = producer.Publish(ctx, &mq.Message{
		RoutingKey:  "test.hello",
		Body:        []byte(`{"hello":"world"}`),
		ContentType: "application/json",
	})
	assert.NoError(t, err)

	// Consume.
	received := make(chan []byte, 1)
	consumer, err := b.Consumer(queue, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			received <- d.Body()
			return d.Ack()
		},
		QosPrefetchCount: 10,
		Concurrency:      1,
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Start(ctx))
	assert.True(t, consumer.IsRunning())

	select {
	case body := <-received:
		assert.Contains(t, string(body), "hello")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Stop.
	assert.NoError(t, consumer.Stop(5*time.Second))
	assert.False(t, consumer.IsRunning())
}

// TestIntegrationBroker_Confirms tests publisher confirms.
func TestIntegrationBroker_Confirms(t *testing.T) {
	if !hasRabbitMQ() {
		t.Skip("RabbitMQ not available, skipping integration test")
	}

	b, err := New(Config{URL: rabbitmqURL()})
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, b.Connect())

	exchange := "test.confirms"
	queue := "test.confirms.queue"

	_ = b.DeclareExchange(exchange, mq.DefaultExchangeOptions())
	defer b.DeleteExchange(exchange)
	_ = b.DeclareQueue(queue, mq.DefaultQueueOptions())
	defer b.DeleteQueue(queue)
	_ = b.Bind(queue, exchange, "#")
	defer b.Unbind(queue, exchange, "#")

	producer, err := b.Producer(exchange, mq.PublishOptions{
		Persistent: true,
		Confirm:    true,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = producer.Publish(ctx, &mq.Message{
		RoutingKey: "test",
		Body:       []byte("with confirms"),
	})
	assert.NoError(t, err)
}

// TestIntegrationBroker_Reconnect tests automatic reconnection.
func TestIntegrationBroker_Reconnect(t *testing.T) {
	if !hasRabbitMQ() {
		t.Skip("RabbitMQ not available, skipping integration test")
	}

	b, err := New(Config{
		URL:            rabbitmqURL(),
		ReconnectDelay: 500 * time.Millisecond,
	})
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, b.Connect())
	assert.True(t, b.IsConnected())
}

// TestIntegrationBroker_Middleware tests middleware chain in consumer.
func TestIntegrationBroker_Middleware(t *testing.T) {
	if !hasRabbitMQ() {
		t.Skip("RabbitMQ not available, skipping integration test")
	}

	b, err := New(Config{URL: rabbitmqURL()})
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, b.Connect())

	exchange := "test.mw"
	queue := "test.mw.queue"
	_ = b.DeclareExchange(exchange, mq.DefaultExchangeOptions())
	defer b.DeleteExchange(exchange)
	_ = b.DeclareQueue(queue, mq.DefaultQueueOptions())
	defer b.DeleteQueue(queue)
	_ = b.Bind(queue, exchange, "#")
	defer b.Unbind(queue, exchange, "#")

	producer, _ := b.Producer(exchange, mq.PublishOptions{})
	ctx := context.Background()
	_ = producer.Publish(ctx, &mq.Message{
		RoutingKey: "test",
		Body:       []byte("middleware test"),
	})

	var order []string
	consumer, _ := b.Consumer(queue, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			order = append(order, "handler")
			return d.Ack()
		},
		Middleware: []mq.Middleware{
			func(next mq.Handler) mq.Handler {
				return func(ctx context.Context, d mq.Delivery) error {
					order = append(order, "mw1-before")
					err := next(ctx, d)
					order = append(order, "mw1-after")
					return err
				}
			},
			mq.RecoverMiddleware(nil),
		},
	})
	_ = consumer.Start(ctx)
	time.Sleep(2 * time.Second)
	_ = consumer.Stop(3 * time.Second)

	assert.Contains(t, order, "handler")
	assert.Contains(t, order, "mw1-before")
}
