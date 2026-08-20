// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// genECPEM generates an EC P-256 private key and returns its PEM encoding.
func genECPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func TestNewAPNsProvider_Config(t *testing.T) {
	keyPEM := genECPEM(t)
	p, err := NewAPNsProvider(ProviderConfig{
		"team_id":    "TEAM123",
		"key_id":     "KEY456",
		"auth_key":   keyPEM,
		"bundle_id":  "com.example.app",
		"production": false,
		"endpoint":   "https://example.com",
	})
	require.NoError(t, err)
	ap, ok := p.(*APNsProvider)
	require.True(t, ok)
	assert.Equal(t, "TEAM123", ap.cfg.TeamID)
	assert.Equal(t, "KEY456", ap.cfg.KeyID)
	assert.Equal(t, "com.example.app", ap.cfg.BundleID)
	assert.False(t, ap.cfg.Production)
	assert.Equal(t, "https://example.com", ap.cfg.Endpoint)
}

func TestNewAPNsProvider_MissingCreds(t *testing.T) {
	_, err := NewAPNsProvider(ProviderConfig{"team_id": "TEAM"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestAPNsProvider_Kind(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
	})
	assert.Equal(t, ProviderAPNs, p.Kind())
}

func TestAPNsProvider_Endpoint(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
	})
	ap := p.(*APNsProvider)
	assert.Equal(t, "https://api.push.apple.com", ap.endpoint())

	p2, _ := NewAPNsProvider(ProviderConfig{
		"team_id":    "TEAM",
		"key_id":     "KEY",
		"auth_key":   keyPEM,
		"production": false,
	})
	ap2 := p2.(*APNsProvider)
	assert.Equal(t, "https://api.sandbox.push.apple.com", ap2.endpoint())
}

func TestAPNsProvider_AuthToken(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
	})
	ap := p.(*APNsProvider)
	tok, err := ap.authToken()
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
	// JWT has three base64url parts
	assert.Len(t, strings.Split(tok, "."), 3)
}

func TestAPNsProvider_BuildPayload(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
	})
	ap := p.(*APNsProvider)
	body, err := ap.buildPayload(Notification{Title: "T", Body: "B", Badge: 3, Sound: "default"})
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	aps, ok := payload["aps"].(map[string]any)
	require.True(t, ok)
	alert, ok := aps["alert"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "T", alert["title"])
	assert.Equal(t, "B", alert["body"])
	assert.EqualValues(t, 3, aps["badge"])
	assert.Equal(t, "default", aps["sound"])
}

func TestAPNsProvider_BuildPayload_Silent(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
	})
	ap := p.(*APNsProvider)
	body, err := ap.buildPayload(Notification{Data: map[string]string{"k": "v"}})
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	aps := payload["aps"].(map[string]any)
	assert.EqualValues(t, 1, aps["content-available"])
}

func TestAPNsProvider_Send_Success(t *testing.T) {
	keyPEM := genECPEM(t)
	var gotPath, gotAuth, gotTopic, gotPushType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("authorization")
		gotTopic = r.Header.Get("apns-topic")
		gotPushType = r.Header.Get("apns-push-type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p, err := NewAPNsProvider(ProviderConfig{
		"team_id":   "TEAM",
		"key_id":    "KEY",
		"auth_key":  keyPEM,
		"bundle_id": "com.example.app",
		"endpoint":  srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "device-token-abc", Platform: PlatformIOS}},
		Notification: Notification{Title: "hello", Body: "world"},
		Priority:     "high",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Accepted)
	assert.Equal(t, "sent", res.Status)
	assert.Contains(t, gotPath, "/3/device/device-token-abc")
	assert.Contains(t, gotAuth, "bearer ")
	assert.Equal(t, "com.example.app", gotTopic)
	assert.Equal(t, "alert", gotPushType)
	assert.Contains(t, string(gotBody), "hello")
}

func TestAPNsProvider_Send_ProviderError(t *testing.T) {
	keyPEM := genECPEM(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		w.Write([]byte(`{"reason":"BadDeviceToken"}`))
	}))
	defer srv.Close()

	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":   "TEAM",
		"key_id":    "KEY",
		"auth_key":  keyPEM,
		"bundle_id": "com.example.app",
		"endpoint":  srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "bad-token", Platform: PlatformIOS}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "BadDeviceToken")
}

func TestAPNsProvider_Send_InvalidRequest(t *testing.T) {
	keyPEM := genECPEM(t)
	p, _ := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": keyPEM,
		"endpoint": "https://example.com",
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestAPNsProvider_Send_BadKey(t *testing.T) {
	p, err := NewAPNsProvider(ProviderConfig{
		"team_id":  "TEAM",
		"key_id":   "KEY",
		"auth_key": "not-a-valid-key",
	})
	require.NoError(t, err)
	res, err := p.Send(context.Background(), SendRequest{
		To:           []DeviceToken{{Token: "tok"}},
		Notification: Notification{Title: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}

func TestParseECPrivateKey_Invalid(t *testing.T) {
	_, err := parseECPrivateKey("not pem")
	require.Error(t, err)
}

func TestParseECPrivateKey_Empty(t *testing.T) {
	_, err := parseECPrivateKey("")
	require.Error(t, err)
}
