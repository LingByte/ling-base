// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qrcode

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	img, err := Generate("https://github.com/LingByte/ling-base", ECLMedium, 256)
	require.NoError(t, err)
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	assert.Equal(t, 256, w)
	assert.Equal(t, 256, h)
}

func TestGenerate_AllLevels(t *testing.T) {
	for _, level := range []ErrorCorrectionLevel{ECLLow, ECLMedium, ECLQuartile, ECLHigh} {
		img, err := Generate("test data", level, 200)
		require.NoError(t, err, "level %d", level)
		assert.Equal(t, 200, img.Bounds().Dx())
	}
}

func TestGenerate_NoScale(t *testing.T) {
	img, err := Generate("hello", ECLMedium, 0)
	require.NoError(t, err)
	// Unscaled QR should be small (1 module per pixel).
	w := img.Bounds().Dx()
	assert.True(t, w > 0 && w < 100, "unscaled QR width = %d", w)
}

func TestGenerate_EmptyText(t *testing.T) {
	_, err := Generate("", ECLMedium, 200)
	assert.Error(t, err)
}

func TestGeneratePNG(t *testing.T) {
	data, err := GeneratePNG("test payload", ECLHigh, 200)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid PNG.
	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qr.png")
	err := Save("save test", path, ECLMedium, 200)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Verify it's a valid PNG.
	f, _ := os.Open(path)
	defer f.Close()
	_, err = png.Decode(f)
	require.NoError(t, err)
}

func TestSave_Error(t *testing.T) {
	err := Save("test", "/nonexistent/dir/qr.png", ECLMedium, 200)
	assert.Error(t, err)
}

func TestGenerateWithLogo(t *testing.T) {
	// Create a simple logo.
	logo := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	img, err := GenerateWithLogo("https://lingbyte.com", ECLHigh, 300, logo, 60)
	require.NoError(t, err)
	assert.Equal(t, 300, img.Bounds().Dx())
	assert.Equal(t, 300, img.Bounds().Dy())

	// Center area should have logo pixels (red).
	center := img.At(150, 150)
	r, g, b, _ := center.RGBA()
	assert.Greater(t, r/256, uint32(200))
	assert.Less(t, g/256, uint32(50))
	assert.Less(t, b/256, uint32(50))
}

func TestGenerateWithLogo_DefaultSize(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	img, err := GenerateWithLogo("logo default size", ECLHigh, 250, logo, 0)
	require.NoError(t, err)
	assert.Equal(t, 250, img.Bounds().Dx())
}

func TestGenerateWithLogo_NilLogo(t *testing.T) {
	img, err := GenerateWithLogo("no logo", ECLMedium, 200, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, 200, img.Bounds().Dx())
}

func TestGenerateWithLogo_EmptyText(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 10, 10))
	_, err := GenerateWithLogo("", ECLHigh, 200, logo, 40)
	assert.Error(t, err)
}

func TestGenerateWithLogo_LogoCappedSize(t *testing.T) {
	// Logo size larger than 1/3 of QR should be capped.
	logo := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	img, err := GenerateWithLogo("cap test", ECLHigh, 200, logo, 150)
	require.NoError(t, err)
	assert.Equal(t, 200, img.Bounds().Dx())
}

func TestGenerateWithLogoPNG(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 30, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 30; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 0, A: 255})
		}
	}
	data, err := GenerateWithLogoPNG("png logo test", ECLHigh, 250, logo, 50)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestSaveWithLogo(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 30, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 30; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 255, A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), "qr_logo.png")
	err := SaveWithLogo("save with logo", path, ECLHigh, 250, logo, 50)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestSaveWithLogo_Error(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 10, 10))
	err := SaveWithLogo("test", "/nonexistent/dir/qr.png", ECLHigh, 200, logo, 40)
	assert.Error(t, err)
}

func TestGenerateWithLogoFile(t *testing.T) {
	// Create a temp logo file.
	logoPath := filepath.Join(t.TempDir(), "logo.png")
	logo := image.NewRGBA(image.Rect(0, 0, 30, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 30; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 0, G: 128, B: 255, A: 255})
		}
	}
	f, _ := os.Create(logoPath)
	png.Encode(f, logo)
	f.Close()

	img, err := GenerateWithLogoFile("file logo test", ECLHigh, 250, logoPath, 50)
	require.NoError(t, err)
	assert.Equal(t, 250, img.Bounds().Dx())
}

func TestGenerateWithLogoFile_NotExist(t *testing.T) {
	_, err := GenerateWithLogoFile("test", ECLHigh, 200, "/nonexistent/logo.png", 40)
	assert.Error(t, err)
}

func TestSaveWithLogoFile(t *testing.T) {
	logoPath := filepath.Join(t.TempDir(), "logo.png")
	logo := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	f, _ := os.Create(logoPath)
	png.Encode(f, logo)
	f.Close()

	outPath := filepath.Join(t.TempDir(), "qr_with_file_logo.png")
	err := SaveWithLogoFile("save file logo", outPath, ECLHigh, 250, logoPath, 50)
	require.NoError(t, err)

	_, err = os.Stat(outPath)
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// Round-trip: generate → decode
// ──────────────────────────────────────────────

func TestRoundTrip_GenerateAndDecode(t *testing.T) {
	text := "https://ling-base.dev/hello/world"
	img, err := Generate(text, ECLHigh, 300)
	require.NoError(t, err)

	decoded, err := Decode(img)
	require.NoError(t, err)
	assert.Equal(t, text, decoded)
}

func TestRoundTrip_PNGBytes(t *testing.T) {
	text := "https://ling-base.dev/round-trip"
	data, err := GeneratePNG(text, ECLHigh, 300)
	require.NoError(t, err)

	decoded, err := DecodeBytes(data)
	require.NoError(t, err)
	assert.Equal(t, text, decoded)
}

func TestRoundTrip_File(t *testing.T) {
	text := "round trip via file"
	path := filepath.Join(t.TempDir(), "roundtrip.png")
	err := Save(text, path, ECLHigh, 300)
	require.NoError(t, err)

	decoded, err := DecodeFile(path)
	require.NoError(t, err)
	assert.Equal(t, text, decoded)
}

func TestDecode_NilImage(t *testing.T) {
	_, err := Decode(nil)
	assert.Error(t, err)
}

func TestDecode_NoQRCode(t *testing.T) {
	// A plain solid-color image has no QR code.
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	_, err := Decode(img)
	assert.Error(t, err)
}

func TestDecodeFile_NotExist(t *testing.T) {
	_, err := DecodeFile("/nonexistent/qr.png")
	assert.Error(t, err)
}

func TestDecodeBytes_InvalidData(t *testing.T) {
	_, err := DecodeBytes([]byte("not an image"))
	assert.Error(t, err)
}

func TestToBoombulerLevel(t *testing.T) {
	tests := []struct {
		level ErrorCorrectionLevel
		want  string
	}{
		{ECLLow, "L"},
		{ECLMedium, "M"},
		{ECLQuartile, "Q"},
		{ECLHigh, "H"},
		{ErrorCorrectionLevel(99), "M"}, // default
	}
	for _, tt := range tests {
		l := toBoombulerLevel(tt.level)
		assert.Equal(t, tt.want, l.String(), "level %d", tt.level)
	}
}
