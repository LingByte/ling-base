// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "unknown-service", cfg.ServiceName)
	assert.True(t, cfg.SetGlobal)
}

func TestNewProvider_Prometheus(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Shutdown(context.Background())

	assert.NotNil(t, p.MeterProvider())
	assert.NotNil(t, p.Meter())
	assert.NotNil(t, p.HTTPHandler())
	assert.NotNil(t, p.Exporter())
}

func TestNewProvider_MissingServiceName(t *testing.T) {
	cfg := Config{
		ServiceName: "",
	}
	_, err := NewProvider(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ServiceName is required")
}

func TestNewProvider_WithCustomRegistry(t *testing.T) {
	reg := promclient.NewRegistry()
	cfg := Config{
		ServiceName:        "custom-reg-test",
		PrometheusRegistry: reg,
		PrometheusGatherer: reg,
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	assert.NotNil(t, p.HTTPHandler())
}

func TestProvider_Counter(t *testing.T) {
	cfg := Config{
		ServiceName: "counter-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	counter, err := p.Meter().Int64Counter("test_counter_total",
		otelmetric.WithDescription("A test counter"))
	require.NoError(t, err)

	counter.Add(context.Background(), 5)
	counter.Add(context.Background(), 3)

	// Verify via HTTP handler.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "test_counter_total")
	assert.Contains(t, body, "8") // 5 + 3
}

func TestProvider_Histogram(t *testing.T) {
	cfg := Config{
		ServiceName: "histogram-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	hist, err := p.Meter().Float64Histogram("test_histogram",
		otelmetric.WithDescription("A test histogram"),
		otelmetric.WithExplicitBucketBoundaries(0.1, 0.5, 1.0, 5.0))
	require.NoError(t, err)

	hist.Record(context.Background(), 0.3)
	hist.Record(context.Background(), 2.0)
	hist.Record(context.Background(), 0.05)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "test_histogram")
}

func TestProvider_Gauge(t *testing.T) {
	cfg := Config{
		ServiceName: "gauge-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	gauge, err := p.Meter().Float64ObservableGauge("test_gauge",
		otelmetric.WithDescription("A test gauge"))
	require.NoError(t, err)

	_, err = p.Meter().RegisterCallback(func(_ context.Context, o otelmetric.Observer) error {
		o.ObserveFloat64(gauge, 42.0)
		return nil
	}, gauge)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "test_gauge")
}

func TestProvider_UpDownCounter(t *testing.T) {
	cfg := Config{
		ServiceName: "updown-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	udc, err := p.Meter().Int64UpDownCounter("test_updown",
		otelmetric.WithDescription("A test up-down counter"))
	require.NoError(t, err)

	udc.Add(context.Background(), 10)
	udc.Add(context.Background(), -3)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "test_updown")
}

func TestProvider_DefaultHistogramBuckets(t *testing.T) {
	cfg := Config{
		ServiceName:             "buckets-test",
		DefaultHistogramBuckets: []float64{1, 5, 10, 50},
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	hist, err := p.Meter().Float64Histogram("bucketed_hist")
	require.NoError(t, err)
	hist.Record(context.Background(), 3)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "bucketed_hist")
}

func TestInit_SetsGlobal(t *testing.T) {
	cfg := Config{
		ServiceName: "global-metrics-test",
		SetGlobal:   true,
	}
	p, err := Init(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// Use global convenience function.
	counter, err := Int64Counter("global_counter")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)
}

func TestProvider_ShutdownMultipleCalls(t *testing.T) {
	cfg := Config{
		ServiceName: "multi-shutdown-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)

	err1 := p.Shutdown(context.Background())
	err2 := p.Shutdown(context.Background())
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestHTTPMiddleware(t *testing.T) {
	cfg := Config{
		ServiceName: "middleware-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	mw, err := NewHTTPMiddleware(p)
	require.NoError(t, err)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics were recorded with attributes.
	metricReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mw2 := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(mw2, metricReq)

	body := mw2.Body.String()
	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds")
	// Verify status label is present.
	assert.Contains(t, body, `status="2xx"`)
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `path="/test"`)
}

func TestHTTPMiddleware_DifferentStatuses(t *testing.T) {
	cfg := Config{
		ServiceName: "status-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	mw, err := NewHTTPMiddleware(p)
	require.NoError(t, err)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))

	for _, path := range []string{"/ok", "/notfound", "/error"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Verify metrics have different status labels.
	metricReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, metricReq)

	body := w.Body.String()
	assert.Contains(t, body, `status="2xx"`)
	assert.Contains(t, body, `status="4xx"`)
	assert.Contains(t, body, `status="5xx"`)
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status int
		expect string
	}{
		{100, "1xx"},
		{200, "2xx"},
		{201, "2xx"},
		{301, "3xx"},
		{404, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
		{0, "unknown"},
		{-1, "unknown"},
	}
	for _, tt := range tests {
		got := statusLabel(tt.status)
		assert.Equal(t, tt.expect, got, "status=%d", tt.status)
	}
}

func TestKeyValueHelpers(t *testing.T) {
	assert.Equal(t, "key", string(KeyValue("key", "value").Key))
	assert.Equal(t, "value", KeyValue("key", "value").Value.AsString())

	assert.Equal(t, "key", string(KeyInt("key", 42).Key))
	assert.Equal(t, int64(42), KeyInt("key", 42).Value.AsInt64())

	assert.Equal(t, "key", string(KeyFloat("key", 3.14).Key))

	assert.Equal(t, "key", string(KeyBool("key", true).Key))
	assert.True(t, KeyBool("key", true).Value.AsBool())
}

func TestProvider_ResourceAttributes(t *testing.T) {
	cfg := Config{
		ServiceName:    "resource-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ResourceAttributes: map[string]string{
			"custom.metric.attr": "metric-value",
		},
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	// Record a metric and check the output includes resource attributes.
	counter, err := p.Meter().Int64Counter("resource_counter")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	// The service_name should appear as a label.
	assert.True(t, strings.Contains(body, "service_name") || strings.Contains(body, "target_info"),
		"metrics should contain service_name or target_info")
}

func TestDefaultHTTPBuckets(t *testing.T) {
	assert.NotEmpty(t, DefaultHTTPBuckets)
	assert.Contains(t, DefaultHTTPBuckets, 0.1)
	assert.Contains(t, DefaultHTTPBuckets, 1.0)
}

func TestProvider_MultipleInstruments(t *testing.T) {
	cfg := Config{
		ServiceName: "multi-instrument-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	meter := p.Meter()

	c1, err := meter.Int64Counter("multi_c1")
	require.NoError(t, err)
	c2, err := meter.Float64Counter("multi_c2")
	require.NoError(t, err)
	h1, err := meter.Int64Histogram("multi_h1")
	require.NoError(t, err)
	h2, err := meter.Float64Histogram("multi_h2")
	require.NoError(t, err)
	udc, err := meter.Float64UpDownCounter("multi_udc")
	require.NoError(t, err)

	ctx := context.Background()
	c1.Add(ctx, 1)
	c2.Add(ctx, 1.5)
	h1.Record(ctx, 10)
	h2.Record(ctx, 2.5)
	udc.Add(ctx, 5)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "multi_c1")
	assert.Contains(t, body, "multi_c2")
	assert.Contains(t, body, "multi_h1")
	assert.Contains(t, body, "multi_h2")
	assert.Contains(t, body, "multi_udc")
}

func TestStatusWriter(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: 200}
	sw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, sw.status)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProvider_DurationTiming(t *testing.T) {
	cfg := Config{
		ServiceName: "timing-test",
	}
	p, err := NewProvider(cfg)
	require.NoError(t, err)
	defer p.Shutdown(context.Background())

	mw, err := NewHTTPMiddleware(p)
	require.NoError(t, err)

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify duration was recorded.
	metricReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mw2 := httptest.NewRecorder()
	p.HTTPHandler().ServeHTTP(mw2, metricReq)

	body := mw2.Body.String()
	assert.Contains(t, body, "http_request_duration_seconds")
}
