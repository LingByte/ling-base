// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"context"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Channel
// ──────────────────────────────────────────────

func TestChannel_NameTypeEnabled(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok}, WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	ch := NewChannel("email-primary", m)

	assert.Equal(t, "email-primary", ch.Name())
	assert.Equal(t, notification.TypeEmail, ch.Type())
	assert.True(t, ch.Enabled())

	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
}

func TestChannel_Send_HTML(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok}, WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	ch := NewChannel("email-primary", m)

	msg := notification.NewEmailMessage("to@x.com", "subj", "text body")
	msg.HTML = "<p>html body</p>"
	err := ch.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, 1, ok.htmlCalls)
	assert.Equal(t, 0, ok.textCalls)
	assert.Equal(t, "subj", ok.lastSubject)
	assert.Equal(t, "<p>html body</p>", ok.lastBody)
}

func TestChannel_Send_Text(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok}, WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	ch := NewChannel("email-primary", m)

	msg := notification.NewEmailMessage("to@x.com", "subj", "text body")
	err := ch.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, 0, ok.htmlCalls)
	assert.Equal(t, 1, ok.textCalls)
	assert.Equal(t, "text body", ok.lastBody)
}

func TestChannel_Send_NoMailer(t *testing.T) {
	ch := NewChannel("email-primary", nil)
	err := ch.Send(context.Background(), notification.NewEmailMessage("to@x.com", "s", "b"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no mailer")
}

// ──────────────────────────────────────────────
// Integration with Dispatcher
// ──────────────────────────────────────────────

func TestChannel_RegisteredWithDispatcher(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok}, WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	ch := NewChannel("email-primary", m)

	d := notification.NewDispatcher()
	d.AddChannel(ch)

	msg := notification.NewEmailMessage("to@x.com", "subj", "text body")
	require.NoError(t, d.Send(context.Background(), msg))
	assert.Equal(t, 1, ok.sendCount)
}
