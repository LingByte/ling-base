// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rocketmq

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/mq"
	"github.com/stretchr/testify/assert"
)

// Interface compliance checks (compile-time).
var (
	_ mq.Broker   = (*Broker)(nil)
	_ mq.Producer = (*Producer)(nil)
	_ mq.Consumer = (*Consumer)(nil)
	_ mq.Delivery = (*Delivery)(nil)
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.NameServer)
	assert.NotEmpty(t, cfg.GroupName)
	assert.True(t, cfg.RetryCount > 0)
}

func TestNew_EmptyNameServer(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NameServer is required")
}

func TestNew_Defaults(t *testing.T) {
	b, err := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.Equal(t, "DEFAULT_PRODUCER", b.cfg.GroupName)
	assert.Equal(t, 2, b.cfg.RetryCount)
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		NameServer:    []string{"127.0.0.1:9876"},
		GroupName:     "my-group",
		InstanceName:  "my-instance",
		RetryCount:    5,
		Credentials:   Credentials{AccessKey: "ak", SecretKey: "sk"},
	}
	b, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "my-group", b.cfg.GroupName)
	assert.Equal(t, 5, b.cfg.RetryCount)
	assert.Equal(t, "ak", b.cfg.Credentials.AccessKey)
}

func TestBroker_IsConnected(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	assert.False(t, b.IsConnected())
}

func TestBroker_IsConnected_AfterConnect(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	assert.NoError(t, b.Connect())
	assert.True(t, b.IsConnected())
}

func TestBroker_Close(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	assert.NoError(t, b.Close())
	assert.False(t, b.IsConnected())
}

func TestBroker_Close_Idempotent(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	assert.NoError(t, b.Close())
	// Second close should not panic.
	assert.NoError(t, b.Close())
}

func TestBroker_Close_NotConnected(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	// Close should be safe even if never connected.
	assert.NoError(t, b.Close())
}

func TestBroker_Connect_AfterClose(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	assert.NoError(t, b.Close())
	err := b.Connect()
	assert.Error(t, err)
	assert.ErrorIs(t, err, mq.ErrClosed)
}

func TestBroker_Metrics_Initial(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	m := b.Metrics()
	assert.Equal(t, int64(0), m.Published)
	assert.Equal(t, int64(0), m.Consumed)
}

func TestBroker_Producer_NotConnected(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_, err := b.Producer("topic", mq.PublishOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, mq.ErrNotConnected)
}

func TestBroker_Producer_EmptyTopic(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()
	_, err := b.Producer("", mq.PublishOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
}

func TestBroker_Consumer_NoHandler(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()
	_, err := b.Consumer("topic", mq.ConsumeOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, mq.ErrNoHandler)
}

func TestBroker_Consumer_EmptyTopic(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()
	_, err := b.Consumer("", mq.ConsumeOptions{
		Handler: func(ctx context.Context, d mq.Delivery) error { return nil },
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is required")
}

func TestBroker_DeclareExchange(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()

	assert.NoError(t, b.DeclareExchange("test.topic", mq.DefaultExchangeOptions()))
	assert.NoError(t, b.DeleteExchange("test.topic"))
}

func TestBroker_DeclareExchange_EmptyName(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()

	err := b.DeclareExchange("", mq.DefaultExchangeOptions())
	assert.Error(t, err)
}

func TestBroker_TopologyNoOps(t *testing.T) {
	b, _ := New(Config{NameServer: []string{"127.0.0.1:9876"}})
	_ = b.Connect()
	defer b.Close()

	assert.NoError(t, b.DeclareQueue("q", mq.DefaultQueueOptions()))
	assert.NoError(t, b.Bind("q", "ex", "rk"))
	assert.NoError(t, b.Unbind("q", "ex", "rk"))
	assert.NoError(t, b.DeleteQueue("q"))
}

func TestCredentials_IsEmpty(t *testing.T) {
	assert.True(t, Credentials{}.isEmpty())
	assert.True(t, Credentials{AccessKey: "ak"}.isEmpty())
	assert.False(t, Credentials{AccessKey: "ak", SecretKey: "sk"}.isEmpty())
}

func TestToHeaderString(t *testing.T) {
	assert.Equal(t, "str", toHeaderString("str"))
	assert.Equal(t, "bytes", toHeaderString([]byte("bytes")))
	assert.Equal(t, "42", toHeaderString(42))
	assert.Equal(t, "100", toHeaderString(int64(100)))
	assert.Equal(t, "true", toHeaderString(true))
	assert.Equal(t, "", toHeaderString(1.5))
}
