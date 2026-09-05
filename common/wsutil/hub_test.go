// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package wsutil_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/wsutil"
)

// ──────────────────────────────────────────────
// EnhancedHub tests
// ──────────────────────────────────────────────

func newEnhancedTestServer(t *testing.T, hub *wsutil.EnhancedHub) (*httptest.Server, string) {
	t.Helper()
	upgrader := wsutil.NewUpgrader(wsutil.DefaultConfig())
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		info := &wsutil.ConnInfo{
			RemoteAddr: r.RemoteAddr,
		}
		_, err = hub.Register(conn, info)
		if err != nil {
			conn.Close()
			return
		}
		hub.ReadLoop(context.Background(), conn, func(i *wsutil.ConnInfo, mt wsutil.MessageType, data []byte) {
			// echo
			if mt == wsutil.TextMessage {
				_ = conn.WriteText(string(data))
			} else {
				_ = conn.WriteBinary(data)
			}
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	return srv, wsURL
}

func TestEnhancedHub_RegisterAndCount(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)

	conn, err := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	// Wait for server to register the connection.
	time.Sleep(50 * time.Millisecond)

	if hub.Count() != 1 {
		t.Errorf("Count = %d, want 1", hub.Count())
	}

	conn2, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn2.Close()
	time.Sleep(50 * time.Millisecond)

	if hub.Count() != 2 {
		t.Errorf("Count = %d, want 2", hub.Count())
	}

	srv.Close()
}

func TestEnhancedHub_MaxConnections(t *testing.T) {
	hub := wsutil.NewEnhancedHub(wsutil.WithMaxConnections(1))
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn1, err := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	if err != nil {
		t.Fatalf("Dial 1 failed: %v", err)
	}
	defer conn1.Close()
	time.Sleep(50 * time.Millisecond)

	if hub.Count() != 1 {
		t.Errorf("Count = %d, want 1", hub.Count())
	}

	// Second connection should be rejected by the hub.
	conn2, err := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	if err != nil {
		t.Fatalf("Dial 2 failed: %v", err)
	}
	defer conn2.Close()
	time.Sleep(50 * time.Millisecond)

	// The server should have closed the second connection.
	if hub.Count() != 1 {
		t.Errorf("Count = %d, want 1 (max exceeded)", hub.Count())
	}
}

func TestEnhancedHub_Login(t *testing.T) {
	var mu sync.Mutex
	var loginCalled bool
	hub := wsutil.NewEnhancedHub(wsutil.WithOnLogin(func(conn *wsutil.Conn, info *wsutil.ConnInfo) error {
		mu.Lock()
		defer mu.Unlock()
		loginCalled = true
		return nil
	}))

	// Simulate register + login without a real server.
	upgrader := wsutil.NewUpgrader(wsutil.DefaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r)
		defer conn.Close()
		info := &wsutil.ConnInfo{RemoteAddr: r.RemoteAddr}
		hub.Register(conn, info)
		hub.Login(conn, "user-1", "alice")
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	// Connect as client.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !loginCalled {
		t.Error("onLogin callback was not called")
	}
}

func TestEnhancedHub_Login_RejectedByCallback(t *testing.T) {
	hub := wsutil.NewEnhancedHub(wsutil.WithOnLogin(func(conn *wsutil.Conn, info *wsutil.ConnInfo) error {
		return errors.New("auth failed")
	}))

	upgrader := wsutil.NewUpgrader(wsutil.DefaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r)
		defer conn.Close()
		info := &wsutil.ConnInfo{}
		hub.Register(conn, info)
		err := hub.Login(conn, "user-1", "alice")
		if err == nil {
			t.Error("Login should have failed")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)
}

func TestEnhancedHub_OnCloseCallback(t *testing.T) {
	var mu sync.Mutex
	var closeCalled bool
	hub := wsutil.NewEnhancedHub(wsutil.WithOnClose(func(conn *wsutil.Conn, info *wsutil.ConnInfo, reason error) {
		mu.Lock()
		defer mu.Unlock()
		closeCalled = true
	}))

	srv, wsURL := newEnhancedTestServer(t, hub)
	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	time.Sleep(50 * time.Millisecond)

	conn.Close()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !closeCalled {
		t.Error("onClose callback was not called")
	}
	srv.Close()
}

func TestEnhancedHub_Broadcast(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Broadcast from server side.
	hub.BroadcastText("hello-all")

	// Read the broadcast message.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != "hello-all" {
		t.Errorf("Got %q, want hello-all", string(data))
	}
}

func TestEnhancedHub_BroadcastFiltered(t *testing.T) {
	hub := wsutil.NewEnhancedHub()

	upgrader := wsutil.NewUpgrader(wsutil.DefaultConfig())
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r)
		defer conn.Close()
		hub.Register(conn, &wsutil.ConnInfo{Name: r.URL.Query().Get("name")})
		time.Sleep(200 * time.Millisecond)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	conn1, _ := wsutil.Dial(context.Background(), wsURL+"?name=alice", wsutil.DefaultConfig())
	defer conn1.Close()
	conn2, _ := wsutil.Dial(context.Background(), wsURL+"?name=bob", wsutil.DefaultConfig())
	defer conn2.Close()
	time.Sleep(50 * time.Millisecond)

	// Broadcast only to "alice".
	hub.BroadcastTextFiltered("hi-alice", func(info *wsutil.ConnInfo) bool {
		return info.Name == "alice"
	})

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn1.Read()
	if err != nil {
		t.Fatalf("conn1 Read failed: %v", err)
	}
	if string(data) != "hi-alice" {
		t.Errorf("conn1 got %q, want hi-alice", string(data))
	}

	// conn2 should not receive the message.
	conn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn2.Read()
	if err == nil {
		t.Error("conn2 should not have received the filtered message")
	}
}

func TestEnhancedHub_SendTo(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	infos := hub.Infos()
	if len(infos) != 1 {
		t.Fatalf("Infos count = %d, want 1", len(infos))
	}
	id := infos[0].ID

	err := hub.SendTextTo(id, "direct-msg")
	if err != nil {
		t.Fatalf("SendTextTo failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != "direct-msg" {
		t.Errorf("Got %q, want direct-msg", string(data))
	}
}

func TestEnhancedHub_SendTo_NotFound(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	err := hub.SendTextTo("nonexistent", "msg")
	if err != wsutil.ErrNotRegistered {
		t.Errorf("Error = %v, want ErrNotRegistered", err)
	}
}

func TestEnhancedHub_UpdateMetadata(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	// Get the server-side connection from the hub.
	conns := hub.Conns()
	if len(conns) != 1 {
		t.Fatalf("Conns count = %d, want 1", len(conns))
	}
	var serverConn *wsutil.Conn
	for c := range conns {
		serverConn = c
	}

	hub.UpdateMetadata(serverConn, "role", "admin")

	infos := hub.Infos()
	if infos[0].Metadata["role"] != "admin" {
		t.Errorf("Metadata[role] = %v, want admin", infos[0].Metadata["role"])
	}
}

func TestEnhancedHub_Close(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	if hub.Count() != 1 {
		t.Errorf("Count = %d, want 1", hub.Count())
	}

	hub.Close()
	time.Sleep(50 * time.Millisecond)

	if hub.Count() != 0 {
		t.Errorf("Count after Close = %d, want 0", hub.Count())
	}
	srv.Close()
}

func TestEnhancedHub_Get(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	infos := hub.Infos()
	if len(infos) != 1 {
		t.Fatalf("Infos count = %d", len(infos))
	}

	info := hub.Get(infos[0].Conn)
	if info == nil {
		t.Error("Get returned nil")
	}
}

func TestEnhancedHub_Conns(t *testing.T) {
	hub := wsutil.NewEnhancedHub()
	srv, wsURL := newEnhancedTestServer(t, hub)
	defer srv.Close()

	conn, _ := wsutil.Dial(context.Background(), wsURL, wsutil.DefaultConfig())
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	conns := hub.Conns()
	if len(conns) != 1 {
		t.Errorf("Conns count = %d, want 1", len(conns))
	}
}
