# helper

Utility functions for relay provider adaptors, covering SSE streaming, response ID generation, and streaming response construction.

## Key Functions

- **`SetEventStreamHeaders`** -- sets SSE headers (`text/event-stream`, `no-cache`, `keep-alive`) on an `http.ResponseWriter`.
- **`StringData`** / **`ObjectData`** -- write an SSE `data:` line to the writer and flush.
- **`Done`** -- writes the SSE `[DONE]` marker.
- **`GetResponseID`** -- generates a `chatcmpl-<id>` response ID from a request ID.
- **`GenerateStartEmptyResponse`** -- creates an empty start chunk for streaming chat completions.
- **`GenerateStopResponse`** -- creates a stop chunk with `finish_reason: "stop"`.
- **`GenerateFinalUsageResponse`** -- creates a final chunk carrying usage stats.

## StreamScanner

- **`StreamScanner`** -- reads SSE `data:` lines from an `io.Reader`.
- **`StreamScannerHandler`** -- iterates SSE lines from an `http.Response`, calling a handler per line until `[DONE]` or EOF.

## Usage

```go
helper.SetEventStreamHeaders(w)
helper.ObjectData(w, helper.GenerateStartEmptyResponse(id, createdAt, model, nil))
helper.Done(w)
```
