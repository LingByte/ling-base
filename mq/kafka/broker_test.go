// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kafka

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
)

// Verify that Broker implements mq.Broker.
func TestBroker_ImplementsBroker(t *testing.T) {
	var _ mq.Broker = (*Broker)(nil)
}

// Verify that Producer implements mq.Producer.
func TestProducer_ImplementsProducer(t *testing.T) {
	var _ mq.Producer = (*Producer)(nil)
}

// Verify that Consumer implements mq.Consumer.
func TestConsumer_ImplementsConsumer(t *testing.T) {
	var _ mq.Consumer = (*Consumer)(nil)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.Brokers)
	assert.NotEmpty(t, cfg.Topic)
	assert.NotEmpty(t, cfg.GroupID)
	assert.True(t, cfg.DialerTimeout > 0)
	assert.True(t, cfg.CommitInterval > 0)
}

func TestNew_EmptyBrokers(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "brokers are required")
}

func TestNew_Defaults(t *testing.T) {
	b, err := New(Config{Brokers: []string{"localhost:9092"}})
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.True(t, b.cfg.DialerTimeout > 0)
	assert.True(t, b.cfg.CommitInterval > 0)
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		Brokers:        []string{"broker1:9092", "broker2:9092"},
		Topic:          "custom-topic",
		GroupID:        "custom-group",
		DialerTimeout:  5_000_000_000,
		CommitInterval: 2_000_000_000,
		EnableTLS:      true,
	}
	b, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "custom-topic", b.cfg.Topic)
	assert.Equal(t, "custom-group", b.cfg.GroupID)
	assert.Len(t, b.cfg.Brokers, 2)
}

func TestBroker_IsConnected(t *testing.T) {
	b, err := New(Config{Brokers: []string{"localhost:9092"}})
	assert.NoError(t, err)
	assert.True(t, b.IsConnected())
}

func TestBroker_IsConnected_AfterClose(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	assert.NoError(t, b.Close())
	assert.False(t, b.IsConnected())
}

func TestBroker_Close(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	assert.NoError(t, b.Close())
}

func TestBroker_Close_Idempotent(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	assert.NoError(t, b.Close())
	// Second close should not panic or error.
	assert.NoError(t, b.Close())
}

func TestBroker_DeclareQueue_NoOp(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	// DeclareQueue is a no-op for Kafka.
	assert.NoError(t, b.DeclareQueue("any-queue", mq.DefaultQueueOptions()))
}

func TestBroker_Bind_NoOp(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	assert.NoError(t, b.Bind("queue", "exchange", "key"))
}

func TestBroker_Unbind_NoOp(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	assert.NoError(t, b.Unbind("queue", "exchange", "key"))
}

func TestBroker_DeleteQueue_NoOp(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	assert.NoError(t, b.DeleteQueue("any-queue"))
}

func TestBroker_Producer_AfterClose(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	_ = b.Close()
	_, err := b.Producer("topic", mq.PublishOptions{})
	assert.Error(t, err)
}

func TestBroker_Producer_EmptyTopic(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	_, err := b.Producer("", mq.PublishOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
}

func TestBroker_Producer_Cached(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}, Topic: "test"})
	defer b.Close()
	p1, err := b.Producer("topic", mq.PublishOptions{})
	assert.NoError(t, err)
	p2, err := b.Producer("topic", mq.PublishOptions{})
	assert.NoError(t, err)
	assert.Same(t, p1, p2) // cached
}

func TestBroker_Consumer_NoHandler(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	_, err := b.Consumer("topic", mq.ConsumeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
}

func TestBroker_Consumer_EmptyTopic(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	_, err := b.Consumer("", mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error { return nil },
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
}

func TestBroker_Metrics_Initial(t *testing.T) {
	b, _ := New(Config{Brokers: []string{"localhost:9092"}})
	defer b.Close()
	m := b.Metrics()
	assert.Equal(t, int64(0), m.Published)
	assert.Equal(t, int64(0), m.Consumed)
}

func TestToInt(t *testing.T) {
	assert.Equal(t, 5, mustToInt(toInt(5)))
	assert.Equal(t, 5, mustToInt(toInt(int32(5))))
	assert.Equal(t, 5, mustToInt(toInt(int64(5))))
	assert.Equal(t, 5, mustToInt(toInt(float64(5))))
	_, ok := toInt("not-a-number")
	assert.False(t, ok)
}

func mustToInt(v int, ok bool) int {
	if !ok {
		panic("toInt returned !ok")
	}
	return v
}
