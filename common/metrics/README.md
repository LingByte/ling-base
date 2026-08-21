# metrics

Reusable wrapper around the OpenTelemetry Go metrics SDK with a Prometheus exporter, handling provider setup, resource attributes, and custom histogram buckets.

## Features

- OpenTelemetry MeterProvider with Prometheus exporter
- Configurable service name, version, namespace, and environment
- Custom default histogram bucket boundaries
- Prometheus-compatible HTTP handler for `/metrics`
- Private Prometheus registry by default (avoids global pollution)
- Optional global MeterProvider registration
- Convenience attribute helpers (KeyValue, KeyInt, KeyFloat, KeyBool)

## Key types

- `Config` -- provider configuration (service info, attributes, buckets, registry)
- `Provider` -- wraps MeterProvider with exporter and HTTP handler

## Key functions

- `Init(cfg)` -- create and optionally register a global Provider
- `NewProvider(cfg)` -- create Provider without setting globals
- `Provider.Meter()` -- get an `otelmetric.Meter` for instrument creation
- `Provider.HTTPHandler()` -- Prometheus `/metrics` endpoint handler
- `Provider.Shutdown(ctx)` -- flush and shut down
- `KeyValue`, `KeyInt`, `KeyFloat`, `KeyBool` -- attribute helpers

## Quick start

```go
import (
    "net/http"
    "github.com/LingByte/ling-base/common/metrics"
)

provider, err := metrics.Init(metrics.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})
if err != nil {
    log.Fatal(err)
}
defer provider.Shutdown(ctx)

http.Handle("/metrics", provider.HTTPHandler())
go http.ListenAndServe(":9090", nil)

counter, _ := provider.Meter().Float64Counter("requests_total")
counter.Add(ctx, 1)
```

## License

MIT
