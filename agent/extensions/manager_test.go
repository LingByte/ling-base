package extensions

import (
	"testing"

	"github.com/LingByte/ling-base/agent/extproto"
)

func TestNewManager(t *testing.T) {
	m := New([]string{"/tmp/test-exts"}, extproto.HelloAckFromHost{
		HostVersion: "test",
		Provider:    "relay",
		Model:       "test-model",
		CWD:         "/tmp",
	})
	if m == nil {
		t.Fatal("New returned nil")
	}
	if len(m.rootDirs) != 1 {
		t.Errorf("expected 1 root dir, got %d", len(m.rootDirs))
	}
}

func TestCommandsEmpty(t *testing.T) {
	m := New(nil, extproto.HelloAckFromHost{})
	cmds := m.Commands()
	if len(cmds) != 0 {
		t.Errorf("expected 0 commands, got %d", len(cmds))
	}
}

func TestFormatExtensionsEmpty(t *testing.T) {
	m := New(nil, extproto.HelloAckFromHost{})
	got := m.FormatExtensions()
	if got != "No extensions loaded." {
		t.Errorf("expected 'No extensions loaded.', got %q", got)
	}
}

func TestHasCommand(t *testing.T) {
	m := New(nil, extproto.HelloAckFromHost{})
	if m.HasCommand("anything") {
		t.Error("HasCommand should return false with no extensions")
	}
}

func TestHasTool(t *testing.T) {
	m := New(nil, extproto.HelloAckFromHost{})
	if m.HasTool("anything") {
		t.Error("HasTool should return false with no extensions")
	}
}
