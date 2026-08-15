// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultHTTPClient is the shared client used by all helpers. It relies
// on the per-request context for timeouts.
var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// errProviderRejected is returned (wrapped) when a provider API rejects
// the request with a non-success code.
var errProviderRejected = errors.New("sms: provider rejected")

// PostForm sends an application/x-www-form-urlencoded POST request.
// When basicUser is non-empty HTTP Basic Auth is applied. The response
// body is returned in full.
func PostForm(ctx context.Context, endpoint string, form url.Values, headers map[string]string, basicUser, basicPass string) ([]byte, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("sms: empty endpoint")
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("sms: build request: %w", err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequest(req)
}

// PostJSON sends a JSON POST request. body may be nil for an empty body.
func PostJSON(ctx context.Context, endpoint string, body []byte, headers map[string]string, basicUser, basicPass string) ([]byte, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("sms: empty endpoint")
	}
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, r)
	if err != nil {
		return nil, fmt.Errorf("sms: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequest(req)
}

// GetURL sends a GET request and returns the response body.
func GetURL(ctx context.Context, endpoint string, headers map[string]string, basicUser, basicPass string) ([]byte, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("sms: empty endpoint")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("sms: build request: %w", err)
	}
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequest(req)
}

// applyHeaders copies the given headers onto the request.
func applyHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

// applyBasicAuth sets Basic Auth when a username is provided.
func applyBasicAuth(req *http.Request, basicUser, basicPass string) {
	if basicUser != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}
}

// doRequest executes the request using the shared client (which honours
// the request context deadline) and returns the full response body. A
// non-2xx status produces an error containing the status and body.
func doRequest(req *http.Request) ([]byte, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sms: http request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sms: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, fmt.Errorf("sms: unexpected status %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, nil
}

// truncate limits s to at most n characters for inclusion in error msgs.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ──────────────────────────────────────────────
// Raw HTTP helpers — return (statusCode, body, error)
// These are used by providers that need to inspect non-2xx response bodies
// to extract provider-specific error messages.
// ──────────────────────────────────────────────

// PostFormRaw sends a form POST and returns the HTTP status code, body, and
// transport error (nil for any HTTP status). The caller is responsible for
// checking the status code.
func PostFormRaw(ctx context.Context, endpoint string, form url.Values, headers map[string]string, basicUser, basicPass string) (int, []byte, error) {
	if endpoint == "" {
		return 0, nil, fmt.Errorf("sms: empty endpoint")
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return 0, nil, fmt.Errorf("sms: build request: %w", err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequestRaw(req)
}

// PostJSONRaw sends a JSON POST and returns the HTTP status code, body, and
// transport error (nil for any HTTP status).
func PostJSONRaw(ctx context.Context, endpoint string, jsonBody []byte, headers map[string]string, basicUser, basicPass string) (int, []byte, error) {
	if endpoint == "" {
		return 0, nil, fmt.Errorf("sms: empty endpoint")
	}
	var r io.Reader
	if jsonBody != nil {
		r = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, r)
	if err != nil {
		return 0, nil, fmt.Errorf("sms: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequestRaw(req)
}

// GetURLRaw sends a GET and returns the HTTP status code, body, and
// transport error (nil for any HTTP status).
func GetURLRaw(ctx context.Context, endpoint string, headers map[string]string, basicUser, basicPass string) (Int int, body []byte, err error) {
	if endpoint == "" {
		return 0, nil, fmt.Errorf("sms: empty endpoint")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("sms: build request: %w", err)
	}
	applyHeaders(req, headers)
	applyBasicAuth(req, basicUser, basicPass)
	return doRequestRaw(req)
}

// doRequestRaw executes the request and returns (statusCode, body, error).
// The error is only for transport-level failures, not HTTP error codes.
func doRequestRaw(req *http.Request) (int, []byte, error) {
	if err := req.Context().Err(); err != nil {
		return 0, nil, err
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("sms: http request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("sms: read response: %w", err)
	}
	return resp.StatusCode, data, nil
}

// ──────────────────────────────────────────────
// Shared utility helpers
// ──────────────────────────────────────────────

// Is2xx reports whether code is in the 200–299 range.
func Is2xx(code int) bool { return code >= 200 && code < 300 }

// NowUnix returns the current Unix timestamp.
func NowUnix() int64 { return time.Now().Unix() }

// TruncateRaw limits s to max characters, appending "…" when truncated.
func TruncateRaw(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// JSONStringAny marshals v to a JSON string, returning "" on error.
func JSONStringAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// CtxOrBackground returns ctx when non-nil, otherwise context.Background.
func CtxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// FirstRecipientStr returns the first recipient's phone number string
// (without the "+" prefix). Returns an error when there are no recipients.
func FirstRecipientStr(req SendRequest) (string, error) {
	if len(req.To) == 0 {
		return "", fmt.Errorf("sms: no recipients")
	}
	n := strings.TrimSpace(req.To[0].Number)
	if n == "" {
		return "", fmt.Errorf("sms: first recipient has empty number")
	}
	return n, nil
}
