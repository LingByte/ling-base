# webhook

A webhook sender with automatic JSON serialization, HMAC-SHA256 request
signing, event filtering, and retry with exponential backoff. Pure
standard library, no external dependencies.

## Features

- JSON payload serialization with `Content-Type: application/json`
- `X-Webhook-Event` and `X-Webhook-Timestamp` headers
- Optional HMAC-SHA256 signing via `X-Webhook-Signature` header
- Event allow-list filtering (skip events the webhook didn't subscribe to)
- Automatic retry with exponential backoff on network/5xx errors
- 4xx errors are not retried (client errors)

## Quick start

```go
import (
    "time"
    "github.com/LingByte/ling-base/common/webhook"
)

sender := webhook.NewSender(
    webhook.WithTimeout(10*time.Second),
    webhook.WithMaxRetries(3),
    webhook.WithRetryInterval(time.Second),
)

wh := &webhook.Webhook{
    URL:    "https://example.com/hook",
    Secret: "s3cr3t",
    Events: []string{"order.created", "order.paid"},
    Active: true,
}

err := sender.SendWithSignature(wh, "order.created", map[string]any{
    "order_id": "12345",
})
```

## Signing

`SendWithSignature` adds an `X-Webhook-Signature` header containing the
lowercase hex-encoded HMAC-SHA256 of the raw JSON payload keyed by
`Webhook.Secret`. Receivers should recompute the HMAC over the raw
request body and compare (constant-time) to verify authenticity:

```go
valid := webhook.VerifySignature(secret, requestBody, sigHeader)
```

## API

### Sender

| Method | Description |
| --- | --- |
| `NewSender(opts...)` | Create a sender |
| `Send(wh, event, payload)` | Send a webhook |
| `SendWithSignature(wh, event, payload)` | Send with HMAC signature |
| `VerifySignature(secret, body, sig)` | Verify a received signature |

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithTimeout(d)` | 10s | Per-attempt HTTP timeout |
| `WithMaxRetries(n)` | 3 | Retries after initial attempt |
| `WithRetryInterval(d)` | 1s | Base backoff interval (`d * 2^n`) |
| `WithHTTPClient(c)` | http.DefaultClient | Custom client (testing) |

### Webhook fields

`URL`, `Secret`, `Events` (empty = all events), `Active`.

## Retry behavior

- Network errors and HTTP 5xx → retried with exponential backoff
- HTTP 4xx → returned immediately, not retried
- Inactive webhook / event not subscribed / empty URL → returned immediately
