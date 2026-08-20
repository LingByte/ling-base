// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// ReplacePlaceholders
// ──────────────────────────────────────────────

func TestReplacePlaceholders_WithVars(t *testing.T) {
	tpl := "Hello {{.Name}}, your code is {{.Code}}."
	out := ReplacePlaceholders(tpl, map[string]any{"Name": "Alice", "Code": 12345})
	assert.Equal(t, "Hello Alice, your code is 12345.", out)
}

func TestReplacePlaceholders_NilVars(t *testing.T) {
	assert.Equal(t, "hello", ReplacePlaceholders("hello", nil))
}

func TestReplacePlaceholders_EmptyVars(t *testing.T) {
	assert.Equal(t, "hello", ReplacePlaceholders("hello", map[string]any{}))
}

func TestReplacePlaceholders_MissingKey(t *testing.T) {
	tpl := "Hello {{.Name}}, code {{.Code}}."
	out := ReplacePlaceholders(tpl, map[string]any{"Name": "Bob"})
	assert.Equal(t, "Hello Bob, code {{.Code}}.", out)
}

func TestReplacePlaceholders_EmptyTemplate(t *testing.T) {
	out := ReplacePlaceholders("", map[string]any{"Name": "Alice"})
	assert.Equal(t, "", out)
}

func TestReplacePlaceholders_MultipleOccurrences(t *testing.T) {
	tpl := "{{.X}} and {{.X}} plus {{.Y}}"
	out := ReplacePlaceholders(tpl, map[string]any{"X": "a", "Y": "b"})
	assert.Equal(t, "a and a plus b", out)
}

func TestReplacePlaceholders_MalformedPlaceholder(t *testing.T) {
	// A malformed placeholder that doesn't match the regex should be left as-is.
	result := ReplacePlaceholders("Hello {{ broken", map[string]any{"broken": "x"})
	assert.Equal(t, "Hello {{ broken", result)
}

func TestReplacePlaceholders_IntValue(t *testing.T) {
	result := ReplacePlaceholders("Code: {{.Code}}", map[string]any{"Code": 12345})
	assert.Equal(t, "Code: 12345", result)
}

func TestReplacePlaceholders_MultiplePlaceholders(t *testing.T) {
	result := ReplacePlaceholders("{{.Greeting}} {{.Name}}, code {{.Code}}", map[string]any{
		"Greeting": "Hello",
		"Name":     "Bob",
		"Code":     1234,
	})
	assert.Equal(t, "Hello Bob, code 1234", result)
}

// ──────────────────────────────────────────────
// ParseSender
// ──────────────────────────────────────────────

func TestParseSender_WithName(t *testing.T) {
	name, addr, err := ParseSender("Alice <alice@example.com>", "Fallback")
	require.NoError(t, err)
	assert.Equal(t, "Alice", name)
	assert.Equal(t, "alice@example.com", addr)
}

func TestParseSender_WithoutName(t *testing.T) {
	name, addr, err := ParseSender("alice@example.com", "Fallback")
	require.NoError(t, err)
	assert.Equal(t, "Fallback", name)
	assert.Equal(t, "alice@example.com", addr)
}

func TestParseSender_EmptyString(t *testing.T) {
	_, _, err := ParseSender("", "Fallback")
	assert.Error(t, err)
}

func TestParseSender_InvalidAddress(t *testing.T) {
	_, _, err := ParseSender("not-an-email", "Fallback")
	assert.Error(t, err)
}

func TestParseSender_EmptyNameInBrackets(t *testing.T) {
	name, addr, err := ParseSender("<alice@example.com>", "Fallback")
	require.NoError(t, err)
	assert.Equal(t, "Fallback", name)
	assert.Equal(t, "alice@example.com", addr)
}

// ──────────────────────────────────────────────
// DefaultRetryPolicy
// ──────────────────────────────────────────────

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	assert.Equal(t, 3, p.MaxAttempts)
	assert.Equal(t, 1*time.Second, p.InitialBackoff)
	assert.Equal(t, 10*time.Second, p.MaxBackoff)
}

// ──────────────────────────────────────────────
// Status constants
// ──────────────────────────────────────────────

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "sent", StatusSent)
	assert.Equal(t, "failed", StatusFailed)
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "delivered", StatusDelivered)
	assert.Equal(t, "soft_bounce", StatusSoftBounce)
	assert.Equal(t, "invalid", StatusInvalid)
}
