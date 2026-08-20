// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_NewProviderFromKind_Mock(t *testing.T) {
	p, err := NewProviderFromKind(ProviderMock, ProviderConfig{})
	require.NoError(t, err)
	assert.Equal(t, ProviderMock, p.Kind())
}

func TestRegistry_UnknownKind(t *testing.T) {
	_, err := NewProviderFromKind(ProviderKind("nope"), ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider kind")
}

func TestRegistry_RegisteredKinds(t *testing.T) {
	kinds := RegisteredKinds()
	// init() registers mock, aliyun, tencent, twilio
	assert.Contains(t, kinds, ProviderMock)
	assert.Contains(t, kinds, ProviderAliyun)
	assert.Contains(t, kinds, ProviderTencent)
	assert.Contains(t, kinds, ProviderTwilio)
	// sorted
	for i := 1; i < len(kinds); i++ {
		assert.True(t, kinds[i-1] <= kinds[i], "kinds not sorted: %v", kinds)
	}
}

func TestRegistry_RegisterAndOverwrite(t *testing.T) {
	kind := ProviderKind("custom-test")
	called := 0
	RegisterProvider(kind, func(cfg ProviderConfig) (Provider, error) {
		called++
		return &MockProvider{}, nil
	})
	p, err := NewProviderFromKind(kind, ProviderConfig{})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, 1, called)

	// overwrite
	RegisterProvider(kind, func(cfg ProviderConfig) (Provider, error) {
		return &MockProvider{ResultMessageID: "v2"}, nil
	})
	p2, err := NewProviderFromKind(kind, ProviderConfig{})
	require.NoError(t, err)
	_, ok := p2.(*MockProvider)
	assert.True(t, ok)
}

func TestRegistry_MustRegisterProvider_Duplicate(t *testing.T) {
	kind := ProviderKind("must-dup")
	MustRegisterProvider(kind, func(cfg ProviderConfig) (Provider, error) {
		return &MockProvider{}, nil
	})
	assert.Panics(t, func() {
		MustRegisterProvider(kind, func(cfg ProviderConfig) (Provider, error) {
			return &MockProvider{}, nil
		})
	})
}

// ──────────────────────────────────────────────
// Registry coverage — all providers
// ──────────────────────────────────────────────

func TestNewProviderFromKind_AllProviders(t *testing.T) {
	tests := []struct {
		kind ProviderKind
		cfg  ProviderConfig
	}{
		{ProviderHuawei, ProviderConfig{"app_key": "k", "app_secret": "s", "sender": "snd"}},
		{ProviderYunpian, ProviderConfig{"api_key": "k"}},
		{ProviderSubmail, ProviderConfig{"app_id": "id", "app_key": "k"}},
		{ProviderLuosimao, ProviderConfig{"api_key": "k"}},
		{ProviderYuntongxun, ProviderConfig{"app_id": "a", "account_sid": "sid", "account_token": "tok"}},
		{ProviderHuyi, ProviderConfig{"api_id": "id", "api_key": "k"}},
		{ProviderJuhe, ProviderConfig{"app_key": "k"}},
		{ProviderBaidu, ProviderConfig{"ak": "a", "sk": "s", "signature_id": "sig"}},
		{ProviderHuaxin, ProviderConfig{"user_id": "u", "password": "p", "base_url": "http://example.com"}},
		{ProviderChuanglan, ProviderConfig{"account": "a", "password": "p"}},
		{ProviderRongcloud, ProviderConfig{"app_key": "k", "app_secret": "s"}},
		{ProviderTiniyo, ProviderConfig{"account_sid": "sid", "token": "tok", "from": "+123"}},
		{ProviderUCloud, ProviderConfig{"public_key": "pk", "private_key": "sk", "project_id": "pid", "region": "cn-bj2"}},
		{ProviderNeteaseYunx, ProviderConfig{"app_key": "k", "app_secret": "s"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			p, err := NewProviderFromKind(tt.kind, tt.cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.kind, p.Kind())
		})
	}
}

func TestNewProviderFromKind_UnknownKind(t *testing.T) {
	_, err := NewProviderFromKind("unknown", nil)
	require.Error(t, err)
}

func TestRegisteredKinds_IncludesAll(t *testing.T) {
	kinds := RegisteredKinds()
	expected := []ProviderKind{
		ProviderAliyun, ProviderTencent, ProviderTwilio, ProviderHuawei,
		ProviderYunpian, ProviderSubmail, ProviderLuosimao,
		ProviderYuntongxun, ProviderHuyi, ProviderJuhe, ProviderBaidu,
		ProviderHuaxin, ProviderChuanglan, ProviderRongcloud,
		ProviderTiniyo, ProviderUCloud, ProviderNeteaseYunx, ProviderMock,
	}
	for _, k := range expected {
		assert.Contains(t, kinds, k, "provider %q should be registered", k)
	}
}
