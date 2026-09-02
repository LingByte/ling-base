// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package wsutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

// newTestServer starts an httptest server that upgrades to WebSocket and
// echoes back any received message. It returns the server and a ws:// URL.
func newTestServer(t *testing.T, cfg Config) (*httptest.Server, string) {
	t.Helper()
	upgrader := NewUpgrader(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, data, err := conn.Read()
			if err != nil {
				return
			}
			if mt == TextMessage {
				_ = conn.WriteText(string(data))
			} else {
				_ = conn.WriteBinary(data)
			}
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	return srv, wsURL
}

// ──────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 30*time.Second, cfg.PingInterval)
	assert.Equal(t, 70*time.Second, cfg.PongTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, int64(1<<20), cfg.MaxMessageSize)
}

// ──────────────────────────────────────────────
// Upgrader
// ──────────────────────────────────────────────

func TestNewUpgrader(t *testing.T) {
	cfg := DefaultConfig()
	u := NewUpgrader(cfg)
	require.NotNil(t, u)
	assert.Equal(t, cfg, u.config)
	assert.NotNil(t, u.upgrader)
}

func TestUpgrader_Upgrade_Success(t *testing.T) {
	cfg := DefaultConfig()
	_, wsURL := newTestServer(t, cfg)

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteText("ping"))
	msg, err := conn.ReadText()
	require.NoError(t, err)
	assert.Equal(t, "ping", msg)
}

func TestUpgrader_Upgrade_BadRequest(t *testing.T) {
	upgrader := NewUpgrader(DefaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force upgrade failure by sending a response before upgrade.
		w.WriteHeader(http.StatusBadRequest)
		_, _ = upgrader.Upgrade(w, r)
	}))
	defer srv.Close()

	// Use a raw HTTP GET (no WebSocket headers) to trigger failure.
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ──────────────────────────────────────────────
// Dial
// ──────────────────────────────────────────────

func TestDial_Success(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()
}

func TestDial_InvalidURL(t *testing.T) {
	_, err := Dial(context.Background(), "http://example.invalid", DefaultConfig())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dial failed")
}

func TestDial_ContextCanceled(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Dial(ctx, wsURL, DefaultConfig())
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Write / Read
// ──────────────────────────────────────────────

func TestConn_WriteText_ReadText(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteText("hello"))
	msg, err := conn.ReadText()
	require.NoError(t, err)
	assert.Equal(t, "hello", msg)
}

func TestConn_WriteBinary_Read(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	payload := []byte{0x00, 0x01, 0x02, 0xFF}
	require.NoError(t, conn.WriteBinary(payload))
	mt, data, err := conn.Read()
	require.NoError(t, err)
	assert.Equal(t, BinaryMessage, mt)
	assert.Equal(t, payload, data)
}

func TestConn_ReadText_NonText(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteBinary([]byte("binary")))
	_, err = conn.ReadText()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected text message")
}

func TestConn_WriteJSON_ReadJSON(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	out := payload{Name: "alice", Age: 30}
	require.NoError(t, conn.WriteJSON(out))

	var got payload
	require.NoError(t, conn.ReadJSON(&got))
	assert.Equal(t, out, got)
}

func TestConn_ReadJSON_Invalid(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteText("{bad json"))
	var v map[string]any
	err = conn.ReadJSON(&v)
	assert.Error(t, err)
}

func TestConn_Read_Closed(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, conn.Close())
	_, _, err = conn.Read()
	assert.Error(t, err)
}

func TestConn_Write_Closed(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	err = conn.WriteText("after close")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Ping / Pong handlers
// ──────────────────────────────────────────────

func TestConn_SetPongHandler(t *testing.T) {
	cfg := DefaultConfig()
	pongReceived := make(chan string, 1)

	upgrader := NewUpgrader(cfg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		// Default ping handler responds with a pong frame; keep reading.
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	conn.SetPongHandler(func(appData string) error {
		pongReceived <- appData
		return nil
	})

	// Pong handler is invoked during reads; start a reader goroutine.
	go func() {
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}()

	require.NoError(t, conn.ws.WriteMessage(websocket.PingMessage, []byte("hello")))
	select {
	case appData := <-pongReceived:
		assert.Equal(t, "hello", appData)
	case <-time.After(2 * time.Second):
		t.Fatal("pong handler not invoked")
	}
}

func TestConn_SetPingHandler(t *testing.T) {
	cfg := DefaultConfig()
	pingReceived := make(chan string, 1)

	upgrader := NewUpgrader(cfg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetPingHandler(func(appData string) error {
			pingReceived <- appData
			return nil
		})
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.ws.WriteMessage(websocket.PingMessage, []byte("pingdata")))
	select {
	case appData := <-pingReceived:
		assert.Equal(t, "pingdata", appData)
	case <-time.After(2 * time.Second):
		t.Fatal("ping handler not invoked")
	}
}

func TestConn_StartPing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PingInterval = 50 * time.Millisecond

	pongCount := make(chan int, 10)
	upgrader := NewUpgrader(cfg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		// Default ping handler responds with pong; just keep reading.
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Pong handler fires during reads on the client.
	conn.SetPongHandler(func(appData string) error {
		pongCount <- 1
		return nil
	})
	go func() {
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}()

	cancel := conn.StartPing()
	time.Sleep(250 * time.Millisecond)
	cancel()

	count := 0
	for {
		select {
		case <-pongCount:
			count++
		default:
			assert.Greater(t, count, 0, "expected at least one pong")
			return
		}
	}
}

// ──────────────────────────────────────────────
// MaxMessageSize
// ──────────────────────────────────────────────

func TestConn_MaxMessageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxMessageSize = 8
	cfg.ReadTimeout = 500 * time.Millisecond

	// Use an echo server so the client can read back small messages.
	_, wsURL := newTestServer(t, cfg)

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Small message within limit: echo round-trip succeeds.
	require.NoError(t, conn.WriteText("short"))
	_, _, err = conn.Read()
	require.NoError(t, err)

	// Large message exceeds limit: server read fails and closes; client read
	// then also fails (either read limit or connection closed).
	require.NoError(t, conn.WriteText("this is way too long"))
	_, _, err = conn.Read()
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// ReadTimeout
// ──────────────────────────────────────────────

func TestConn_ReadTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReadTimeout = 50 * time.Millisecond

	upgrader := NewUpgrader(cfg)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		// Don't send anything; client should time out on read.
		<-r.Context().Done()
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = conn.Read()
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Hub
// ──────────────────────────────────────────────

func TestHub_Register_Unregister(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)

	_, wsURL := newTestServer(t, DefaultConfig())
	c1, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c1.Close()
	c2, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c2.Close()

	hub.Register(c1)
	hub.Register(c2)
	hub.mu.RLock()
	assert.Len(t, hub.conns, 2)
	hub.mu.RUnlock()

	hub.Unregister(c1)
	hub.mu.RLock()
	assert.Len(t, hub.conns, 1)
	hub.mu.RUnlock()

	hub.Unregister(c2)
	hub.mu.RLock()
	assert.Empty(t, hub.conns)
	hub.mu.RUnlock()
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()

	// Server registers connections to the hub and keeps reading (to keep
	// the connection alive). The hub broadcasts to server-side conns, and
	// the clients receive the data.
	upgrader := NewUpgrader(DefaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		hub.Register(conn)
		defer hub.Unregister(conn)
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c1, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c1.Close()
	c2, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c2.Close()

	// Wait for both connections to register on server side.
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.conns) == 2
	}, 2*time.Second, 20*time.Millisecond)

	hub.Broadcast([]byte("hello-binary"))

	// Both clients should receive the broadcast.
	for i, c := range []*Conn{c1, c2} {
		done := make(chan []byte, 1)
		go func() {
			mt, data, err := c.Read()
			if err != nil {
				done <- nil
				return
			}
			_ = mt
			done <- data
		}()
		select {
		case data := <-done:
			assert.Equal(t, []byte("hello-binary"), data, "client %d", i)
		case <-time.After(2 * time.Second):
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}
}

func TestHub_BroadcastText(t *testing.T) {
	hub := NewHub()

	upgrader := NewUpgrader(DefaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		hub.Register(conn)
		defer hub.Unregister(conn)
		for {
			if _, _, err := conn.Read(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c1, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c1.Close()
	c2, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer c2.Close()

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.conns) == 2
	}, 2*time.Second, 20*time.Millisecond)

	hub.BroadcastText("broadcast-text")

	for i, c := range []*Conn{c1, c2} {
		done := make(chan string, 1)
		go func() {
			msg, err := c.ReadText()
			if err != nil {
				done <- ""
				return
			}
			done <- msg
		}()
		select {
		case msg := <-done:
			assert.Equal(t, "broadcast-text", msg, "client %d", i)
		case <-time.After(2 * time.Second):
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}
}

func TestHub_Close(t *testing.T) {
	hub := NewHub()

	_, wsURL := newTestServer(t, DefaultConfig())
	c1, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	c2, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)

	hub.Register(c1)
	hub.Register(c2)
	hub.Close()

	hub.mu.RLock()
	assert.Empty(t, hub.conns)
	hub.mu.RUnlock()
}

func TestHub_Broadcast_NoConns(t *testing.T) {
	hub := NewHub()
	assert.NotPanics(t, func() {
		hub.Broadcast([]byte("nobody"))
		hub.BroadcastText("nobody")
	})
}

// ──────────────────────────────────────────────
// JSON encode error path
// ──────────────────────────────────────────────

func TestConn_WriteJSON_Error(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	defer conn.Close()

	// channel cannot be JSON-marshaled.
	err = conn.WriteJSON(make(chan int))
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// WriteTimeout applied
// ──────────────────────────────────────────────

func TestConn_WriteTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WriteTimeout = 1 * time.Nanosecond

	_, wsURL := newTestServer(t, cfg)
	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	// Very short write timeout; large write may fail. At minimum the deadline
	// is set without panicking.
	_ = conn.WriteText("x")
}

// ──────────────────────────────────────────────
// MessageType constants
// ──────────────────────────────────────────────

func TestMessageType_Constants(t *testing.T) {
	assert.Equal(t, MessageType(websocket.TextMessage), TextMessage)
	assert.Equal(t, MessageType(websocket.BinaryMessage), BinaryMessage)
}

// ──────────────────────────────────────────────
// StartPing default interval
// ──────────────────────────────────────────────

func TestConn_StartPing_ZeroInterval(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PingInterval = 0

	_, wsURL := newTestServer(t, cfg)
	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	cancel := conn.StartPing()
	time.Sleep(60 * time.Millisecond)
	cancel()
}

// ──────────────────────────────────────────────
// WriteTimeout = 0 (no deadline) path
// ──────────────────────────────────────────────

func TestConn_WriteTimeout_Zero(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WriteTimeout = 0

	_, wsURL := newTestServer(t, cfg)
	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteText("no-deadline"))
	msg, err := conn.ReadText()
	require.NoError(t, err)
	assert.Equal(t, "no-deadline", msg)
}

// ──────────────────────────────────────────────
// ReadText / ReadJSON error from underlying read
// ──────────────────────────────────────────────

func TestConn_ReadText_ReadError(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	_, err = conn.ReadText()
	assert.Error(t, err)
}

func TestConn_ReadJSON_ReadError(t *testing.T) {
	_, wsURL := newTestServer(t, DefaultConfig())
	conn, err := Dial(context.Background(), wsURL, DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	var v map[string]any
	err = conn.ReadJSON(&v)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// StartPing write error (closed connection)
// ──────────────────────────────────────────────

func TestConn_StartPing_WriteError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PingInterval = 20 * time.Millisecond

	_, wsURL := newTestServer(t, cfg)
	conn, err := Dial(context.Background(), wsURL, cfg)
	require.NoError(t, err)

	cancel := conn.StartPing()
	// Close the connection so the next ping write fails and the goroutine exits.
	require.NoError(t, conn.Close())
	time.Sleep(60 * time.Millisecond)
	cancel()
}

// ──────────────────────────────────────────────
// unused import guard
// ──────────────────────────────────────────────

var _ = json.Marshal
