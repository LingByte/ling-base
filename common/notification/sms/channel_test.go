// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/common/notification"
)

func TestChannel_NameTypeEnabled(t *testing.T) {
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: &MockProvider{ResultMessageID: "x"}, Enabled: true},
	})
	ch := NewChannel("sms-primary", ms)
	assert.Equal(t, "sms-primary", ch.Name())
	assert.Equal(t, notification.TypeSMS, ch.Type())
	assert.True(t, ch.Enabled())
}

func TestChannel_SetEnabled(t *testing.T) {
	ch := NewChannel("c", NewMultiSender(nil))
	assert.True(t, ch.Enabled())
	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
}

func TestChannel_Send_Success(t *testing.T) {
	mock := &MockProvider{ResultMessageID: "mid"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("sms-primary", ms)

	err := ch.Send(context.Background(), notification.Message{
		Type:        notification.TypeSMS,
		PhoneNumber: "13800000000",
		CountryCode: 86,
		Body:        "hello",
		SignName:    "LingByte",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "13800000000", mock.SentRequests[0].To[0].Number)
	assert.Equal(t, 86, mock.SentRequests[0].To[0].CountryCode)
	assert.Equal(t, "hello", mock.SentRequests[0].Message.Content)
	assert.Equal(t, "LingByte", mock.SentRequests[0].Message.SignName)
}

func TestChannel_Send_UsesToField(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type: notification.TypeSMS,
		To:   "8613800000000",
		Body: "hi",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "8613800000000", mock.SentRequests[0].To[0].Number)
}

func TestChannel_Send_TemplateAndData(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type:     notification.TypeSMS,
		To:       "13800000000",
		Template: "TPL_1",
		Data:     map[string]any{"code": "1234"},
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "TPL_1", mock.SentRequests[0].Message.Template)
	assert.Equal(t, "1234", mock.SentRequests[0].Message.Data["code"])
}

func TestChannel_Send_NoRecipient(t *testing.T) {
	ch := NewChannel("c", NewMultiSender([]SenderChannel{
		{Name: "m", Provider: &MockProvider{}, Enabled: true},
	}))
	err := ch.Send(context.Background(), notification.Message{
		Type: notification.TypeSMS,
		Body: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipient")
}

func TestChannel_Send_NilSender(t *testing.T) {
	ch := &Channel{name: "c", sender: nil, enabled: true}
	err := ch.Send(context.Background(), notification.Message{
		Type:        notification.TypeSMS,
		PhoneNumber: "13800000000",
		Body:        "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sender")
}

func TestChannel_Send_Failure(t *testing.T) {
	mock := &MockProvider{ShouldFail: true}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type:        notification.TypeSMS,
		PhoneNumber: "13800000000",
		Body:        "hi",
	})
	require.Error(t, err)
}

func TestGuessCountryCode(t *testing.T) {
	assert.Equal(t, 86, guessCountryCode("8613800000000"))
	assert.Equal(t, 1, guessCountryCode("15551234567"))
	assert.Equal(t, 0, guessCountryCode("999"))
	assert.Equal(t, 0, guessCountryCode(""))
}

func TestChannel_ImplementsNotificationChannel(t *testing.T) {
	var _ notification.Channel = (*Channel)(nil)
}
