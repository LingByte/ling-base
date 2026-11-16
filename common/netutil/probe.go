// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package netutil

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	// ErrProbeEmptyURL is returned when the probe URL is empty.
	ErrProbeEmptyURL = errors.New("netutil/probe: url must not be empty")
	// ErrProbeUnsupportedMethod is returned for unsupported HTTP methods.
	ErrProbeUnsupportedMethod = errors.New("netutil/probe: unsupported HTTP method")
	// ErrProbeBuildRequest wraps request construction failures.
	ErrProbeBuildRequest = errors.New("netutil/probe: build request failed")
	// ErrProbeNetwork wraps transport / dial / TLS failures.
	ErrProbeNetwork = errors.New("netutil/probe: network error")
	// ErrProbeValidate wraps response validation failures.
	ErrProbeValidate = errors.New("netutil/probe: validation failed")
	// ErrProbeMarshalBody wraps BodyJSON marshal failures.
	ErrProbeMarshalBody = errors.New("netutil/probe: marshal body failed")
)

var allowedProbeMethods = map[string]struct{}{
	http.MethodGet: {}, http.MethodHead: {}, http.MethodPost: {},
	http.MethodPut: {}, http.MethodPatch: {}, http.MethodDelete: {},
	http.MethodOptions: {},
}

const (
	defaultProbeMaxBody   int64 = 1 << 20 // 1 MiB
	defaultProbeUserAgent       = "ling-base/netutil-probe"
	defaultProbeTimeout         = 30 * time.Second
	bodyTruncatedMarker         = "\n...[truncated]"
)

// ──────────────────────────────────────────────
// Request / Expect / Result
// ──────────────────────────────────────────────

// ProbeRequest describes a single HTTP probe.
//
// Timeout is applied via context.WithTimeout on top of the caller context.
// The Prober's http.Client.Timeout is left at 0 so only the request context
// governs cancellation (avoids confusing double timeouts).
type ProbeRequest struct {
	URL           string
	Method        string            // defaults to GET; must be a standard method
	Headers       map[string]string // keys are canonicalized via http.Header
	Params        map[string]string
	Body          string
	BodyJSON      any // marshaled if Body is empty; supports $${var} in string leaves
	ContentType   string
	Timeout       time.Duration // default 30s; clipped by outer ctx deadline
	Expect        ProbeExpect
	SkipTLSVerify bool
	Variables     map[string]string // "$${key}" substitution

	// FollowRedirects controls redirect following. nil = use Prober default.
	FollowRedirects *bool
	// MaxRedirects caps redirects when following (default 10).
	MaxRedirects int

	// ProxyURL optionally overrides the HTTP proxy for this request.
	ProxyURL string
	// TLSServerName sets TLS SNI / ServerName.
	TLSServerName string
}

// Clone returns a deep-ish copy safe for concurrent / looped mutation.
func (r ProbeRequest) Clone() ProbeRequest {
	out := r
	if r.Headers != nil {
		out.Headers = make(map[string]string, len(r.Headers))
		for k, v := range r.Headers {
			out.Headers[k] = v
		}
	}
	if r.Params != nil {
		out.Params = make(map[string]string, len(r.Params))
		for k, v := range r.Params {
			out.Params[k] = v
		}
	}
	if r.Variables != nil {
		out.Variables = make(map[string]string, len(r.Variables))
		for k, v := range r.Variables {
			out.Variables[k] = v
		}
	}
	if r.FollowRedirects != nil {
		v := *r.FollowRedirects
		out.FollowRedirects = &v
	}
	return out
}

// ProbeExpect defines response validation rules.
type ProbeExpect struct {
	StatusCode      int // 0 → any 2xx
	BodyContains    string
	BodyNotContains string
	BodyJSONPath    string // e.g. "data.0.name" or "items.-1.id"
	Validator       func(statusCode int, body []byte, headers http.Header) error
}

// ProbeTiming breaks down request latency phases (best-effort via httptrace).
type ProbeTiming struct {
	DNS     time.Duration
	Connect time.Duration
	TLS     time.Duration
	TTFB    time.Duration // time to first response byte
	Total   time.Duration
}

// ProbeResult holds the outcome of a probe execution.
type ProbeResult struct {
	Success    bool
	StatusCode int
	Body       string
	Truncated  bool
	Headers    http.Header
	Duration   time.Duration
	Timing     ProbeTiming
	Error      string
	Err        error // typed cause when available (errors.Is / As)
	Extracted  map[string]string
	Timestamp  time.Time
}

// ──────────────────────────────────────────────
// Prober
// ──────────────────────────────────────────────

// Prober executes HTTP probes.
//
// Concurrency: a Prober is safe for concurrent use after NewProber returns.
// Per-request TLS/proxy overrides clone Transport; the shared client is not mutated.
type Prober struct {
	client          *http.Client
	insecureOnce    sync.Once
	insecureClient  *http.Client
	maxBody         int64
	userAgent       string
	followRedirects bool
	maxRedirects    int
	rootCAs         *x509.CertPool
	mu              sync.Mutex // guards insecureClient init + custom clones map if needed
}

// ProberOption configures a [Prober].
type ProberOption func(*Prober)

// WithProbeMaxBodySize sets the maximum response body size to read (default 1 MiB).
func WithProbeMaxBodySize(n int64) ProberOption {
	return func(p *Prober) {
		if n > 0 {
			p.maxBody = n
		}
	}
}

// WithProbeUserAgent sets the default User-Agent.
func WithProbeUserAgent(ua string) ProberOption {
	return func(p *Prober) { p.userAgent = ua }
}

// WithProbeHTTPClient sets a custom *http.Client (Timeout should be 0; use request context).
func WithProbeHTTPClient(c *http.Client) ProberOption {
	return func(p *Prober) { p.client = c }
}

// WithProbeFollowRedirects sets the default redirect policy.
func WithProbeFollowRedirects(follow bool, max int) ProberOption {
	return func(p *Prober) {
		p.followRedirects = follow
		if max > 0 {
			p.maxRedirects = max
		}
	}
}

// WithProbeRootCAs sets custom trusted roots for TLS verification.
func WithProbeRootCAs(pool *x509.CertPool) ProberOption {
	return func(p *Prober) { p.rootCAs = pool }
}

// NewProber creates a Prober with sensible defaults.
func NewProber(opts ...ProberOption) *Prober {
	p := &Prober{
		maxBody:         defaultProbeMaxBody,
		userAgent:       defaultProbeUserAgent,
		followRedirects: true,
		maxRedirects:    10,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	if p.client == nil {
		p.client = newProbeHTTPClient(false, "", "", p.rootCAs, nil)
	}
	applyRedirectPolicy(p.client, p.followRedirects, p.maxRedirects)
	return p
}

// Execute runs a single HTTP probe and returns the result.
func (p *Prober) Execute(ctx context.Context, req ProbeRequest) ProbeResult {
	result := ProbeResult{Timestamp: time.Now(), Headers: make(http.Header)}
	req = req.Clone()

	if strings.TrimSpace(req.URL) == "" {
		result.Err = ErrProbeEmptyURL
		result.Error = ErrProbeEmptyURL.Error()
		return result
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	method := strings.ToUpper(req.Method)
	if _, ok := allowedProbeMethods[method]; !ok {
		result.Err = fmt.Errorf("%w: %s", ErrProbeUnsupportedMethod, req.Method)
		result.Error = result.Err.Error()
		return result
	}
	req.Method = method
	if req.Timeout <= 0 {
		req.Timeout = defaultProbeTimeout
	}

	if err := applyProbeVariables(&req); err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrProbeMarshalBody, err)
		result.Error = result.Err.Error()
		return result
	}

	start := time.Now()
	defer func() { result.Duration = time.Since(start); result.Timing.Total = result.Duration }()

	// Request timeout nests under the caller context; the shorter deadline wins.
	probeCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	timing := &result.Timing
	probeCtx = withProbeTrace(probeCtx, timing)

	httpReq, err := p.buildRequest(probeCtx, req)
	if err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrProbeBuildRequest, err)
		result.Error = result.Err.Error()
		return result
	}

	client, err := p.clientFor(req)
	if err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrProbeBuildRequest, err)
		result.Error = result.Err.Error()
		return result
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrProbeNetwork, err)
		result.Error = result.Err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Headers = resp.Header.Clone()

	limited := io.LimitReader(resp.Body, p.maxBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		result.Err = fmt.Errorf("%w: read body: %v", ErrProbeNetwork, err)
		result.Error = result.Err.Error()
		return result
	}
	if int64(len(body)) > p.maxBody {
		result.Truncated = true
		body = append(body[:p.maxBody], []byte(bodyTruncatedMarker)...)
	}
	result.Body = string(body)

	if err := p.validate(resp.StatusCode, body, resp.Header, req.Expect, &result); err != nil {
		result.Err = fmt.Errorf("%w: %v", ErrProbeValidate, err)
		result.Error = result.Err.Error()
		return result
	}

	result.Success = true
	return result
}

func (p *Prober) buildRequest(ctx context.Context, req ProbeRequest) (*http.Request, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if len(req.Params) > 0 {
		q := parsedURL.Query()
		for k, v := range req.Params {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	} else if req.BodyJSON != nil {
		b, err := json.Marshal(req.BodyJSON)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrProbeMarshalBody, err)
		}
		bodyReader = bytes.NewReader(b)
		if req.ContentType == "" {
			req.ContentType = "application/json"
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v) // canonicalizes MIME header keys
	}
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	if p.userAgent != "" && httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", p.userAgent)
	}
	return httpReq, nil
}

func (p *Prober) validate(statusCode int, body []byte, headers http.Header, expect ProbeExpect, result *ProbeResult) error {
	if expect.StatusCode != 0 {
		if statusCode != expect.StatusCode {
			return fmt.Errorf("status code %d, expected %d", statusCode, expect.StatusCode)
		}
	} else if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("status code %d is not 2xx", statusCode)
	}

	if expect.BodyContains != "" && !bytes.Contains(body, []byte(expect.BodyContains)) {
		return fmt.Errorf("body does not contain %q", expect.BodyContains)
	}
	if expect.BodyNotContains != "" && bytes.Contains(body, []byte(expect.BodyNotContains)) {
		return fmt.Errorf("body contains %q (expected not to)", expect.BodyNotContains)
	}

	if expect.BodyJSONPath != "" {
		val, err := extractProbeJSONPath(body, expect.BodyJSONPath)
		if err != nil {
			return fmt.Errorf("json path %q: %w", expect.BodyJSONPath, err)
		}
		if result.Extracted == nil {
			result.Extracted = make(map[string]string)
		}
		result.Extracted[expect.BodyJSONPath] = val
	}

	if expect.Validator != nil {
		if err := expect.Validator(statusCode, body, headers); err != nil {
			return fmt.Errorf("validator: %w", err)
		}
	}
	return nil
}

func (p *Prober) clientFor(req ProbeRequest) (*http.Client, error) {
	follow := p.followRedirects
	if req.FollowRedirects != nil {
		follow = *req.FollowRedirects
	}
	maxRedir := p.maxRedirects
	if req.MaxRedirects > 0 {
		maxRedir = req.MaxRedirects
	}

	needsClone := req.SkipTLSVerify || req.ProxyURL != "" || req.TLSServerName != "" ||
		req.FollowRedirects != nil || req.MaxRedirects > 0

	if !needsClone {
		return p.client, nil
	}

	if req.SkipTLSVerify && req.ProxyURL == "" && req.TLSServerName == "" &&
		req.FollowRedirects == nil && req.MaxRedirects == 0 {
		return p.getInsecureClient(), nil
	}

	base := p.client
	if req.SkipTLSVerify {
		base = p.getInsecureClient()
	}
	cloned, err := cloneHTTPClient(base, req.SkipTLSVerify, req.ProxyURL, req.TLSServerName, p.rootCAs)
	if err != nil {
		return nil, err
	}
	applyRedirectPolicy(cloned, follow, maxRedir)
	return cloned, nil
}

func (p *Prober) getInsecureClient() *http.Client {
	p.insecureOnce.Do(func() {
		c, err := cloneHTTPClient(p.client, true, "", "", p.rootCAs)
		if err != nil {
			c = newProbeHTTPClient(true, "", "", p.rootCAs, p.client.Jar)
		}
		applyRedirectPolicy(c, p.followRedirects, p.maxRedirects)
		p.insecureClient = c
	})
	return p.insecureClient
}

func newProbeHTTPClient(insecure bool, proxyURL, serverName string, roots *x509.CertPool, jar http.CookieJar) *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure, //nolint:gosec // intentional for probe SkipTLSVerify
			RootCAs:            roots,
			ServerName:         serverName,
			MinVersion:         tls.VersionTLS12,
		},
	}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err == nil {
			tr.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{
		Timeout:   0, // request context owns deadlines
		Transport: tr,
		Jar:       jar,
	}
}

func cloneHTTPClient(base *http.Client, insecure bool, proxyURL, serverName string, roots *x509.CertPool) (*http.Client, error) {
	if base == nil {
		return newProbeHTTPClient(insecure, proxyURL, serverName, roots, nil), nil
	}
	out := *base
	out.Timeout = 0

	var tr *http.Transport
	switch bt := base.Transport.(type) {
	case *http.Transport:
		tr = bt.Clone()
	case nil:
		tr = newProbeHTTPClient(false, "", "", roots, nil).Transport.(*http.Transport)
	default:
		// Unknown transport: build a fresh one rather than mutating shared state.
		c := newProbeHTTPClient(insecure, proxyURL, serverName, roots, base.Jar)
		return c, nil
	}

	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		tr.TLSClientConfig = tr.TLSClientConfig.Clone()
	}
	tr.TLSClientConfig.InsecureSkipVerify = insecure
	if roots != nil {
		tr.TLSClientConfig.RootCAs = roots
	}
	if serverName != "" {
		tr.TLSClientConfig.ServerName = serverName
	}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		tr.Proxy = http.ProxyURL(u)
	}
	if tr.MaxIdleConns == 0 {
		tr.MaxIdleConns = 100
	}
	if tr.IdleConnTimeout == 0 {
		tr.IdleConnTimeout = 90 * time.Second
	}
	out.Transport = tr
	return &out, nil
}

func applyRedirectPolicy(c *http.Client, follow bool, max int) {
	if c == nil {
		return
	}
	if !follow {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return
	}
	if max <= 0 {
		max = 10
	}
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= max {
			return fmt.Errorf("stopped after %d redirects", max)
		}
		return nil
	}
}

func withProbeTrace(ctx context.Context, timing *ProbeTiming) context.Context {
	var dnsStart, connectStart, tlsStart, gotConn time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS = time.Since(dnsStart)
			}
		},
		ConnectStart: func(_, _ string) { connectStart = time.Now() },
		ConnectDone: func(_, _ string, _ error) {
			if !connectStart.IsZero() {
				timing.Connect = time.Since(connectStart)
			}
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			if !tlsStart.IsZero() {
				timing.TLS = time.Since(tlsStart)
			}
		},
		GotConn: func(httptrace.GotConnInfo) { gotConn = time.Now() },
		GotFirstResponseByte: func() {
			if !gotConn.IsZero() {
				timing.TTFB = time.Since(gotConn)
			}
		},
	}
	return httptrace.WithClientTrace(ctx, trace)
}

// ──────────────────────────────────────────────
// Variable substitution
// ──────────────────────────────────────────────

func applyProbeVariables(req *ProbeRequest) error {
	vars := req.Variables
	req.URL = substituteProbeVars(req.URL, vars)
	for k, v := range req.Headers {
		req.Headers[k] = substituteProbeVars(v, vars)
	}
	for k, v := range req.Params {
		req.Params[k] = substituteProbeVars(v, vars)
	}
	req.Body = substituteProbeVars(req.Body, vars)
	req.ProxyURL = substituteProbeVars(req.ProxyURL, vars)
	req.TLSServerName = substituteProbeVars(req.TLSServerName, vars)

	if req.BodyJSON != nil {
		substituted, err := substituteJSONVars(req.BodyJSON, vars)
		if err != nil {
			return err
		}
		req.BodyJSON = substituted
	}
	return nil
}

// substituteProbeVars replaces "$${key}" patterns. Longer keys are applied first
// to avoid short-key collisions (e.g. a vs aa).
func substituteProbeVars(s string, vars map[string]string) string {
	if vars == nil || !strings.Contains(s, "$${") {
		return s
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		s = strings.ReplaceAll(s, "$${"+k+"}", vars[k])
	}
	return s
}

func substituteJSONVars(v any, vars map[string]string) (any, error) {
	if vars == nil {
		return v, nil
	}
	switch x := v.(type) {
	case string:
		return substituteProbeVars(x, vars), nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			sv, err := substituteJSONVars(vv, vars)
			if err != nil {
				return nil, err
			}
			out[substituteProbeVars(k, vars)] = sv
		}
		return out, nil
	case map[string]string:
		out := make(map[string]string, len(x))
		for k, vv := range x {
			out[substituteProbeVars(k, vars)] = substituteProbeVars(vv, vars)
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			sv, err := substituteJSONVars(vv, vars)
			if err != nil {
				return nil, err
			}
			out[i] = sv
		}
		return out, nil
	default:
		// Re-encode typed structs to map/slice so nested strings can be rewritten.
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var generic any
		if err := json.Unmarshal(b, &generic); err != nil {
			return nil, err
		}
		return substituteJSONVars(generic, vars)
	}
}

// ──────────────────────────────────────────────
// JSON path extraction
// ──────────────────────────────────────────────

func extractProbeJSONPath(body []byte, path string) (string, error) {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("unmarshal json: %w", err)
	}

	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return "", fmt.Errorf("key %q not found", part)
			}
			current = next
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("expected numeric index, got %q", part)
			}
			if idx < 0 {
				idx = len(v) + idx
			}
			if idx < 0 || idx >= len(v) {
				return "", fmt.Errorf("index %d out of range [0,%d)", idx, len(v))
			}
			current = v[idx]
		default:
			return "", fmt.Errorf("cannot traverse %q on %T", part, current)
		}
	}

	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		// Prefer integer formatting when the value is integral.
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "", nil
	case json.Number:
		return v.String(), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

// LoadProbeRootCAs loads PEM certificates from paths into a cert pool.
func LoadProbeRootCAs(paths ...string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, p := range paths {
		pem, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("netutil/probe: no certificates found in %s", p)
		}
	}
	return pool, nil
}
