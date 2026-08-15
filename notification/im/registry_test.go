// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Registry
// ──────────────────────────────────────────────

func TestRegisteredProviders(t *testing.T) {
	names := RegisteredProviders()
	assert.Contains(t, names, "wecom")
	assert.Contains(t, names, "feishu")
}

func TestNewProviderFromConfig_WeCom(t *testing.T) {
	cfg := `{"webhook_url":"https://example.com/wecom","corp_id":"c","agent_id":"a","secret":"s"}`
	p, err := NewProviderFromConfig("wecom", cfg)
	require.NoError(t, err)
	assert.Equal(t, "wecom", p.Kind())
}

func TestNewProviderFromConfig_Feishu(t *testing.T) {
	cfg := `{"webhook_url":"https://example.com/feishu","secret":"sec","app_id":"id","app_secret":"as"}`
	p, err := NewProviderFromConfig("FEISHU", cfg)
	require.NoError(t, err)
	assert.Equal(t, "feishu", p.Kind())
}

func TestNewProviderFromConfig_Unknown(t *testing.T) {
	_, err := NewProviderFromConfig("unknown", `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestNewProviderFromConfig_InvalidJSON(t *testing.T) {
	_, err := NewProviderFromConfig("wecom", `{not json`)
	require.Error(t, err)
}

func TestRegisterProvider_Custom(t *testing.T) {
	RegisterProvider("custom-test", func(configJSON string) (Provider, error) {
		return NewWeComProvider(WeComConfig{WebhookURL: configJSON}), nil
	})
	defer func() {
		registryMu.Lock()
		delete(registry, "custom-test")
		registryMu.Unlock()
	}()

	p, err := NewProviderFromConfig("Custom-Test", "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "wecom", p.Kind())
}

func TestRegisterProvider_Overwrite(t *testing.T) {
	original, exists := getFactory("wecom")
	require.True(t, exists)

	RegisterProvider("wecom", func(configJSON string) (Provider, error) {
		return nil, fmt.Errorf("overwritten")
	})

	_, err := NewProviderFromConfig("wecom", `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overwritten")

	// Restore original.
	registryMu.Lock()
	registry["wecom"] = original
	registryMu.Unlock()
}

func TestNewProviderFromConfig_EmptyConfig(t *testing.T) {
	// Empty config should still work for wecom (it just creates with empty URL).
	p, err := NewProviderFromConfig("wecom", `{}`)
	require.NoError(t, err)
	assert.Equal(t, "wecom", p.Kind())
}
