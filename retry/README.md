# retry

A flexible retry framework with configurable backoff strategies, max attempts,
timeouts, and retryable-error filtering.

## Features

- **Backoff strategies**: exponential backoff (with jitter), fixed interval, no delay
- **Custom backoff**: implement the `Backoff` interface for custom strategies
- **Max attempts**: fixed count or unlimited (bounded by context/timeout)
- **Overall timeout**: deadline across all attempts combined
- **Retryable errors**: filter which errors should be retried via `WithRetryIf`
- **Callbacks**: `OnRetry`, `OnSuccess`, `OnError` hooks for observability
- **Decorator pattern**: wrap any function to produce a retried version
- **Context-aware**: respects context cancellation and deadlines
- **Zero dependencies**: only uses the standard library

## Quick start

```go
import "github.com/LingByte/ling-base/retry"

// Basic retry with exponential backoff
err := retry.Do(ctx, func(ctx context.Context) error {
    return callRemoteAPI(ctx)
},
    retry.WithMaxAttempts(5),
    retry.WithExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0, true),
)

// Only retry on transient errors
err := retry.Do(ctx, op,
    retry.WithMaxAttempts(3),
    retry.WithFixedInterval(500*time.Millisecond),
    retry.WithRetryIf(func(err error) bool {
        var netErr net.Error
        return errors.As(err, &netErr) && netErr.Timeout()
    }),
)
```

## Backoff strategies

### Exponential backoff

```go
retry.WithExponentialBackoff(
    base:     100*time.Millisecond,  // initial delay
    maxDelay: 10*time.Second,        // upper bound (0 = no cap)
    factor:   2.0,                   // growth multiplier
    jitter:   true,                  // ±25% random jitter
)
```

Delay = `base * factor^attempt`, capped at `maxDelay`.
With jitter enabled, a random ±25% offset is added to avoid thundering-herd.

### Fixed interval

```go
retry.WithFixedInterval(500 * time.Millisecond)
```

Constant delay between every attempt.

### No backoff

```go
retry.WithNoBackoff()
```

Retry immediately with zero delay.

### Custom backoff

```go
type LinearBackoff struct{ Step time.Duration }
func (l LinearBackoff) NextDelay(attempt int) time.Duration {
    return l.Step * time.Duration(attempt+1)
}

retry.WithBackoff(LinearBackoff{Step: 100 * time.Millisecond})
```

## Decorator pattern

Wrap any function to produce a retried version:

```go
retriedGet := retry.Decorate(
    func(ctx context.Context, url string) (*http.Response, error) {
        return http.Get(url)
    },
    retry.WithMaxAttempts(3),
    retry.WithFixedInterval(500*time.Millisecond),
)
resp, err := retriedGet(ctx, "https://example.com")
```

For functions with no arguments:

```go
retriedFn := retry.Decorate0(
    func(ctx context.Context) error { return doWork(ctx) },
    retry.WithMaxAttempts(5),
)
err := retriedFn(ctx)
```

## Options

| Option | Description |
|--------|-------------|
| `WithMaxAttempts(n)` | Total attempts including first (default: 3, `<0` = unlimited) |
| `WithTimeout(d)` | Overall deadline for all attempts (0 = no timeout) |
| `WithBackoff(b)` | Custom backoff strategy |
| `WithExponentialBackoff(...)` | Convenience: exponential backoff |
| `WithFixedInterval(d)` | Convenience: fixed interval |
| `WithNoBackoff()` | Convenience: retry immediately |
| `WithRetryIf(fn)` | Predicate to filter retryable errors |
| `WithOnRetry(fn)` | Callback before each retry |
| `WithOnSuccess(fn)` | Callback after success |
| `WithOnError(fn)` | Callback when all attempts exhausted |

## License

MIT
