# opentelemetry

Unified, one-call setup for OpenTelemetry traces, metrics, and logs — the three pillars of observability.

It builds on the standalone `tracing` and `metrics` packages and adds the logs signal (still experimental in the OTel Go SDK) to provide a single entry point for application bootstrap. A Zap-to-OTel logs bridge lets existing `zap.Logger` usage be forwarded to the OTel logs pipeline.

## Key types

- `Config` — configures the unified SDK (service info, trace/metrics/log exporters, sampling)
- `SDK` — handle holding all three providers; call `Shutdown` to flush
- `Init(ctx, cfg) (*SDK, error)` — create and register providers globally
- `LogProvider` — wraps the SDK LoggerProvider
- `ZapOTelCore` — `zapcore.Core` that bridges zap entries to OTel logs
- Exporter kinds: `TraceExporterOTLPGRPC/HTTP/Stdout/Noop`, `MetricsExporterPrometheus/Noop`, `LogExporterOTLPGRPC/HTTP/Stdout/Noop`

## Quick start

```go
import "github.com/LingByte/ling-base/common/opentelemetry"

otelSDK, err := opentelemetry.Init(ctx, opentelemetry.Config{
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    Environment:     "production",
    TraceExporter:   opentelemetry.TraceExporterOTLPGRPC,
    OTLPEndpoint:    "localhost:4317",
    MetricsExporter: opentelemetry.MetricsExporterPrometheus,
    LogExporter:     opentelemetry.LogExporterOTLPGRPC,
    SampleRatio:     0.1,
})
if err != nil {
    log.Fatal(err)
}
defer otelSDK.Shutdown(ctx)

http.Handle("/metrics", otelSDK.MetricsHTTPHandler())
```
