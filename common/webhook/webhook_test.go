// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer returns an httptest.Server whose handler records the
// last request it received and can be configured to return a given
// status code (optionally failing the first n requests).
func newTestServer(t *testing.T, status int, failFirst int32) (*httptest.Server, *atomic.Value, *atomic.Int32) {
	t.Helper()
	var lastReq atomic.Value
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lastReq.Store(map[string]any{
			"headers":    r.Header,
			"body":       body,
			"method":     r.Method,
			"path":       r.URL.Path,
			"event":      r.Header.Get("X-Webhook-Event"),
			"timestamp":  r.Header.Get("X-Webhook-Timestamp"),
			"signature":  r.Header.Get("X-Webhook-Signature"),
			"contentType": r.Header.Get("Content-Type"),
		})
		n := calls.Add(1)
		if n <= failFirst {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(status)
	}))
	return srv, &lastReq, &calls
}

func TestSender_Send(t *testing.T) {
	srv, lastReq, calls := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	wh := &Webhook{URL: srv.URL, Events: []string{"order.created"}, Active: true}

	err := sender.Send(wh, "order.created", map[string]any{"order_id": "123"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())

	req := lastReq.Load().(map[string]any)
	assert.Equal(t, http.MethodPost, req["method"])
	assert.Equal(t, "application/json", req["contentType"])
	assert.Equal(t, "order.created", req["event"])
	assert.NotEmpty(t, req["timestamp"])
	assert.NotContains(t, req["headers"].(http.Header), "X-Webhook-Signature")
	assert.JSONEq(t, `{"order_id":"123"}`, string(req["body"].([]byte)))
}

func TestSender_SendWithSignature(t *testing.T) {
	srv, lastReq, calls := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	secret := "topsecret"
	wh := &Webhook{URL: srv.URL, Secret: secret, Events: []string{"user.created"}, Active: true}

	err := sender.SendWithSignature(wh, "user.created", map[string]any{"id": "u1"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())

	req := lastReq.Load().(map[string]any)
	sig := req["signature"].(string)
	assert.NotEmpty(t, sig)
	// Verify the signature matches HMAC-SHA256 of the body.
	body := req["body"].([]byte)
	assert.True(t, VerifySignature(secret, body, sig))
	// A wrong secret must not verify.
	assert.False(t, VerifySignature("wrong", body, sig))
}

func TestSender_Retry(t *testing.T) {
	// Fail the first 2 attempts, succeed on the 3rd.
	srv, _, calls := newTestServer(t, http.StatusOK, 2)
	defer srv.Close()

	sender := NewSender(
		WithHTTPClient(srv.Client()),
		WithMaxRetries(3),
		WithRetryInterval(10*time.Millisecond),
	)
	wh := &Webhook{URL: srv.URL, Active: true}

	err := sender.Send(wh, "evt", map[string]any{"k": "v"})
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestSender_RetryExhausted(t *testing.T) {
	// Always fail.
	srv, _, calls := newTestServer(t, http.StatusOK, 1000)
	defer srv.Close()

	sender := NewSender(
		WithHTTPClient(srv.Client()),
		WithMaxRetries(2),
		WithRetryInterval(10*time.Millisecond),
	)
	wh := &Webhook{URL: srv.URL, Active: true}

	err := sender.Send(wh, "evt", map[string]any{"k": "v"})
	require.Error(t, err)
	// 1 initial + 2 retries = 3 attempts.
	assert.Equal(t, int32(3), calls.Load())
}

func TestSender_EventFilter(t *testing.T) {
	srv, _, calls := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	wh := &Webhook{URL: srv.URL, Events: []string{"a", "b"}, Active: true}

	// Subscribed event is sent.
	err := sender.Send(wh, "a", "x")
	require.NoError(t, err)

	// Non-subscribed event is skipped (no request, no retry).
	err = sender.Send(wh, "c", "x")
	require.ErrorIs(t, err, ErrEventNotSubscribed)
	assert.Equal(t, int32(1), calls.Load())

	// Empty events list means subscribed to all.
	wh.Events = nil
	err = sender.Send(wh, "anything", "x")
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestSender_InactiveAndEmptyURL(t *testing.T) {
	sender := NewSender()

	err := sender.Send(&Webhook{URL: "http://x", Active: false}, "e", "p")
	assert.ErrorIs(t, err, ErrWebhookInactive)

	err = sender.Send(&Webhook{URL: "", Active: true}, "e", "p")
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestSender_NilWebhook(t *testing.T) {
	sender := NewSender()
	err := sender.Send(nil, "e", "p")
	assert.Error(t, err)
}

func TestSender_4xxNoRetry(t *testing.T) {
	srv, _, calls := newTestServer(t, http.StatusBadRequest, 0)
	defer srv.Close()

	sender := NewSender(
		WithHTTPClient(srv.Client()),
		WithMaxRetries(3),
		WithRetryInterval(10*time.Millisecond),
	)
	wh := &Webhook{URL: srv.URL, Active: true}

	err := sender.Send(wh, "e", "p")
	require.Error(t, err)
	// 4xx is not retried.
	assert.Equal(t, int32(1), calls.Load())
}

func TestVerifySignature(t *testing.T) {
	secret := "abc"
	body := []byte(`{"hello":"world"}`)
	mac := hmacSha256Hex(secret, body)
	assert.True(t, VerifySignature(secret, body, mac))
	assert.False(t, VerifySignature(secret, body, "deadbeef"))
	assert.False(t, VerifySignature("other", body, mac))
}

// hmacSha256Hex is a test helper that computes the hex HMAC independently
// of the package internals, for cross-checking VerifySignature.
func hmacSha256Hex(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

func TestWithTimeout(t *testing.T) {
	s := NewSender(WithTimeout(5 * time.Second))
	assert.Equal(t, 5*time.Second, s.timeout)
	// Non-positive durations are ignored (keeps default).
	s2 := NewSender(WithTimeout(0))
	assert.Equal(t, 10*time.Second, s2.timeout)
	s3 := NewSender(WithTimeout(-1 * time.Second))
	assert.Equal(t, 10*time.Second, s3.timeout)
}

func TestWithMaxRetries_NegativeIgnored(t *testing.T) {
	s := NewSender(WithMaxRetries(-1))
	assert.Equal(t, 3, s.maxRetries)
}

func TestWithRetryInterval_NonPositiveIgnored(t *testing.T) {
	s := NewSender(WithRetryInterval(0))
	assert.Equal(t, 1*time.Second, s.retryInterval)
}

func TestWithHTTPClient_NilIgnored(t *testing.T) {
	s := NewSender(WithHTTPClient(nil))
	assert.NotNil(t, s.client)
}

func TestSender_SendWithSignature_EmptySecret(t *testing.T) {
	srv, lastReq, _ := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	// Empty secret => no signature header, behaves like Send.
	wh := &Webhook{URL: srv.URL, Secret: "", Events: []string{"e"}, Active: true}
	err := sender.SendWithSignature(wh, "e", "x")
	require.NoError(t, err)
	req := lastReq.Load().(map[string]any)
	assert.Empty(t, req["signature"])
}

func TestSender_NilPayload(t *testing.T) {
	srv, lastReq, _ := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	wh := &Webhook{URL: srv.URL, Active: true}
	err := sender.Send(wh, "e", nil)
	require.NoError(t, err)
	req := lastReq.Load().(map[string]any)
	assert.JSONEq(t, `null`, string(req["body"].([]byte)))
}

func TestSender_MarshalError(t *testing.T) {
	srv, _, _ := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()))
	wh := &Webhook{URL: srv.URL, Active: true}
	// A channel cannot be JSON-marshaled.
	err := sender.Send(wh, "e", make(chan int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal payload")
}

func TestSender_NetworkErrorRetried(t *testing.T) {
	// Point at a closed port to trigger network errors that are retried.
	sender := NewSender(
		WithHTTPClient(&http.Client{Timeout: 200 * time.Millisecond}),
		WithMaxRetries(1),
		WithRetryInterval(5*time.Millisecond),
	)
	wh := &Webhook{URL: "http://127.0.0.1:1/hook", Active: true}
	err := sender.Send(wh, "e", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send failed after 2 attempts")
}

func TestSender_NewRequestError(t *testing.T) {
	sender := NewSender(WithMaxRetries(1), WithRetryInterval(time.Millisecond))
	// An invalid URL with a control character causes http.NewRequest to fail.
	wh := &Webhook{URL: "http://example.com/\x00", Active: true}
	err := sender.Send(wh, "e", "p")
	require.Error(t, err)
}

func TestSender_DefaultClientWhenNil(t *testing.T) {
	srv, _, calls := newTestServer(t, http.StatusOK, 0)
	defer srv.Close()

	// Construct a sender with a nil client by overriding after construction.
	sender := NewSender()
	sender.client = nil
	wh := &Webhook{URL: srv.URL, Active: true}
	err := sender.Send(wh, "e", "p")
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestSender_5xxRetried(t *testing.T) {
	srv, _, calls := newTestServer(t, http.StatusBadGateway, 1000)
	defer srv.Close()

	sender := NewSender(
		WithHTTPClient(srv.Client()),
		WithMaxRetries(2),
		WithRetryInterval(5*time.Millisecond),
	)
	wh := &Webhook{URL: srv.URL, Active: true}
	err := sender.Send(wh, "e", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
	// 1 initial + 2 retries.
	assert.Equal(t, int32(3), calls.Load())
}

func TestSender_4xxErrorMessage(t *testing.T) {
	srv, _, _ := newTestServer(t, http.StatusNotFound, 0)
	defer srv.Close()

	sender := NewSender(WithHTTPClient(srv.Client()), WithMaxRetries(3))
	wh := &Webhook{URL: srv.URL, Active: true}
	err := sender.Send(wh, "e", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client error status 404")
}

func TestRetryableError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	re := &retryableError{err: inner}
	assert.Equal(t, "inner", re.Error())
	assert.ErrorIs(t, re, inner)
}

func TestVerifySignature_CaseInsensitive(t *testing.T) {
	secret := "abc"
	body := []byte(`{"x":1}`)
	mac := hmacSha256Hex(secret, body)
	// Uppercase signature should still verify (lowercased internally).
	assert.True(t, VerifySignature(secret, body, strings.ToUpper(mac)))
}

func TestVerifySignature_EmptyInputs(t *testing.T) {
	assert.True(t, VerifySignature("", []byte(""), hmacSha256Hex("", []byte(""))))
	assert.False(t, VerifySignature("", []byte("x"), ""))
}

func TestEventSubscribed_AllEvents(t *testing.T) {
	assert.True(t, eventSubscribed(nil, "anything"))
	assert.True(t, eventSubscribed([]string{}, "anything"))
	assert.True(t, eventSubscribed([]string{"a", "b"}, "a"))
	assert.False(t, eventSubscribed([]string{"a", "b"}, "c"))
}
