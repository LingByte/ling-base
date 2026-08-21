# meter

Usage metering for AI API calls. Records per-call usage (tokens, images, audio seconds, video seconds) and provides aggregation and query capabilities. This package measures usage only; cost calculation is left to the consuming application.

## Key Types

- **`Usage`** -- metering data for a single call (input/output/cached/reasoning tokens, image count, audio/video seconds, request count). Has a `Merge` method.
- **`UsageRecord`** -- a stored metering event with provider, model, mode, usage, and timestamp.
- **`UsageQuery`** -- filter for querying records (provider, model, mode, time range).
- **`UsageStats`** -- aggregated summary with totals and breakdowns by provider/model/mode.
- **`UsageAggregate`** -- one bucket in a grouped aggregation.
- **`Meter`** -- interface with `Record`, `Query`, `Aggregate`, and `Close` methods.

## Implementations

- **`NewMemoryMeter`** -- in-memory implementation suitable for testing and single-process usage.

## Usage

```go
m := meter.NewMemoryMeter()
defer m.Close()

m.Record(ctx, &meter.UsageRecord{
    Provider: "openai",
    Model:    "gpt-4o-mini",
    Usage:    meter.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
})

stats, _ := m.Query(ctx, nil)
agg, _ := m.Aggregate(ctx, nil, "model")
```
