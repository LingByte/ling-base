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

func TestJuheProvider_Kind(t *testing.T) {
	p, err := NewJuheProvider(ProviderConfig{"app_key": "test-key"})
	require.NoError(t, err)
	assert.Equal(t, ProviderJuhe, p.Kind())
}

func TestJuheProvider_MissingConfig(t *testing.T) {
	_, err := NewJuheProvider(ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_key")
}

func TestJuheProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error_code":0,"reason":"ok","result":{}}`)
	}))
	defer srv.Close()

	p, err := NewJuheProvider(ProviderConfig{
		"app_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "tpl-1", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
}

func TestJuheProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"error_code":205401,"reason":"bad template","result":null}`)
	}))
	defer srv.Close()

	p, err := NewJuheProvider(ProviderConfig{
		"app_key": "test-key", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "bad"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad template")
}

func TestJuheProvider_RequiresTemplate(t *testing.T) {
	p, err := NewJuheProvider(ProviderConfig{"app_key": "test-key"})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires template id")
}

func TestJuheProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewJuheProvider(ProviderConfig{
		"app_key": "k", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "12345"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
