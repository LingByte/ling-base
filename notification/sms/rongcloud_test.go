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

func TestRongcloudProvider_Kind(t *testing.T) {
	p, err := NewRongcloudProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderRongcloud, p.Kind())
}

func TestRongcloudProvider_MissingConfig(t *testing.T) {
	_, err := NewRongcloudProvider(ProviderConfig{"app_key": "test-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_key and app_secret")
}

func TestRongcloudProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sms/sendCode.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":200,"sessionId":"session-123","errorMessage":""}`)
	}))
	defer srv.Close()

	p, err := NewRongcloudProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "tpl-1"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "session-123", res.MessageID)
}

func TestRongcloudProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":1000,"sessionId":"","errorMessage":"bad template"}`)
	}))
	defer srv.Close()

	p, err := NewRongcloudProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "bad"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad template")
}

func TestRongcloudProvider_RequiresTemplate(t *testing.T) {
	p, err := NewRongcloudProvider(ProviderConfig{
		"app_key": "test-key", "app_secret": "test-secret",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires template")
}

func TestRongcloudProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewRongcloudProvider(ProviderConfig{
		"app_key": "k", "app_secret": "s", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
