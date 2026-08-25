// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────
// ICO tests
// ──────────────────────────────────────────────

func TestToICO_DefaultSizes(t *testing.T) {
	img := createTestImage() // 100x100
	data, err := ToICO(img, nil)
	if err != nil {
		t.Fatalf("ToICO failed: %v", err)
	}
	if len(data) < 6 {
		t.Fatalf("ICO data too short: %d bytes", len(data))
	}
	// ICONDIR: reserved=0, type=1, count
	reserved := binary.LittleEndian.Uint16(data[0:2])
	icoType := binary.LittleEndian.Uint16(data[2:4])
	count := binary.LittleEndian.Uint16(data[4:6])
	if reserved != 0 {
		t.Errorf("reserved = %d, want 0", reserved)
	}
	if icoType != 1 {
		t.Errorf("type = %d, want 1", icoType)
	}
	if int(count) != len(DefaultICOSizes) {
		t.Errorf("count = %d, want %d", count, len(DefaultICOSizes))
	}
}

func TestToICO_CustomSizes(t *testing.T) {
	img := createTestImage()
	sizes := []int{64, 32, 16}
	data, err := ToICO(img, sizes)
	if err != nil {
		t.Fatalf("ToICO failed: %v", err)
	}
	count := binary.LittleEndian.Uint16(data[4:6])
	if int(count) != len(sizes) {
		t.Errorf("count = %d, want %d", count, len(sizes))
	}
}

func TestToICO_InvalidSize(t *testing.T) {
	img := createTestImage()
	_, err := ToICO(img, []int{0})
	if err == nil {
		t.Error("expected error for size 0, got nil")
	}
}

func TestToICO_NonSquareSource(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	data, err := ToICO(img, []int{48})
	if err != nil {
		t.Fatalf("ToICO non-square failed: %v", err)
	}
	if len(data) < 22 {
		t.Fatalf("ICO data too short: %d", len(data))
	}
}

func TestSaveICO(t *testing.T) {
	img := createTestImage()
	path := filepath.Join(t.TempDir(), "test.ico")
	if err := SaveICO(img, path, []int{32, 16}); err != nil {
		t.Fatalf("SaveICO failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() < 100 {
		t.Errorf("ICO file too small: %d bytes", info.Size())
	}
}

// ──────────────────────────────────────────────
// WebP encode tests
// ──────────────────────────────────────────────

func TestSaveWebP_Lossy(t *testing.T) {
	img := createTestImage()
	path := filepath.Join(t.TempDir(), "test.webp")
	if err := SaveWebP(img, path, 80); err != nil {
		t.Fatalf("SaveWebP lossy failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	// WebP files start with RIFF....WEBP
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Errorf("invalid WebP header: %q", data[:minInt(12, len(data))])
	}
}

func TestSaveWebP_Lossless(t *testing.T) {
	img := createTestImage()
	path := filepath.Join(t.TempDir(), "test_lossless.webp")
	if err := SaveWebP(img, path, -1); err != nil {
		t.Fatalf("SaveWebP lossless failed: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Error("lossless WebP file is empty")
	}
}

func TestEncodeWebP_RoundTrip(t *testing.T) {
	img := createTestImage()
	var buf bytes.Buffer
	if err := Encode(&buf, img, FormatWebP, 75); err != nil {
		t.Fatalf("Encode WebP failed: %v", err)
	}
	decoded, format, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode WebP failed: %v", err)
	}
	if format != FormatWebP {
		t.Errorf("format = %s, want webp", format)
	}
	w, h := Dimensions(decoded)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
}

// ──────────────────────────────────────────────
// Animated GIF tests
// ──────────────────────────────────────────────

func createAnimatedGIF(t *testing.T, frames, w, h int) *gif.GIF {
	t.Helper()
	g := &gif.GIF{
		LoopCount: 0,
		Delay:     make([]int, frames),
		Disposal:  make([]byte, frames),
		Image:     make([]*image.Paletted, frames),
	}
	pal := color.Palette{
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 0, G: 255, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}
	for i := 0; i < frames; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, w, h), pal)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				frame.SetColorIndex(x, y, uint8((x+y+i)%len(pal)))
			}
		}
		g.Image[i] = frame
		g.Delay[i] = 10
		g.Disposal[i] = gif.DisposalBackground
	}
	return g
}

func TestEncodeAnimatedGIF_RoundTrip(t *testing.T) {
	g := createAnimatedGIF(t, 3, 50, 50)
	var buf bytes.Buffer
	if err := EncodeAnimatedGIF(&buf, g); err != nil {
		t.Fatalf("EncodeAnimatedGIF failed: %v", err)
	}
	decoded, err := DecodeAnimatedGIF(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeAnimatedGIF failed: %v", err)
	}
	if len(decoded.Image) != 3 {
		t.Errorf("frame count = %d, want 3", len(decoded.Image))
	}
}

func TestResizeAnimatedGIF(t *testing.T) {
	g := createAnimatedGIF(t, 3, 50, 50)
	out := ResizeAnimatedGIF(g, 25, 25)
	if len(out.Image) != 3 {
		t.Errorf("frame count = %d, want 3", len(out.Image))
	}
	w, h := Dimensions(out.Image[0])
	if w != 25 || h != 25 {
		t.Errorf("frame[0] = %dx%d, want 25x25", w, h)
	}
	// Delays should be preserved.
	for i := range out.Delay {
		if out.Delay[i] != g.Delay[i] {
			t.Errorf("delay[%d] = %d, want %d", i, out.Delay[i], g.Delay[i])
		}
	}
}

func TestOptimizeAnimatedGIF(t *testing.T) {
	g := createAnimatedGIF(t, 2, 100, 100)
	data, err := OptimizeAnimatedGIF(g, 50)
	if err != nil {
		t.Fatalf("OptimizeAnimatedGIF failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("OptimizeAnimatedGIF returned empty data")
	}
	// Verify the result is a valid GIF.
	decoded, err := DecodeAnimatedGIF(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decoded optimized gif failed: %v", err)
	}
	if len(decoded.Image) != 2 {
		t.Errorf("frame count = %d, want 2", len(decoded.Image))
	}
}

func TestResizeAnimatedGIFFile(t *testing.T) {
	g := createAnimatedGIF(t, 2, 40, 40)
	inPath := filepath.Join(t.TempDir(), "in.gif")
	f, err := os.Create(inPath)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := EncodeAnimatedGIF(f, g); err != nil {
		f.Close()
		t.Fatalf("encode failed: %v", err)
	}
	f.Close()

	outPath := filepath.Join(t.TempDir(), "out.gif")
	if err := ResizeAnimatedGIFFile(inPath, outPath, 20, 20); err != nil {
		t.Fatalf("ResizeAnimatedGIFFile failed: %v", err)
	}
	decoded, err := DecodeAnimatedGIF(mustOpen(t, outPath))
	if err != nil {
		t.Fatalf("decode out failed: %v", err)
	}
	w, h := Dimensions(decoded.Image[0])
	if w != 20 || h != 20 {
		t.Errorf("frame[0] = %dx%d, want 20x20", w, h)
	}
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	return f
}

// ──────────────────────────────────────────────
// Filter tests (Vintage / Cool / Warm)
// ──────────────────────────────────────────────

func TestVintage(t *testing.T) {
	img := createTestImage()
	out := Vintage(img, 0.8)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
	// Center pixel should differ from original (strength > 0).
	orig := img.At(50, 50)
	result := out.At(50, 50)
	origR, origG, origB, _ := orig.RGBA()
	resR, resG, resB, _ := result.RGBA()
	if origR == resR && origG == resG && origB == resB {
		t.Error("Vintage(0.8) produced no change at center pixel")
	}
}

func TestVintage_ZeroStrength(t *testing.T) {
	img := createTestImage()
	out := Vintage(img, 0)
	if out != img {
		t.Error("Vintage(0) should return original image")
	}
}

func TestCool(t *testing.T) {
	img := createTestImage()
	out := Cool(img, 0.7)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestWarm(t *testing.T) {
	img := createTestImage()
	out := Warm(img, 0.7)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestFilters_ClampStrength(t *testing.T) {
	img := createTestImage()
	// strength > 1 should be clamped, not panic.
	_ = Vintage(img, 2)
	_ = Cool(img, 2)
	_ = Warm(img, 2)
}

// ──────────────────────────────────────────────
// Effect tests (Vignette / AddNoise / Pixelate)
// ──────────────────────────────────────────────

func TestVignette(t *testing.T) {
	img := createTestImage()
	out := Vignette(img, 0.8, 0.5)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
	// Corner pixel should be darker than center pixel.
	corner := out.At(0, 0)
	center := out.At(50, 50)
	cR, cG, cB, _ := corner.RGBA()
	eR, eG, eB, _ := center.RGBA()
	if cR+eR == 0 && cG+eG == 0 && cB+eB == 0 {
		// Skip if both are zero (unlikely with gradient test image).
		return
	}
	if cR > eR {
		t.Errorf("corner R = %d, center R = %d; corner should be darker", cR, eR)
	}
}

func TestVignette_ZeroStrength(t *testing.T) {
	img := createTestImage()
	out := Vignette(img, 0, 0.5)
	if out != img {
		t.Error("Vignette(0, ...) should return original image")
	}
}

func TestAddNoise_Monochrome(t *testing.T) {
	img := createTestImage()
	rng := rand.New(rand.NewSource(42))
	out := AddNoise(img, 30, true, rng)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
	// At least some pixels should differ.
	diffs := 0
	for y := 0; y < 100; y += 10 {
		for x := 0; x < 100; x += 10 {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			r2, g2, b2, _ := out.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				diffs++
			}
		}
	}
	if diffs == 0 {
		t.Error("AddNoise produced no changes")
	}
}

func TestAddNoise_Color(t *testing.T) {
	img := createTestImage()
	rng := rand.New(rand.NewSource(42))
	out := AddNoise(img, 30, false, rng)
	if out == nil {
		t.Fatal("AddNoise color returned nil")
	}
}

func TestAddNoise_ZeroAmount(t *testing.T) {
	img := createTestImage()
	out := AddNoise(img, 0, true, nil)
	if out != img {
		t.Error("AddNoise(0, ...) should return original image")
	}
}

func TestPixelate(t *testing.T) {
	img := createTestImage()
	out := Pixelate(img, 10)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
	// Within a block, all pixels should be the same color.
	c1 := out.At(0, 0)
	c2 := out.At(5, 5)
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 {
		t.Errorf("pixels in same block differ: (0,0)=%d,%d,%d (5,5)=%d,%d,%d",
			r1, g1, b1, r2, g2, b2)
	}
}

func TestPixelate_BlockSizeOne(t *testing.T) {
	img := createTestImage()
	out := Pixelate(img, 1)
	if out != img {
		t.Error("Pixelate(1) should return original image")
	}
}

func TestPixelate_LargeBlock(t *testing.T) {
	img := createTestImage()
	out := Pixelate(img, 200) // larger than image
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Errorf("dimensions = %dx%d, want 100x100", w, h)
	}
}

// ──────────────────────────────────────────────
// ParseHexColor tests
// ──────────────────────────────────────────────

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		want    color.RGBA
		wantErr bool
	}{
		{"#f80", color.RGBA{R: 255, G: 136, B: 0, A: 255}, false},
		{"f80", color.RGBA{R: 255, G: 136, B: 0, A: 255}, false},
		{"#F80", color.RGBA{R: 255, G: 136, B: 0, A: 255}, false},
		{"#f80a", color.RGBA{R: 255, G: 136, B: 0, A: 170}, false},
		{"#ff8800", color.RGBA{R: 255, G: 136, B: 0, A: 255}, false},
		{"#ff8800aa", color.RGBA{R: 255, G: 136, B: 0, A: 170}, false},
		{"#000000", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"#ffffff", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"", color.RGBA{}, true},
		{"#", color.RGBA{}, true},
		{"#ff", color.RGBA{}, true},
		{"#fffff", color.RGBA{}, true},
		{"#fffffff", color.RGBA{}, true},
		{"#gggggg", color.RGBA{}, true},
	}
	for _, tt := range tests {
		got, err := ParseHexColor(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseHexColor(%q) expected error, got %v", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseHexColor(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseHexColor(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFormatHexColor(t *testing.T) {
	tests := []struct {
		input color.Color
		want  string
	}{
		{color.RGBA{R: 255, G: 136, B: 0, A: 255}, "#ff8800"},
		{color.RGBA{R: 255, G: 136, B: 0, A: 170}, "#ff8800aa"},
		{color.RGBA{R: 0, G: 0, B: 0, A: 255}, "#000000"},
		{color.RGBA{R: 255, G: 255, B: 255, A: 255}, "#ffffff"},
	}
	for _, tt := range tests {
		got := FormatHexColor(tt.input)
		if got != tt.want {
			t.Errorf("FormatHexColor(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseHexColor_RoundTrip(t *testing.T) {
	original := color.RGBA{R: 123, G: 45, B: 67, A: 255}
	hex := FormatHexColor(original)
	parsed, err := ParseHexColor(hex)
	if err != nil {
		t.Fatalf("ParseHexColor(%q) failed: %v", hex, err)
	}
	if parsed != original {
		t.Errorf("round-trip mismatch: got %v, want %v", parsed, original)
	}
}

func TestMustParseHexColor(t *testing.T) {
	c := MustParseHexColor("#ff8800")
	if c.R != 255 || c.G != 136 || c.B != 0 || c.A != 255 {
		t.Errorf("MustParseHexColor = %v, want {255 136 0 255}", c)
	}
}

func TestMustParseHexColor_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseHexColor did not panic on invalid input")
		}
	}()
	MustParseHexColor("#invalid")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
