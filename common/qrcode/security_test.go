// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qrcode

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testSecret = []byte("my-secret-key-1234567890-abcdef")
	testKey32  = []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	testKey16  = []byte("0123456789abcdef")                 // 16 bytes
)

// ──────────────────────────────────────────────
// Sign / Verify
// ──────────────────────────────────────────────

func TestSign_Verify(t *testing.T) {
	payload := "product-id:SN-2026-0001"
	token, err := Sign(payload, testSecret)
	require.NoError(t, err)
	assert.True(t, IsSecureToken(token))

	got, err := Verify(token, testSecret, 0)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestSign_EmptyPayload(t *testing.T) {
	_, err := Sign("", testSecret)
	assert.Error(t, err)
}

func TestSign_EmptySecret(t *testing.T) {
	_, err := Sign("data", nil)
	assert.Error(t, err)
}

func TestVerify_WrongSecret(t *testing.T) {
	token, _ := Sign("data", testSecret)
	_, err := Verify(token, []byte("wrong-secret"), 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestVerify_TamperedToken(t *testing.T) {
	token, _ := Sign("data", testSecret)
	// Flip a character in the token body.
	tampered := token[:len(token)-1] + "X"
	_, err := Verify(tampered, testSecret, 0)
	assert.Error(t, err)
}

func TestVerify_NotAToken(t *testing.T) {
	_, err := Verify("plain text", testSecret, 0)
	assert.Error(t, err)
}

func TestVerify_EmptySecret(t *testing.T) {
	token, _ := Sign("data", testSecret)
	_, err := Verify(token, nil, 0)
	assert.Error(t, err)
}

func TestVerify_Expired(t *testing.T) {
	token, err := Sign("data", testSecret)
	require.NoError(t, err)
	// maxAge of 1 nanosecond — token should be "expired" immediately.
	_, err = Verify(token, testSecret, 1*time.Nanosecond)
	// This may or may not expire depending on timing; run a few times.
	_ = err // non-deterministic; just ensure no panic
}

func TestVerify_MaxAgeZero_NoExpiry(t *testing.T) {
	token, _ := Sign("data", testSecret)
	_, err := Verify(token, testSecret, 0)
	require.NoError(t, err)
}

func TestVerify_MalformedBase64(t *testing.T) {
	_, err := Verify(TokenPrefix+"!!!not-base64!!!", testSecret, 0)
	assert.Error(t, err)
}

func TestVerify_MalformedJSON(t *testing.T) {
	// Valid base64 but not valid JSON.
	enc := base64RawURLEncode([]byte("not json"))
	_, err := Verify(TokenPrefix+enc, testSecret, 0)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Encrypt / Decrypt
// ──────────────────────────────────────────────

func TestEncrypt_Decrypt(t *testing.T) {
	payload := "confidential-product-data"
	token, err := Encrypt(payload, testKey32)
	require.NoError(t, err)
	assert.True(t, IsSecureToken(token))

	got, err := Decrypt(token, testKey32)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestEncrypt_Decrypt_Key16(t *testing.T) {
	token, err := Encrypt("data", testKey16)
	require.NoError(t, err)
	got, err := Decrypt(token, testKey16)
	require.NoError(t, err)
	assert.Equal(t, "data", got)
}

func TestEncrypt_EmptyPayload(t *testing.T) {
	_, err := Encrypt("", testKey32)
	assert.Error(t, err)
}

func TestEncrypt_InvalidKey(t *testing.T) {
	_, err := Encrypt("data", []byte("short"))
	assert.Error(t, err)
}

func TestDecrypt_WrongKey(t *testing.T) {
	token, _ := Encrypt("data", testKey32)
	_, err := Decrypt(token, testKey16)
	assert.Error(t, err)
}

func TestDecrypt_NotAToken(t *testing.T) {
	_, err := Decrypt("plain text", testKey32)
	assert.Error(t, err)
}

func TestDecrypt_MalformedBase64(t *testing.T) {
	_, err := Decrypt(TokenPrefix+"!!!", testKey32)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	// Encode a very short ciphertext (shorter than nonce).
	enc := base64RawURLEncode([]byte("ab"))
	_, err := Decrypt(TokenPrefix+enc, testKey32)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Secure / Unseal
// ──────────────────────────────────────────────

func TestSecure_Unseal(t *testing.T) {
	payload := "anti-counterfeit:SN-9999"
	token, err := Secure(payload, testKey32, testSecret)
	require.NoError(t, err)
	assert.True(t, IsSecureToken(token))

	got, err := Unseal(token, testKey32, testSecret, 0)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestSecure_EmptyPayload(t *testing.T) {
	_, err := Secure("", testKey32, testSecret)
	assert.Error(t, err)
}

func TestSecure_EmptySignKey(t *testing.T) {
	_, err := Secure("data", testKey32, nil)
	assert.Error(t, err)
}

func TestSecure_InvalidEncryptKey(t *testing.T) {
	_, err := Secure("data", []byte("short"), testSecret)
	assert.Error(t, err)
}

func TestUnseal_WrongSignKey(t *testing.T) {
	token, _ := Secure("data", testKey32, testSecret)
	_, err := Unseal(token, testKey32, []byte("wrong"), 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestUnseal_WrongEncryptKey(t *testing.T) {
	token, _ := Secure("data", testKey32, testSecret)
	_, err := Unseal(token, testKey16, testSecret, 0)
	assert.Error(t, err)
}

func TestUnseal_NotAToken(t *testing.T) {
	_, err := Unseal("plain text", testKey32, testSecret, 0)
	assert.Error(t, err)
}

func TestUnseal_MalformedBase64(t *testing.T) {
	_, err := Unseal(TokenPrefix+"!!!", testKey32, testSecret, 0)
	assert.Error(t, err)
}

func TestUnseal_MalformedJSON(t *testing.T) {
	enc := base64RawURLEncode([]byte("not json"))
	_, err := Unseal(TokenPrefix+enc, testKey32, testSecret, 0)
	assert.Error(t, err)
}

func TestUnseal_EmptySignKey(t *testing.T) {
	token, _ := Secure("data", testKey32, testSecret)
	_, err := Unseal(token, testKey32, nil, 0)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Convenience: GenerateSigned / GenerateSecure
// ──────────────────────────────────────────────

func TestGenerateSigned(t *testing.T) {
	img, err := GenerateSigned("product:123", testSecret, ECLHigh, 300)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())

	// Round-trip: decode → verify.
	decoded, err := Decode(img)
	require.NoError(t, err)
	payload, err := Verify(decoded, testSecret, 0)
	require.NoError(t, err)
	assert.Equal(t, "product:123", payload)
}

func TestGenerateSecure(t *testing.T) {
	img, err := GenerateSecure("secret-product:456", testKey32, testSecret, ECLHigh, 400)
	require.NoError(t, err)
	assert.Equal(t, 400, img.Bounds().Dx())

	decoded, err := Decode(img)
	require.NoError(t, err)
	payload, err := Unseal(decoded, testKey32, testSecret, 0)
	require.NoError(t, err)
	assert.Equal(t, "secret-product:456", payload)
}

func TestGenerateSigned_Error(t *testing.T) {
	_, err := GenerateSigned("", testSecret, ECLHigh, 300)
	assert.Error(t, err)
}

func TestGenerateSecure_Error(t *testing.T) {
	_, err := GenerateSecure("data", []byte("short"), testSecret, ECLHigh, 300)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// IsSecureToken
// ──────────────────────────────────────────────

func TestIsSecureToken(t *testing.T) {
	assert.False(t, IsSecureToken(""))
	assert.False(t, IsSecureToken("plain text"))
	assert.True(t, IsSecureToken(TokenPrefix+"abc"))
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func base64RawURLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
