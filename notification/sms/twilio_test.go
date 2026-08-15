// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTwilioProvider_Config(t *testing.T) {
	p, err := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
		"from":        "+15550000000",
		"endpoint":    "https://example.com",
	})
	require.NoError(t, err)
	tp, ok := p.(*TwilioProvider)
	require.True(t, ok)
	assert.Equal(t, "SID", tp.cfg.AccountSID)
	assert.Equal(t, "TOK", tp.cfg.Token)
	assert.Equal(t, "+15550000000", tp.cfg.From)
	assert.Equal(t, "https://example.com", tp.cfg.Endpoint)
}

func TestNewTwilioProvider_Defaults(t *testing.T) {
	p, err := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
	})
	require.NoError(t, err)
	tp := p.(*TwilioProvider)
	assert.Equal(t, "https://api.twilio.com", tp.cfg.Endpoint)
}

func TestNewTwilioProvider_MissingCreds(t *testing.T) {
	_, err := NewTwilioProvider(ProviderConfig{"account_sid": "SID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestTwilioProvider_Kind(t *testing.T) {
	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
	})
	assert.Equal(t, ProviderTwilio, p.Kind())
}

func TestTwilioProvider_Send_Success(t *testing.T) {
	var gotPath, user, pass string
	var gotForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		user, pass, _ = r.BasicAuth()
		body, _ := io.ReadAll(r.Body)
		gotForm = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sid":"SM123","status":"queued","error_code":0}`))
	}))
	defer srv.Close()

	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
		"from":        "+15550000000",
		"endpoint":    srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "15551234567", CountryCode: 1}},
		Message: Message{Content: "hello"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Accepted)
	assert.Equal(t, "queued", res.Status)
	assert.Equal(t, "SM123", res.MessageID)
	assert.Contains(t, gotPath, "/Accounts/SID/Messages.json")
	assert.Equal(t, "SID", user)
	assert.Equal(t, "TOK", pass)
	assert.Contains(t, gotForm, "To=%2B115551234567")
	assert.Contains(t, gotForm, "Body=hello")
}

func TestTwilioProvider_Send_FromExtras(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "From=%2B19999999999")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sid":"SM1","status":"sent","error_code":0}`))
	}))
	defer srv.Close()

	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
		"from":        "+15550000000",
		"endpoint":    srv.URL,
	})
	_, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "15551234567", CountryCode: 1}},
		Message: Message{Content: "hi"},
		Extras:  map[string]any{"from": "+19999999999"},
	})
	require.NoError(t, err)
}

func TestTwilioProvider_Send_Failed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sid":"SM2","status":"failed","error_code":21211,"error_message":"invalid number"}`))
	}))
	defer srv.Close()

	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
		"endpoint":    srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "0000"}},
		Message: Message{Content: "hi"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid number")
}

func TestTwilioProvider_Send_InvalidRequest(t *testing.T) {
	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestTwilioProvider_Send_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	p, _ := NewTwilioProvider(ProviderConfig{
		"account_sid": "SID",
		"token":       "TOK",
		"endpoint":    srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "15551234567"}},
		Message: Message{Content: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}
