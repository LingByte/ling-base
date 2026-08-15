// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockProvider_Success(t *testing.T) {
	m := &MockProvider{ResultMessageID: "mid-1"}
	req := SendRequest{
		To:      []PhoneNumber{{Number: "13800000000", CountryCode: 86}},
		Message: Message{Content: "hi"},
	}
	res, err := m.Send(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, ProviderMock, res.Provider)
	assert.True(t, res.Accepted)
	assert.Equal(t, "mid-1", res.MessageID)
	assert.Equal(t, "sent", res.Status)
	assert.NotZero(t, res.SentAtUnix)
	assert.Len(t, m.SentRequests, 1)
}

func TestMockProvider_DefaultMessageID(t *testing.T) {
	m := &MockProvider{}
	res, err := m.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.NoError(t, err)
	assert.Equal(t, "mock-message-id", res.MessageID)
}

func TestMockProvider_Failure(t *testing.T) {
	m := &MockProvider{ShouldFail: true}
	res, err := m.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "configured failure")
	assert.Contains(t, err.Error(), "configured failure")
	assert.Len(t, m.SentRequests, 1)
}

func TestNewMockProvider_FromConfig(t *testing.T) {
	p, err := NewMockProvider(ProviderConfig{
		"should_fail": true,
		"message_id":  "cfg-mid",
		"status":      "queued",
	})
	require.NoError(t, err)
	m, ok := p.(*MockProvider)
	require.True(t, ok)
	assert.True(t, m.ShouldFail)
	assert.Equal(t, "cfg-mid", m.ResultMessageID)
	assert.Equal(t, "queued", m.ResultStatus)
}

func TestNewMockProvider_NilConfig(t *testing.T) {
	p, err := NewMockProvider(nil)
	require.NoError(t, err)
	assert.Equal(t, ProviderMock, p.Kind())
}

func TestMockProvider_CancelledContext(t *testing.T) {
	m := &MockProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.Send(ctx, SendRequest{To: []PhoneNumber{{Number: "1"}}, Message: Message{Content: "x"}})
	require.Error(t, err)
}

func TestToBool(t *testing.T) {
	b, ok := toBool(true)
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = toBool("true")
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = toBool("no")
	assert.True(t, ok)
	assert.False(t, b)

	b, ok = toBool(1)
	assert.True(t, ok)
	assert.True(t, b)

	b, ok = toBool(float64(0))
	assert.True(t, ok)
	assert.False(t, b)

	_, ok = toBool(nil)
	assert.False(t, ok)
}
