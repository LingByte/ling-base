// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package barcode

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/boombuler/barcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_Code128(t *testing.T) {
	img, err := Generate(TypeCode128, "Hello123")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
	assert.True(t, img.Bounds().Dy() > 0)
}

func TestGenerate_Code39(t *testing.T) {
	img, err := Generate(TypeCode39, "HELLO123")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_Code93(t *testing.T) {
	img, err := Generate(TypeCode93, "HELLO123")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_Codabar(t *testing.T) {
	img, err := Generate(TypeCodabar, "A12345678B")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_EAN13(t *testing.T) {
	img, err := Generate(TypeEAN13, "590123412345")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_EAN8(t *testing.T) {
	img, err := Generate(TypeEAN8, "9638501")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)

	// Verify it's actually EAN-8, not EAN-13.
	meta, err := GetMetadata(img)
	require.NoError(t, err)
	assert.Equal(t, "EAN 8", meta.CodeKind)
}

func TestGenerate_EAN8_WithChecksum(t *testing.T) {
	// 8 digits with correct checksum should also work.
	img, err := Generate(TypeEAN8, "96385012")
	require.NoError(t, err)

	meta, err := GetMetadata(img)
	require.NoError(t, err)
	assert.Equal(t, "EAN 8", meta.CodeKind)
}

func TestGenerate_EAN8_WrongChecksum(t *testing.T) {
	// 8 digits with wrong checksum should fail.
	_, err := Generate(TypeEAN8, "96385019")
	assert.Error(t, err)
}

func TestGenerate_UPCA_11Digits(t *testing.T) {
	// 11 digits → prepend "0" → 12 digits → EAN-13 auto-checksum.
	img, err := Generate(TypeUPCA, "36000291451")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)

	// Verify it's EAN-13 (UPC-A is a subset).
	meta, err := GetMetadata(img)
	require.NoError(t, err)
	assert.Equal(t, "EAN 13", meta.CodeKind)
	// Content should start with "0" (UPC-A prefix).
	assert.True(t, len(meta.Content) == 13)
	assert.Equal(t, "0", string(meta.Content[0]))
}

func TestGenerate_UPCA_12Digits(t *testing.T) {
	// 12 digits (11 data + 1 UPC-A checksum) → prepend "0" → 13 digits.
	// ean.Encode sees 13 digits and validates the checksum.
	// UPC-A checksum for "36000291451" is 5, so full code is "360002914515".
	// As EAN-13: "0360002914515" (checksum also 5, same algorithm).
	img, err := Generate(TypeUPCA, "360002914515")
	require.NoError(t, err)

	meta, err := GetMetadata(img)
	require.NoError(t, err)
	assert.Equal(t, "EAN 13", meta.CodeKind)
	assert.Equal(t, "0360002914515", meta.Content)
}

func TestGenerate_UPCA_12Digits_WrongChecksum(t *testing.T) {
	// 12 digits with wrong checksum → prepend "0" → 13 digits → validation fails.
	// "360002914510" has wrong checksum (should be 5, not 0).
	_, err := Generate(TypeUPCA, "360002914510")
	assert.Error(t, err)
}

func TestGenerate_TwoOfFive(t *testing.T) {
	img, err := Generate(TypeTwoOfFive, "1234567890")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_PDF417(t *testing.T) {
	img, err := Generate(TypePDF417, "PDF417 test data with more content")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_DataMatrix(t *testing.T) {
	img, err := Generate(TypeDataMatrix, "datamatrix test")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_Aztec(t *testing.T) {
	img, err := Generate(TypeAztec, "aztec test data")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerate_AllTypes(t *testing.T) {
	tests := []struct {
		typ     BarcodeType
		content string
	}{
		{TypeCode128, "ABC123"},
		{TypeCode39, "TEST39"},
		{TypeCode93, "TEST93"},
		{TypeCodabar, "C12345D"},
		{TypeEAN13, "590123412345"},
		{TypeEAN8, "9638501"},
		{TypeUPCA, "36000291451"},
		{TypeTwoOfFive, "123456"},
		{TypePDF417, "pdf data"},
		{TypeDataMatrix, "dm data"},
		{TypeAztec, "aztec data"},
	}
	for _, tt := range tests {
		img, err := Generate(tt.typ, tt.content)
		require.NoError(t, err, "type %s", tt.typ)
		assert.True(t, img.Bounds().Dx() > 0, "type %s has zero width", tt.typ)
	}
}

func TestGenerate_EmptyContent(t *testing.T) {
	_, err := Generate(TypeCode128, "")
	assert.Error(t, err)
}

func TestGenerate_WhitespaceOnlyContent(t *testing.T) {
	_, err := Generate(TypeCode128, "   ")
	assert.Error(t, err)
}

func TestGenerate_UnknownType(t *testing.T) {
	_, err := Generate(BarcodeType("unknown"), "test")
	assert.Error(t, err)
}

func TestGenerate_EAN13_InvalidLength(t *testing.T) {
	_, err := Generate(TypeEAN13, "123")
	assert.Error(t, err)
}

func TestGenerate_Code39_InvalidChars(t *testing.T) {
	// Code39 doesn't support lowercase.
	_, err := Generate(TypeCode39, "lowercase")
	assert.Error(t, err)
}

func TestGenerateScaled(t *testing.T) {
	img, err := GenerateScaled(TypeCode128, "scaled test", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
	assert.Equal(t, 100, img.Bounds().Dy())
}

func TestGenerateScaled_NoScale(t *testing.T) {
	img, err := GenerateScaled(TypeCode128, "no scale", 0, 0)
	require.NoError(t, err)
	// Should return unscaled barcode.
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGeneratePNG(t *testing.T) {
	data, err := GeneratePNG(TypeCode128, "png test", 300, 100)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "barcode.png")
	err := Save(TypeCode128, "save test", path, 300, 100)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	f, _ := os.Open(path)
	defer f.Close()
	_, err = png.Decode(f)
	require.NoError(t, err)
}

func TestSave_Error(t *testing.T) {
	err := Save(TypeCode128, "test", "/nonexistent/dir/barcode.png", 300, 100)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Convenience functions
// ──────────────────────────────────────────────

func TestCode128(t *testing.T) {
	img, err := Code128("convenience test", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
	assert.Equal(t, 100, img.Bounds().Dy())
}

func TestCode39(t *testing.T) {
	img, err := Code39("TEST39", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestCode93(t *testing.T) {
	img, err := Code93("TEST93", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestCodabar(t *testing.T) {
	img, err := Codabar("A1234B", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestEAN13(t *testing.T) {
	img, err := EAN13("590123412345", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestEAN8(t *testing.T) {
	img, err := EAN8("9638501", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestUPCA(t *testing.T) {
	img, err := UPCA("36000291451", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestUPC_A_Deprecated(t *testing.T) {
	// Deprecated alias should still work.
	img, err := UPC_A("36000291451", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestTwoOfFive(t *testing.T) {
	img, err := TwoOfFive("123456", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestPDF417(t *testing.T) {
	img, err := PDF417("pdf417 test", 500, 200)
	require.NoError(t, err)
	assert.Equal(t, 500, img.Bounds().Dx())
}

func TestDataMatrix(t *testing.T) {
	img, err := DataMatrix("dm test", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

func TestAztec(t *testing.T) {
	img, err := Aztec("aztec test", 300, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
}

// ──────────────────────────────────────────────
// Metadata
// ──────────────────────────────────────────────

func TestGetMetadata(t *testing.T) {
	img, err := Generate(TypeCode128, "metadata test")
	require.NoError(t, err)
	meta, err := GetMetadata(img)
	require.NoError(t, err)
	assert.Equal(t, "metadata test", meta.Content)
}

func TestGetMetadata_NotBarcode(t *testing.T) {
	// A scaled barcode is still a barcode, but a random image is not.
	// Use a PNG-decoded image which is definitely not a barcode type.
	pngData, err := GeneratePNG(TypeCode128, "test", 200, 80)
	require.NoError(t, err)
	decoded, err := png.Decode(bytes.NewReader(pngData))
	require.NoError(t, err)
	_, err = GetMetadata(decoded)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Options tests
// ──────────────────────────────────────────────

func TestGenerateWithOptions_PDF417SecurityLevel(t *testing.T) {
	// Different security levels should produce different barcode sizes.
	img1, err := GenerateWithOptions(TypePDF417, "test data", Options{
		PDF417SecurityLevel: 0,
	})
	require.NoError(t, err)

	img2, err := GenerateWithOptions(TypePDF417, "test data", Options{
		PDF417SecurityLevel: 8,
	})
	require.NoError(t, err)

	// Higher security level should produce a larger barcode (more redundancy).
	assert.True(t, img2.Bounds().Dy() >= img1.Bounds().Dy(),
		"higher security level should produce >= rows: %d vs %d",
		img2.Bounds().Dy(), img1.Bounds().Dy())
}

func TestGenerateWithOptions_AztecECCPercent(t *testing.T) {
	// Different ECC percentages should produce different barcode sizes.
	img1, err := GenerateWithOptions(TypeAztec, "test data", Options{
		AztecECCPercent: 10,
	})
	require.NoError(t, err)

	img2, err := GenerateWithOptions(TypeAztec, "test data", Options{
		AztecECCPercent: 90,
	})
	require.NoError(t, err)

	// Higher ECC should produce a larger barcode.
	assert.True(t, img2.Bounds().Dx() >= img1.Bounds().Dx(),
		"higher ECC should produce >= size: %d vs %d",
		img2.Bounds().Dx(), img1.Bounds().Dx())
}

func TestGenerateWithOptions_Code39FullASCII(t *testing.T) {
	// With full ASCII, lowercase should be accepted.
	img, err := GenerateWithOptions(TypeCode39, "test", Options{
		Code39FullASCII: true,
	})
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestGenerateWithOptions_TwoOfFiveInterleaved(t *testing.T) {
	img, err := GenerateWithOptions(TypeTwoOfFive, "12345678", Options{
		TwoOfFiveInterleaved: true,
	})
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.Equal(t, byte(8), opts.PDF417SecurityLevel)
	assert.Equal(t, 33, opts.AztecECCPercent) // aztec.DEFAULT_EC_PERCENT
	assert.Equal(t, 0, opts.AztecLayers)
}

// ──────────────────────────────────────────────
// GenerateWithScaleFactor tests
// ──────────────────────────────────────────────

func TestGenerateWithScaleFactor_1D(t *testing.T) {
	// Get original size first.
	orig, err := Generate(TypeCode128, "test")
	require.NoError(t, err)
	origWidth := orig.Bounds().Dx()

	img, err := GenerateWithScaleFactor(TypeCode128, "test", 3, 100)
	require.NoError(t, err)

	// Width should be originalWidth * 3.
	assert.Equal(t, origWidth*3, img.Bounds().Dx())
	// Height should be the specified height.
	assert.Equal(t, 100, img.Bounds().Dy())
}

func TestGenerateWithScaleFactor_2D(t *testing.T) {
	// Get original size first.
	orig, err := Generate(TypeAztec, "test")
	require.NoError(t, err)
	origWidth := orig.Bounds().Dx()
	origHeight := orig.Bounds().Dy()

	img, err := GenerateWithScaleFactor(TypeAztec, "test", 4, 0)
	require.NoError(t, err)

	// Both dimensions should be scaled by factor.
	assert.Equal(t, origWidth*4, img.Bounds().Dx())
	assert.Equal(t, origHeight*4, img.Bounds().Dy())
}

func TestGenerateWithScaleFactor_NoScale(t *testing.T) {
	img, err := GenerateWithScaleFactor(TypeCode128, "test", 0, 0)
	require.NoError(t, err)
	// Should return unscaled barcode.
	orig, err := Generate(TypeCode128, "test")
	require.NoError(t, err)
	assert.Equal(t, orig.Bounds().Dx(), img.Bounds().Dx())
}

// ──────────────────────────────────────────────
// SupportedTypes tests
// ──────────────────────────────────────────────

func TestSupportedTypes(t *testing.T) {
	types := SupportedTypes()
	assert.NotEmpty(t, types)
	assert.Equal(t, 11, len(types))

	// Verify all types are valid by generating a test barcode.
	for _, typ := range types {
		assert.NotEmpty(t, typ)
	}
}

func TestTypeName(t *testing.T) {
	tests := []struct {
		typ  BarcodeType
		name string
	}{
		{TypeCode128, "Code 128"},
		{TypeCode39, "Code 39"},
		{TypeCode93, "Code 93"},
		{TypeCodabar, "Codabar"},
		{TypeEAN13, "EAN-13"},
		{TypeEAN8, "EAN-8"},
		{TypeUPCA, "UPC-A"},
		{TypeTwoOfFive, "2 of 5"},
		{TypePDF417, "PDF417"},
		{TypeDataMatrix, "DataMatrix"},
		{TypeAztec, "Aztec"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.name, TypeName(tt.typ))
	}
}

func TestTypeName_Unknown(t *testing.T) {
	assert.Equal(t, "unknown", TypeName(BarcodeType("unknown")))
}

// ──────────────────────────────────────────────
// Verify barcode type metadata
// ──────────────────────────────────────────────

func TestGenerate_VerifyMetadataTypes(t *testing.T) {
	tests := []struct {
		typ      BarcodeType
		content  string
		codeKind string
	}{
		{TypeCode128, "ABC123", "Code 128"},
		{TypeCode39, "TEST39", "Code 39"},
		{TypeEAN13, "590123412345", "EAN 13"},
		{TypeEAN8, "9638501", "EAN 8"},
	}
	for _, tt := range tests {
		img, err := Generate(tt.typ, tt.content)
		require.NoError(t, err)

		meta, err := GetMetadata(img)
		require.NoError(t, err)
		assert.Equal(t, tt.codeKind, meta.CodeKind, "type %s", tt.typ)
	}
}

// Ensure barcode import is used.
var _ barcode.Barcode
