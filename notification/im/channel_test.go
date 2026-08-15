// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"fmt"
	"testing"

	"github.com/LingByte/ling-base/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Channel
// ──────────────────────────────────────────────

// fakeProvider is a test double for Provider.
type fakeProvider struct {
	kind      string
	lastMsg   Message
	sendErr   error
	sendCalls int
}

func (f *fakeProvider) Kind() string { return f.kind }
func (f *fakeProvider) Send(ctx context.Context, msg Message) error {
	f.sendCalls++
	f.lastMsg = msg
	return f.sendErr
}

func TestChannel_NameTypeEnabled(t *testing.T) {
	fp := &fakeProvider{kind: "wecom"}
	c := NewChannel("ops-bot", fp)
	assert.Equal(t, "ops-bot", c.Name())
	assert.Equal(t, notification.TypeIM, c.Type())
	assert.True(t, c.Enabled())
}

func TestChannel_SetEnabled(t *testing.T) {
	c := NewChannel("x", &fakeProvider{})
	c.SetEnabled(false)
	assert.False(t, c.Enabled())
}

func TestChannel_Send_Success(t *testing.T) {
	fp := &fakeProvider{kind: "wecom"}
	c := NewChannel("ops-bot", fp)

	err := c.Send(context.Background(), notification.NewIMMessage("Title", "Content"))
	require.NoError(t, err)
	assert.Equal(t, 1, fp.sendCalls)
	assert.Equal(t, "Title", fp.lastMsg.Title)
	assert.Equal(t, "Content", fp.lastMsg.Content)
}

func TestChannel_Send_FallbackFields(t *testing.T) {
	fp := &fakeProvider{kind: "feishu"}
	c := NewChannel("fb", fp)

	// Message without Title/Content but with Subject/Body should fall back.
	msg := notification.Message{Type: notification.TypeIM, Subject: "S", Body: "B"}
	err := c.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "S", fp.lastMsg.Title)
	assert.Equal(t, "B", fp.lastMsg.Content)
}

func TestChannel_Send_ProviderError(t *testing.T) {
	fp := &fakeProvider{kind: "wecom", sendErr: assertAnError("boom")}
	c := NewChannel("ops-bot", fp)
	err := c.Send(context.Background(), notification.NewIMMessage("t", "c"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestChannel_Send_NilProvider(t *testing.T) {
	c := NewChannel("empty", nil)
	err := c.Send(context.Background(), notification.NewIMMessage("t", "c"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no provider")
}

func TestChannel_Send_Disabled(t *testing.T) {
	fp := &fakeProvider{kind: "wecom"}
	c := NewChannel("test", fp)
	c.SetEnabled(false)

	// Even when disabled, Send should still work (enabled is just a flag
	// for the dispatcher to check).
	err := c.Send(context.Background(), notification.NewIMMessage("t", "c"))
	require.NoError(t, err)
	assert.Equal(t, 1, fp.sendCalls)
}

func TestChannel_Send_EmptyMessage(t *testing.T) {
	fp := &fakeProvider{kind: "wecom"}
	c := NewChannel("test", fp)

	// Empty message should still be sent.
	err := c.Send(context.Background(), notification.NewIMMessage("", ""))
	require.NoError(t, err)
	assert.Equal(t, 1, fp.sendCalls)
	assert.Equal(t, "", fp.lastMsg.Title)
	assert.Equal(t, "", fp.lastMsg.Content)
}

// ──────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────

// assertAnError returns a new error with the given message.
func assertAnError(msg string) error {
	return fmt.Errorf("%s", msg)
}

// getFactory retrieves a factory from the registry (for testing).
func getFactory(name string) (func(string) (Provider, error), bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}
