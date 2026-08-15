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

func TestUCloudProvider_Kind(t *testing.T) {
	p, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2",
	})
	require.NoError(t, err)
	assert.Equal(t, ProviderUCloud, p.Kind())
}

func TestUCloudProvider_MissingConfig(t *testing.T) {
	_, err := NewUCloudProvider(ProviderConfig{"public_key": "pk"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public_key and private_key")
}

func TestUCloudProvider_MissingProjectID(t *testing.T) {
	_, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")
}

func TestUCloudProvider_MissingRegion(t *testing.T) {
	_, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region")
}

func TestUCloudProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "SendUSMSMessage", r.URL.Query().Get("Action"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Action":"SendUSMSMessageResponse","RetCode":0,"Message":"","SessionNo":"session-789"}`)
	}))
	defer srv.Close()

	p, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2",
		"api_base": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{
		Template: "tpl-1", Data: map[string]string{"code": "1234"},
	}))
	require.NoError(t, err)
	assert.True(t, res.Accepted)
	assert.Equal(t, "session-789", res.MessageID)
}

func TestUCloudProvider_Send_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Action":"SendUSMSMessageResponse","RetCode":230,"Message":"invalid template","SessionNo":""}`)
	}))
	defer srv.Close()

	p, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2",
		"api_base": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), phoneReq(Message{Template: "bad"}))
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "invalid template")
}

func TestUCloudProvider_RequiresTemplate(t *testing.T) {
	p, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2",
	})
	require.NoError(t, err)
	_, err = p.Send(context.Background(), phoneReq(Message{Content: "hello"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires templateId")
}

func TestUCloudProvider_Send_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	p, err := NewUCloudProvider(ProviderConfig{
		"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2", "api_base": srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "TPL_001"},
	})
	require.Error(t, err)
	assert.False(t, res.Accepted)
}

// ──────────────────────────────────────────────
// UCloud helpers coverage
// ──────────────────────────────────────────────

func TestUCloudEncodeValue(t *testing.T) {
	assert.Equal(t, "", ucloudEncodeValue(nil))
	assert.Equal(t, "hello", ucloudEncodeValue("hello"))
	assert.Equal(t, "true", ucloudEncodeValue(true))
	assert.Equal(t, "false", ucloudEncodeValue(false))
	assert.Equal(t, "42", ucloudEncodeValue(42))
	assert.Equal(t, "42", ucloudEncodeValue(int64(42)))
	assert.Equal(t, "42", ucloudEncodeValue(uint64(42)))
	assert.Equal(t, "42", ucloudEncodeValue(float64(42)))
	assert.Equal(t, "42.5", ucloudEncodeValue(float64(42.5)))
	assert.Equal(t, `{"a":"b"}`, ucloudEncodeValue(map[string]string{"a": "b"}))
	assert.Equal(t, `{"a":"b"}`, ucloudEncodeValue(map[string]any{"a": "b"}))
}

func TestUCloudFlattenParams(t *testing.T) {
	params := map[string]any{
		"Action": "Test",
		"List":   []string{"a", "b"},
		"Map":    map[string]any{"key": "val"},
		"Nil":    nil,
	}
	out := ucloudFlattenParams(params)
	assert.Equal(t, "Test", out["Action"])
	assert.Equal(t, "a", out["List.0"])
	assert.Equal(t, "b", out["List.1"])
	assert.Equal(t, "val", out["Map.key"])
	_, hasNil := out["Nil"]
	assert.False(t, hasNil)
}

func TestUCloudSignString(t *testing.T) {
	params := map[string]any{
		"Action":    "Test",
		"PublicKey": "pk",
	}
	signStr := ucloudSignString(params, "privateKey")
	assert.Contains(t, signStr, "ActionTest")
	assert.Contains(t, signStr, "PublicKeypk")
	assert.Contains(t, signStr, "privateKey")
}
