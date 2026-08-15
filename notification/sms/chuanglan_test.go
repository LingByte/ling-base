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

func TestChuanglanProvider_Kind(t *testing.T) {
	p, err := NewChuanglanProvider(ProviderConfig{
		"account": "test-account", "password": "test-pass",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderChuanglan, p.Kind())
}

func TestChuanglanProvider_MissingConfig(t *testing.T) {
	_, err := NewChuanglanProvider(ProviderConfig{"account": "test-account"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account and password")
}

func TestChuanglanProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"0","msg":"ok","error":""}`)
	}))
	defer srv.Close()

	p, err := NewChuanglanProvider(ProviderConfig{
		"account": "test-account", "password": "test-pass", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
}

func TestChuanglanProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"1","msg":"bad phone","error":""}`)
	}))
	defer srv.Close()

	p, err := NewChuanglanProvider(ProviderConfig{
		"account": "test-account", "password": "test-pass", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad phone")
}

func TestChuanglanProvider_RequiresContent(t *testing.T) {
	p, err := NewChuanglanProvider(ProviderConfig{
		"account": "test-account", "password": "test-pass",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{}))
	require.Error(t, err)
	// ValidateBasic fires first for empty content+template.
	assert.Contains(t, err.Error(), "content or template is required")
}

func TestChuanglanProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewChuanglanProvider(ProviderConfig{
		"account": "a", "password": "p", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
