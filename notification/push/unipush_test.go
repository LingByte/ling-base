// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUniPushProvider_Config(t *testing.T) {
	p, err := NewUniPushProvider(ProviderConfig{
		"app_id":     "APP123",
		"app_key":    "KEY",
		"app_secret": "SECRET",
		"endpoint":   "https://example.com",
	})
	require.NoError(t, err)
	up, ok := p.(*UniPushProvider)
	require.True(t, ok)
	assert.Equal(t, "APP123", up.cfg.AppID)
	assert.Equal(t, "KEY", up.cfg.AppKey)
	assert.Equal(t, "SECRET", up.cfg.AppSecret)
	assert.Equal(t, "https://example.com", up.cfg.Endpoint)
}

func TestNewUniPushProvider_MissingCreds(t *testing.T) {
	_, err := NewUniPushProvider(ProviderConfig{"app_id": "a"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestUniPushProvider_Kind(t *testing.T) {
	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
	})
	assert.Equal(t, ProviderUniPush, p.Kind())
}

func TestUniPushProvider_Endpoint(t *testing.T) {
	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "app123",
		"app_secret": "s",
	})
	up := p.(*UniPushProvider)
	assert.Equal(t, "https://push-api.cloud.huawei.com/v1/app123/messages:send", up.endpoint())
}

func TestUniPushProvider_BuildMessage(t *testing.T) {
	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
	})
	up := p.(*UniPushProvider)
	body, err := up.buildMessage(SendRequest{
		To: []DeviceToken{
			{Token: "tok1", Platform: PlatformAndroid},
			{Token: "tok2", Platform: PlatformHuawei},
		},
		Notification: Notification{Title: "T", Body: "B", Icon: "ic", ClickAction: "https://example.com"},
		Priority:     "high",
		TimeToLive:   60,
	})
	require.NoError(t, err)
	var msg uniPushMessage
	require.NoError(t, json.Unmarshal(body, &msg))
	assert.Equal(t, []string{"tok1", "tok2"}, msg.Message.Token)
	assert.Equal(t, "T", msg.Message.Notification.Title)
	assert.Equal(t, "B", msg.Message.Notification.Body)
	assert.Equal(t, "ic", msg.Message.Notification.Icon)
	assert.Equal(t, 3, msg.Message.Notification.ClickAction.Type)
	assert.Equal(t, "https://example.com", msg.Message.Notification.ClickAction.URL)
	assert.Equal(t, "high", msg.Message.Android.Priority)
	assert.Equal(t, "60s", msg.Message.Android.TTL)
}

func TestUniPushProvider_Send_Success(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	var gotAuth, gotBody string
	pushSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"80000000","msg":"success","requestId":"req-123"}`))
	}))
	defer pushSrv.Close()

	p, err := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
		"endpoint":   pushSrv.URL,
		"token_uri":  tokenSrv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "device-tok", Platform: PlatformAndroid}},
		Notification: Notification{Title: "hello", Body: "world"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Accepted)
	assert.Equal(t, "sent", res.Status)
	assert.Equal(t, "req-123", res.MessageID)
	assert.Equal(t, "Bearer test-token", gotAuth)
	assert.Contains(t, gotBody, "device-tok")
}

func TestUniPushProvider_Send_ProviderError(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	pushSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"80000001","msg":"invalid token","requestId":"req-2"}`))
	}))
	defer pushSrv.Close()

	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
		"endpoint":   pushSrv.URL,
		"token_uri":  tokenSrv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformAndroid}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid token")
}

func TestUniPushProvider_Send_iOSRejected(t *testing.T) {
	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformIOS}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "APNs")
}

func TestUniPushProvider_Send_InvalidRequest(t *testing.T) {
	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestUniPushProvider_Send_TokenExchangeFailure(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid"}`))
	}))
	defer tokenSrv.Close()

	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
		"endpoint":   "https://push.example.com",
		"token_uri":  tokenSrv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformAndroid}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, err.Error(), "token exchange")
}

func TestUniPushProvider_Send_HTTPError(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	pushSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":"internal","msg":"server error"}`))
	}))
	defer pushSrv.Close()

	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
		"endpoint":   pushSrv.URL,
		"token_uri":  tokenSrv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformAndroid}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}

func TestUniPushProvider_AccessToken_Cached(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"cached-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	p, _ := NewUniPushProvider(ProviderConfig{
		"app_id":     "a",
		"app_secret": "s",
		"token_uri":  tokenSrv.URL,
	})
	up := p.(*UniPushProvider)

	tok1, err := up.accessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cached-token", tok1)

	// Second call should return cached token without hitting the server.
	tok2, err := up.accessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, tok1, tok2)
}
