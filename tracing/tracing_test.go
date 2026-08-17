// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "unknown-service", cfg.ServiceName)
	assert.Equal(t, ExporterNoop, cfg.Exporter)
	assert.Equal(t, 1.0, cfg.SampleRatio)
	assert.True(t, cfg.SetGlobal)
}

func TestNewProvider_NoopExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Exporter:       ExporterNoop,
		SampleRatio:    1.0,
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Shutdown(ctx)

	assert.NotNil(t, p.TracerProvider())
	assert.NotNil(t, p.Tracer())

	// Start a span and verify it works.
	ctx, span := p.Tracer().Start(ctx, "test-operation")
	assert.NotNil(t, span)
	assert.True(t, span.IsRecording())
	span.End()
}

func TestNewProvider_StdoutExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "test-service",
		Exporter:    ExporterStdout,
		SampleRatio: 1.0,
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer p.Shutdown(ctx)

	_, span := p.Tracer().Start(ctx, "stdout-test")
	span.End()
	// Flush by shutting down.
	require.NoError(t, p.Shutdown(ctx))
}

func TestNewProvider_MissingServiceName(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "",
		Exporter:    ExporterNoop,
	}
	_, err := NewProvider(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ServiceName is required")
}

func TestNewProvider_UnknownExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "test-service",
		Exporter:    ExporterKind("unknown"),
	}
	_, err := NewProvider(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown exporter kind")
}

func TestNewProvider_ResourceAttributes(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "2.0.0",
		Environment:    "staging",
		Exporter:       ExporterNoop,
		ResourceAttributes: map[string]string{
			"custom.attr": "custom-value",
		},
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)
	defer p.Shutdown(ctx)

	res := p.Resource()
	attrs := res.Attributes()
	found := false
	for _, attr := range attrs {
		if attr.Key == attribute.Key("custom.attr") && attr.Value.AsString() == "custom-value" {
			found = true
		}
	}
	assert.True(t, found, "custom resource attribute should be present")
}

func TestNewProvider_SampleRatio(t *testing.T) {
	tests := []struct {
		ratio float64
	}{
		{0.0}, // never sample
		{1.0}, // always sample
		{0.5}, // 50% sampling
	}
	for _, tt := range tests {
		ctx := context.Background()
		cfg := Config{
			ServiceName: "test-service",
			Exporter:    ExporterNoop,
			SampleRatio: tt.ratio,
		}
		p, err := NewProvider(ctx, cfg)
		require.NoError(t, err)
		defer p.Shutdown(ctx)
	}
}

func TestInit_SetsGlobal(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "global-test",
		Exporter:    ExporterNoop,
		SetGlobal:   true,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	// Verify global tracer provider is set.
	tp := otel.GetTracerProvider()
	assert.NotNil(t, tp)

	// Start a span via the global API.
	_, span := Start(ctx, "global-span")
	assert.NotNil(t, span)
	span.End()
}

func TestInit_DoesNotSetGlobal(t *testing.T) {
	ctx := context.Background()
	original := otel.GetTracerProvider()
	cfg := Config{
		ServiceName: "non-global-test",
		Exporter:    ExporterNoop,
		SetGlobal:   false,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	// Global should be unchanged.
	assert.Equal(t, original, otel.GetTracerProvider())
}

func TestStart_WithAttributes(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "attr-test",
		Exporter:    ExporterNoop,
		SetGlobal:   true,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	_, span := Start(ctx, "attr-operation",
		oteltrace.WithAttributes(
			attribute.String("key1", "value1"),
			attribute.Int("key2", 42),
		))
	assert.NotNil(t, span)
	span.End()
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "ctx-test",
		Exporter:    ExporterNoop,
		SetGlobal:   true,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	ctx, span := Start(ctx, "parent-span")
	defer span.End()

	childSpan := SpanFromContext(ctx)
	assert.NotNil(t, childSpan)
}

func TestContextWithSpan(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "ctx-with-span-test",
		Exporter:    ExporterNoop,
		SetGlobal:   true,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	_, span := Start(ctx, "test-span")
	defer span.End()

	newCtx := ContextWithSpan(ctx, span)
	spanFromCtx := SpanFromContext(newCtx)
	assert.NotNil(t, spanFromCtx)
}

func TestProvider_ShutdownMultipleCalls(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "multi-shutdown",
		Exporter:    ExporterNoop,
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)

	// Multiple shutdowns should be safe.
	err1 := p.Shutdown(ctx)
	err2 := p.Shutdown(ctx)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		endpoint string
		expect   bool
	}{
		{"localhost:4317", true},
		{"127.0.0.1:4317", true},
		{"0.0.0.0:4317", true},
		{"example.com:4317", false},
		{"", false}, // empty string is not localhost
	}
	for _, tt := range tests {
		got := isLocalhost(tt.endpoint)
		assert.Equal(t, tt.expect, got, "endpoint=%q", tt.endpoint)
	}
}

func TestIsTracingEnabled(t *testing.T) {
	// Before Init, the global provider is a noop.
	enabled := IsTracingEnabled()
	_ = enabled // Just verify it doesn't panic.

	ctx := context.Background()
	cfg := Config{
		ServiceName: "enabled-test",
		Exporter:    ExporterNoop,
		SetGlobal:   true,
	}
	shutdown, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer shutdown(ctx)

	enabled = IsTracingEnabled()
	assert.True(t, enabled)
}

func TestProvider_TracerProviderType(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "type-test",
		Exporter:    ExporterNoop,
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)
	defer p.Shutdown(ctx)

	tp := p.TracerProvider()
	assert.NotNil(t, tp, "TracerProvider should not be nil")
}

func TestNewProvider_WithBatchOptions(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:            "batch-test",
		Exporter:               ExporterNoop,
		SpanBatchTimeout:       2 * time.Second,
		SpanMaxQueueSize:       1024,
		SpanMaxExportBatchSize: 256,
	}
	p, err := NewProvider(ctx, cfg)
	require.NoError(t, err)
	defer p.Shutdown(ctx)
}
