// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

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
	assert.Contains(t, kinds, ProviderMock)
	assert.Contains(t, kinds, ProviderAPNs)
	assert.Contains(t, kinds, ProviderFCM)
	assert.Contains(t, kinds, ProviderUniPush)
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

func TestNewProviderFromKind_UnknownKind(t *testing.T) {
	_, err := NewProviderFromKind("unknown", nil)
	require.Error(t, err)
}

func TestRegisteredKinds_IncludesAll(t *testing.T) {
	kinds := RegisteredKinds()
	expected := []ProviderKind{ProviderAPNs, ProviderFCM, ProviderUniPush, ProviderMock}
	for _, k := range expected {
		assert.Contains(t, kinds, k, "provider %q should be registered", k)
	}
}
