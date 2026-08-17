// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "unknown-service", cfg.ServiceName)
	assert.Equal(t, TraceExporterNoop, cfg.TraceExporter)
	assert.Equal(t, MetricsExporterNoop, cfg.MetricsExporter)
	assert.Equal(t, LogExporterNoop, cfg.LogExporter)
	assert.True(t, cfg.SetGlobal)
}

func TestInit_AllNoop(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TraceExporter:   TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:     LogExporterNoop,
		SetGlobal:       true,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, sdk)
	defer sdk.Shutdown(ctx)

	assert.NotNil(t, sdk.Traces)
	// Metrics and Logs are nil for noop exporters.
	// Actually, Metrics is nil for noop, but Logs returns a noop provider.
	assert.Nil(t, sdk.Metrics) // noop metrics doesn't create a provider
	assert.NotNil(t, sdk.Logs)
}

func TestInit_MissingServiceName(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "",
	}
	_, err := Init(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ServiceName is required")
}

func TestInit_WithPrometheus(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:     "prom-test",
		TraceExporter:   TraceExporterNoop,
		MetricsExporter: MetricsExporterPrometheus,
		LogExporter:     LogExporterNoop,
		SetGlobal:       true,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	assert.NotNil(t, sdk.Metrics)
	assert.NotNil(t, sdk.MetricsHTTPHandler())
}

func TestInit_WithStdoutTrace(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "stdout-trace-test",
		TraceExporter: TraceExporterStdout,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:   LogExporterNoop,
		SetGlobal:     false,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	assert.NotNil(t, sdk.Traces)

	// Start a span.
	ctx, span := sdk.Traces.Tracer().Start(ctx, "stdout-trace-span")
	span.End()
}

func TestInit_WithStdoutLog(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "stdout-log-test",
		TraceExporter: TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:   LogExporterStdout,
		SetGlobal:     false,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	assert.NotNil(t, sdk.Logs)
	assert.NotNil(t, sdk.Logs.Logger())

	// Emit a log record.
	var rec otellog.Record
	rec.SetTimestamp(time.Now())
	rec.SetBody(otellog.StringValue("test log message"))
	rec.SetSeverity(otellog.SeverityInfo)
	sdk.Logs.Logger().Emit(ctx, rec)
}

func TestInit_UnknownTraceExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "bad-trace-test",
		TraceExporter: TraceExporterKind("unknown"),
	}
	_, err := Init(ctx, cfg)
	require.Error(t, err)
}

func TestInit_UnknownLogExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "bad-log-test",
		TraceExporter: TraceExporterNoop,
		LogExporter:   LogExporterKind("unknown"),
	}
	_, err := Init(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log exporter kind")
}

func TestSDK_ShutdownMultipleCalls(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "multi-shutdown-test",
		TraceExporter: TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:   LogExporterNoop,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)

	err1 := sdk.Shutdown(ctx)
	err2 := sdk.Shutdown(ctx)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestLogProvider_ShutdownMultipleCalls(t *testing.T) {
	ctx := context.Background()
	cfg := logConfig{
		ServiceName: "lp-shutdown-test",
		Exporter:    LogExporterNoop,
	}
	lp, err := newLogProvider(ctx, cfg)
	require.NoError(t, err)

	err1 := lp.Shutdown(ctx)
	err2 := lp.Shutdown(ctx)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestLogProvider_Noop(t *testing.T) {
	ctx := context.Background()
	cfg := logConfig{
		ServiceName: "noop-log-test",
		Exporter:    LogExporterNoop,
	}
	lp, err := newLogProvider(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, lp.Logger())
	// Noop provider has nil LoggerProvider.
	assert.Nil(t, lp.LoggerProvider())
}

func TestZapOTelCore(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "zap-bridge-test",
		TraceExporter: TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:   LogExporterStdout,
		SetGlobal:     false,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	core := NewZapOTelCore(sdk.Logs.Logger())
	logger := zap.New(core)

	logger.Info("test info message",
		zap.String("key", "value"),
		zap.Int("count", 42),
		zap.Bool("flag", true),
	)
	logger.Error("test error message",
		zap.String("error", "something failed"),
	)
}

func TestZapOTelCore_WithLevel(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:   "zap-level-test",
		TraceExporter: TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:   LogExporterStdout,
		SetGlobal:     false,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	core := NewZapOTelCoreWithLevel(sdk.Logs.Logger(), zapcore.ErrorLevel)
	logger := zap.New(core)

	// Info should be filtered out (level is Error).
	logger.Info("this should be filtered")
	// Error should pass through.
	logger.Error("this should pass")
}

func TestZapOTelCore_Enabled(t *testing.T) {
	ctx := context.Background()
	cfg := logConfig{
		ServiceName: "zap-enabled-test",
		Exporter:    LogExporterNoop,
	}
	lp, err := newLogProvider(ctx, cfg)
	require.NoError(t, err)

	core := NewZapOTelCore(lp.Logger())
	c := core.(*ZapOTelCore)

	assert.False(t, c.Enabled(zapcore.DebugLevel))
	assert.True(t, c.Enabled(zapcore.InfoLevel))
	assert.True(t, c.Enabled(zapcore.WarnLevel))
	assert.True(t, c.Enabled(zapcore.ErrorLevel))
}

func TestZapOTelCore_With(t *testing.T) {
	ctx := context.Background()
	cfg := logConfig{
		ServiceName: "zap-with-test",
		Exporter:    LogExporterNoop,
	}
	lp, err := newLogProvider(ctx, cfg)
	require.NoError(t, err)

	core := NewZapOTelCore(lp.Logger())
	core2 := core.With([]zapcore.Field{
		zap.String("persistent", "value"),
	})

	c2 := core2.(*ZapOTelCore)
	assert.Len(t, c2.fields, 1)
	assert.Equal(t, "persistent", string(c2.fields[0].Key))
}

func TestZapOTelCore_Sync(t *testing.T) {
	ctx := context.Background()
	cfg := logConfig{
		ServiceName: "zap-sync-test",
		Exporter:    LogExporterNoop,
	}
	lp, err := newLogProvider(ctx, cfg)
	require.NoError(t, err)

	core := NewZapOTelCore(lp.Logger())
	assert.NoError(t, core.Sync())
}

func TestZapLevelToOTel(t *testing.T) {
	tests := []struct {
		zapLevel zapcore.Level
		expect   otellog.Severity
	}{
		{zapcore.DebugLevel, otellog.SeverityDebug},
		{zapcore.InfoLevel, otellog.SeverityInfo},
		{zapcore.WarnLevel, otellog.SeverityWarn},
		{zapcore.ErrorLevel, otellog.SeverityError},
		{zapcore.DPanicLevel, otellog.SeverityFatal1},
		{zapcore.PanicLevel, otellog.SeverityFatal2},
		{zapcore.FatalLevel, otellog.SeverityFatal3},
	}
	for _, tt := range tests {
		got := zapLevelToOTel(tt.zapLevel)
		assert.Equal(t, tt.expect, got)
	}
}

func TestZapFieldToOTel(t *testing.T) {
	tests := []struct {
		name    string
		field   zapcore.Field
		wantKey string
	}{
		{"string", zap.String("k", "v"), "k"},
		{"int", zap.Int64("k", 42), "k"},
		{"float", zap.Float64("k", 3.14), "k"},
		{"bool", zap.Bool("k", true), "k"},
		{"duration", zap.Duration("k", time.Second), "k"},
		{"error", zap.Error(fmt.Errorf("test")), "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kv := zapFieldToOTel(tt.field)
			assert.Equal(t, tt.wantKey, string(kv.Key))
		})
	}
}

func TestInit_ResourceAttributes(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:    "resource-attr-test",
		TraceExporter:  TraceExporterNoop,
		MetricsExporter: MetricsExporterNoop,
		LogExporter:    LogExporterNoop,
		ResourceAttributes: map[string]string{
			"custom.otel.attr": "otel-value",
		},
		SetGlobal: false,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	// Check trace resource.
	res := sdk.Traces.Resource()
	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == attribute.Key("custom.otel.attr") && attr.Value.AsString() == "otel-value" {
			found = true
		}
	}
	assert.True(t, found, "custom resource attribute should be present in traces")
}

func TestIsLocalhostLog(t *testing.T) {
	assert.True(t, isLocalhostLog("localhost:4317"))
	assert.True(t, isLocalhostLog("127.0.0.1:4317"))
	assert.True(t, isLocalhostLog("0.0.0.0:4317"))
	assert.False(t, isLocalhostLog("example.com:4317"))
}

func TestInit_FullPipeline(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:      "full-pipeline-test",
		ServiceVersion:   "1.0.0",
		Environment:      "test",
		TraceExporter:    TraceExporterNoop,
		MetricsExporter:  MetricsExporterPrometheus,
		LogExporter:      LogExporterNoop,
		SampleRatio:      1.0,
		SpanBatchTimeout: 2 * time.Second,
		SetGlobal:        true,
	}
	sdk, err := Init(ctx, cfg)
	require.NoError(t, err)
	defer sdk.Shutdown(ctx)

	// Verify all three signals.
	assert.NotNil(t, sdk.Traces)
	assert.NotNil(t, sdk.Metrics)
	assert.NotNil(t, sdk.Logs)
	assert.NotNil(t, sdk.MetricsHTTPHandler())

	// Use the tracing.
	_, span := sdk.Traces.Tracer().Start(ctx, "full-pipeline-span")
	span.End()

	// Use the metrics.
	counter, err := sdk.Metrics.Meter().Int64Counter("pipeline_counter")
	require.NoError(t, err)
	counter.Add(ctx, 1)
}
