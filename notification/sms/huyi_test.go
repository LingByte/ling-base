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

func TestHuyiProvider_Kind(t *testing.T) {
	p, err := NewHuyiProvider(ProviderConfig{
		"api_id": "test-id", "api_key": "test-key",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderHuyi, p.Kind())
}

func TestHuyiProvider_MissingConfig(t *testing.T) {
	_, err := NewHuyiProvider(ProviderConfig{"api_id": "test-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_id and api_key")
}

func TestHuyiProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"SubmitResult":{"smsid":"sms-123","code":2,"msg":"ok"}}`)
	}))
	defer srv.Close()

	p, err := NewHuyiProvider(ProviderConfig{
		"api_id": "test-id", "api_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "sms-123", res.MessageID)
}

func TestHuyiProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"SubmitResult":{"smsid":"","code":405,"msg":"bad content"}}`)
	}))
	defer srv.Close()

	p, err := NewHuyiProvider(ProviderConfig{
		"api_id": "test-id", "api_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad content")
}

func TestHuyiProvider_RequiresContentOrTemplate(t *testing.T) {
	p, err := NewHuyiProvider(ProviderConfig{
		"api_id": "test-id", "api_key": "test-key",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{}))
	require.Error(t, err)
	// ValidateBasic fires first for empty content+template.
	assert.Contains(t, err.Error(), "content or template is required")
}

func TestHuyiProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewHuyiProvider(ProviderConfig{
		"api_id": "id", "api_key": "k", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
