// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kafka

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kafkaBrokers returns the Kafka brokers from the KAFKA_BROKERS env var
// or empty if not set.
func kafkaBrokers() []string {
	if b := os.Getenv("KAFKA_BROKERS"); b != "" {
		return []string{b}
	}
	return nil
}

// hasKafka checks if a Kafka broker is reachable.
func hasKafka() bool {
	brokers := kafkaBrokers()
	if len(brokers) == 0 {
		return false
	}
	b, err := New(Config{
		Brokers:       brokers,
		DialerTimeout: 2 * time.Second,
	})
	if err != nil {
		return false
	}
	defer b.Close()
	return b.Connect() == nil
}

// TestIntegrationBroker is a full integration test that requires a
// running Kafka instance. Skipped if no broker is available.
func TestIntegrationBroker(t *testing.T) {
	if !hasKafka() {
		t.Skip("Kafka not available (set KAFKA_BROKERS env var), skipping integration test")
	}

	brokers := kafkaBrokers()
	b, err := New(Config{
		Brokers:        brokers,
		Topic:          "ling-base-test",
		GroupID:        "ling-base-test-group",
		DialerTimeout:  5 * time.Second,
		CommitInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer b.Close()

	require.NoError(t, b.Connect())
	assert.True(t, b.IsConnected())

	topic := "ling-base-integration-test"

	// Publish a message.
	producer, err := b.Producer(topic, mq.PublishOptions{Persistent: true})
	require.NoError(t, err)

	ctx := context.Background()
	err = producer.Publish(ctx, &mq.Message{
		ID:          "test-msg-1",
		Body:        []byte(`{"hello":"kafka"}`),
		ContentType: "application/json",
		Headers:     map[string]any{"x-test": "integration"},
	})
	assert.NoError(t, err)

	// Consume the message.
	received := make(chan []byte, 1)
	consumer, err := b.Consumer(topic, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			received <- d.Body()
			return d.Ack()
		},
		Concurrency: 1,
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Start(ctx))
	assert.True(t, consumer.IsRunning())

	select {
	case body := <-received:
		assert.Contains(t, string(body), "kafka")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Stop.
	assert.NoError(t, consumer.Stop(5*time.Second))
	assert.False(t, consumer.IsRunning())
}

// TestIntegrationBroker_AutoAck tests auto-ack consumption.
func TestIntegrationBroker_AutoAck(t *testing.T) {
	if !hasKafka() {
		t.Skip("Kafka not available (set KAFKA_BROKERS env var), skipping integration test")
	}

	brokers := kafkaBrokers()
	b, err := New(Config{
		Brokers:        brokers,
		Topic:          "ling-base-test",
		GroupID:        "ling-base-test-group-autoack",
		DialerTimeout:  5 * time.Second,
		CommitInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	defer b.Close()

	topic := "ling-base-integration-test-autoack"

	producer, err := b.Producer(topic, mq.PublishOptions{})
	require.NoError(t, err)

	ctx := context.Background()
	_ = producer.Publish(ctx, &mq.Message{
		ID:   "autoack-msg",
		Body: []byte("autoack-payload"),
	})

	received := make(chan []byte, 1)
	consumer, err := b.Consumer(topic, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			received <- d.Body()
			return nil // return nil, auto-ack handles commit
		},
		AutoAck:     true,
		Concurrency: 1,
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Start(ctx))

	select {
	case body := <-received:
		assert.Equal(t, "autoack-payload", string(body))
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	_ = consumer.Stop(5 * time.Second)
}
