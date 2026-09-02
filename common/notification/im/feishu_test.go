// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// FeishuProvider
// ──────────────────────────────────────────────

func TestFeishuProvider_Kind(t *testing.T) {
	p := NewFeishuProvider(FeishuConfig{})
	assert.Equal(t, "feishu", p.Kind())
}

func TestFeishuProvider_EmptyURLError(t *testing.T) {
	p := NewFeishuProvider(FeishuConfig{})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook url is empty")
}

func TestFeishuProvider_SendText(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "Alert", Content: "hello world"})
	require.NoError(t, err)

	assert.Equal(t, "text", got["msg_type"])
	content, ok := got["content"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alert\nhello world", content["text"])
	// no signature fields when secret unset
	_, hasTS := got["timestamp"]
	_, hasSign := got["sign"]
	assert.False(t, hasTS)
	assert.False(t, hasSign)
}

func TestFeishuProvider_SendWithSecret(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"StatusCode":0}`))
	}))
	defer srv.Close()

	secret := "my-secret"
	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL, Secret: secret})
	err := p.Send(context.Background(), Message{Title: "T", Content: "C"})
	require.NoError(t, err)

	tsStr, ok := got["timestamp"].(string)
	require.True(t, ok, "timestamp present")
	assert.NotEmpty(t, tsStr)

	sign, ok := got["sign"].(string)
	require.True(t, ok, "sign present")
	assert.NotEmpty(t, sign)

	// verify the signature matches the documented algorithm
	expected, err := feishuSign(parseInt64(t, tsStr), secret)
	require.NoError(t, err)
	assert.Equal(t, expected, sign)
}

func TestFeishuProvider_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestFeishuProvider_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}

func TestFeishuProvider_SendWithSecret_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL, Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}

// ──────────────────────────────────────────────
// feishuSign edge cases
// ──────────────────────────────────────────────

func TestFeishuSign_DirectCall(t *testing.T) {
	sign, err := feishuSign(1700000000, "test-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, sign)

	// Verify the sign is deterministic for the same inputs.
	sign2, err := feishuSign(1700000000, "test-secret")
	require.NoError(t, err)
	assert.Equal(t, sign, sign2)

	// Different inputs produce different signs.
	sign3, err := feishuSign(1700000001, "test-secret")
	require.NoError(t, err)
	assert.NotEqual(t, sign, sign3)
}

func TestFeishuSign_EmptySecret(t *testing.T) {
	sign, err := feishuSign(1700000000, "")
	require.NoError(t, err)
	assert.NotEmpty(t, sign)
}

func TestFeishuSign_DifferentSecretsProduceDifferentSigns(t *testing.T) {
	sign1, _ := feishuSign(1700000000, "secret1")
	sign2, _ := feishuSign(1700000000, "secret2")
	assert.NotEqual(t, sign1, sign2)
}

func TestFeishuProvider_SendWithSecret_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`unavailable`))
	}))
	defer srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL, Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestFeishuProvider_SendWithSecret_VerifyPayload(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"StatusCode":0}`))
	}))
	defer srv.Close()

	p := NewFeishuProvider(FeishuConfig{WebhookURL: srv.URL, Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "T", Content: "C"})
	require.NoError(t, err)

	// Verify payload has both timestamp and sign.
	assert.Contains(t, got, "timestamp")
	assert.Contains(t, got, "sign")
	assert.Equal(t, "text", got["msg_type"])
}

// ──────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────

func parseInt64(t *testing.T, s string) int64 {
	t.Helper()
	v, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err)
	return v
}
