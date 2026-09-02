// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// DingTalkProvider
// ──────────────────────────────────────────────

func TestDingTalkProvider_Kind(t *testing.T) {
	p := NewDingTalkProvider(DingTalkConfig{})
	assert.Equal(t, "dingtalk", p.Kind())
}

func TestDingTalkProvider_EmptyURLError(t *testing.T) {
	p := NewDingTalkProvider(DingTalkConfig{})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook url is empty")
}

func TestDingTalkProvider_SendText(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "Alert", Content: "hello world"})
	require.NoError(t, err)

	assert.Equal(t, "text", got["msgtype"])
	text, ok := got["text"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alert\nhello world", text["content"])
}

func TestDingTalkProvider_SendWithSecret(t *testing.T) {
	var got map[string]any
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	secret := "SECmy-secret"
	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL + "?access_token=test-token", Secret: secret})
	err := p.Send(context.Background(), Message{Title: "T", Content: "C"})
	require.NoError(t, err)

	// Verify the URL contains timestamp and sign parameters.
	parsed, err := url.Parse(gotURL)
	require.NoError(t, err)
	q := parsed.Query()
	assert.NotEmpty(t, q.Get("timestamp"))
	assert.NotEmpty(t, q.Get("sign"))
	assert.Equal(t, "test-token", q.Get("access_token"))

	// Verify the signature matches the documented algorithm.
	tsStr := q.Get("timestamp")
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	require.NoError(t, err)
	expected, err := dingTalkSign(ts, secret)
	require.NoError(t, err)
	assert.Equal(t, expected, q.Get("sign"))
}

func TestDingTalkProvider_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestDingTalkProvider_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}

func TestDingTalkProvider_SendWithSecret_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL + "?access_token=tk", Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
}

func TestDingTalkProvider_AutoBuildURL(t *testing.T) {
	// When WebhookURL is empty but AccessToken is set, the URL should
	// be built automatically.
	p := NewDingTalkProvider(DingTalkConfig{AccessToken: "my-token"})
	assert.Contains(t, p.cfg.WebhookURL, "access_token=my-token")
	assert.Contains(t, p.cfg.WebhookURL, "oapi.dingtalk.com")
}

// ──────────────────────────────────────────────
// dingTalkSign edge cases
// ──────────────────────────────────────────────

func TestDingTalkSign_DirectCall(t *testing.T) {
	sign, err := dingTalkSign(1700000000000, "test-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, sign)

	// Verify the sign is deterministic for the same inputs.
	sign2, err := dingTalkSign(1700000000000, "test-secret")
	require.NoError(t, err)
	assert.Equal(t, sign, sign2)

	// Different inputs produce different signs.
	sign3, err := dingTalkSign(1700000000001, "test-secret")
	require.NoError(t, err)
	assert.NotEqual(t, sign, sign3)
}

func TestDingTalkSign_EmptySecret(t *testing.T) {
	sign, err := dingTalkSign(1700000000000, "")
	require.NoError(t, err)
	assert.NotEmpty(t, sign)
}

func TestDingTalkProvider_BuildSignedURL_InvalidURL(t *testing.T) {
	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: "://[invalid", Secret: "secret"})
	_, err := p.buildSignedURL()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse webhook url")
}

func TestDingTalkProvider_SendWithSecret_InvalidURL(t *testing.T) {
	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: "://[invalid", Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse webhook url")
}

func TestDingTalkProvider_BuildSignedURL_Valid(t *testing.T) {
	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=tk", Secret: "secret"})
	signedURL, err := p.buildSignedURL()
	require.NoError(t, err)
	assert.Contains(t, signedURL, "timestamp=")
	assert.Contains(t, signedURL, "sign=")
	assert.Contains(t, signedURL, "access_token=tk")
}

func TestDingTalkProvider_SendWithSecret_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer srv.Close()

	p := NewDingTalkProvider(DingTalkConfig{WebhookURL: srv.URL + "?access_token=tk", Secret: "secret"})
	err := p.Send(context.Background(), Message{Title: "t", Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
