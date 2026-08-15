// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rocketmq

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rocketmqNameServer returns the name-server address from the
// ROCKETMQ_NAMESERVER env var, or empty if not set.
func rocketmqNameServer() []string {
	if ns := os.Getenv("ROCKETMQ_NAMESERVER"); ns != "" {
		return []string{ns}
	}
	return nil
}

// TestIntegrationBroker is a full integration test that requires a
// running RocketMQ name server. Skipped if ROCKETMQ_NAMESERVER is not
// set.
func TestIntegrationBroker(t *testing.T) {
	ns := rocketmqNameServer()
	if len(ns) == 0 {
		t.Skip("ROCKETMQ_NAMESERVER not set, skipping integration test")
	}

	b, err := New(Config{
		NameServer: ns,
		GroupName:  "ling-base-test",
		RetryCount: 2,
	})
	require.NoError(t, err)
	defer b.Close()

	require.NoError(t, b.Connect())
	assert.True(t, b.IsConnected())

	topic := fmt.Sprintf("test.lingbase.%d", time.Now().UnixNano())

	// Publish.
	producer, err := b.Producer(topic, mq.PublishOptions{Persistent: true})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = producer.Publish(ctx, &mq.Message{
		Body:        []byte(`{"hello":"world"}`),
		ContentType: "application/json",
		RoutingKey:  "hello",
	})
	assert.NoError(t, err)

	// Consume.
	received := make(chan []byte, 1)
	consumer, err := b.Consumer(topic, mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error {
			select {
			case received <- d.Body():
			default:
			}
			return d.Ack()
		},
		Concurrency: 1,
	})
	require.NoError(t, err)

	require.NoError(t, consumer.Start(ctx))
	assert.True(t, consumer.IsRunning())

	select {
	case body := <-received:
		assert.Contains(t, string(body), "hello")
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	// Stop.
	assert.NoError(t, consumer.Stop(5 * time.Second))
	assert.False(t, consumer.IsRunning())
}
