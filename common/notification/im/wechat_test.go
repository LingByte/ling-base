// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// WeChatProvider
// ──────────────────────────────────────────────

func TestWeChatProvider_Kind(t *testing.T) {
	p := NewWeChatProvider(WeChatConfig{})
	assert.Equal(t, "wechat", p.Kind())
}

func TestWeChatProvider_EmptyURLError(t *testing.T) {
	p := NewWeChatProvider(WeChatConfig{})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook url is empty")
}

func TestWeChatProvider_SendText(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	p := NewWeChatProvider(WeChatConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "Alert", Content: "hello world"})
	require.NoError(t, err)

	assert.Equal(t, "text", got["msgtype"])
	text, ok := got["text"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alert\nhello world", text["content"])
}

func TestWeChatProvider_SendEmptyTitle(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	p := NewWeChatProvider(WeChatConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Content: "just content"})
	require.NoError(t, err)

	assert.Equal(t, "text", got["msgtype"])
	text, ok := got["text"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "\njust content", text["content"])
}

func TestWeChatProvider_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer srv.Close()

	p := NewWeChatProvider(WeChatConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWeChatProvider_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewWeChatProvider(WeChatConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}
