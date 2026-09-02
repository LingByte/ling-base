# sse

Server-Sent Events (SSE) support for both server and client sides, using only
the Go standard library.

## Features

- Server-side `Writer` that emits spec-compliant SSE events with auto-flush
- Client-side `Client` + `StreamReader` for streaming events from any SSE endpoint
- `SetHeaders` helper for the standard SSE response headers
- `ParseEvent` for parsing SSE text blocks (useful in tests)
- Multi-line data, comments, retry hints, and custom event types

## Key functions

- `NewWriter(w)`, `SetHeaders(w)`
- `(*Writer) Write/WriteData/WriteComment/WriteRetry/Close`
- `NewClient(url)`, `(*Client) WithHTTPClient(c)`, `(*Client) Connect(ctx)`
- `(*StreamReader) Next/Err/Close`
- `ParseEvent(data)`

## Quick start

```go
import "github.com/LingByte/ling-base/common/sse"

// Server
func handler(w http.ResponseWriter, r *http.Request) {
    sse.SetHeaders(w)
    sw, _ := sse.NewWriter(w)
    defer sw.Close()
    sw.WriteData("hello")
    sw.Write(&sse.Event{ID: "1", Event: "ping", Data: "pong"})
}

// Client
client := sse.NewClient("http://example.com/events")
stream, _ := client.Connect(ctx)
defer stream.Close()
for {
    ev, err := stream.Next()
    if err != nil { break }
    log.Println(ev.Data)
}
```

## License

MIT
