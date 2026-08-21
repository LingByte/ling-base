# realtime

Framework-independent WebSocket abstraction for OpenAI-compatible Realtime API sessions. Bridges a client WebSocket connection with an upstream provider WebSocket, forwarding events bidirectionally while accumulating usage metrics.

## Key Types

- **`Conn`** -- minimal WebSocket connection interface (`ReadMessage`, `WriteMessage`, `Close`). Satisfied by `*gorilla/websocket.Conn`.
- **`Connector`** -- interface for dialing an upstream realtime WebSocket. Implementations are provider-specific.
- **`SessionConfig`** -- parameters for opening a session (model, voice, instructions, modalities, audio formats, tools, API key, base URL).
- **`Session`** -- bridges a client and upstream connection, accumulates usage, and optionally records to a `meter.Meter`.
- **`OpenAIConnector`** -- `Connector` implementation for the OpenAI Realtime API.

## Usage

```go
session := realtime.NewSession(clientConn, realtime.SessionConfig{
    Model:             "gpt-4o-realtime-preview",
    Modalities:        []string{"text", "audio"},
    InputAudioFormat:  "pcm16",
    OutputAudioFormat: "pcm16",
    APIKey:            apiKey,
})
session.SetMeter(meter.NewMemoryMeter(), "openai")

connector := realtime.NewOpenAIConnector(apiKey, "gpt-4o-realtime-preview")
if err := session.Connect(ctx, connector); err != nil {
    log.Printf("session ended: %v", err)
}
fmt.Printf("Usage: %+v\n", session.Usage())
```
