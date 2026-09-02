// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("secret", "sitekey")
	assert.NotNil(t, c)
	assert.Equal(t, "secret", c.secretKey)
	assert.Equal(t, "sitekey", c.siteKey)
	assert.Equal(t, TurnstileEndpoint, c.endpoint)
	assert.NotNil(t, c.httpClient)
}

func TestWithHTTPClient(t *testing.T) {
	c := NewClient("s", "k")
	hc := &http.Client{}
	c.WithHTTPClient(hc)
	assert.Equal(t, hc, c.httpClient)
}

func TestWithEndpoint(t *testing.T) {
	c := NewClient("s", "k")
	c.WithEndpoint("https://custom.example.com/verify")
	assert.Equal(t, "https://custom.example.com/verify", c.endpoint)
}

func TestIsTokenValid(t *testing.T) {
	assert.False(t, IsTokenValid(""))
	assert.False(t, IsTokenValid("short"))
	assert.False(t, IsTokenValid("12345678")) // 8 chars < 10
	assert.True(t, IsTokenValid("0123456789")) // 10 chars
	assert.True(t, IsTokenValid(strings.Repeat("a", 100)))
	assert.False(t, IsTokenValid(strings.Repeat("a", 4097))) // > 4096
	assert.True(t, IsTokenValid(strings.Repeat("a", 4096)))  // exactly 4096
}

func TestVerify_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "secret", r.Form.Get("secret"))
		assert.Equal(t, "test-token-1234567890", r.Form.Get("response"))
		assert.Equal(t, "1.2.3.4", r.Form.Get("remoteip"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"success":      true,
			"action":       "login",
			"challenge_ts": "2026-01-01T00:00:00Z",
			"hostname":     "example.com",
			"metadata":     map[string]any{"result": "passed"},
		}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	resp, err := c.Verify(context.Background(), "test-token-1234567890", "1.2.3.4")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "login", resp.Action)
	assert.Equal(t, "example.com", resp.Hostname)
	assert.Equal(t, "2026-01-01T00:00:00Z", resp.ChallengeTS)
}

func TestVerify_Failed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"success":     false,
			"error-codes": []string{"invalid-input-response", "timeout-or-duplicate"},
		}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	resp, err := c.Verify(context.Background(), "test-token-1234567890", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorCodes, "invalid-input-response")
}

func TestVerify_InvalidToken(t *testing.T) {
	c := NewClient("secret", "sitekey")
	_, err := c.Verify(context.Background(), "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestVerify_Non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	_, err := c.Verify(context.Background(), "test-token-1234567890", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestVerify_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	_, err := c.Verify(context.Background(), "test-token-1234567890", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestVerify_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	_, err := c.Verify(context.Background(), "test-token-1234567890", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verify request")
}

func TestVerifyRequest_FormField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "form-token-1234567890", r.Form.Get("response"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"success": true}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	r := httptest.NewRequest("POST", "/submit", strings.NewReader("cf-turnstile-response=form-token-1234567890"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.RemoteAddr = "1.2.3.4:5678"

	resp, err := c.VerifyRequest(context.Background(), r, "cf-turnstile-response")
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestVerifyRequest_HeaderFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"success": true}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	r := httptest.NewRequest("POST", "/submit", nil)
	r.Header.Set("cf-turnstile-response", "header-token-1234567890")
	r.RemoteAddr = "1.2.3.4:5678"

	resp, err := c.VerifyRequest(context.Background(), r, "cf-turnstile-response")
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestVerifyRequest_EmptyToken(t *testing.T) {
	c := NewClient("secret", "sitekey")
	r := httptest.NewRequest("POST", "/submit", nil)
	_, err := c.VerifyRequest(context.Background(), r, "cf-turnstile-response")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestVerifyRequest_IPv6RemoteAddr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, r.ParseForm())
		// IPv6 addr should not be stripped (contains multiple colons).
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"success": true}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	r := httptest.NewRequest("POST", "/submit", strings.NewReader("cf-turnstile-response=token-1234567890"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.RemoteAddr = "[::1]:5678"

	resp, err := c.VerifyRequest(context.Background(), r, "cf-turnstile-response")
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestRenderHTML(t *testing.T) {
	c := NewClient("secret", "0x4AAAAAAA")
	html := c.RenderHTML()
	assert.Contains(t, html, ScriptURL)
	assert.Contains(t, html, `data-sitekey="0x4AAAAAAA"`)
	assert.Contains(t, html, "cf-turnstile")
	assert.Contains(t, html, "async")
	assert.Contains(t, html, "defer")
}

func TestRenderHTMLWithCallback(t *testing.T) {
	c := NewClient("secret", "0x4AAAAAAA")
	html := c.RenderHTMLWithCallback("onTurnstileSuccess")
	assert.Contains(t, html, ScriptURL)
	assert.Contains(t, html, `data-sitekey="0x4AAAAAAA"`)
	assert.Contains(t, html, `data-callback="onTurnstileSuccess"`)
}

func TestVerify_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"success": true}))
	}))
	defer srv.Close()

	c := NewClient("secret", "sitekey").WithEndpoint(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	_, err := c.Verify(ctx, "test-token-1234567890", "")
	require.Error(t, err)
}
