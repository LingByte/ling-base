# wsutil

WebSocket utilities built on top of [gorilla/websocket](https://github.com/gorilla/websocket).

## Features

- `Conn` -- connection wrapper with configurable timeouts and ping/pong handling
- `Upgrader` -- HTTP → WebSocket upgrade helper
- `Dial` -- client-side dialer with context support
- `Hub` -- connection manager with broadcast (binary & text) and close-all
- Read/write helpers for text, binary, and JSON messages
- Automatic ping goroutine (`StartPing`) with cancel function

## Key types

- `Config` -- tuning parameters (`PingInterval`, `PongTimeout`, `WriteTimeout`, `ReadTimeout`, `MaxMessageSize`)
- `Conn` -- wrapped WebSocket connection
- `Upgrader` -- server-side upgrader
- `Hub` -- broadcast hub
- `MessageType` -- `TextMessage` / `BinaryMessage`

## Key functions

- `DefaultConfig()` -- sensible defaults
- `NewUpgrader(config)` -- create an upgrader
- `Dial(ctx, url, config)` -- connect as a client
- `(*Conn) WriteText`, `WriteBinary`, `WriteJSON`, `Read`, `ReadText`, `ReadJSON`
- `(*Conn) SetPingHandler`, `SetPongHandler`, `StartPing`, `Close`
- `NewHub()`, `(*Hub) Register`, `Unregister`, `Broadcast`, `BroadcastText`, `Close`

## Quick start

```go
import "github.com/LingByte/ling-base/common/wsutil"

// Server
upgrader := wsutil.NewUpgrader(wsutil.DefaultConfig())
conn, err := upgrader.Upgrade(w, r)
if err != nil { log.Fatal(err) }
defer conn.Close()
cancel := conn.StartPing()
defer cancel()
_ = conn.WriteText("welcome")

// Client
conn, err := wsutil.Dial(ctx, "ws://localhost:8080/ws", wsutil.DefaultConfig())
defer conn.Close()
msg, _ := conn.ReadText()

// Hub broadcast
hub := wsutil.NewHub()
hub.Register(conn)
hub.BroadcastText("hello everyone")
```

## License

MIT
