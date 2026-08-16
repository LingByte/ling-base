package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// stubWSServer is a minimal WS echo server for BaseAgent lifecycle tests.
type stubWSServer struct {
	t        *testing.T
	server   *httptest.Server
	url      string
	onConnect func(conn *websocket.Conn)
	onMessage func(conn *websocket.Conn, msg []byte)
}

func newStubWSServer(t *testing.T, onConnect func(*websocket.Conn), onMessage func(*websocket.Conn, []byte)) *stubWSServer {
	s := &stubWSServer{t: t, onConnect: onConnect, onMessage: onMessage}
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		if s.onConnect != nil {
			s.onConnect(conn)
		}
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if s.onMessage != nil {
				s.onMessage(conn, msg)
			}
		}
	}))
	s.url = "ws" + strings.TrimPrefix(s.server.URL, "http")
	return s
}

func (s *stubWSServer) close() { s.server.Close() }

func TestBaseAgentLifecycle(t *testing.T) {
	var mu sync.Mutex
	var gotEvents []EventType
	stub := newStubWSServer(t, func(conn *websocket.Conn) {
		// Send a session.created-like message so Dispatch can fire open.
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"session.created"}`))
	}, nil)
	defer stub.close()

	dialer := *websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(context.Background(), stub.url, nil)
	if err != nil {
		t.Fatalf("dial: %v (resp=%v)", err, resp)
	}
	defer resp.Body.Close()

	ba := NewBaseAgent("test", Options{
		OnEvent: func(ev Event) {
			mu.Lock()
			gotEvents = append(gotEvents, ev.Type)
			mu.Unlock()
		},
	}, 16)
	ba.SetConn(conn)
	ctx, cancel := context.WithCancel(context.Background())
	ba.SetRootContext(ctx, cancel)
	ba.Dispatch = func(raw []byte) {
		var head struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(raw, &head)
		if head.Type == "session.created" {
			ba.FireOnce(Event{Type: EventSessionOpen})
		}
	}
	ba.StartLoops()

	// Wait for session open.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(gotEvents)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	if len(gotEvents) == 0 || gotEvents[0] != EventSessionOpen {
		t.Fatalf("gotEvents = %v, want EventSessionOpen first", gotEvents)
	}
	mu.Unlock()

	if !ba.HasFiredOpen() {
		t.Error("HasFiredOpen = false, want true")
	}

	// Close should be idempotent.
	if err := ba.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := ba.Close(); err != nil {
		t.Errorf("Close twice: %v", err)
	}
	if !ba.IsClosed() {
		t.Error("IsClosed = false after Close")
	}

	// Wait for session close event.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		last := gotEvents[len(gotEvents)-1]
		mu.Unlock()
		if last == EventSessionClose {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	last := gotEvents[len(gotEvents)-1]
	mu.Unlock()
	if last != EventSessionClose {
		t.Errorf("last event = %v, want EventSessionClose", last)
	}
}

func TestBaseAgentSendJSONAfterClose(t *testing.T) {
	ba := NewBaseAgent("test", Options{OnEvent: func(Event) {}}, 4)
	ctx, cancel := context.WithCancel(context.Background())
	ba.SetRootContext(ctx, cancel)
	ba.closed.Store(true)
	err := ba.SendJSON(map[string]any{"type": "x"}, false)
	if !errors.Is(err, ErrAgentClosed) {
		t.Errorf("err = %v, want ErrAgentClosed", err)
	}
}

func TestBaseAgentPushAudioAfterClose(t *testing.T) {
	ba := NewBaseAgent("test", Options{OnEvent: func(Event) {}}, 4)
	ctx, cancel := context.WithCancel(context.Background())
	ba.SetRootContext(ctx, cancel)
	ba.closed.Store(true)
	err := ba.SendRaw([]byte("x"), false)
	if !errors.Is(err, ErrAgentClosed) {
		t.Errorf("err = %v, want ErrAgentClosed", err)
	}
}

func TestBaseAgentNonBlockingDrop(t *testing.T) {
	ba := NewBaseAgent("test", Options{OnEvent: func(Event) {}}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	ba.SetRootContext(ctx, cancel)
	// Fill the channel.
	ba.sendCh <- []byte("first")
	// Non-blocking send should not block.
	done := make(chan struct{})
	go func() {
		_ = ba.SendRaw([]byte("second"), true)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("non-blocking send blocked")
	}
}

func TestBaseAgentEmitSuppressesAfterClose(t *testing.T) {
	var mu sync.Mutex
	var got []EventType
	ba := NewBaseAgent("test", Options{
		OnEvent: func(ev Event) {
			mu.Lock()
			got = append(got, ev.Type)
			mu.Unlock()
		},
	}, 4)
	ba.closed.Store(true)
	ba.Emit(Event{Type: EventAssistantText, Text: "should be suppressed"})
	ba.Emit(Event{Type: EventSessionClose})
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || got[0] != EventSessionClose {
		t.Errorf("got = %v, want only EventSessionClose", got)
	}
}

func TestBaseAgentFireOnceDedup(t *testing.T) {
	var count int
	ba := NewBaseAgent("test", Options{
		OnEvent: func(ev Event) {
			if ev.Type == EventSessionOpen {
				count++
			}
		},
	}, 4)
	ba.FireOnce(Event{Type: EventSessionOpen})
	ba.FireOnce(Event{Type: EventSessionOpen})
	if count != 1 {
		t.Errorf("FireOnce fired %d times, want 1", count)
	}
}
