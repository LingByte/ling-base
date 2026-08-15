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

func TestBaiduProvider_Kind(t *testing.T) {
	p, err := NewBaiduProvider(ProviderConfig{
		"ak": "test-ak", "sk": "test-sk", "signature_id": "sig-1",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderBaidu, p.Kind())
}

func TestBaiduProvider_MissingConfig(t *testing.T) {
	_, err := NewBaiduProvider(ProviderConfig{"ak": "test-ak"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ak and sk")
}

func TestBaiduProvider_MissingSignature(t *testing.T) {
	_, err := NewBaiduProvider(ProviderConfig{
		"ak": "test-ak", "sk": "test-sk",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature_id")
}

func TestBaiduProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/sendSms", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"1000","message":"ok","data":[{"messageId":"msg-123","phoneNumber":"8613800138000"}]}`)
	}))
	defer srv.Close()

	p, err := NewBaiduProvider(ProviderConfig{
		"ak": "test-ak", "sk": "test-sk", "signature_id": "sig-1",
		"endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "tpl-1", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "msg-123", res.MessageID)
}

func TestBaiduProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":"1001","message":"invalid signature","data":[]}`)
	}))
	defer srv.Close()

	p, err := NewBaiduProvider(ProviderConfig{
		"ak": "test-ak", "sk": "test-sk", "signature_id": "sig-1",
		"endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "tpl-1"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid signature")
}

func TestBaiduProvider_RequiresTemplate(t *testing.T) {
	p, err := NewBaiduProvider(ProviderConfig{
		"ak": "test-ak", "sk": "test-sk", "signature_id": "sig-1",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires template")
}

func TestBaiduProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewBaiduProvider(ProviderConfig{
		"ak": "a", "sk": "s", "signature_id": "sig", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
