// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package barcode

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

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
}

func TestGenerate_UPCA(t *testing.T) {
	// UPC-A: 11 digits (checksum auto-calculated, leading 0 prepended for EAN-13).
	img, err := Generate(TypeUPCA, "36000291451")
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
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

func TestUPC_A(t *testing.T) {
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
