// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ MailReader = (*IMAPReader)(nil)

// ──────────────────────────────────────────────
// parseUID
// ──────────────────────────────────────────────

func TestParseUID_Valid(t *testing.T) {
	uid, err := parseUID("12345")
	assert.NoError(t, err)
	assert.Equal(t, uint32(12345), uint32(uid))
}

func TestParseUID_Empty(t *testing.T) {
	_, err := parseUID("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseUID_NonNumeric(t *testing.T) {
	_, err := parseUID("abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-numeric")
}

func TestParseUID_Mixed(t *testing.T) {
	_, err := parseUID("12abc34")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-numeric")
}

func TestParseUID_Zero(t *testing.T) {
	_, err := parseUID("0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseUID_LargeNumber(t *testing.T) {
	uid, err := parseUID("4294967295") // max uint32
	assert.NoError(t, err)
	assert.Equal(t, uint32(4294967295), uint32(uid))
}

// ──────────────────────────────────────────────
// IMAPReader closed state
// ──────────────────────────────────────────────

func TestIMAPReader_ReadMessages_Closed(t *testing.T) {
	r := &IMAPReader{client: nil}
	_, err := r.ReadMessages(5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestIMAPReader_ReadRecentMessages_Closed(t *testing.T) {
	r := &IMAPReader{client: nil}
	_, err := r.ReadRecentMessages(5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestIMAPReader_MarkRead_Closed(t *testing.T) {
	r := &IMAPReader{client: nil}
	err := r.MarkRead("123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestIMAPReader_DeleteMessage_Closed(t *testing.T) {
	r := &IMAPReader{client: nil}
	err := r.DeleteMessage("123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestIMAPReader_Close_NilClient(t *testing.T) {
	r := &IMAPReader{client: nil}
	err := r.Close()
	assert.NoError(t, err) // no-op when already closed
}

// parseUID error paths are covered by TestParseUID_* tests above.
// MarkRead/DeleteMessage with invalid UID on a nil client will hit
// the "closed" check first, which is already tested.

// ──────────────────────────────────────────────
// IMAPConfig defaults
// ──────────────────────────────────────────────

func TestNewIMAPReader_EmptyHost(t *testing.T) {
	_, err := NewIMAPReader(IMAPConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}
