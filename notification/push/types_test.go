// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package push

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBasic_Valid(t *testing.T) {
	req := SendRequest{
		To:           []DeviceToken{{Token: "abc123", Platform: PlatformIOS}},
		Notification: Notification{Title: "hello"},
	}
	assert.NoError(t, ValidateBasic(req))
}

func TestValidateBasic_BodyOnly(t *testing.T) {
	req := SendRequest{
		To:           []DeviceToken{{Token: "abc123", Platform: PlatformAndroid}},
		Notification: Notification{Body: "hello"},
	}
	assert.NoError(t, ValidateBasic(req))
}

func TestValidateBasic_EmptyTo(t *testing.T) {
	req := SendRequest{
		Notification: Notification{Title: "hello"},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipients")
}

func TestValidateBasic_EmptyToken(t *testing.T) {
	req := SendRequest{
		To:           []DeviceToken{{Token: "  "}},
		Notification: Notification{Title: "hello"},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty token")
}

func TestValidateBasic_EmptyTitleAndBody(t *testing.T) {
	req := SendRequest{
		To: []DeviceToken{{Token: "abc123"}},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title or body")
}

func TestFirstDeviceToken_Valid(t *testing.T) {
	req := SendRequest{
		To: []DeviceToken{
			{Token: "first", Platform: PlatformIOS},
			{Token: "second", Platform: PlatformAndroid},
		},
	}
	tok, err := FirstDeviceToken(req)
	require.NoError(t, err)
	assert.Equal(t, "first", tok.Token)
	assert.Equal(t, PlatformIOS, tok.Platform)
}

func TestFirstDeviceToken_Empty(t *testing.T) {
	_, err := FirstDeviceToken(SendRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients")
}

func TestFirstDeviceToken_EmptyToken(t *testing.T) {
	_, err := FirstDeviceToken(SendRequest{
		To: []DeviceToken{{Token: "  "}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty token")
}

func TestProviderKind_Constants(t *testing.T) {
	assert.Equal(t, ProviderKind("apns"), ProviderAPNs)
	assert.Equal(t, ProviderKind("fcm"), ProviderFCM)
	assert.Equal(t, ProviderKind("unipush"), ProviderUniPush)
	assert.Equal(t, ProviderKind("mock"), ProviderMock)
}

func TestPlatform_Constants(t *testing.T) {
	assert.Equal(t, Platform("ios"), PlatformIOS)
	assert.Equal(t, Platform("android"), PlatformAndroid)
	assert.Equal(t, Platform("huawei"), PlatformHuawei)
}

// ensure context import is used
var _ = context.Background
