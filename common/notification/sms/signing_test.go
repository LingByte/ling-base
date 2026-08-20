// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sms

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSHA1Hex_KnownValue(t *testing.T) {
	// sha1("abc") = a9993e364706816aba3e25717850c26c9cd0d89d
	assert.Equal(t, "a9993e364706816aba3e25717850c26c9cd0d89d", SHA1Hex("abc"))
	// cross-check
	sum := sha1.Sum([]byte("abc"))
	assert.Equal(t, hex.EncodeToString(sum[:]), SHA1Hex("abc"))
}

func TestSHA1Hex_Empty(t *testing.T) {
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", SHA1Hex(""))
}

func TestSHA256Base64_KnownValue(t *testing.T) {
	// sha256("abc") base64
	sum := sha256.Sum256([]byte("abc"))
	want := base64.StdEncoding.EncodeToString(sum[:])
	assert.Equal(t, want, SHA256Base64("abc"))
	assert.NotEmpty(t, want)
}

func TestSHA256Hex_KnownValue(t *testing.T) {
	// sha256("abc") hex = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", SHA256Hex("abc"))
}

func TestMD5Hex_KnownValue(t *testing.T) {
	// md5("abc") = 900150983cd24fb0d6963f7d28e17f72
	assert.Equal(t, "900150983cd24fb0d6963f7d28e17f72", MD5Hex("abc"))
	sum := md5.Sum([]byte("abc"))
	assert.Equal(t, hex.EncodeToString(sum[:]), MD5Hex("abc"))
}

func TestMD5Hex_Empty(t *testing.T) {
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", MD5Hex(""))
}

func TestRandHex_NonEmpty(t *testing.T) {
	s := RandHex(16)
	assert.Len(t, s, 32) // 16 bytes -> 32 hex chars
	// all hex chars
	for _, c := range s {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "non-hex char %q", c)
	}
}

func TestRandHex_DifferentCalls(t *testing.T) {
	a := RandHex(8)
	b := RandHex(8)
	assert.Len(t, a, 16)
	assert.Len(t, b, 16)
	assert.NotEqual(t, a, b)
}

func TestRandHex_ZeroOrNegative(t *testing.T) {
	assert.Equal(t, "", RandHex(0))
	assert.Equal(t, "", RandHex(-1))
}

func TestHMACSHA1Base64(t *testing.T) {
	got := HMACSHA1Base64("secret&", "GET&%2F&foo")
	assert.NotEmpty(t, got)
	// decode to verify it's valid base64
	_, err := base64.StdEncoding.DecodeString(got)
	assert.NoError(t, err)
}

func TestHMACSHA256Base64(t *testing.T) {
	got := HMACSHA256Base64("key", "message")
	assert.NotEmpty(t, got)
	_, err := base64.StdEncoding.DecodeString(got)
	assert.NoError(t, err)
}
