// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHuaweiProvider_Kind(t *testing.T) {
	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret", "sender": "test-sender",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderHuawei, p.Kind())
}

func TestHuaweiProvider_MissingConfig(t *testing.T) {
	_, err := NewHuaweiProvider(ProviderConfig{"app_key": "test-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_key and app_secret")
}

func TestHuaweiProvider_MissingSender(t *testing.T) {
	_, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sender")
}

func TestHuaweiProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sms/batchSendSms/v1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"000000","description":"success","result":[{"status":"000000","smsMsgId":"msg-123"}]}`)
	}))
	defer srv.Close()

	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret",
		"sender": "test-sender", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "SMS_123", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "msg-123", res.MessageID)
}

func TestHuaweiProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"E200027","description":"invalid template","result":[]}`)
	}))
	defer srv.Close()

	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret",
		"sender": "test-sender", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "bad-template"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid template")
}

func TestHuaweiProvider_RequiresTemplate(t *testing.T) {
	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret", "sender": "test-sender",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires template")
}

func TestHuaweiProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "k", "app_secret": "s", "sender": "snd", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}

func TestHuaweiProvider_Send_EmptyRecipients(t *testing.T) {
	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "k", "app_secret": "s", "sender": "snd",
	})
	require.NoError(t, err)

	_, err = p.Send(context.Background(), SendRequest{
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
}

func TestHuaweiProvider_Send_CancelledContext(t *testing.T) {
	p, err := NewHuaweiProvider(ProviderConfig{
		"app_key": "k", "app_secret": "s", "sender": "snd",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = p.Send(ctx, SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
}
