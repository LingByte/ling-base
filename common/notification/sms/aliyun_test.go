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

func TestNewAliyunProvider_Config(t *testing.T) {
	p, err := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
		"sign_name":         "LingByte",
		"endpoint":          "https://example.com",
		"content_template":  "TPL",
		"content_param_key": "code",
	})
	require.NoError(t, err)
	ap, ok := p.(*AliyunProvider)
	require.True(t, ok)
	assert.Equal(t, "AKID", ap.cfg.AccessKeyID)
	assert.Equal(t, "SECRET", ap.cfg.AccessKeySecret)
	assert.Equal(t, "LingByte", ap.cfg.SignName)
	assert.Equal(t, "https://example.com", ap.cfg.Endpoint)
	assert.Equal(t, "TPL", ap.cfg.ContentTemplate)
	assert.Equal(t, "code", ap.cfg.ContentParamKey)
}

func TestNewAliyunProvider_DefaultEndpoint(t *testing.T) {
	p, err := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
	})
	require.NoError(t, err)
	ap := p.(*AliyunProvider)
	assert.Equal(t, "https://dysmsapi.aliyuncs.com", ap.cfg.Endpoint)
	assert.Equal(t, "content", ap.cfg.ContentParamKey)
}

func TestNewAliyunProvider_MissingCreds(t *testing.T) {
	_, err := NewAliyunProvider(ProviderConfig{"access_key_id": "AKID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestAliyunProvider_Kind(t *testing.T) {
	p, _ := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
	})
	assert.Equal(t, ProviderAliyun, p.Kind())
}

func TestAliyunProvider_Sign(t *testing.T) {
	p, _ := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
	})
	ap := p.(*AliyunProvider)
	form := ap.buildCommonParams()
	form.Set("PhoneNumbers", "13800000000")
	form.Set("SignName", "LingByte")
	sig := ap.sign(form)
	assert.NotEmpty(t, sig)
	// signature must be valid base64
	// (HMACSHA1Base64 returns base64)
	require.NotEmpty(t, sig)
	// signing the same form twice with same nonce should be deterministic
	sig2 := ap.sign(form)
	assert.Equal(t, sig, sig2)
}

func TestAliyunProvider_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = body
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SendSmsResponse>
  <Code>OK</Code>
  <Message>OK</Message>
  <RequestId>req-1</RequestId>
  <BizId>biz-1</BizId>
</SendSmsResponse>`))
	}))
	defer srv.Close()

	p, err := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
		"sign_name":         "LingByte",
		"endpoint":          srv.URL,
	})
	require.NoError(t, err)

	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000", CountryCode: 86}},
		Message: Message{Content: "hello", Template: "TPL_001", Data: map[string]string{"k": "v"}},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Accepted)
	assert.Equal(t, "OK", res.Status)
	assert.Equal(t, "biz-1", res.MessageID)
	assert.Contains(t, res.Raw, "SendSmsResponse")
}

func TestAliyunProvider_Send_ProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SendSmsResponse>
  <Code>isv.BUSINESS_LIMIT_CONTROL</Code>
  <Message>send frequency too high</Message>
  <RequestId>req-2</RequestId>
</SendSmsResponse>`))
	}))
	defer srv.Close()

	p, _ := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
		"endpoint":          srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.NoError(t, err) // provider error code is not a Go error
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
	assert.Contains(t, res.Error, "frequency")
}

func TestAliyunProvider_Send_InvalidRequest(t *testing.T) {
	p, _ := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
		"endpoint":          "https://example.com",
	})
	_, err := p.Send(context.Background(), SendRequest{})
	require.Error(t, err)
}

func TestAliyunProvider_Send_BadXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not xml`))
	}))
	defer srv.Close()

	p, _ := NewAliyunProvider(ProviderConfig{
		"access_key_id":     "AKID",
		"access_key_secret": "SECRET",
		"endpoint":          srv.URL,
	})
	res, err := p.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800000000"}},
		Message: Message{Content: "hi"},
	})
	require.Error(t, err)
	require.NotNil(t, res)
	assert.False(t, res.Accepted)
}

func TestPercentEncode(t *testing.T) {
	assert.Equal(t, "%20", percentEncode(" "))
	assert.Equal(t, "%2A", percentEncode("*"))
	assert.Equal(t, "~", percentEncode("~"))
	assert.Equal(t, "abc", percentEncode("abc"))
}

func TestJsonString(t *testing.T) {
	assert.Equal(t, "", jsonString(nil))
	out := jsonString(map[string]string{"a": "1", "b": "2"})
	assert.Contains(t, out, `"a":"1"`)
	assert.Contains(t, out, `"b":"2"`)
}

func TestJsonStringPart_Escaping(t *testing.T) {
	assert.Equal(t, `"a\"b"`, jsonStringPart(`a"b`))
	assert.Equal(t, `"a\\b"`, jsonStringPart(`a\b`))
	assert.Equal(t, `"a\nb"`, jsonStringPart("a\nb"))
}
