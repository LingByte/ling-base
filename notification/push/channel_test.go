// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/notification"
)

func TestChannel_NameTypeEnabled(t *testing.T) {
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: &MockProvider{ResultMessageID: "x"}, Enabled: true},
	})
	ch := NewChannel("push-primary", ms)
	assert.Equal(t, "push-primary", ch.Name())
	assert.Equal(t, notification.TypePush, ch.Type())
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
	ch := NewChannel("push-primary", ms)

	err := ch.Send(context.Background(), notification.Message{
		Type:  notification.TypePush,
		To:    "device-token-123",
		Title: "hello",
		Body:  "world",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "device-token-123", mock.SentRequests[0].To[0].Token)
	assert.Equal(t, "hello", mock.SentRequests[0].Notification.Title)
	assert.Equal(t, "world", mock.SentRequests[0].Notification.Body)
}

func TestChannel_Send_UsesDataToken(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type:  notification.TypePush,
		Data:  map[string]any{"token": "data-token"},
		Title: "hi",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "data-token", mock.SentRequests[0].To[0].Token)
}

func TestChannel_Send_UsesDeviceTokenKey(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type: notification.TypePush,
		Data: map[string]any{"device_token": "dt-token"},
		Body: "hi",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, "dt-token", mock.SentRequests[0].To[0].Token)
}

func TestChannel_Send_ExtrasPlatform(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type:   notification.TypePush,
		To:     "tok",
		Body:   "hi",
		Extras: map[string]any{"platform": "android"},
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.Equal(t, PlatformAndroid, mock.SentRequests[0].To[0].Platform)
}

func TestChannel_Send_NoRecipient(t *testing.T) {
	ch := NewChannel("c", NewMultiSender([]SenderChannel{
		{Name: "m", Provider: &MockProvider{}, Enabled: true},
	}))
	err := ch.Send(context.Background(), notification.Message{
		Type:  notification.TypePush,
		Title: "hi",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipient device token")
}

func TestChannel_Send_NilSender(t *testing.T) {
	ch := &Channel{name: "c", sender: nil, enabled: true}
	err := ch.Send(context.Background(), notification.Message{
		Type: notification.TypePush,
		To:   "tok",
		Body: "hi",
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
		Type: notification.TypePush,
		To:   "tok",
		Body: "hi",
	})
	require.Error(t, err)
}

func TestChannel_Send_DataExcludesTokenKey(t *testing.T) {
	mock := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "m", Provider: mock, Enabled: true},
	})
	ch := NewChannel("c", ms)
	err := ch.Send(context.Background(), notification.Message{
		Type: notification.TypePush,
		Data: map[string]any{"token": "tok", "order_id": "123"},
		Body: "hi",
	})
	require.NoError(t, err)
	require.Len(t, mock.SentRequests, 1)
	assert.NotContains(t, mock.SentRequests[0].Notification.Data, "token")
	assert.Equal(t, "123", mock.SentRequests[0].Notification.Data["order_id"])
}

func TestGuessPlatform(t *testing.T) {
	assert.Equal(t, PlatformIOS, guessPlatform(notification.Message{}))
	assert.Equal(t, PlatformAndroid, guessPlatform(notification.Message{
		Extras: map[string]any{"platform": "android"},
	}))
	assert.Equal(t, PlatformHuawei, guessPlatform(notification.Message{
		Data: map[string]any{"platform": "huawei"},
	}))
	assert.Equal(t, PlatformIOS, guessPlatform(notification.Message{
		Extras: map[string]any{"platform": "unknown"},
	}))
}

func TestTokensFromAny(t *testing.T) {
	out := tokensFromAny("abc", PlatformIOS)
	assert.Len(t, out, 1)
	assert.Equal(t, "abc", out[0].Token)

	out = tokensFromAny([]string{"x", "y"}, PlatformAndroid)
	assert.Len(t, out, 1)
	assert.Equal(t, "x", out[0].Token)

	out = tokensFromAny([]any{"a", "b"}, PlatformHuawei)
	assert.Len(t, out, 2)
	assert.Equal(t, "a", out[0].Token)
	assert.Equal(t, "b", out[1].Token)

	out = tokensFromAny(123, PlatformIOS)
	assert.Empty(t, out)
}

func TestChannel_ImplementsNotificationChannel(t *testing.T) {
	var _ notification.Channel = (*Channel)(nil)
}
