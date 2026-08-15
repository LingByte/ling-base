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

func TestTiniyoProvider_Kind(t *testing.T) {
	p, err := NewTiniyoProvider(ProviderConfig{
		"account_sid": "sid", "token": "tok", "from": "+1000",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderTiniyo, p.Kind())
}

func TestTiniyoProvider_MissingConfig(t *testing.T) {
	_, err := NewTiniyoProvider(ProviderConfig{"account_sid": "sid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account_sid, token and from")
}

func TestTiniyoProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/2010-04-01/Accounts/")
		assert.Contains(t, r.URL.Path, "/Messages.json")
		user, _, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "sid", user)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"sid":"SM-123","status":"queued","message":""}`)
	}))
	defer srv.Close()

	p, err := NewTiniyoProvider(ProviderConfig{
		"account_sid": "sid", "token": "tok", "from": "+1000",
		"base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "SM-123", res.MessageID)
	assert.Equal(t, "queued", res.Status)
}

func TestTiniyoProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"sid":"","status":"","message":"invalid from number"}`)
	}))
	defer srv.Close()

	p, err := NewTiniyoProvider(ProviderConfig{
		"account_sid": "sid", "token": "tok", "from": "+1000",
		"base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid from number")
}

func TestTiniyoProvider_RequiresContent(t *testing.T) {
	p, err := NewTiniyoProvider(ProviderConfig{
		"account_sid": "sid", "token": "tok", "from": "+1000",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Template: "tpl"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires content")
}

func TestTiniyoProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewTiniyoProvider(ProviderConfig{
		"account_sid": "sid", "token": "tok", "from": "+1234567890", "base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
