// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ssh_test

import (
	"encoding/json"
	"testing"

	"github.com/LingByte/ling-base/common/ssh"
)

func TestConfig_Defaults(t *testing.T) {
	term := ssh.NewTerminal(ssh.Config{
		Host:     "localhost:22",
		User:     "root",
		Password: "secret",
	})

	// Access internal config via behavior — we test defaults by checking
	// that NewTerminal doesn't panic and accepts the config.
	if term == nil {
		t.Fatal("NewTerminal returned nil")
	}
}

func TestClientMessage_JSON(t *testing.T) {
	// Test command message.
	cmdMsg := ssh.ClientMessage{
		Type: ssh.MsgTypeCmd,
		Cmd:  "ls -la\n",
	}
	data, err := json.Marshal(cmdMsg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ssh.ClientMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Type != ssh.MsgTypeCmd {
		t.Errorf("Type = %q, want %q", decoded.Type, ssh.MsgTypeCmd)
	}
	if decoded.Cmd != "ls -la\n" {
		t.Errorf("Cmd = %q", decoded.Cmd)
	}

	// Test resize message.
	resizeMsg := ssh.ClientMessage{
		Type: ssh.MsgTypeResize,
		Cols: 120,
		Rows: 40,
	}
	data, err = json.Marshal(resizeMsg)
	if err != nil {
		t.Fatalf("Marshal resize: %v", err)
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal resize: %v", err)
	}
	if decoded.Type != ssh.MsgTypeResize {
		t.Errorf("Type = %q, want %q", decoded.Type, ssh.MsgTypeResize)
	}
	if decoded.Cols != 120 {
		t.Errorf("Cols = %d, want 120", decoded.Cols)
	}
	if decoded.Rows != 40 {
		t.Errorf("Rows = %d, want 40", decoded.Rows)
	}
}

func TestTerminal_Close_WithoutRun(t *testing.T) {
	term := ssh.NewTerminal(ssh.Config{
		Host:     "localhost:22",
		User:     "root",
		Password: "secret",
	})

	// Close should not panic even if Run was never called.
	if err := term.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestTerminal_Done_WithoutRun(t *testing.T) {
	term := ssh.NewTerminal(ssh.Config{
		Host:     "localhost:22",
		User:     "root",
		Password: "secret",
	})

	// Done should return a non-nil channel.
	done := term.Done()
	if done == nil {
		t.Error("Done returned nil channel")
	}
}

func TestDialClient_InvalidHost(t *testing.T) {
	_, err := ssh.DialClient(ssh.Config{
		Host:           "localhost:1", // port 1 should fail
		User:           "root",
		Password:       "secret",
		ConnectTimeout: 1, // 1ns timeout for fast failure
	})
	if err == nil {
		t.Error("DialClient should fail for invalid host")
	}
}

func TestRunCommand_InvalidHost(t *testing.T) {
	_, err := ssh.RunCommand(ssh.Config{
		Host:           "localhost:1",
		User:           "root",
		Password:       "secret",
		ConnectTimeout: 1,
	}, "echo hello")
	if err == nil {
		t.Error("RunCommand should fail for invalid host")
	}
}

func TestAuthMethods_NoAuth(t *testing.T) {
	// Should fail when no auth method is configured.
	_, err := ssh.DialClient(ssh.Config{
		Host:           "localhost:22",
		User:           "root",
		ConnectTimeout: 1,
	})
	if err == nil {
		t.Error("DialClient should fail without auth")
	}
}

// ──────────────────────────────────────────────
// Mock WebSocket connection
// ──────────────────────────────────────────────

type mockWSConn struct {
	readData  [][]byte
	readIdx   int
	written   [][]byte
	closeErr  error
}

func (m *mockWSConn) ReadMessage() (int, []byte, error) {
	if m.readIdx >= len(m.readData) {
		return 0, nil, m.closeErr
	}
	data := m.readData[m.readIdx]
	m.readIdx++
	return 1, data, nil
}

func (m *mockWSConn) WriteMessage(messageType int, data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.written = append(m.written, cp)
	return nil
}

func (m *mockWSConn) Close() error {
	return nil
}

func TestTerminal_Run_AuthFailure(t *testing.T) {
	term := ssh.NewTerminal(ssh.Config{
		Host:           "localhost:1",
		User:           "root",
		Password:       "secret",
		ConnectTimeout: 1,
	})

	ws := &mockWSConn{}
	err := term.Run(ws)
	if err == nil {
		t.Error("Run should fail for invalid SSH host")
	}

	// Should have written an error message to the WebSocket.
	if len(ws.written) == 0 {
		t.Error("WebSocket should have received an error message")
	}
}
