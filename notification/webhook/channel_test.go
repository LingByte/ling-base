// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Channel
// ──────────────────────────────────────────────

func TestChannel_Name(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannel("wh-primary", d, WebhookConfig{URL: "https://example.com", Enabled: true})
	assert.Equal(t, "wh-primary", ch.Name())
}

func TestChannel_Type(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannel("wh", d, WebhookConfig{})
	assert.Equal(t, "webhook", string(ch.Type()))
}

func TestChannel_Enabled(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannel("wh", d, WebhookConfig{Enabled: true})
	assert.True(t, ch.Enabled())

	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
}

func TestChannel_Send_Success(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	ch := NewChannel("wh", d, WebhookConfig{URL: server.URL, Enabled: true})

	err := ch.Send(context.Background(), notification.NewWebhookMessage("test.event", server.URL, map[string]any{"key": "val"}))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestChannel_Send_Disabled(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannel("wh", d, WebhookConfig{URL: "https://example.com", Enabled: false})
	ch.SetEnabled(false)

	err := ch.Send(context.Background(), notification.NewWebhookMessage("test", "https://example.com", nil))
	assert.NoError(t, err) // disabled = skip
}

func TestChannel_Send_NilDispatcher(t *testing.T) {
	ch := NewChannel("wh", nil, WebhookConfig{Enabled: true})
	err := ch.Send(context.Background(), notification.NewWebhookMessage("test", "https://example.com", nil))
	assert.Error(t, err)
}

func TestChannel_Send_DefaultEvent(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	ch := NewChannel("wh", d, WebhookConfig{URL: server.URL, Enabled: true})

	// Message with no Event field.
	msg := notification.NewWebhookMessage("", server.URL, map[string]any{"x": 1})
	err := ch.Send(context.Background(), msg)
	require.NoError(t, err)

	var payload Payload
	require.NoError(t, json.Unmarshal(receivedBody, &payload))
	assert.Equal(t, "notification", payload.Event) // default event
}
