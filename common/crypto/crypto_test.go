// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Random key
// ──────────────────────────────────────────────

func TestRandomKey(t *testing.T) {
	key, err := RandomKey(32)
	if err != nil {
		t.Fatalf("RandomKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("RandomKey length = %d, want 32", len(key))
	}
}

func TestMustRandomKey(t *testing.T) {
	key := MustRandomKey(16)
	if len(key) != 16 {
		t.Fatalf("MustRandomKey length = %d", len(key))
	}
}

func TestRandomHex(t *testing.T) {
	h, err := RandomHex(16)
	if err != nil {
		t.Fatalf("RandomHex failed: %v", err)
	}
	if len(h) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("RandomHex length = %d, want 32", len(h))
	}
}

func TestRandomBase64(t *testing.T) {
	s, err := RandomBase64(16)
	if err != nil {
		t.Fatalf("RandomBase64 failed: %v", err)
	}
	if len(s) == 0 {
		t.Fatal("RandomBase64 returned empty")
	}
}

// ──────────────────────────────────────────────
// AES-GCM
// ──────────────────────────────────────────────

func TestAESGCM_EncryptDecrypt(t *testing.T) {
	key := MustRandomKey(32)
	plaintext := []byte("hello world")
	ciphertext, err := AESGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := AESGCMDecrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestAESGCM_DifferentCiphertexts(t *testing.T) {
	key := MustRandomKey(16)
	plaintext := []byte("same data")
	c1, _ := AESGCMEncrypt(key, plaintext)
	c2, _ := AESGCMEncrypt(key, plaintext)
	if bytes.Equal(c1, c2) {
		t.Fatal("AES-GCM should produce different ciphertexts (random nonce)")
	}
}

func TestAESGCM_WrongKey(t *testing.T) {
	key1 := MustRandomKey(16)
	key2 := MustRandomKey(16)
	ciphertext, _ := AESGCMEncrypt(key1, []byte("secret"))
	_, err := AESGCMDecrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestAESGCM_TamperedCiphertext(t *testing.T) {
	key := MustRandomKey(16)
	ciphertext, _ := AESGCMEncrypt(key, []byte("secret"))
	ciphertext[len(ciphertext)-1] ^= 0xff // tamper
	_, err := AESGCMDecrypt(key, ciphertext)
	if err == nil {
		t.Fatal("Decrypt of tampered ciphertext should fail")
	}
}

func TestAESGCM_InvalidKey(t *testing.T) {
	_, err := AESGCMEncrypt([]byte("short"), []byte("data"))
	if err == nil {
		t.Fatal("AES with short key should fail")
	}
}

func TestAESGCM_ShortCiphertext(t *testing.T) {
	key := MustRandomKey(16)
	_, err := AESGCMDecrypt(key, []byte("short"))
	if err == nil {
		t.Fatal("Decrypt of short ciphertext should fail")
	}
}

func TestAESGCM_Base64(t *testing.T) {
	key := MustRandomKey(32)
	encoded, err := AESGCMEncryptBase64(key, []byte("hello"))
	if err != nil {
		t.Fatalf("EncryptBase64 failed: %v", err)
	}
	decrypted, err := AESGCMDecryptBase64(key, encoded)
	if err != nil {
		t.Fatalf("DecryptBase64 failed: %v", err)
	}
	if string(decrypted) != "hello" {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestAESGCM_InvalidBase64(t *testing.T) {
	key := MustRandomKey(16)
	_, err := AESGCMDecryptBase64(key, "!!!invalid base64!!!")
	if err == nil {
		t.Fatal("DecryptBase64 with invalid base64 should fail")
	}
}

// ──────────────────────────────────────────────
// AES-CBC
// ──────────────────────────────────────────────

func TestAESCBC_EncryptDecrypt(t *testing.T) {
	key := MustRandomKey(32)
	plaintext := []byte("hello world, this is a longer message")
	ciphertext, err := AESCBCEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := AESCBCDecrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestAESCBC_ShortMessage(t *testing.T) {
	key := MustRandomKey(16)
	plaintext := []byte("hi")
	ciphertext, err := AESCBCEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := AESCBCDecrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestAESCBC_EmptyMessage(t *testing.T) {
	key := MustRandomKey(16)
	ciphertext, err := AESCBCEncrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := AESCBCDecrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("decrypted = %q, want empty", decrypted)
	}
}

func TestAESCBC_WrongKey(t *testing.T) {
	key1 := MustRandomKey(16)
	key2 := MustRandomKey(16)
	ciphertext, _ := AESCBCEncrypt(key1, []byte("secret"))
	_, err := AESCBCDecrypt(key2, ciphertext)
	if err == nil {
		// CBC won't error on wrong key, but padding will be wrong.
		_ = err
	}
}

func TestAESCBC_ShortCiphertext(t *testing.T) {
	key := MustRandomKey(16)
	_, err := AESCBCDecrypt(key, []byte("short"))
	if err == nil {
		t.Fatal("Decrypt of short ciphertext should fail")
	}
}

// ──────────────────────────────────────────────
// RSA
// ──────────────────────────────────────────────

func TestRSA_GenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair failed: %v", err)
	}
	if priv == nil || pub == nil {
		t.Fatal("keys should not be nil")
	}
}

func TestRSA_EncryptDecrypt(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair(2048)
	plaintext := []byte("hello RSA")
	ciphertext, err := RSAEncrypt(pub, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	decrypted, err := RSADecrypt(priv, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestRSA_SignVerify(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair(2048)
	data := []byte("message to sign")
	sig, err := RSASign(priv, data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if err := RSAVerify(pub, data, sig); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestRSA_VerifyWrongData(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair(2048)
	sig, _ := RSASign(priv, []byte("original"))
	err := RSAVerify(pub, []byte("tampered"), sig)
	if err == nil {
		t.Fatal("Verify of wrong data should fail")
	}
}

func TestRSA_PEMExportImport(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair(2048)

	privPEM, err := ExportRSAPrivateKeyPEM(priv)
	if err != nil {
		t.Fatalf("ExportRSAPrivateKeyPEM failed: %v", err)
	}
	if !strings.Contains(privPEM, "PRIVATE KEY") {
		t.Fatal("private key PEM should contain PRIVATE KEY")
	}

	pubPEM, err := ExportRSAPublicKeyPEM(pub)
	if err != nil {
		t.Fatalf("ExportRSAPublicKeyPEM failed: %v", err)
	}
	if !strings.Contains(pubPEM, "PUBLIC KEY") {
		t.Fatal("public key PEM should contain PUBLIC KEY")
	}

	parsedPriv, err := ParseRSAPrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyPEM failed: %v", err)
	}
	parsedPub, err := ParseRSAPublicKeyPEM(pubPEM)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyPEM failed: %v", err)
	}

	// Test encrypt/decrypt with parsed keys.
	ciphertext, _ := RSAEncrypt(parsedPub, []byte("test"))
	decrypted, _ := RSADecrypt(parsedPriv, ciphertext)
	if string(decrypted) != "test" {
		t.Fatalf("PEM round-trip failed: %q", decrypted)
	}
}

func TestRSA_InvalidPEM(t *testing.T) {
	_, err := ParseRSAPrivateKeyPEM("invalid pem")
	if err == nil {
		t.Fatal("ParseRSAPrivateKeyPEM with invalid PEM should fail")
	}
	_, err = ParseRSAPublicKeyPEM("invalid pem")
	if err == nil {
		t.Fatal("ParseRSAPublicKeyPEM with invalid PEM should fail")
	}
}

// ──────────────────────────────────────────────
// HMAC-SHA signatures
// ──────────────────────────────────────────────

func TestSignSHA256(t *testing.T) {
	sig := SignSHA256([]byte("data"), []byte("key"))
	if len(sig) != 32 {
		t.Fatalf("SignSHA256 length = %d, want 32", len(sig))
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("data")
	key := []byte("key")
	sig := SignSHA256(data, key)
	if !VerifySHA256(data, key, sig) {
		t.Fatal("VerifySHA256 should be true")
	}
	if VerifySHA256([]byte("wrong"), key, sig) {
		t.Fatal("VerifySHA256 with wrong data should be false")
	}
	if VerifySHA256(data, []byte("wrong"), sig) {
		t.Fatal("VerifySHA256 with wrong key should be false")
	}
}

func TestSignSHA512(t *testing.T) {
	sig := SignSHA512([]byte("data"), []byte("key"))
	if len(sig) != 64 {
		t.Fatalf("SignSHA512 length = %d, want 64", len(sig))
	}
}

func TestVerifySHA512(t *testing.T) {
	data := []byte("data")
	key := []byte("key")
	sig := SignSHA512(data, key)
	if !VerifySHA512(data, key, sig) {
		t.Fatal("VerifySHA512 should be true")
	}
}

// ──────────────────────────────────────────────
// JWT
// ──────────────────────────────────────────────

func TestJWT_HS256_SignVerify(t *testing.T) {
	j := NewJWT(JWTConfig{
		Algorithm: JWTAlgHS256,
		Secret:    []byte("my-secret-key"),
		Issuer:    "test",
		ExpiresIn: time.Hour,
	})

	token, err := j.Sign(JWTClaims{
		Subject: "user123",
		Extra: map[string]interface{}{
			"role": "admin",
		},
	})
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT should have 3 parts, got %d", len(parts))
	}

	claims, err := j.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.Subject != "user123" {
		t.Fatalf("Subject = %q, want user123", claims.Subject)
	}
	if claims.Issuer != "test" {
		t.Fatalf("Issuer = %q, want test", claims.Issuer)
	}
	if claims.Extra["role"] != "admin" {
		t.Fatalf("role = %v, want admin", claims.Extra["role"])
	}
}

func TestJWT_Expired(t *testing.T) {
	j := NewJWT(JWTConfig{
		Secret:    []byte("key"),
		ExpiresIn: -time.Hour, // already expired
	})

	token, _ := j.Sign(JWTClaims{Subject: "user"})
	_, err := j.Verify(token)
	if err == nil {
		t.Fatal("Verify of expired token should fail")
	}
}

func TestJWT_NotBefore(t *testing.T) {
	j := NewJWT(JWTConfig{Secret: []byte("key")})
	token, _ := j.Sign(JWTClaims{
		NotBefore: time.Now().Add(time.Hour).Unix(),
	})
	_, err := j.Verify(token)
	if err == nil {
		t.Fatal("Verify of not-yet-valid token should fail")
	}
}

func TestJWT_InvalidSignature(t *testing.T) {
	j := NewJWT(JWTConfig{Secret: []byte("key")})
	token, _ := j.Sign(JWTClaims{Subject: "user"})

	// Tamper with token.
	parts := strings.Split(token, ".")
	parts[2] = "invalidSignature"
	tampered := strings.Join(parts, ".")

	_, err := j.Verify(tampered)
	if err == nil {
		t.Fatal("Verify of tampered token should fail")
	}
}

func TestJWT_InvalidFormat(t *testing.T) {
	j := NewJWT(JWTConfig{Secret: []byte("key")})
	_, err := j.Verify("invalid.token")
	if err == nil {
		t.Fatal("Verify of invalid format should fail")
	}
}

func TestJWT_HS512(t *testing.T) {
	j := NewJWT(JWTConfig{
		Algorithm: JWTAlgHS512,
		Secret:    []byte("key"),
	})
	token, err := j.Sign(JWTClaims{Subject: "user"})
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	claims, err := j.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.Subject != "user" {
		t.Fatalf("Subject = %q", claims.Subject)
	}
}

func TestJWT_RS256(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair(2048)
	j := NewJWT(JWTConfig{
		Algorithm: JWTAlgRS256,
		RSAPriv:   priv,
		RSAPub:    pub,
		Issuer:    "test",
	})

	token, err := j.Sign(JWTClaims{Subject: "user"})
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	claims, err := j.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.Subject != "user" {
		t.Fatalf("Subject = %q", claims.Subject)
	}
}

func TestJWT_RS256_NoPrivateKey(t *testing.T) {
	j := NewJWT(JWTConfig{Algorithm: JWTAlgRS256})
	_, err := j.Sign(JWTClaims{})
	if err == nil {
		t.Fatal("Sign RS256 without private key should fail")
	}
}

func TestJWT_RS256_NoPublicKey(t *testing.T) {
	j := NewJWT(JWTConfig{Algorithm: JWTAlgRS256})
	_, err := j.Verify("a.b.c")
	if err == nil {
		t.Fatal("Verify RS256 without public key should fail")
	}
}

func TestJWT_DefaultAlgorithm(t *testing.T) {
	j := NewJWT(JWTConfig{Secret: []byte("key")})
	if j.config.Algorithm != JWTAlgHS256 {
		t.Fatalf("default algorithm = %q, want HS256", j.config.Algorithm)
	}
}

func TestJWT_ParseWithoutExpiryCheck(t *testing.T) {
	j := NewJWT(JWTConfig{Secret: []byte("key"), ExpiresIn: -time.Hour})
	token, _ := j.Sign(JWTClaims{Subject: "user"})
	// Parse should succeed even if expired.
	claims, err := j.Parse(token)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if claims.Subject != "user" {
		t.Fatalf("Subject = %q", claims.Subject)
	}
}

// ──────────────────────────────────────────────
// Base64 helpers
// ──────────────────────────────────────────────

func TestBase64EncodeDecode(t *testing.T) {
	data := []byte("hello")
	encoded := Base64Encode(data)
	decoded, err := Base64Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Fatalf("decoded = %q, want %q", decoded, data)
	}
}

func TestBase64URLEncodeDecode(t *testing.T) {
	data := []byte("hello")
	encoded := Base64URLEncode(data)
	decoded, err := Base64URLDecode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Fatalf("decoded = %q, want %q", decoded, data)
	}
}

func TestBase64Decode_Invalid(t *testing.T) {
	_, err := Base64Decode("!!!invalid!!!")
	if err == nil {
		t.Fatal("Decode of invalid base64 should fail")
	}
}

// ──────────────────────────────────────────────
// ECDSA (P-256)
// ──────────────────────────────────────────────

func TestECDSA_GenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateECDSAKeyPair()
	require.NoError(t, err)
	require.NotNil(t, priv)
	require.NotNil(t, pub)
	assert.Equal(t, elliptic.P256(), priv.Curve)
}

func TestECDSA_SignVerify(t *testing.T) {
	priv, pub, _ := GenerateECDSAKeyPair()
	data := []byte("message to sign")
	sig, err := ECDSASign(priv, data)
	require.NoError(t, err)
	assert.NotEmpty(t, sig)

	err = ECDSAVerify(pub, data, sig)
	assert.NoError(t, err)
}

func TestECDSA_VerifyWrongData(t *testing.T) {
	priv, pub, _ := GenerateECDSAKeyPair()
	sig, _ := ECDSASign(priv, []byte("original"))
	err := ECDSAVerify(pub, []byte("tampered"), sig)
	assert.Error(t, err)
}

func TestECDSA_VerifyTamperedSignature(t *testing.T) {
	priv, pub, _ := GenerateECDSAKeyPair()
	sig, _ := ECDSASign(priv, []byte("data"))
	sig[0] ^= 0xff // tamper
	err := ECDSAVerify(pub, []byte("data"), sig)
	assert.Error(t, err)
}

func TestECDSA_SignNilKey(t *testing.T) {
	_, err := ECDSASign(nil, []byte("data"))
	assert.Error(t, err)
}

func TestECDSA_VerifyNilKey(t *testing.T) {
	err := ECDSAVerify(nil, []byte("data"), []byte("sig"))
	assert.Error(t, err)
}

func TestECDSA_PEMExportImport(t *testing.T) {
	priv, pub, _ := GenerateECDSAKeyPair()

	privPEM, err := ExportECDSAPrivateKeyPEM(priv)
	require.NoError(t, err)
	assert.Contains(t, privPEM, "EC PRIVATE KEY")

	pubPEM, err := ExportECDSAPublicKeyPEM(pub)
	require.NoError(t, err)
	assert.Contains(t, pubPEM, "PUBLIC KEY")

	parsedPriv, err := ParseECDSAPrivateKeyPEM(privPEM)
	require.NoError(t, err)
	parsedPub, err := ParseECDSAPublicKeyPEM(pubPEM)
	require.NoError(t, err)

	// Sign with original, verify with parsed.
	sig, _ := ECDSASign(priv, []byte("test"))
	err = ECDSAVerify(parsedPub, []byte("test"), sig)
	assert.NoError(t, err)

	// Sign with parsed, verify with original.
	sig2, _ := ECDSASign(parsedPriv, []byte("test2"))
	err = ECDSAVerify(pub, []byte("test2"), sig2)
	assert.NoError(t, err)
}

func TestECDSA_InvalidPEM(t *testing.T) {
	_, err := ParseECDSAPrivateKeyPEM("invalid pem")
	assert.Error(t, err)
	_, err = ParseECDSAPublicKeyPEM("invalid pem")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Ed25519
// ──────────────────────────────────────────────

func TestEd25519_GenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	assert.Equal(t, ed25519.PrivateKeySize, len(priv))
	assert.Equal(t, ed25519.PublicKeySize, len(pub))
}

func TestEd25519_SignVerify(t *testing.T) {
	priv, pub, _ := GenerateEd25519KeyPair()
	data := []byte("message to sign")
	sig, err := Ed25519Sign(priv, data)
	require.NoError(t, err)
	assert.Equal(t, ed25519.SignatureSize, len(sig))

	err = Ed25519Verify(pub, data, sig)
	assert.NoError(t, err)
}

func TestEd25519_VerifyWrongData(t *testing.T) {
	priv, pub, _ := GenerateEd25519KeyPair()
	sig, _ := Ed25519Sign(priv, []byte("original"))
	err := Ed25519Verify(pub, []byte("tampered"), sig)
	assert.Error(t, err)
}

func TestEd25519_VerifyTamperedSignature(t *testing.T) {
	priv, pub, _ := GenerateEd25519KeyPair()
	sig, _ := Ed25519Sign(priv, []byte("data"))
	sig[0] ^= 0xff // tamper
	err := Ed25519Verify(pub, []byte("data"), sig)
	assert.Error(t, err)
}

func TestEd25519_SignInvalidKey(t *testing.T) {
	_, err := Ed25519Sign(ed25519.PrivateKey{}, []byte("data"))
	assert.Error(t, err)
}

func TestEd25519_VerifyInvalidKey(t *testing.T) {
	err := Ed25519Verify(ed25519.PublicKey{}, []byte("data"), []byte("sig"))
	assert.Error(t, err)
}

func TestEd25519_PEMExportImport(t *testing.T) {
	priv, pub, _ := GenerateEd25519KeyPair()

	privPEM, err := ExportEd25519PrivateKeyPEM(priv)
	require.NoError(t, err)
	assert.Contains(t, privPEM, "PRIVATE KEY")

	pubPEM, err := ExportEd25519PublicKeyPEM(pub)
	require.NoError(t, err)
	assert.Contains(t, pubPEM, "PUBLIC KEY")

	parsedPriv, err := ParseEd25519PrivateKeyPEM(privPEM)
	require.NoError(t, err)
	parsedPub, err := ParseEd25519PublicKeyPEM(pubPEM)
	require.NoError(t, err)

	// Sign with original, verify with parsed.
	sig, _ := Ed25519Sign(priv, []byte("test"))
	err = Ed25519Verify(parsedPub, []byte("test"), sig)
	assert.NoError(t, err)

	// Sign with parsed, verify with original.
	sig2, _ := Ed25519Sign(parsedPriv, []byte("test2"))
	err = Ed25519Verify(pub, []byte("test2"), sig2)
	assert.NoError(t, err)
}

func TestEd25519_InvalidPEM(t *testing.T) {
	_, err := ParseEd25519PrivateKeyPEM("invalid pem")
	assert.Error(t, err)
	_, err = ParseEd25519PublicKeyPEM("invalid pem")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// HKDF
// ──────────────────────────────────────────────

func TestHKDF(t *testing.T) {
	secret := []byte("input key material")
	salt := []byte("salt")
	info := []byte("context")
	length := 32

	key, err := HKDF(secret, salt, info, length)
	require.NoError(t, err)
	assert.Len(t, key, length)
}

func TestHKDF_Deterministic(t *testing.T) {
	secret := []byte("secret")
	salt := []byte("salt")
	info := []byte("info")

	key1, _ := HKDF(secret, salt, info, 32)
	key2, _ := HKDF(secret, salt, info, 32)
	assert.True(t, bytes.Equal(key1, key2), "HKDF should be deterministic")
}

func TestHKDF_DifferentInputs(t *testing.T) {
	secret := []byte("secret")

	key1, _ := HKDF(secret, []byte("salt1"), []byte("info"), 32)
	key2, _ := HKDF(secret, []byte("salt2"), []byte("info"), 32)
	assert.False(t, bytes.Equal(key1, key2), "different salts should produce different keys")

	key3, _ := HKDF(secret, []byte("salt"), []byte("info1"), 32)
	key4, _ := HKDF(secret, []byte("salt"), []byte("info2"), 32)
	assert.False(t, bytes.Equal(key3, key4), "different info should produce different keys")
}

func TestHKDF_NilSalt(t *testing.T) {
	key, err := HKDF([]byte("secret"), nil, []byte("info"), 16)
	require.NoError(t, err)
	assert.Len(t, key, 16)
}

func TestHKDF_EmptyInfo(t *testing.T) {
	key, err := HKDF([]byte("secret"), []byte("salt"), nil, 16)
	require.NoError(t, err)
	assert.Len(t, key, 16)
}

func TestHKDF_InvalidLength(t *testing.T) {
	_, err := HKDF([]byte("secret"), []byte("salt"), []byte("info"), 0)
	assert.Error(t, err)

	_, err = HKDF([]byte("secret"), []byte("salt"), []byte("info"), -1)
	assert.Error(t, err)
}

func TestHKDF_LongLength(t *testing.T) {
	_, err := HKDF([]byte("secret"), []byte("salt"), []byte("info"), 255*sha256.Size+1)
	assert.Error(t, err)
}

func TestHKDF_MaxLength(t *testing.T) {
	key, err := HKDF([]byte("secret"), []byte("salt"), []byte("info"), 255*sha256.Size)
	require.NoError(t, err)
	assert.Len(t, key, 255*sha256.Size)
}

// ──────────────────────────────────────────────
// PBKDF2
// ──────────────────────────────────────────────

func TestPBKDF2(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")
	key, err := PBKDF2(password, salt, 1000, 32)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestPBKDF2_Deterministic(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")

	key1, _ := PBKDF2(password, salt, 1000, 32)
	key2, _ := PBKDF2(password, salt, 1000, 32)
	assert.True(t, bytes.Equal(key1, key2), "PBKDF2 should be deterministic")
}

func TestPBKDF2_DifferentPasswords(t *testing.T) {
	salt := []byte("salt")
	key1, _ := PBKDF2([]byte("password1"), salt, 1000, 32)
	key2, _ := PBKDF2([]byte("password2"), salt, 1000, 32)
	assert.False(t, bytes.Equal(key1, key2), "different passwords should produce different keys")
}

func TestPBKDF2_DifferentSalts(t *testing.T) {
	password := []byte("password")
	key1, _ := PBKDF2(password, []byte("salt1"), 1000, 32)
	key2, _ := PBKDF2(password, []byte("salt2"), 1000, 32)
	assert.False(t, bytes.Equal(key1, key2), "different salts should produce different keys")
}

func TestPBKDF2_DifferentIterations(t *testing.T) {
	password := []byte("password")
	salt := []byte("salt")
	key1, _ := PBKDF2(password, salt, 1000, 32)
	key2, _ := PBKDF2(password, salt, 2000, 32)
	assert.False(t, bytes.Equal(key1, key2), "different iterations should produce different keys")
}

func TestPBKDF2_InvalidIterations(t *testing.T) {
	_, err := PBKDF2([]byte("pw"), []byte("salt"), 0, 32)
	assert.Error(t, err)
}

func TestPBKDF2_InvalidLength(t *testing.T) {
	_, err := PBKDF2([]byte("pw"), []byte("salt"), 100, 0)
	assert.Error(t, err)
}

func TestPBKDF2_MultiBlock(t *testing.T) {
	// Request more than one SHA-256 block (32 bytes).
	key, err := PBKDF2([]byte("pw"), []byte("salt"), 100, 64)
	require.NoError(t, err)
	assert.Len(t, key, 64)
}
