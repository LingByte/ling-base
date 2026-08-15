// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package crypto provides encryption and signing utilities:
//
//   - AES: GCM and CBC modes with padding
//   - RSA: key generation, encryption/decryption, signing/verification
//   - HMAC-SHA signatures
//   - JWT: create and parse JSON Web Tokens (HS256/RS256)
//
// # Quick start
//
//	// AES-GCM
//	key := crypto.RandomKey(32) // AES-256
//	ciphertext, _ := crypto.AESGCMEncrypt(key, []byte("secret"))
//	plaintext, _ := crypto.AESGCMDecrypt(key, ciphertext)
//
//	// RSA
//	priv, pub, _ := crypto.GenerateRSAKeyPair(2048)
//	enc, _ := crypto.RSAEncrypt(pub, []byte("data"))
//	dec, _ := crypto.RSADecrypt(priv, enc)
//
//	// JWT
//	token, _ := crypto.NewJWT(crypto.JWTConfig{Secret: []byte("key")}).Sign(claims)
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// Random key generation
// ──────────────────────────────────────────────

// RandomKey returns n cryptographically secure random bytes.
func RandomKey(n int) ([]byte, error) {
	key := make([]byte, n)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("crypto: generate key: %w", err)
	}
	return key, nil
}

// MustRandomKey is like RandomKey but panics on error.
func MustRandomKey(n int) []byte {
	key, err := RandomKey(n)
	if err != nil {
		panic(err)
	}
	return key
}

// ──────────────────────────────────────────────
// AES-GCM
// ──────────────────────────────────────────────

// AESGCMEncrypt encrypts plaintext using AES-GCM with the given key.
// The key must be 16, 24, or 32 bytes (AES-128/192/256).
// Returns nonce + ciphertext concatenated.
func AESGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// AESGCMDecrypt decrypts data produced by AESGCMEncrypt.
func AESGCMDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: GCM decrypt: %w", err)
	}
	return plaintext, nil
}

// AESGCMEncryptBase64 encrypts and returns base64-encoded result.
func AESGCMEncryptBase64(key, plaintext []byte) (string, error) {
	data, err := AESGCMEncrypt(key, plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// AESGCMDecryptBase64 decrypts a base64-encoded ciphertext.
func AESGCMDecryptBase64(key []byte, encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("crypto: base64 decode: %w", err)
	}
	return AESGCMDecrypt(key, data)
}

// ──────────────────────────────────────────────
// AES-CBC (with PKCS7 padding)
// ──────────────────────────────────────────────

// AESCBCEncrypt encrypts plaintext using AES-CBC with PKCS7 padding.
// The key must be 16, 24, or 32 bytes. A random IV is prepended to the output.
func AESCBCEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: AES cipher: %w", err)
	}
	// PKCS7 padding.
	blockSize := block.BlockSize()
	padded := pkcs7Pad(plaintext, blockSize)

	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("crypto: IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)

	return append(iv, ciphertext...), nil
}

// AESCBCDecrypt decrypts data produced by AESCBCEncrypt.
func AESCBCDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: AES cipher: %w", err)
	}
	blockSize := block.BlockSize()
	if len(data) < blockSize {
		return nil, errors.New("crypto: ciphertext too short")
	}
	iv, ciphertext := data[:blockSize], data[blockSize:]
	if len(ciphertext)%blockSize != 0 {
		return nil, errors.New("crypto: ciphertext not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	return pkcs7Unpad(plaintext, blockSize)
}

// pkcs7Pad applies PKCS7 padding.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}

// pkcs7Unpad removes PKCS7 padding.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("crypto: invalid padding")
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > blockSize {
		return nil, errors.New("crypto: invalid padding length")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, errors.New("crypto: invalid padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}

// ──────────────────────────────────────────────
// RSA
// ──────────────────────────────────────────────

// GenerateRSAKeyPair generates an RSA key pair of the given bit size.
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("crypto: generate RSA key: %w", err)
	}
	return priv, &priv.PublicKey, nil
}

// RSAEncrypt encrypts plaintext using RSA-OAEP with SHA-256.
func RSAEncrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: RSA encrypt: %w", err)
	}
	return ciphertext, nil
}

// RSADecrypt decrypts ciphertext using RSA-OAEP with SHA-256.
func RSADecrypt(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: RSA decrypt: %w", err)
	}
	return plaintext, nil
}

// RSASign signs data using RSASSA-PKCS1-v1.5 with SHA-256.
func RSASign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	hashed := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, 0, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: RSA sign: %w", err)
	}
	return sig, nil
}

// RSAVerify verifies an RSASSA-PKCS1-v1.5 signature.
func RSAVerify(pub *rsa.PublicKey, data, sig []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(pub, 0, hashed[:], sig)
}

// ExportRSAPrivateKeyPEM exports a private key as PEM-encoded string.
func ExportRSAPrivateKeyPEM(priv *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("crypto: marshal private key: %w", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return string(pemBlock), nil
}

// ExportRSAPublicKeyPEM exports a public key as PEM-encoded string.
func ExportRSAPublicKeyPEM(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("crypto: marshal public key: %w", err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return string(pemBlock), nil
}

// ParseRSAPrivateKeyPEM parses a PEM-encoded RSA private key.
func ParseRSAPrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("crypto: invalid PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1.
		key2, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("crypto: parse private key: %w", err)
		}
		return key2, nil
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("crypto: not an RSA private key")
	}
	return rsaKey, nil
}

// ParseRSAPublicKeyPEM parses a PEM-encoded RSA public key.
func ParseRSAPublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("crypto: invalid PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1.
		key2, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("crypto: parse public key: %w", err)
		}
		return key2, nil
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("crypto: not an RSA public key")
	}
	return rsaKey, nil
}

// ──────────────────────────────────────────────
// HMAC-SHA signatures
// ──────────────────────────────────────────────

// SignSHA256 signs data using HMAC-SHA256.
func SignSHA256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifySHA256 verifies an HMAC-SHA256 signature in constant time.
func VerifySHA256(data, key, sig []byte) bool {
	expected := SignSHA256(data, key)
	return hmac.Equal(expected, sig)
}

// SignSHA512 signs data using HMAC-SHA512.
func SignSHA512(data, key []byte) []byte {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifySHA512 verifies an HMAC-SHA512 signature in constant time.
func VerifySHA512(data, key, sig []byte) bool {
	expected := SignSHA512(data, key)
	return hmac.Equal(expected, sig)
}

// ──────────────────────────────────────────────
// JWT (JSON Web Token)
// ──────────────────────────────────────────────

// JWTAlgorithm represents a JWT signing algorithm.
type JWTAlgorithm string

const (
	JWTAlgHS256 JWTAlgorithm = "HS256"
	JWTAlgHS512 JWTAlgorithm = "HS512"
	JWTAlgRS256 JWTAlgorithm = "RS256"
)

// JWTConfig configures the JWT signer.
type JWTConfig struct {
	Algorithm JWTAlgorithm    // default HS256
	Secret    []byte          // for HS256/HS512
	RSAPriv   *rsa.PrivateKey // for RS256 signing
	RSAPub    *rsa.PublicKey  // for RS256 verification
	Issuer    string          // iss claim
	Audience  string          // aud claim
	ExpiresIn time.Duration   // exp claim
}

// JWTClaims represents JWT claims.
type JWTClaims struct {
	Issuer    string                 `json:"iss,omitempty"`
	Subject   string                 `json:"sub,omitempty"`
	Audience  string                 `json:"aud,omitempty"`
	ExpiresAt int64                  `json:"exp,omitempty"`
	NotBefore int64                  `json:"nbf,omitempty"`
	IssuedAt  int64                  `json:"iat,omitempty"`
	ID        string                 `json:"jti,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

// JWT is a JWT token signer/verifier.
type JWT struct {
	config JWTConfig
}

// NewJWT creates a new JWT signer with the given config.
func NewJWT(cfg JWTConfig) *JWT {
	if cfg.Algorithm == "" {
		cfg.Algorithm = JWTAlgHS256
	}
	return &JWT{config: cfg}
}

// Sign creates a signed JWT token from the given claims.
func (j *JWT) Sign(claims JWTClaims) (string, error) {
	header := map[string]string{
		"alg": string(j.config.Algorithm),
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("crypto: JWT header: %w", err)
	}

	now := time.Now()
	if claims.Issuer == "" {
		claims.Issuer = j.config.Issuer
	}
	if claims.Audience == "" {
		claims.Audience = j.config.Audience
	}
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now.Unix()
	}
	if claims.NotBefore == 0 {
		claims.NotBefore = now.Unix()
	}
	if claims.ExpiresAt == 0 && j.config.ExpiresIn != 0 {
		claims.ExpiresAt = now.Add(j.config.ExpiresIn).Unix()
	}

	// Build payload map.
	payload := map[string]interface{}{}
	if claims.Issuer != "" {
		payload["iss"] = claims.Issuer
	}
	if claims.Subject != "" {
		payload["sub"] = claims.Subject
	}
	if claims.Audience != "" {
		payload["aud"] = claims.Audience
	}
	if claims.ExpiresAt != 0 {
		payload["exp"] = claims.ExpiresAt
	}
	if claims.NotBefore != 0 {
		payload["nbf"] = claims.NotBefore
	}
	if claims.IssuedAt != 0 {
		payload["iat"] = claims.IssuedAt
	}
	if claims.ID != "" {
		payload["jti"] = claims.ID
	}
	for k, v := range claims.Extra {
		payload[k] = v
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("crypto: JWT payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	sig, err := j.sign([]byte(signingInput))
	if err != nil {
		return "", err
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}

// Parse parses and verifies a JWT token, returning the claims.
// Does not check expiration; use Verify for full validation.
func (j *JWT) Parse(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("crypto: JWT: invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("crypto: JWT: decode signature: %w", err)
	}

	if err := j.verify([]byte(signingInput), sig); err != nil {
		return nil, fmt.Errorf("crypto: JWT: %w", err)
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("crypto: JWT: decode payload: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return nil, fmt.Errorf("crypto: JWT: parse payload: %w", err)
	}

	claims := &JWTClaims{Extra: map[string]interface{}{}}
	for k, v := range raw {
		switch k {
		case "iss":
			if s, ok := v.(string); ok {
				claims.Issuer = s
			}
		case "sub":
			if s, ok := v.(string); ok {
				claims.Subject = s
			}
		case "aud":
			if s, ok := v.(string); ok {
				claims.Audience = s
			}
		case "exp":
			if f, ok := v.(float64); ok {
				claims.ExpiresAt = int64(f)
			}
		case "nbf":
			if f, ok := v.(float64); ok {
				claims.NotBefore = int64(f)
			}
		case "iat":
			if f, ok := v.(float64); ok {
				claims.IssuedAt = int64(f)
			}
		case "jti":
			if s, ok := v.(string); ok {
				claims.ID = s
			}
		default:
			claims.Extra[k] = v
		}
	}
	return claims, nil
}

// Verify parses, verifies, and checks expiration of a JWT token.
func (j *JWT) Verify(token string) (*JWTClaims, error) {
	claims, err := j.Parse(token)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return nil, errors.New("crypto: JWT: token expired")
	}
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, errors.New("crypto: JWT: token not valid yet")
	}
	return claims, nil
}

// sign signs the signing input using the configured algorithm.
func (j *JWT) sign(data []byte) ([]byte, error) {
	switch j.config.Algorithm {
	case JWTAlgHS256:
		return SignSHA256(data, j.config.Secret), nil
	case JWTAlgHS512:
		return SignSHA512(data, j.config.Secret), nil
	case JWTAlgRS256:
		if j.config.RSAPriv == nil {
			return nil, errors.New("crypto: JWT: RS256 requires private key")
		}
		return RSASign(j.config.RSAPriv, data)
	default:
		return nil, fmt.Errorf("crypto: JWT: unsupported algorithm %q", j.config.Algorithm)
	}
}

// verify verifies the signature using the configured algorithm.
func (j *JWT) verify(data, sig []byte) error {
	switch j.config.Algorithm {
	case JWTAlgHS256:
		if !VerifySHA256(data, j.config.Secret, sig) {
			return errors.New("invalid signature")
		}
		return nil
	case JWTAlgHS512:
		if !VerifySHA512(data, j.config.Secret, sig) {
			return errors.New("invalid signature")
		}
		return nil
	case JWTAlgRS256:
		if j.config.RSAPub == nil {
			return errors.New("RS256 requires public key")
		}
		return RSAVerify(j.config.RSAPub, data, sig)
	default:
		return fmt.Errorf("unsupported algorithm %q", j.config.Algorithm)
	}
}

// ──────────────────────────────────────────────
// Base64 helpers
// ──────────────────────────────────────────────

// Base64Encode returns standard base64 encoding.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes a standard base64 string.
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// Base64URLEncode returns URL-safe base64 encoding (no padding).
func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes a URL-safe base64 string (no padding).
func Base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// ──────────────────────────────────────────────
// Random ID generation
// ──────────────────────────────────────────────

// RandomHex returns n cryptographically secure random bytes as a hex string.
func RandomHex(n int) (string, error) {
	b, err := RandomKey(n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// RandomBase64 returns n cryptographically secure random bytes as base64.
func RandomBase64(n int) (string, error) {
	b, err := RandomKey(n)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// RandomReader returns the crypto/rand reader for io.Copy usage.
func RandomReader() io.Reader {
	return rand.Reader
}
