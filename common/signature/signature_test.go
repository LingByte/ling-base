// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package signature

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignHMACSHA256_Verify(t *testing.T) {
	key := []byte("secret")
	sig := SignHMACSHA256("hello", key)
	assert.NotEmpty(t, sig)
	assert.True(t, VerifyHMACSHA256("hello", key, sig))
	assert.False(t, VerifyHMACSHA256("hello2", key, sig))
	assert.False(t, VerifyHMACSHA256("hello", []byte("other"), sig))
}

func TestSignHMACSHA512_Verify(t *testing.T) {
	key := []byte("secret")
	sig := SignHMACSHA512("hello", key)
	assert.NotEmpty(t, sig)
	assert.True(t, VerifyHMACSHA512("hello", key, sig))
	assert.False(t, VerifyHMACSHA512("hello2", key, sig))
}

func TestSignMD5(t *testing.T) {
	// Known MD5 of "hello".
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", SignMD5("hello"))
}

func TestSignSortedParams(t *testing.T) {
	params := url.Values{}
	params.Set("foo", "bar")
	params.Set("abc", "123")
	params.Set("zzz", "last")
	sig := SignSortedParams(params, "mykey")
	assert.NotEmpty(t, sig)
	// Deterministic.
	assert.Equal(t, sig, SignSortedParams(params, "mykey"))
	// Different key yields different sig.
	assert.NotEqual(t, sig, SignSortedParams(params, "otherkey"))
}

func TestSignSortedParams_Order(t *testing.T) {
	// Verify the sorted order is applied regardless of insertion order.
	p1 := url.Values{}
	p1.Set("b", "2")
	p1.Set("a", "1")
	p2 := url.Values{}
	p2.Set("a", "1")
	p2.Set("b", "2")
	assert.Equal(t, SignSortedParams(p1, "k"), SignSortedParams(p2, "k"))
}

func TestSignRequest_Verify(t *testing.T) {
	key := []byte("secret")
	params := url.Values{}
	params.Set("q", "hello")
	params.Set("limit", "10")
	body := []byte(`{"x":1}`)
	sig := SignRequest("POST", "https://api.example.com/v1/items", params, body, key)
	require.NotEmpty(t, sig)

	// SignRequest does not mutate params (timestamp is only embedded in payload).
	require.Empty(t, params.Get("timestamp"))

	// Recompute with the same now-timestamp to confirm determinism of payload.
	// We can't know the exact timestamp SignRequest used, so just verify the
	// signature is non-empty and reproducible in shape via signRequestAt.
	params.Set("timestamp", "1700000000")
	expected := signRequestAt("POST", "https://api.example.com/v1/items", params, body, key, "1700000000")
	assert.NotEmpty(t, expected)
}

func TestVerifyRequest_FreshAndExpired(t *testing.T) {
	key := []byte("secret")
	params := url.Values{}
	params.Set("q", "hello")
	params.Set("timestamp", "1700000000")
	body := []byte(`{}`)

	sig := signRequestAt("GET", "https://example.com", params, body, key, "1700000000")

	// maxAge <= 0 disables freshness check.
	assert.True(t, VerifyRequest("GET", "https://example.com", params, body, key, sig, 0))

	// Fresh timestamp.
	now := time.Now().Unix()
	params.Set("timestamp", int64ToStr(now))
	sigNow := signRequestAt("GET", "https://example.com", params, body, key, int64ToStr(now))
	assert.True(t, VerifyRequest("GET", "https://example.com", params, body, key, sigNow, 5*time.Minute))

	// Expired timestamp.
	params.Set("timestamp", int64ToStr(now-int64((10*time.Minute).Seconds())))
	sigOld := signRequestAt("GET", "https://example.com", params, body, key, int64ToStr(now-int64((10*time.Minute).Seconds())))
	assert.False(t, VerifyRequest("GET", "https://example.com", params, body, key, sigOld, 5*time.Minute))
}

func TestVerifyRequest_BadTimestamp(t *testing.T) {
	params := url.Values{}
	params.Set("timestamp", "not-a-number")
	assert.False(t, VerifyRequest("GET", "https://example.com", params, nil, []byte("k"), "sig", 5*time.Minute))
}

func TestVerifyRequest_MissingTimestamp(t *testing.T) {
	params := url.Values{}
	assert.False(t, VerifyRequest("GET", "https://example.com", params, nil, []byte("k"), "sig", 5*time.Minute))
}

func TestSignSortedParams_Empty(t *testing.T) {
	// Empty params still produce a deterministic signature (just the key is signed).
	sig := SignSortedParams(url.Values{}, "mykey")
	assert.NotEmpty(t, sig)
	// Deterministic.
	assert.Equal(t, sig, SignSortedParams(url.Values{}, "mykey"))
}

func TestSignSortedParams_NilParams(t *testing.T) {
	// nil params should not panic and should produce a signature based only on the key.
	sig := SignSortedParams(nil, "mykey")
	assert.NotEmpty(t, sig)
	assert.Equal(t, sig, SignSortedParams(nil, "mykey"))
}

func TestSortedParamsString_Nil(t *testing.T) {
	assert.Equal(t, "", sortedParamsString(nil))
}

func TestSortedParamsString_SkipsSignAndEmpty(t *testing.T) {
	params := url.Values{}
	params.Set("sign", "should-be-skipped")
	params.Set("foo", "")
	params.Set("bar", "baz")
	// "sign" and empty values must be skipped; only bar=baz remains.
	assert.Equal(t, "bar=baz", sortedParamsString(params))
}

func TestSortedParamsString_MultiValues(t *testing.T) {
	params := url.Values{}
	params.Add("k", "v1")
	params.Add("k", "v2")
	params.Set("a", "1")
	// Multiple values for the same key are emitted as separate entries.
	assert.Equal(t, "a=1&k=v1&k=v2", sortedParamsString(params))
}

func TestSignRequest_PayloadShape(t *testing.T) {
	// Verify the canonical payload built by buildRequestPayload.
	params := url.Values{}
	params.Set("b", "2")
	params.Set("a", "1")
	body := []byte(`body`)
	payload := buildRequestPayload("POST", "https://example.com", params, body, "1700000000")
	expected := "POST\nhttps://example.com\na=1&b=2\nbody\n1700000000"
	assert.Equal(t, expected, payload)
}

func TestSignRequest_NilParamsAndBody(t *testing.T) {
	// SignRequest with nil params and nil body should not panic.
	sig := SignRequest("GET", "https://example.com", nil, nil, []byte("secret"))
	assert.NotEmpty(t, sig)
}

func TestVerifyRequest_WrongSig(t *testing.T) {
	params := url.Values{}
	params.Set("timestamp", "1700000000")
	// A bogus signature must not verify.
	assert.False(t, VerifyRequest("GET", "https://example.com", params, nil, []byte("secret"), "badsig", 0))
}

func TestVerifyRequest_NegativeMaxAge(t *testing.T) {
	// Negative maxAge disables freshness check (only > 0 triggers the check).
	params := url.Values{}
	params.Set("timestamp", "1700000000")
	sig := signRequestAt("GET", "https://example.com", params, nil, []byte("secret"), "1700000000")
	assert.True(t, VerifyRequest("GET", "https://example.com", params, nil, []byte("secret"), sig, -1))
}

func int64ToStr(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}
