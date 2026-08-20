// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command otel-demo demonstrates the ling-base tracing, metrics, and
// opentelemetry packages.
//
// It starts a unified OTel SDK with:
//   - Tracing: stdout exporter (prints spans to console)
//   - Metrics: Prometheus exporter (served on /metrics)
//   - Logs: stdout exporter (prints log records to console)
//
// Then it simulates some work with spans, metrics, and a Zap→OTel log
// bridge, and serves the /metrics endpoint until interrupted.
//
// Usage:
//
//	go run ./cmd/otel-demo
//	go run ./cmd/otel-demo -addr :9090
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LingByte/ling-base/common/metrics"
	"github.com/LingByte/ling-base/common/opentelemetry"
	"github.com/LingByte/ling-base/common/tracing"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func main() {
	addr := flag.String("addr", ":9090", "address for Prometheus metrics server")
	flag.Parse()

	ctx := context.Background()

	// ── Initialize the unified OTel SDK ──
	fmt.Println("=== Initializing OpenTelemetry SDK ===")
	sdk, err := opentelemetry.Init(ctx, opentelemetry.Config{
		ServiceName:     "otel-demo",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		TraceExporter:   opentelemetry.TraceExporterStdout,
		MetricsExporter: opentelemetry.MetricsExporterPrometheus,
		LogExporter:     opentelemetry.LogExporterStdout,
		SampleRatio:     1.0,
		SetGlobal:       true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "init otel: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		fmt.Println("\n=== Shutting down OTel SDK ===")
		if err := sdk.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		}
	}()
	fmt.Println("  Traces:  stdout exporter")
	fmt.Println("  Metrics: Prometheus exporter")
	fmt.Println("  Logs:    stdout exporter")

	// ── Create instruments ──
	meter := sdk.Metrics.Meter()
	requestsTotal, err := meter.Int64Counter("demo_requests_total",
		otelmetric.WithDescription("Total demo requests"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create counter: %v\n", err)
		os.Exit(1)
	}

	requestDuration, err := meter.Float64Histogram("demo_request_duration_seconds",
		otelmetric.WithDescription("Demo request duration in seconds"),
		otelmetric.WithExplicitBucketBoundaries(metrics.DefaultHTTPBuckets...))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create histogram: %v\n", err)
		os.Exit(1)
	}

	activeGauge, err := meter.Int64ObservableGauge("demo_active_connections",
		otelmetric.WithDescription("Active demo connections"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create gauge: %v\n", err)
		os.Exit(1)
	}

	var activeCount int64
	_, err = meter.RegisterCallback(func(_ context.Context, o otelmetric.Observer) error {
		o.ObserveInt64(activeGauge, activeCount)
		return nil
	}, activeGauge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "register callback: %v\n", err)
		os.Exit(1)
	}

	// ── Create a Zap logger bridged to OTel logs ──
	zapCore := opentelemetry.NewZapOTelCore(sdk.Logs.Logger())
	logger := zap.New(zapCore)

	// ── Start the Prometheus metrics server ──
	http.Handle("/metrics", sdk.MetricsHTTPHandler())
	go func() {
		fmt.Printf("\n=== Prometheus metrics server on %s/metrics ===\n", *addr)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			fmt.Fprintf(os.Stderr, "metrics server: %v\n", err)
		}
	}()

	// ── Simulate work ──
	fmt.Println("\n=== Simulating work (Ctrl+C to stop) ===")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				activeCount = rng.Int63n(100) + 1

				// Create a span for each "request".
				ctx, span := tracing.Start(ctx, "demo-request",
					oteltrace.WithAttributes(
						attribute.String("request.method", "GET"),
						attribute.String("request.path", "/api/data"),
					))
				start := time.Now()

				// Simulate processing.
				time.Sleep(time.Duration(rng.Intn(100)) * time.Millisecond)

				duration := time.Since(start).Seconds()
				requestsTotal.Add(ctx, 1,
					otelmetric.WithAttributes(
						attribute.String("method", "GET"),
						attribute.String("status", "200"),
					))
				requestDuration.Record(ctx, duration)

				logger.Info("processed request",
					zap.String("method", "GET"),
					zap.String("path", "/api/data"),
					zap.Float64("duration_s", duration),
					zap.Int64("active", activeCount),
				)

				span.End()
			case <-ctx.Done():
				return
			}
		}
	}()

	// ── Wait for interrupt ──
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\n=== Received interrupt signal ===")
}
