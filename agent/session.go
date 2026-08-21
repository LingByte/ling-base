// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/ling-base/relay"
	"github.com/google/uuid"
)

// Session is a persistent conversation state. It can be saved to disk
// and resumed later.
type Session struct {
	ID       string          `json:"id"`
	Model    string          `json:"model"`
	System   string          `json:"system"`
	Messages []relay.Message `json:"messages"`
	Created  time.Time       `json:"created"`
	Updated  time.Time       `json:"updated"`

	// Cost tracking (cumulative).
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NewSession creates a new session.
func NewSession(model, system string) *Session {
	return &Session{
		ID:      uuid.NewString(),
		Model:   model,
		System:  system,
		Created: time.Now(),
		Updated: time.Now(),
	}
}

// Save writes the session to a JSON file at the given path.
func (s *Session) Save(path string) error {
	s.Updated = time.Now()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("session: create dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("session: write: %w", err)
	}
	return nil
}

// LoadSession reads a session from a JSON file.
func LoadSession(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session: read: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("session: unmarshal: %w", err)
	}
	return &s, nil
}

// DefaultSessionDir returns the default session storage directory.
// - macOS: ~/Library/Application Support/ling-agent/sessions
// - Linux: ~/.local/state/ling-agent/sessions
// - Windows: %LOCALAPPDATA%\ling-agent\sessions
func DefaultSessionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	switch {
	case filepath.Separator == '\\' && os.Getenv("LOCALAPPDATA") != "":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "ling-agent", "sessions")
	case darwinOrLinuxHome(home):
		return filepath.Join(home, "Library", "Application Support", "ling-agent", "sessions")
	default:
		return filepath.Join(home, ".local", "state", "ling-agent", "sessions")
	}
}

func darwinOrLinuxHome(home string) bool {
	// Check if we're on macOS by looking for the Library directory.
	if _, err := os.Stat(filepath.Join(home, "Library")); err == nil {
		return true
	}
	return false
}

// SessionPath returns the full path for a session file.
func SessionPath(id string) string {
	return filepath.Join(DefaultSessionDir(), id+".json")
}

// SyncFromAgent copies the agent's current state into the session.
func (s *Session) SyncFromAgent(a *Agent) {
	s.Messages = a.Messages()
	s.Updated = time.Now()
}

// ApplyToAgent loads the session's state into an agent.
func (s *Session) ApplyToAgent(a *Agent) {
	a.SetMessages(s.Messages)
}
