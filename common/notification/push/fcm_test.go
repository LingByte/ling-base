// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// genRSAPEM generates an RSA 2048 private key and returns its PEM encoding.
func genRSAPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
}

// genServiceAccountJSON builds a service account JSON key string.
func genServiceAccountJSON(t *testing.T, tokenURI string) string {
	t.Helper()
	return fmt.Sprintf(`{
		"type": "service_account",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"private_key": %q,
		"token_uri": %q
	}`, genRSAPEM(t), tokenURI)
}

func TestNewFCMProvider_Config(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, err := NewFCMProvider(ProviderConfig{
		"project_id":          "my-project",
		"service_account_key": sa,
		"endpoint":            "https://example.com",
	})
	require.NoError(t, err)
	fp, ok := p.(*FCMProvider)
	require.True(t, ok)
	assert.Equal(t, "my-project", fp.cfg.ProjectID)
	assert.Equal(t, "https://example.com", fp.cfg.Endpoint)
	assert.Equal(t, "test@test-project.iam.gserviceaccount.com", fp.sa.ClientEmail)
}

func TestNewFCMProvider_MissingCreds(t *testing.T) {
	_, err := NewFCMProvider(ProviderConfig{"project_id": "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestNewFCMProvider_BadServiceAccount(t *testing.T) {
	_, err := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": "not json",
	})
	require.Error(t, err)
}

func TestFCMProvider_Kind(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
	})
	assert.Equal(t, ProviderFCM, p.Kind())
}

func TestFCMProvider_Endpoint(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "my-project",
		"service_account_key": sa,
	})
	fp := p.(*FCMProvider)
	assert.Equal(t, "https://fcm.googleapis.com/v1/projects/my-project/messages:send", fp.endpoint())
}

func TestFCMProvider_BuildMessage(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
	})
	fp := p.(*FCMProvider)
	body, err := fp.buildMessage(SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformAndroid}},
		Notification: Notification{Title: "T", Body: "B", Icon: "ic", Color: "#FF0000"},
		Priority:     "high",
		CollapseKey:  "ck",
		TimeToLive:   60,
	})
	require.NoError(t, err)
	var msg fcmMessage
	require.NoError(t, json.Unmarshal(body, &msg))
	assert.Equal(t, "tok", msg.Message.Token)
	assert.Equal(t, "T", msg.Message.Notification.Title)
	assert.Equal(t, "B", msg.Message.Notification.Body)
	require.NotNil(t, msg.Message.Android)
	assert.Equal(t, "high", msg.Message.Android.Priority)
	assert.Equal(t, "ck", msg.Message.Android.CollapseKey)
	assert.Equal(t, "60s", msg.Message.Android.TTL)
	require.NotNil(t, msg.Message.Android.Notification)
	assert.Equal(t, "ic", msg.Message.Android.Notification.Icon)
	assert.Equal(t, "#FF0000", msg.Message.Android.Notification.Color)
}

func TestFCMProvider_BuildMessage_iOS(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
	})
	fp := p.(*FCMProvider)
	body, err := fp.buildMessage(SendRequest{
		To:           []DeviceToken{{Token: "tok", Platform: PlatformIOS}},
		Notification: Notification{Title: "T", Body: "B", Badge: 5, Sound: "default"},
	})
	require.NoError(t, err)
	var msg fcmMessage
	require.NoError(t, json.Unmarshal(body, &msg))
	require.NotNil(t, msg.Message.APNs)
	aps, ok := msg.Message.APNs.Payload["aps"].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 5, aps["badge"])
	assert.Equal(t, "default", aps["sound"])
}

func TestFCMProvider_Send_Success(t *testing.T) {
	// Mock OAuth2 token server.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	sa := genServiceAccountJSON(t, tokenSrv.URL)

	// Mock FCM messages:send server.
	var gotAuth, gotBody string
	fcmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"projects/p/messages/123"}`))
	}))
	defer fcmSrv.Close()

	p, err := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
		"endpoint":            fcmSrv.URL,
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
	assert.Equal(t, "projects/p/messages/123", res.MessageID)
	assert.Equal(t, "Bearer test-token", gotAuth)
	assert.Contains(t, gotBody, "device-tok")
}

func TestFCMProvider_Send_ProviderError(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	sa := genServiceAccountJSON(t, tokenSrv.URL)

	fcmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":400,"message":"INVALID_ARGUMENT","status":"INVALID_ARGUMENT"}}`))
	}))
	defer fcmSrv.Close()

	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
		"endpoint":            fcmSrv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok"}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "INVALID_ARGUMENT")
}

func TestFCMProvider_Send_InvalidRequest(t *testing.T) {
	sa := genServiceAccountJSON(t, "https://oauth2.googleapis.com/token")
	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestFCMProvider_Send_TokenExchangeFailure(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer tokenSrv.Close()

	sa := genServiceAccountJSON(t, tokenSrv.URL)

	p, _ := NewFCMProvider(ProviderConfig{
		"project_id":          "p",
		"service_account_key": sa,
		"endpoint":            "https://fcm.example.com",
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok"}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, err.Error(), "token exchange")
}

func TestParseServiceAccount_MissingFields(t *testing.T) {
	_, err := parseServiceAccount(`{"client_email":"a@b.com"}`, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestParseServiceAccount_BadJSON(t *testing.T) {
	_, err := parseServiceAccount("not json", "")
	require.Error(t, err)
}
