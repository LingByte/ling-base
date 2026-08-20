// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package totp

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDefaults(t *testing.T) {
	key, err := Generate(Options{
		Issuer:      "MyApp",
		AccountName: "alice@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.NotEmpty(t, key.Secret())
	assert.NotEmpty(t, key.URL())
	assert.Equal(t, "MyApp", key.Issuer())
	assert.Equal(t, "alice@example.com", key.AccountName())
	assert.Equal(t, uint64(DefaultPeriod), key.Period())
	assert.Equal(t, DigitsSix, key.Digits())
	assert.Equal(t, AlgorithmSHA1, key.Algorithm())
	assert.True(t, strings.HasPrefix(key.URL(), "otpauth://totp/"))
}

func TestGenerateCustomOptions(t *testing.T) {
	key, err := Generate(Options{
		Issuer:      "TestApp",
		AccountName: "bob@test.com",
		Period:      60,
		Digits:      DigitsEight,
		Algorithm:   AlgorithmSHA256,
		SecretSize:  64,
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(60), key.Period())
	assert.Equal(t, DigitsEight, key.Digits())
	assert.Equal(t, AlgorithmSHA256, key.Algorithm())
}

func TestGenerateErrors(t *testing.T) {
	_, err := Generate(Options{AccountName: "a@b.com"})
	assert.ErrorIs(t, err, ErrInvalidIssuer)

	_, err = Generate(Options{Issuer: "MyApp"})
	assert.ErrorIs(t, err, ErrInvalidAccount)

	_, err = Generate(Options{Issuer: "  ", AccountName: "a@b.com"})
	assert.ErrorIs(t, err, ErrInvalidIssuer)
}

func TestValidateRoundTrip(t *testing.T) {
	key, err := Generate(Options{
		Issuer:      "Test",
		AccountName: "rt@test.com",
	})
	require.NoError(t, err)

	code, err := CurrentCode(key.Secret(), nil)
	require.NoError(t, err)
	require.Len(t, code, 6)

	assert.True(t, Validate(code, key.Secret(), nil))
	assert.False(t, Validate("000000", key.Secret(), nil))
}

func TestValidateEmptyInputs(t *testing.T) {
	assert.False(t, Validate("", "SECRET", nil))
	assert.False(t, Validate("123456", "", nil))
	assert.False(t, Validate("", "", nil))
}

func TestValidateWhitespaceTrimmed(t *testing.T) {
	key, err := Generate(Options{Issuer: "T", AccountName: "w@test.com"})
	require.NoError(t, err)
	code, err := CurrentCode(key.Secret(), nil)
	require.NoError(t, err)

	assert.True(t, Validate("  "+code+"  ", "  "+key.Secret()+"  ", nil))
}

func TestValidateAtSpecificTime(t *testing.T) {
	key, err := Generate(Options{Issuer: "T", AccountName: "at@test.com"})
	require.NoError(t, err)

	code, err := CurrentCode(key.Secret(), nil)
	require.NoError(t, err)
	now := time.Now().UTC()

	// Current code should validate at current time.
	assert.True(t, ValidateAt(code, key.Secret(), now, nil))

	// Same code should not validate 2 minutes ago (outside ±1 window of ±30s).
	assert.False(t, ValidateAt(code, key.Secret(), now.Add(-2*time.Minute), nil))

	// With skew=5 (±150s), it should validate at -2 minutes (120s).
	assert.True(t, ValidateAt(code, key.Secret(), now.Add(-2*time.Minute), &ValidateOptions{Skew: 5}))
}

func TestValidateStrictSkew(t *testing.T) {
	key, err := Generate(Options{Issuer: "T", AccountName: "s@test.com"})
	require.NoError(t, err)

	now := time.Now().UTC()
	code, err := CurrentCode(key.Secret(), nil)
	require.NoError(t, err)

	// Strict skew=0: a code from 31 seconds ago should fail.
	assert.False(t, ValidateAt(code, key.Secret(), now.Add(-31*time.Second), &ValidateOptions{Skew: 0}))
}

func TestCurrentCodeEmptySecret(t *testing.T) {
	_, err := CurrentCode("", nil)
	assert.ErrorIs(t, err, ErrInvalidSecret)
}

func TestHOTPRoundTrip(t *testing.T) {
	key, err := GenerateHOTP(Options{
		Issuer:      "HOTPTest",
		AccountName: "hotp@test.com",
	})
	require.NoError(t, err)
	require.NotNil(t, key)
	assert.True(t, strings.HasPrefix(key.URL(), "otpauth://hotp/"))

	code, err := HOTPCode(key.Secret(), 0)
	require.NoError(t, err)
	assert.True(t, ValidateHOTP(code, key.Secret(), 0))
	assert.False(t, ValidateHOTP(code, key.Secret(), 1))
}

func TestHOTPValidateEmpty(t *testing.T) {
	assert.False(t, ValidateHOTP("", "SECRET", 0))
	assert.False(t, ValidateHOTP("123456", "", 0))
}

func TestHOTPCodeEmptySecret(t *testing.T) {
	_, err := HOTPCode("", 0)
	assert.ErrorIs(t, err, ErrInvalidSecret)
}

func TestHOTPGenerateErrors(t *testing.T) {
	_, err := GenerateHOTP(Options{AccountName: "a@b.com"})
	assert.ErrorIs(t, err, ErrInvalidIssuer)

	_, err = GenerateHOTP(Options{Issuer: "App"})
	assert.ErrorIs(t, err, ErrInvalidAccount)
}

// ──────────────────────────────────────────────
// QR code rendering
// ──────────────────────────────────────────────

func TestQRPNG(t *testing.T) {
	key, err := Generate(Options{Issuer: "Test", AccountName: "qr@test.com"})
	require.NoError(t, err)

	png, err := QRPNG(key.URL(), 128)
	require.NoError(t, err)
	assert.NotEmpty(t, png)
	// PNG magic bytes.
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, png[:4])
}

func TestQRPNGDefaultSize(t *testing.T) {
	key, err := Generate(Options{Issuer: "T", AccountName: "d@test.com"})
	require.NoError(t, err)
	png, err := QRPNG(key.URL(), 0)
	require.NoError(t, err)
	assert.NotEmpty(t, png)
}

func TestQRPNGEmptyURL(t *testing.T) {
	_, err := QRPNG("", 256)
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestQRDataURL(t *testing.T) {
	key, err := Generate(Options{Issuer: "Test", AccountName: "du@test.com"})
	require.NoError(t, err)

	url, err := QRDataURL(key.URL(), 64)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "data:image/png;base64,"))
}

// ──────────────────────────────────────────────
// Backup / recovery codes
// ──────────────────────────────────────────────

func TestGenerateBackupCodeDefaults(t *testing.T) {
	code, err := GenerateBackupCode(BackupOptions{})
	require.NoError(t, err)
	// Default length 8 + 1 separator = 9 chars.
	assert.Len(t, code, 9)
	assert.Contains(t, code, "-")
}

func TestGenerateBackupCodeTooShort(t *testing.T) {
	_, err := GenerateBackupCode(BackupOptions{Length: 3})
	assert.ErrorIs(t, err, ErrInvalidBackupLength)
}

func TestGenerateBackupCodesDefaults(t *testing.T) {
	codes, hashes, err := GenerateBackupCodes(BackupOptions{})
	require.NoError(t, err)
	assert.Len(t, codes, DefaultBackupCount)
	assert.Len(t, hashes, DefaultBackupCount)

	// All codes should be unique.
	seen := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		seen[c] = struct{}{}
	}
	assert.Len(t, seen, len(codes), "codes should be unique")

	// All hashes should be non-empty and unique.
	seenH := make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		assert.NotEmpty(t, h)
		seenH[h] = struct{}{}
	}
	assert.Len(t, seenH, len(hashes), "hashes should be unique")
}

func TestGenerateBackupCodesCustomCount(t *testing.T) {
	codes, _, err := GenerateBackupCodes(BackupOptions{Count: 5, Length: 12})
	require.NoError(t, err)
	assert.Len(t, codes, 5)
}

func TestGenerateBackupCodesNegativeCount(t *testing.T) {
	_, _, err := GenerateBackupCodes(BackupOptions{Count: -1})
	assert.ErrorIs(t, err, ErrInvalidBackupCount)
}

func TestValidateBackupCodeRoundTrip(t *testing.T) {
	codes, hashes, err := GenerateBackupCodes(BackupOptions{Count: 5})
	require.NoError(t, err)
	for i, c := range codes {
		assert.True(t, ValidateBackupCode(c, hashes[i]))
	}
}

func TestValidateBackupCodeSetFindsMatch(t *testing.T) {
	codes, hashes, err := GenerateBackupCodes(BackupOptions{Count: 5})
	require.NoError(t, err)

	idx, ok := ValidateBackupCodeSet(codes[2], hashes)
	require.True(t, ok)
	assert.Equal(t, 2, idx)

	idx, ok = ValidateBackupCodeSet("WRONG-CODE", hashes)
	assert.False(t, ok)
	assert.Equal(t, -1, idx)
}

func TestValidateBackupCodeCaseInsensitive(t *testing.T) {
	codes, hashes, err := GenerateBackupCodes(BackupOptions{Count: 3})
	require.NoError(t, err)
	lower := strings.ToLower(codes[0])
	assert.True(t, ValidateBackupCode(lower, hashes[0]))
}

func TestValidateBackupCodeWithoutSeparator(t *testing.T) {
	codes, hashes, err := GenerateBackupCodes(BackupOptions{Count: 3})
	require.NoError(t, err)
	noSep := strings.ReplaceAll(codes[0], "-", "")
	assert.True(t, ValidateBackupCode(noSep, hashes[0]))
}

func TestValidateBackupCodeEmptyInputs(t *testing.T) {
	assert.False(t, ValidateBackupCode("", "somehash"))
	assert.False(t, ValidateBackupCode("ABCD-EFGH", ""))
}

func TestHashBackupCodeConsistency(t *testing.T) {
	h1 := HashBackupCode("A1B2-C3D4")
	h2 := HashBackupCode("a1b2c3d4")
	h3 := HashBackupCode("A1B2C3D4")
	assert.Equal(t, h1, h2)
	assert.Equal(t, h1, h3)
}

func TestNormalizeBackupCode(t *testing.T) {
	assert.Equal(t, "A1B2C3D4", NormalizeBackupCode("a1b2-c3d4"))
	assert.Equal(t, "A1B2C3D4", NormalizeBackupCode("A1B2 C3D4"))
	assert.Equal(t, "A1B2C3D4", NormalizeBackupCode("a1b2_c3d4"))
	assert.Equal(t, "", NormalizeBackupCode(""))
}

func TestBackupCodesUnambiguousCharset(t *testing.T) {
	for i := 0; i < 100; i++ {
		code, err := GenerateBackupCode(BackupOptions{Length: 20})
		require.NoError(t, err)
		norm := NormalizeBackupCode(code)
		for _, c := range norm {
			assert.NotContains(t, "01OI", string(c), "ambiguous char in code: %s", code)
		}
	}
}
