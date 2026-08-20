// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Helpers (shared across test files in package webhook)
// ──────────────────────────────────────────────

// mockDeliveryStore is an in-memory DeliveryStore used by tests. It is
// safe for concurrent use and supports injecting an error from
// GetPendingRetries via pendingErr.
type mockDeliveryStore struct {
	mu         sync.Mutex
	created    []DeliveryLog
	updated    []DeliveryLog
	pending    []DeliveryLog
	pendingErr error
}

func (m *mockDeliveryStore) CreateDelivery(log DeliveryLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, log)
	return nil
}

func (m *mockDeliveryStore) UpdateDelivery(log DeliveryLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updated = append(m.updated, log)
	return nil
}

func (m *mockDeliveryStore) GetPendingRetries(limit int) ([]DeliveryLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pendingErr != nil {
		return nil, m.pendingErr
	}
	if limit >= len(m.pending) {
		return m.pending, nil
	}
	return m.pending[:limit], nil
}

func (m *mockDeliveryStore) GetDelivery(id string) (*DeliveryLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, l := range m.created {
		if l.ID == id {
			return &l, nil
		}
	}
	return nil, nil
}

// ──────────────────────────────────────────────
// SignPayload / VerifySignature
// ──────────────────────────────────────────────

func TestSignPayload(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	secret := "mysecret"

	sig := SignPayload(payload, secret)

	// Verify it matches a manually computed HMAC.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expected, sig)
	assert.NotEmpty(t, sig)
}

func TestSignPayload_EmptySecret(t *testing.T) {
	sig := SignPayload([]byte("test"), "")
	// With empty key, HMAC still produces a value.
	assert.NotEmpty(t, sig)
}

func TestVerifySignature(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	secret := "mysecret"
	sig := SignPayload(payload, secret)

	assert.True(t, VerifySignature(payload, secret, sig))
	assert.False(t, VerifySignature(payload, "wrong", sig))
	assert.False(t, VerifySignature(payload, secret, "wrong"))
}

// ──────────────────────────────────────────────
// ValidateURL
// ──────────────────────────────────────────────

func TestValidateURL_Valid(t *testing.T) {
	tests := []struct {
		url string
		ok  bool
	}{
		{"https://example.com/webhook", true},
		{"http://example.com/hook", true},
		{"https://api.example.com/v1/webhook", true},
		{"https://example.com", true},
	}
	for _, tt := range tests {
		err := ValidateURL(tt.url, false)
		if tt.ok {
			assert.NoError(t, err, "expected OK for %s", tt.url)
		} else {
			assert.Error(t, err, "expected error for %s", tt.url)
		}
	}
}

func TestValidateURL_InvalidScheme(t *testing.T) {
	err := ValidateURL("ftp://example.com/hook", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestValidateURL_Localhost(t *testing.T) {
	urls := []string{
		"http://localhost:8080/hook",
		"http://127.0.0.1:8080/hook",
		"http://0.0.0.0:8080/hook",
		"http://[::1]:8080/hook",
	}
	for _, u := range urls {
		err := ValidateURL(u, false)
		assert.Error(t, err, "expected error for %s", u)
	}
}

func TestValidateURL_PrivateIPs(t *testing.T) {
	urls := []string{
		"http://10.0.0.1/hook",
		"http://192.168.1.1/hook",
		"http://172.16.0.1/hook",
	}
	for _, u := range urls {
		err := ValidateURL(u, false)
		assert.Error(t, err, "expected error for %s", u)
	}
}

func TestValidateURL_AllowPrivate(t *testing.T) {
	err := ValidateURL("http://localhost:8080/hook", true)
	assert.NoError(t, err)

	err = ValidateURL("http://10.0.0.1/hook", true)
	assert.NoError(t, err)
}

func TestValidateURL_EmptyHost(t *testing.T) {
	err := ValidateURL("http:///hook", false)
	assert.Error(t, err)
}

func TestValidateURL_InvalidURL(t *testing.T) {
	err := ValidateURL("://broken", false)
	assert.Error(t, err)
}

func TestValidateURL_AllowPrivateLocalhost(t *testing.T) {
	err := ValidateURL("http://localhost:8080", true)
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// Dispatch
// ──────────────────────────────────────────────

func TestDispatch_Success(t *testing.T) {
	var receivedBody []byte
	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Secret:  "testsecret",
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "user.created", map[string]any{"user_id": "123"})
	require.NoError(t, err)

	// Verify body.
	var payload Payload
	require.NoError(t, json.Unmarshal(receivedBody, &payload))
	assert.Equal(t, "user.created", payload.Event)
	assert.Equal(t, "123", payload.Data["user_id"])

	// Verify signature.
	expectedSig := SignPayload(receivedBody, "testsecret")
	assert.Equal(t, expectedSig, receivedSig)
}

func TestDispatch_NoSecret(t *testing.T) {
	var receivedSig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	require.NoError(t, err)
	assert.Empty(t, receivedSig)
}

func TestDispatch_Disabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: false,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	assert.NoError(t, err) // skip is not an error
}

func TestDispatch_DisabledConfig(t *testing.T) {
	d := NewDispatcher()
	err := d.Dispatch(context.Background(), WebhookConfig{Enabled: false, URL: "http://example.com"}, "event", nil)
	require.NoError(t, err)
}

func TestDispatch_EventNotSubscribed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
		Events:  []string{"user.created"},
	}

	err := d.Dispatch(context.Background(), cfg, "user.deleted", nil)
	assert.NoError(t, err) // skip is not an error
}

func TestDispatch_EventNotMatched(t *testing.T) {
	d := NewDispatcher()
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     "http://example.com",
		Events:  []string{"user.created"},
	}, "user.deleted", nil)
	require.NoError(t, err)
}

func TestDispatch_EmptyEventsSendsAll(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
		Events:  nil, // empty = all events
	}

	err := d.Dispatch(context.Background(), cfg, "any.event", nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDispatch_AllEventsMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     srv.URL,
	}, "any.event", map[string]any{"key": "value"})
	require.NoError(t, err)
}

func TestDispatch_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestDispatch_InvalidURL(t *testing.T) {
	d := NewDispatcher()
	cfg := WebhookConfig{
		URL:     "ftp://bad.example.com",
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestDispatch_PrivateURLRejected(t *testing.T) {
	d := NewDispatcher()
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     "http://127.0.0.1:9999",
	}, "event", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestDispatch_WithStore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	require.NoError(t, err)

	assert.Len(t, store.created, 1)
	assert.Len(t, store.updated, 1)
	assert.Equal(t, StatusSent, store.updated[0].Status)
}

func TestDispatch_SuccessWithStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true))
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     srv.URL,
	}, "test.event", map[string]any{"key": "value"})
	require.NoError(t, err)
}

func TestDispatch_FailureWithStore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true), WithMaxAttempts(5))
	cfg := WebhookConfig{
		URL:     server.URL,
		Enabled: true,
	}

	err := d.Dispatch(context.Background(), cfg, "test", nil)
	assert.Error(t, err)

	// Should have created and then updated.
	assert.Len(t, store.created, 1)
	assert.Len(t, store.updated, 1)
	assert.Equal(t, StatusFailed, store.updated[0].Status)
	assert.Equal(t, 1, store.updated[0].RetryCount)
}

func TestDispatch_Failure_Logged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true))
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     srv.URL,
	}, "test.event", map[string]any{"key": "value"})
	require.Error(t, err)
	// The delivery should have been logged (created as pending, then updated).
	require.Len(t, store.created, 1)
	assert.Equal(t, StatusPending, store.created[0].Status)
	require.Len(t, store.updated, 1)
	// Default maxAttempts is 3; first failure with RetryCount=1 < 3, so status is "failed".
	assert.Equal(t, StatusFailed, store.updated[0].Status)
}

func TestDispatch_WithSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Webhook-Signature")
		assert.NotEmpty(t, sig)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher(WithAllowPrivateURLs(true))
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled: true,
		URL:     srv.URL,
		Secret:  "test-secret",
	}, "test.event", map[string]any{"key": "value"})
	require.NoError(t, err)
}

func TestDispatch_CustomMaxAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true))
	err := d.Dispatch(context.Background(), WebhookConfig{
		Enabled:     true,
		URL:         srv.URL,
		MaxAttempts: 1,
	}, "test.event", map[string]any{"key": "value"})
	require.Error(t, err)
	// The created record is pending; the updated record has the final status.
	require.Len(t, store.updated, 1)
	// With MaxAttempts=1 and first failure (RetryCount=1), should go to DLQ.
	assert.Equal(t, StatusDLQ, store.updated[0].Status)
}

// ──────────────────────────────────────────────
// ProcessRetries
// ──────────────────────────────────────────────

func TestProcessRetries_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload, _ := json.Marshal(Payload{Event: "test"})
	store := &mockDeliveryStore{
		pending: []DeliveryLog{
			{
				ID:      "retry-1",
				URL:     server.URL,
				Event:   "test",
				Payload: payload,
				Status:  StatusPending,
			},
		},
	}

	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true))
	count := d.ProcessRetries(context.Background())

	assert.Equal(t, 1, count)
	assert.Len(t, store.updated, 1)
	assert.Equal(t, StatusSent, store.updated[0].Status)
}

func TestProcessRetries_NoPending(t *testing.T) {
	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store))
	count := d.ProcessRetries(context.Background())
	assert.Equal(t, 0, count)
}

func TestProcessRetries_EmptyPending(t *testing.T) {
	store := &mockDeliveryStore{}
	d := NewDispatcher(WithStore(store))
	n := d.ProcessRetries(context.Background())
	assert.Equal(t, 0, n)
}

func TestProcessRetries_StoreError(t *testing.T) {
	store := &mockDeliveryStore{pendingErr: fmt.Errorf("store error")}
	d := NewDispatcher(WithStore(store))
	n := d.ProcessRetries(context.Background())
	assert.Equal(t, 0, n)
}

func TestProcessRetries_ExhaustedToDLQ(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	payload, _ := json.Marshal(Payload{Event: "test"})
	store := &mockDeliveryStore{
		pending: []DeliveryLog{
			{
				ID:          "retry-1",
				URL:         server.URL,
				Event:       "test",
				Payload:     payload,
				Status:      StatusPending,
				RetryCount:  4, // one more = 5 = max
				NextRetryAt: time.Now().Add(-time.Minute),
			},
		},
	}

	d := NewDispatcher(WithStore(store), WithAllowPrivateURLs(true), WithMaxAttempts(5))
	count := d.ProcessRetries(context.Background())

	assert.Equal(t, 1, count)
	assert.Equal(t, StatusDLQ, store.updated[0].Status)
	assert.Equal(t, 5, store.updated[0].RetryCount)
}

func TestProcessRetries_SuccessThenDLQ(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	store.pending = []DeliveryLog{
		{
			ID:         "retry-1",
			URL:        srv.URL,
			Event:      "test.event",
			Payload:    []byte(`{"event":"test.event"}`),
			Status:     StatusFailed,
			RetryCount: 0,
			CreatedAt:  time.Now(),
		},
	}
	d := NewDispatcher(WithStore(store), WithMaxAttempts(3))
	n := d.ProcessRetries(context.Background())
	assert.Equal(t, 1, n)
	require.Len(t, store.updated, 1)
	assert.Equal(t, StatusSent, store.updated[0].Status)
}

func TestProcessRetries_FailureThenDLQ(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	store.pending = []DeliveryLog{
		{
			ID:         "retry-dlq",
			URL:        srv.URL,
			Event:      "test.event",
			Payload:    []byte(`{"event":"test.event"}`),
			Status:     StatusFailed,
			RetryCount: 2, // Already retried twice, next is 3rd which >= maxAttempts(3)
			CreatedAt:  time.Now(),
		},
	}
	d := NewDispatcher(WithStore(store), WithMaxAttempts(3))
	n := d.ProcessRetries(context.Background())
	assert.Equal(t, 1, n)
	require.Len(t, store.updated, 1)
	assert.Equal(t, StatusDLQ, store.updated[0].Status)
}

func TestProcessRetries_FailureThenRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockDeliveryStore{}
	store.pending = []DeliveryLog{
		{
			ID:         "retry-fail",
			URL:        srv.URL,
			Event:      "test.event",
			Payload:    []byte(`{"event":"test.event"}`),
			Status:     StatusFailed,
			RetryCount: 0,
			CreatedAt:  time.Now(),
		},
	}
	d := NewDispatcher(WithStore(store), WithMaxAttempts(3))
	n := d.ProcessRetries(context.Background())
	assert.Equal(t, 1, n)
	require.Len(t, store.updated, 1)
	assert.Equal(t, StatusFailed, store.updated[0].Status)
	assert.Equal(t, 1, store.updated[0].RetryCount)
	assert.True(t, store.updated[0].NextRetryAt.After(time.Now()))
}

// ──────────────────────────────────────────────
// doSend edge cases
// ──────────────────────────────────────────────

func TestDoSend_InvalidURL(t *testing.T) {
	d := NewDispatcher()
	err := d.doSend(context.Background(), WebhookConfig{URL: "http://[::1]:bad"}, []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create request")
}

func TestDoSend_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	d := NewDispatcher()
	err := d.doSend(context.Background(), WebhookConfig{URL: srv.URL}, []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send")
}

func TestDoSend_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	d := NewDispatcher()
	err := d.doSend(context.Background(), WebhookConfig{URL: srv.URL}, []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestDoSend_WithSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Webhook-Signature")
		assert.NotEmpty(t, sig)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher()
	err := d.doSend(context.Background(), WebhookConfig{URL: srv.URL, Secret: "test-secret"}, []byte(`{"event":"test"}`))
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// checkHostNotPrivate edge cases
// ──────────────────────────────────────────────

func TestCheckHostNotPrivate_Localhost(t *testing.T) {
	err := checkHostNotPrivate("localhost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")
}

func TestCheckHostNotPrivate_127001(t *testing.T) {
	err := checkHostNotPrivate("127.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")
}

func TestCheckHostNotPrivate_0000(t *testing.T) {
	err := checkHostNotPrivate("0.0.0.0")
	require.Error(t, err)
}

func TestCheckHostNotPrivate_IPv6Loopback(t *testing.T) {
	err := checkHostNotPrivate("::1")
	require.Error(t, err)
}

func TestCheckHostNotPrivate_PrivateIP(t *testing.T) {
	err := checkHostNotPrivate("10.0.0.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

func TestCheckHostNotPrivate_172Private(t *testing.T) {
	err := checkHostNotPrivate("172.16.0.1")
	require.Error(t, err)
}

func TestCheckHostNotPrivate_192Private(t *testing.T) {
	err := checkHostNotPrivate("192.168.1.1")
	require.Error(t, err)
}

func TestCheckHostNotPrivate_LinkLocal(t *testing.T) {
	err := checkHostNotPrivate("169.254.1.1")
	require.Error(t, err)
}

func TestCheckHostNotPrivate_PublicDomain(t *testing.T) {
	err := checkHostNotPrivate("example.com")
	require.NoError(t, err)
}

func TestCheckHostNotPrivate_PublicIP(t *testing.T) {
	err := checkHostNotPrivate("8.8.8.8")
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// containsEvent edge cases
// ──────────────────────────────────────────────

func TestContainsEvent_Empty(t *testing.T) {
	assert.False(t, containsEvent(nil, "test"))
	assert.False(t, containsEvent([]string{}, "test"))
}

func TestContainsEvent_CaseInsensitive(t *testing.T) {
	assert.True(t, containsEvent([]string{"User.Created"}, "user.created"))
	assert.True(t, containsEvent([]string{"user.created"}, "USER.CREATED"))
}

func TestContainsEvent_NotFound(t *testing.T) {
	assert.False(t, containsEvent([]string{"a", "b"}, "c"))
}

// ──────────────────────────────────────────────
// backoffDuration
// ──────────────────────────────────────────────

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		retry int
		min   time.Duration
		max   time.Duration
	}{
		{0, 2 * time.Second, 2 * time.Second},
		{1, 2 * time.Second, 2 * time.Second},
		{2, 4 * time.Second, 4 * time.Second},
		{3, 8 * time.Second, 8 * time.Second},
		{4, 16 * time.Second, 16 * time.Second},
		{5, 32 * time.Second, 32 * time.Second},
		{10, 32 * time.Second, 32 * time.Second}, // capped
	}
	for _, tt := range tests {
		d := backoffDuration(tt.retry)
		assert.GreaterOrEqual(t, d, tt.min, "retry=%d", tt.retry)
		assert.LessOrEqual(t, d, tt.max, "retry=%d", tt.retry)
	}
}

func TestBackoffDuration_Zero(t *testing.T) {
	d := backoffDuration(0)
	assert.Equal(t, 2*time.Second, d)
}

func TestBackoffDuration_Negative(t *testing.T) {
	d := backoffDuration(-1)
	assert.Equal(t, 2*time.Second, d)
}

func TestBackoffDuration_Growth(t *testing.T) {
	d1 := backoffDuration(1)
	d2 := backoffDuration(2)
	d3 := backoffDuration(3)
	assert.Equal(t, 2*time.Second, d1)
	assert.Equal(t, 4*time.Second, d2)
	assert.Equal(t, 8*time.Second, d3)
}

// ──────────────────────────────────────────────
// WithHTTPClient
// ──────────────────────────────────────────────

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	d := NewDispatcher(WithHTTPClient(custom))
	assert.Same(t, custom, d.httpClient)
}

func TestWithHTTPClient_Nil(t *testing.T) {
	d := NewDispatcher(WithHTTPClient(nil))
	// Nil client should not override the default.
	assert.NotNil(t, d.httpClient)
}
