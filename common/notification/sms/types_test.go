// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBasic_Valid(t *testing.T) {
	req := SendRequest{
		To:      []PhoneNumber{{Number: "13800000000", CountryCode: 86}},
		Message: Message{Content: "hello"},
	}
	assert.NoError(t, ValidateBasic(req))
}

func TestValidateBasic_TemplateOnly(t *testing.T) {
	req := SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Template: "TPL_001"},
	}
	assert.NoError(t, ValidateBasic(req))
}

func TestValidateBasic_EmptyTo(t *testing.T) {
	req := SendRequest{
		Message: Message{Content: "hello"},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipients")
}

func TestValidateBasic_EmptyNumber(t *testing.T) {
	req := SendRequest{
		To:      []PhoneNumber{{Number: "  "}},
		Message: Message{Content: "hello"},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty number")
}

func TestValidateBasic_EmptyContentAndTemplate(t *testing.T) {
	req := SendRequest{
		To: []PhoneNumber{{Number: "13800000000"}},
	}
	err := ValidateBasic(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content or template")
}

func TestFirstRecipient_Valid(t *testing.T) {
	req := SendRequest{
		To: []PhoneNumber{
			{Number: "13800000000", CountryCode: 86},
			{Number: "13900000000", CountryCode: 86},
		},
	}
	got, err := FirstRecipient(req)
	require.NoError(t, err)
	assert.Equal(t, "8613800000000", got)
}

func TestFirstRecipient_NoCountryCode(t *testing.T) {
	req := SendRequest{
		To: []PhoneNumber{{Number: "13800000000"}},
	}
	got, err := FirstRecipient(req)
	require.NoError(t, err)
	assert.Equal(t, "13800000000", got)
}

func TestFirstRecipient_Empty(t *testing.T) {
	_, err := FirstRecipient(SendRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients")
}

func TestFirstRecipient_EmptyNumber(t *testing.T) {
	_, err := FirstRecipient(SendRequest{To: []PhoneNumber{{Number: ""}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty number")
}

func TestNormalizeContent_WithSignPresent(t *testing.T) {
	out := NormalizeContent("【LingByte】hello", "【LingByte】")
	assert.Equal(t, "【LingByte】hello", out)
}

func TestNormalizeContent_WithoutSign(t *testing.T) {
	out := NormalizeContent("hello", "【LingByte】")
	assert.Equal(t, "hello 【LingByte】", out)
}

func TestNormalizeContent_EmptyFallback(t *testing.T) {
	out := NormalizeContent("hello", "")
	assert.Equal(t, "hello", out)
}

func TestNormalizeContent_EmptyContent(t *testing.T) {
	assert.Equal(t, "", NormalizeContent("", "【LingByte】"))
}

func TestNormalizeContent_PlainSign(t *testing.T) {
	out := NormalizeContent("hello", "LingByte")
	assert.Equal(t, "hello LingByte", out)
}

func TestNormalizeContent_AlreadyContainsPlainSign(t *testing.T) {
	out := NormalizeContent("hello from LingByte inc", "LingByte")
	assert.Equal(t, "hello from LingByte inc", out)
}

func TestProviderKindConstants(t *testing.T) {
	assert.Equal(t, ProviderKind("aliyun"), ProviderAliyun)
	assert.Equal(t, ProviderKind("tencent"), ProviderTencent)
	assert.Equal(t, ProviderKind("twilio"), ProviderTwilio)
	assert.Equal(t, ProviderKind("huawei"), ProviderHuawei)
	assert.Equal(t, ProviderKind("yunpian"), ProviderYunpian)
	assert.Equal(t, ProviderKind("submail"), ProviderSubmail)
	assert.Equal(t, ProviderKind("luosimao"), ProviderLuosimao)
	assert.Equal(t, ProviderKind("mock"), ProviderMock)
}

// ──────────────────────────────────────────────
// Utility helpers coverage
// ──────────────────────────────────────────────

func TestTruncateRaw(t *testing.T) {
	assert.Equal(t, "hello", TruncateRaw("hello", 10))
	assert.Equal(t, "hel…", TruncateRaw("hello", 3))
	assert.Equal(t, "hello", TruncateRaw("  hello  ", 10)) // trims spaces
	assert.Equal(t, "", TruncateRaw("", 10))
	assert.Equal(t, "hello", TruncateRaw("hello", 0)) // max=0 means no truncation
}

func TestJSONStringAny(t *testing.T) {
	assert.Equal(t, `{"a":"b"}`, JSONStringAny(map[string]string{"a": "b"}))
	assert.Equal(t, `"hello"`, JSONStringAny("hello"))
	assert.Equal(t, "null", JSONStringAny(nil))
	// Channels can't be marshaled.
	assert.Equal(t, "", JSONStringAny(make(chan int)))
}

func TestCtxOrBackground(t *testing.T) {
	ctx := context.Background()
	result := CtxOrBackground(ctx)
	assert.NotNil(t, result)
	assert.NotNil(t, CtxOrBackground(nil))
}

func TestFirstRecipientStr_Empty(t *testing.T) {
	_, err := FirstRecipientStr(SendRequest{})
	require.Error(t, err)
}

func TestFirstRecipientStr_EmptyNumber(t *testing.T) {
	_, err := FirstRecipientStr(SendRequest{
		To: []PhoneNumber{{Number: ""}},
	})
	require.Error(t, err)
}

func TestIs2xx(t *testing.T) {
	assert.True(t, Is2xx(200))
	assert.True(t, Is2xx(299))
	assert.False(t, Is2xx(199))
	assert.False(t, Is2xx(300))
	assert.False(t, Is2xx(500))
}

// ──────────────────────────────────────────────
// PhoneNumber.String coverage
// ──────────────────────────────────────────────

func TestPhoneNumber_String(t *testing.T) {
	tests := []struct {
		name string
		p    PhoneNumber
		want string
	}{
		{"with country code", PhoneNumber{Number: "13800138000", CountryCode: 86}, "+8613800138000"},
		{"without country code", PhoneNumber{Number: "13800138000"}, "13800138000"},
		{"empty number", PhoneNumber{Number: ""}, ""},
		{"US country code", PhoneNumber{Number: "5551234567", CountryCode: 1}, "+15551234567"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}
