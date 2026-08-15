// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package inbox

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Channel basics
// ──────────────────────────────────────────────

func TestChannel_Basics(t *testing.T) {
	store := NewMemoryStore()
	ch := NewChannel("inbox-primary", store)

	assert.Equal(t, "inbox-primary", ch.Name())
	assert.Equal(t, notification.TypeInbox, ch.Type())
	assert.True(t, ch.Enabled())

	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
}

// ──────────────────────────────────────────────
// Channel.Send
// ──────────────────────────────────────────────

func TestChannel_Send_Success(t *testing.T) {
	store := NewMemoryStore()
	ch := NewChannel("inbox-primary", store)

	// full inbox message
	err := ch.Send(context.Background(), notification.Message{
		Type:        notification.TypeInbox,
		UserID:      "u1",
		Title:       "Welcome",
		Content:     "Hello there",
		ActionURL:   "https://example.com",
		ActionLabel: "Open",
	})
	require.NoError(t, err)

	got, err := store.GetByID("u1", "1")
	require.NoError(t, err)
	assert.Equal(t, "Welcome", got.Title)
	assert.Equal(t, "Hello there", got.Content)
	assert.Equal(t, "https://example.com", got.ActionURL)
	assert.Equal(t, "Open", got.ActionLabel)

	// fallback fields: To/Subject/Body when UserID/Title/Content empty
	err = ch.Send(context.Background(), notification.Message{
		Type:    notification.TypeInbox,
		To:      "u2",
		Subject: "Fallback Title",
		Body:    "Fallback Body",
	})
	require.NoError(t, err)
	got, err = store.GetByID("u2", "2")
	require.NoError(t, err)
	assert.Equal(t, "Fallback Title", got.Title)
	assert.Equal(t, "Fallback Body", got.Content)
}

func TestChannel_Send_Failure(t *testing.T) {
	store := NewMemoryStore()
	ch := NewChannel("inbox-primary", store)

	// missing userID (and To) -> error
	err := ch.Send(context.Background(), notification.Message{
		Type:    notification.TypeInbox,
		Subject: "No user",
		Body:    "x",
	})
	assert.Error(t, err)

	// missing title -> error
	err = ch.Send(context.Background(), notification.Message{
		Type:    notification.TypeInbox,
		UserID:  "u1",
		Content: "x",
	})
	assert.Error(t, err)
}

func TestChannel_NoStore(t *testing.T) {
	ch := NewChannel("inbox-primary", nil)
	err := ch.Send(context.Background(), notification.Message{
		Type:   notification.TypeInbox,
		UserID: "u1",
		Title:  "x",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no store")
}

// ──────────────────────────────────────────────
// Compile-time interface checks
// ──────────────────────────────────────────────

var _ notification.Channel = (*Channel)(nil)
