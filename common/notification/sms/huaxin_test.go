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

func TestHuaxinProvider_Kind(t *testing.T) {
	p, err := NewHuaxinProvider(ProviderConfig{
		"user_id": "test-user", "password": "test-pass", "base_url": "http://example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderHuaxin, p.Kind())
}

func TestHuaxinProvider_MissingConfig(t *testing.T) {
	_, err := NewHuaxinProvider(ProviderConfig{"user_id": "test-user"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_id and password")
}

func TestHuaxinProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sms.aspx", r.URL.Path)
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><returnsms><returnstatus>Success</returnstatus><message>ok</message><taskID>123456</taskID><successCounts>1</successCounts></returnsms>`)
	}))
	defer srv.Close()

	p, err := NewHuaxinProvider(ProviderConfig{
		"user_id": "test-user", "password": "test-pass", "base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "123456", res.MessageID)
}

func TestHuaxinProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><returnsms><returnstatus>Fail</returnstatus><message>bad account</message><taskID></taskID><successCounts>0</successCounts></returnsms>`)
	}))
	defer srv.Close()

	p, err := NewHuaxinProvider(ProviderConfig{
		"user_id": "test-user", "password": "test-pass", "base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "bad account")
}

func TestHuaxinProvider_RequiresContent(t *testing.T) {
	p, err := NewHuaxinProvider(ProviderConfig{
		"user_id": "test-user", "password": "test-pass", "base_url": "http://example.com",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{}))
	require.Error(t, err)
	// ValidateBasic fires first for empty content+template.
	assert.Contains(t, err.Error(), "content or template is required")
}

func TestHuaxinProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewHuaxinProvider(ProviderConfig{
		"user_id": "u", "password": "p", "base_url": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}
