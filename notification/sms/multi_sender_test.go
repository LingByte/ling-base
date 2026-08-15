// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiSender_FirstSucceeds(t *testing.T) {
	a := &MockProvider{ResultMessageID: "a"}
	b := &MockProvider{ResultMessageID: "b"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
		{Name: "b", Provider: b, Enabled: true},
	})
	res, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.NoError(t, err)
	assert.Equal(t, "a", res.MessageID)
	assert.Len(t, a.SentRequests, 1)
	assert.Empty(t, b.SentRequests)
}

func TestMultiSender_FirstFailsSecondSucceeds(t *testing.T) {
	a := &MockProvider{ShouldFail: true}
	b := &MockProvider{ResultMessageID: "b"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
		{Name: "b", Provider: b, Enabled: true},
	})
	res, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "b", res.MessageID)
	assert.Len(t, a.SentRequests, 1)
	assert.Len(t, b.SentRequests, 1)
}

func TestMultiSender_AllFail(t *testing.T) {
	a := &MockProvider{ShouldFail: true}
	b := &MockProvider{ShouldFail: true}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
		{Name: "b", Provider: b, Enabled: true},
	})
	res, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, err.Error(), "all channels failed")
}

func TestMultiSender_NoEnabledChannels(t *testing.T) {
	a := &MockProvider{}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: false},
	})
	_, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled channels")
}

func TestMultiSender_NoChannels(t *testing.T) {
	ms := NewMultiSender(nil)
	_, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
}

func TestMultiSender_RoundRobin(t *testing.T) {
	a := &MockProvider{ResultMessageID: "a"}
	b := &MockProvider{ResultMessageID: "b"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
		{Name: "b", Provider: b, Enabled: true},
	})
	req := SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	}
	// first send uses index 0 -> a
	res, _ := ms.Send(context.Background(), req)
	assert.Equal(t, "a", res.MessageID)
	// after success, pointer advances to 1 -> b
	res, _ = ms.Send(context.Background(), req)
	assert.Equal(t, "b", res.MessageID)
	// wraps back to 0 -> a
	res, _ = ms.Send(context.Background(), req)
	assert.Equal(t, "a", res.MessageID)
}

func TestMultiSender_SetStartingIndex(t *testing.T) {
	a := &MockProvider{ResultMessageID: "a"}
	b := &MockProvider{ResultMessageID: "b"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
		{Name: "b", Provider: b, Enabled: true},
	})
	ms.SetStartingIndex(1)
	res, _ := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	assert.Equal(t, "b", res.MessageID)
}

func TestMultiSender_SkipsDisabledAndNil(t *testing.T) {
	a := &MockProvider{ResultMessageID: "a"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "disabled", Provider: &MockProvider{}, Enabled: false},
		{Name: "nil", Provider: nil, Enabled: true},
		{Name: "a", Provider: a, Enabled: true},
	})
	res, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.NoError(t, err)
	assert.Equal(t, "a", res.MessageID)
}

func TestMultiSender_CancelledContext(t *testing.T) {
	a := &MockProvider{ResultMessageID: "a"}
	ms := NewMultiSender([]SenderChannel{
		{Name: "a", Provider: a, Enabled: true},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ms.Send(ctx, SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
}

// errProvider returns a non-nil error result without a Go error to
// exercise the "all channels failed" path where err is nil.
type errProvider struct{}

func (errProvider) Kind() ProviderKind { return ProviderMock }
func (errProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	return &SendResult{Accepted: false, Status: "failed", Error: "rejected"}, nil
}

func TestMultiSender_RejectedResultNoGoError(t *testing.T) {
	ms := NewMultiSender([]SenderChannel{
		{Name: "r", Provider: errProvider{}, Enabled: true},
	})
	res, err := ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "1"}},
		Message: Message{Content: "x"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}

// ensure errors import is used
var _ = errors.New
