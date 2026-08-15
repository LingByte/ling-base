// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.URL)
	assert.True(t, cfg.DialerTimeout > 0)
	assert.True(t, cfg.ReconnectDelay > 0)
	assert.True(t, cfg.ChannelCacheSize > 0)
}

func TestNew_EmptyURL(t *testing.T) {
	_, err := New(Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestNew_Defaults(t *testing.T) {
	b, err := New(Config{URL: "amqp://localhost:5672/"})
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.True(t, b.cfg.DialerTimeout > 0)
	assert.True(t, b.cfg.ReconnectDelay > 0)
	assert.True(t, b.cfg.ChannelCacheSize > 0)
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		URL:              "amqp://localhost:5672/",
		DialerTimeout:    5_000_000_000,  // 5s in ns
		ReconnectDelay:   3_000_000_000,
		Heartbeat:        15_000_000_000,
		ChannelCacheSize: 32,
	}
	b, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 32, b.cfg.ChannelCacheSize)
}

func TestBroker_IsConnected_NotConnected(t *testing.T) {
	b, _ := New(Config{URL: "amqp://localhost:5672/"})
	assert.False(t, b.IsConnected())
}

func TestBroker_Close_NotConnected(t *testing.T) {
	b, _ := New(Config{URL: "amqp://localhost:5672/"})
	// Close should be safe even if never connected.
	assert.NoError(t, b.Close())
}

func TestBroker_Close_Idempotent(t *testing.T) {
	b, _ := New(Config{URL: "amqp://localhost:5672/"})
	assert.NoError(t, b.Close())
	// Second close should not panic.
	assert.NoError(t, b.Close())
}

func TestBroker_Connect_NotClosed(t *testing.T) {
	b, _ := New(Config{URL: "amqp://invalid:1"})
	err := b.Connect()
	assert.Error(t, err) // can't connect to invalid
}

func TestBroker_Metrics_Initial(t *testing.T) {
	b, _ := New(Config{URL: "amqp://localhost:5672/"})
	m := b.Metrics()
	assert.Equal(t, int64(0), m.Published)
	assert.Equal(t, int64(0), m.Consumed)
}

func TestFormatExpiration(t *testing.T) {
	assert.Equal(t, "", formatExpiration(0))
	assert.Equal(t, "", formatExpiration(-1))
	assert.Equal(t, "1000", formatExpiration(1_000_000_000)) // 1s = 1000ms
	assert.Equal(t, "500", formatExpiration(500_000_000))   // 0.5s = 500ms
}

func TestDialer(t *testing.T) {
	d := dialer(1_000_000_000)
	assert.NotNil(t, d)
	// We can't test the actual dial without a server, but the function
	// should be callable.
}
