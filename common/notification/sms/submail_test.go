// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Shared helpers (used by multiple provider test files)
// ──────────────────────────────────────────────

// rewriteTransport rewrites every outgoing request's scheme/host to the
// given target URL while preserving the original path and query. It is
// used to redirect providers whose endpoints are hardcoded (e.g. Submail)
// to a local httptest server.
type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(t.target)
	if err == nil {
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.Host = u.Host
	}
	rt := t.base
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}

// withRedirectedClient swaps the HTTP client used by the package-level
// defaultClient so that all requests are redirected to target. The
// returned func restores the original client.
func withRedirectedClient(target string) func() {
	orig := defaultClient.HTTPClient
	defaultClient.HTTPClient = &http.Client{
		Transport: &rewriteTransport{target: target},
		Timeout:   30 * time.Second,
	}
	return func() { defaultClient.HTTPClient = orig }
}

func phoneReq(msg Message) SendRequest {
	return SendRequest{
		To:      []PhoneNumber{{Number: "13800138000", CountryCode: 86}},
		Message: msg,
	}
}

// ──────────────────────────────────────────────
// Submail
// ──────────────────────────────────────────────

func TestSubmailProvider_Kind(t *testing.T) {
	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "test-id", "app_key": "test-key",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderSubmail, p.Kind())
}

func TestSubmailProvider_MissingConfig(t *testing.T) {
	_, err := NewSubmailProvider(ProviderConfig{"app_id": "test-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_id and app_key")
}

func TestSubmailProvider_Send_TemplateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/message/xsend.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","send_id":"send-123","msg":"ok"}`)
	}))
	defer srv.Close()
	restore := withRedirectedClient(srv.URL)
	defer restore()

	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "test-id", "app_key": "test-key",
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "project-1", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "send-123", res.MessageID)
}

func TestSubmailProvider_Send_ContentSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/message/send.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","send_id":"send-456","msg":"ok"}`)
	}))
	defer srv.Close()
	restore := withRedirectedClient(srv.URL)
	defer restore()

	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "test-id", "app_key": "test-key",
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "send-456", res.MessageID)
}

func TestSubmailProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"error","send_id":"","msg":"bad appid"}`)
	}))
	defer srv.Close()
	restore := withRedirectedClient(srv.URL)
	defer restore()

	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "test-id", "app_key": "test-key",
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "project-1"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad appid")
}

func TestSubmailProvider_RequiresTemplateOrContent(t *testing.T) {
	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "test-id", "app_key": "test-key",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{}))
	require.Error(t, err)
	// ValidateBasic fires first for empty content+template.
	assert.Contains(t, err.Error(), "content or template is required")
}

func TestSubmailProvider_Send_EmptyRecipients(t *testing.T) {
	p, err := NewSubmailProvider(ProviderConfig{"app_id": "id", "app_key": "k"})
	require.NoError(t, err)

	_, err = p.Send(context.Background(), SendRequest{
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
}

func TestSubmailProvider_Send_ContentMode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","send_id":"submail-content-1"}`)
	}))
	defer srv.Close()

	// Submail uses hardcoded endpoints, so we need to swap the HTTP client.
	original := defaultClient.HTTPClient
	defaultClient.HTTPClient = &http.Client{}
	defer func() { defaultClient.HTTPClient = original }()

	// Use a transport that redirects to our test server.
	defaultClient.HTTPClient.Transport = &rewriteTransport{target: srv.URL}

	p, err := NewSubmailProvider(ProviderConfig{
		"app_id": "id", "app_key": "k",
	})
	require.NoError(t, err)

	// Content mode (no template, no project).
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "submail-content-1", res.MessageID)
}
