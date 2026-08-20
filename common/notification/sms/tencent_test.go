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

func TestNewTencentProvider_Config(t *testing.T) {
	p, err := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"sign_name":  "LingByte",
		"region":     "ap-beijing",
		"endpoint":   "sms.example.com",
	})
	require.NoError(t, err)
	tp, ok := p.(*TencentProvider)
	require.True(t, ok)
	assert.Equal(t, "APPID", tp.cfg.SdkAppID)
	assert.Equal(t, "SID", tp.cfg.SecretID)
	assert.Equal(t, "SKEY", tp.cfg.SecretKey)
	assert.Equal(t, "LingByte", tp.cfg.SignName)
	assert.Equal(t, "ap-beijing", tp.cfg.Region)
	assert.Equal(t, "sms.example.com", tp.cfg.Endpoint)
}

func TestNewTencentProvider_Defaults(t *testing.T) {
	p, err := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
	})
	require.NoError(t, err)
	tp := p.(*TencentProvider)
	assert.Equal(t, "ap-guangzhou", tp.cfg.Region)
	assert.Equal(t, "sms.tencentcloudapi.com", tp.cfg.Endpoint)
}

func TestNewTencentProvider_MissingCreds(t *testing.T) {
	_, err := NewTencentProvider(ProviderConfig{"sdk_app_id": "APPID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestTencentProvider_Kind(t *testing.T) {
	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
	})
	assert.Equal(t, ProviderTencent, p.Kind())
}

func TestTencentProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		assert.NotEmpty(t, r.Header.Get("Authorization"))
		assert.Equal(t, "SendSms", r.Header.Get("X-TC-Action"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"SendStatusSet":[{"SerialNo":"sn-1","PhoneNumber":"+8613800000000","Code":"Ok","Message":"send success"}],"RequestId":"req-1"}}`))
	}))
	defer srv.Close()

	p, err := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"sign_name":  "LingByte",
		"endpoint":   srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000", CountryCode: 86}},
		Message: Message{Template: "TPL_001", Data: map[string]string{"k": "v"}},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Accepted)
	assert.Equal(t, "Ok", res.Status)
	assert.Equal(t, "sn-1", res.MessageID)
}

func TestTencentProvider_Send_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"Error":{"Code":"AuthFailure","Message":"secret invalid"},"RequestId":"req-2"}}`))
	}))
	defer srv.Close()

	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"endpoint":   srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "secret invalid")
}

func TestTencentProvider_Send_BadStatusSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"SendStatusSet":[],"RequestId":"req-3"}}`))
	}))
	defer srv.Close()

	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"endpoint":   srv.URL,
	})
	_, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty SendStatusSet")
}

func TestTencentProvider_Send_FailedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"SendStatusSet":[{"SerialNo":"","Code":"LimitExceeded","Message":"too many"}],"RequestId":"req-4"}}`))
	}))
	defer srv.Close()

	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"endpoint":   srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "too many")
}

func TestTencentProvider_Send_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"endpoint":   srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}

func TestEndpointURL(t *testing.T) {
	assert.Equal(t, "https://sms.tencentcloudapi.com", endpointURL("sms.tencentcloudapi.com"))
	assert.Equal(t, "http://localhost:1234", endpointURL("http://localhost:1234"))
	assert.Equal(t, "https://x.com", endpointURL("https://x.com"))
}

func TestHostOf(t *testing.T) {
	assert.Equal(t, "sms.tencentcloudapi.com", hostOf("https://sms.tencentcloudapi.com"))
	assert.Equal(t, "localhost:1234", hostOf("http://localhost:1234"))
	assert.Equal(t, "plain.com", hostOf("plain.com"))
}

func TestTencentProvider_Send_InvalidRequest(t *testing.T) {
	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestTencentProvider_BuildSignature(t *testing.T) {
	p, _ := NewTencentProvider(ProviderConfig{
		"sdk_app_id": "APPID",
		"secret_id":  "SID",
		"secret_key": "SKEY",
		"endpoint":   "sms.tencentcloudapi.com",
	})
	tp := p.(*TencentProvider)
	headers := tp.buildSignature([]byte(`{"a":"b"}`), 1700000000, "2023-11-14", "sms")
	assert.NotEmpty(t, headers["Authorization"])
	assert.Contains(t, headers["Authorization"], "TC3-HMAC-SHA256")
	assert.Contains(t, headers["Authorization"], "Credential:SID/")
	assert.Equal(t, "SendSms", headers["X-TC-Action"])
	assert.Equal(t, "2021-01-11", headers["X-TC-Version"])
}

func TestHexEncode(t *testing.T) {
	assert.Equal(t, "00ff", hexEncode([]byte{0x00, 0xff}))
	assert.Equal(t, "", hexEncode(nil))
}
