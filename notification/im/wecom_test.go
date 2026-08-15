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
// WeComProvider
// ──────────────────────────────────────────────

func TestWeComProvider_Kind(t *testing.T) {
	p := NewWeComProvider(WeComConfig{})
	assert.Equal(t, "wecom", p.Kind())
}

func TestWeComProvider_EmptyURLError(t *testing.T) {
	p := NewWeComProvider(WeComConfig{})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook url is empty")
}

func TestWeComProvider_SendText(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	p := NewWeComProvider(WeComConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "Alert", Content: "plain text body"})
	require.NoError(t, err)

	assert.Equal(t, "text", got["msgtype"])
	text, ok := got["text"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alert\nplain text body", text["content"])
}

func TestWeComProvider_SendMarkdown(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	p := NewWeComProvider(WeComConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "Alert", Content: "# Heading\n\n**bold** text"})
	require.NoError(t, err)

	assert.Equal(t, "markdown", got["msgtype"])
	md, ok := got["markdown"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, md["content"], "# Heading")
}

func TestWeComProvider_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer srv.Close()

	p := NewWeComProvider(WeComConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWeComProvider_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewWeComProvider(WeComConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}

func TestWeComProvider_SendMarkdownVariations(t *testing.T) {
	tests := []struct {
		name    string
		content string
		isMD    bool
	}{
		{"heading", "# Hello", true},
		{"bold", "text **bold**", true},
		{"code", "text `code`", true},
		{"bullet", "- item", true},
		{"asterisk", "* item", true},
		{"numbered", "1. item", true},
		{"image", "![alt](url)", true},
		{"link", "[text](url)", true},
		{"plain", "just plain text", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isMD, isMarkdown(tt.content))
		})
	}
}

// ──────────────────────────────────────────────
// isMarkdown
// ──────────────────────────────────────────────

func TestIsMarkdown(t *testing.T) {
	assert.True(t, isMarkdown("# heading"))
	assert.True(t, isMarkdown("text with **bold**"))
	assert.True(t, isMarkdown("text with `code`"))
	assert.True(t, isMarkdown("- list item"))
	assert.False(t, isMarkdown("plain text only"))
	assert.False(t, isMarkdown(""))
}

// ──────────────────────────────────────────────
// postJSON edge cases
// ──────────────────────────────────────────────

func TestPostJSON_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // Close immediately to force connection error

	err := postJSON(context.Background(), srv.URL, map[string]any{"msg": "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send request")
}

func TestPostJSON_InvalidURL(t *testing.T) {
	// Use an invalid URL that will cause http.NewRequest to fail.
	err := postJSON(context.Background(), "http://[::1]:namedport", map[string]any{"msg": "test"})
	require.Error(t, err)
	// Could be either "new request" or "send request" depending on the Go version
	assert.NotNil(t, err)
}

func TestPostJSON_MarshalError(t *testing.T) {
	// A channel cannot be marshaled to JSON.
	err := postJSON(context.Background(), "http://localhost:9999", make(chan int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal payload")
}
