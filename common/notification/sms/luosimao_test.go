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

func TestLuosimaoProvider_Kind(t *testing.T) {
	p, err := NewLuosimaoProvider(ProviderConfig{"api_key": "test-key"})
	require.NoError(t, err)
	assert.Equal(t, ProviderLuosimao, p.Kind())
}

func TestLuosimaoProvider_MissingConfig(t *testing.T) {
	_, err := NewLuosimaoProvider(ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

func TestLuosimaoProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "api", user)
		assert.Equal(t, "test-key", pass)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":0,"msg":"ok"}`)
	}))
	defer srv.Close()

	p, err := NewLuosimaoProvider(ProviderConfig{
		"api_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
}

func TestLuosimaoProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error":10000,"msg":"bad mobile"}`)
	}))
	defer srv.Close()

	p, err := NewLuosimaoProvider(ProviderConfig{
		"api_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad mobile")
}

func TestLuosimaoProvider_RequiresContent(t *testing.T) {
	p, err := NewLuosimaoProvider(ProviderConfig{"api_key": "test-key"})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Template: "tpl"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires content")
}

func TestLuosimaoProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewLuosimaoProvider(ProviderConfig{
		"api_key": "k", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}

func TestLuosimaoProvider_Send_EmptyRecipients(t *testing.T) {
	p, err := NewLuosimaoProvider(ProviderConfig{"api_key": "k"})
	require.NoError(t, err)

	_, err = p.Send(context.Background(), SendRequest{
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
}
