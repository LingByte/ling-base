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

func TestYuntongxunProvider_Kind(t *testing.T) {
	p, err := NewYuntongxunProvider(ProviderConfig{
		"app_id": "test-id", "account_sid": "sid", "account_token": "token",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderYuntongxun, p.Kind())
}

func TestYuntongxunProvider_MissingConfig(t *testing.T) {
	_, err := NewYuntongxunProvider(ProviderConfig{"app_id": "test-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_id, account_sid and account_token")
}

func TestYuntongxunProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/2013-12-26/Accounts/")
		assert.Contains(t, r.URL.Path, "/SMS/TemplateSMS")
		assert.NotEmpty(t, r.URL.Query().Get("sig"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"statusCode":"000000","statusMsg":"ok","templateSMS":{"smsMessageSid":"sid-123","dateCreated":"2026"}}`)
	}))
	defer srv.Close()

	p, err := NewYuntongxunProvider(ProviderConfig{
		"app_id": "test-id", "account_sid": "sid", "account_token": "token",
		"endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "tpl-1", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "sid-123", res.MessageID)
}

func TestYuntongxunProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"statusCode":"160038","statusMsg":"invalid template","templateSMS":{}}`)
	}))
	defer srv.Close()

	p, err := NewYuntongxunProvider(ProviderConfig{
		"app_id": "test-id", "account_sid": "sid", "account_token": "token",
		"endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "bad"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid template")
}

func TestYuntongxunProvider_RequiresTemplate(t *testing.T) {
	p, err := NewYuntongxunProvider(ProviderConfig{
		"app_id": "test-id", "account_sid": "sid", "account_token": "token",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires template")
}

func TestYuntongxunProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewYuntongxunProvider(ProviderConfig{
		"app_id": "a", "account_sid": "sid", "account_token": "tok", "endpoint": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
