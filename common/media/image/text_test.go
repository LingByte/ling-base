// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// Ensure image import is used.
var _ = image.NewRGBA

func TestTextWatermark(t *testing.T) {
	img := createTestImage()
	out := TextWatermark(img, "Hi", 10, 10, TextWatermarkOptions{
		FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("TextWatermark dims = %dx%d", w, h)
	}
}

func TestTextWatermark_BottomRight(t *testing.T) {
	img := createTestImage()
	out := TextWatermarkBottomRight(img, "© LingByte", TextWatermarkOptions{
		FontSize: 12,
		Color:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity:  0.8,
		Padding:  5,
	})
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("TextWatermarkBottomRight dimensions = %dx%d, want 100x100", w, h)
	}
	found := false
	for y := 80; y < 100 && !found; y++ {
		for x := 60; x < 100 && !found; x++ {
			r, g, _, _ := out.At(x, y).RGBA()
			if r/256 > 200 && g/256 > 200 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("TextWatermarkBottomRight did not render visible text in bottom-right")
	}
}

func TestTextWatermark_Center(t *testing.T) {
	img := createTestImage()
	out := TextWatermarkCenter(img, "HELLO", TextWatermarkOptions{
		FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("TextWatermarkCenter dimensions = %dx%d", w, h)
	}
	found := false
	for y := 40; y < 60 && !found; y++ {
		for x := 30; x < 70 && !found; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			if r/256 > 200 && g/256 > 200 && b/256 > 200 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("TextWatermarkCenter did not render visible text in center")
	}
}

func TestTextWatermark_TiledRotated(t *testing.T) {
	img := createTestImage()
	out := TextWatermarkTiled(img, "DEMO", TextWatermarkOptions{
		FontSize: 14, Opacity: 0.5, Angle: -30, Padding: 30,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("TextWatermarkTiled dimensions = %dx%d, want 100x100", w, h)
	}
	found := false
	for y := 0; y < 100 && !found; y++ {
		for x := 0; x < 100 && !found; x++ {
			r, _, _, _ := out.At(x, y).RGBA()
			orig, _, _, _ := img.At(x, y).RGBA()
			if r > orig+50*256 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("TextWatermarkTiled did not render any visible text")
	}
}

func TestTextWatermark_Defaults(t *testing.T) {
	img := createTestImage()
	out := TextWatermarkCenter(img, "TEST", TextWatermarkOptions{})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("default TextWatermarkCenter dimensions = %dx%d", w, h)
	}
}

func TestTextWatermark_FontSelection(t *testing.T) {
	img := createTestImage()
	for _, name := range []string{FontGoRegular, FontGoBold, FontGoMedium, FontGoItalic, FontGoMono} {
		out := TextWatermarkCenter(img, "Ag", TextWatermarkOptions{
			Font: name, FontSize: 20, Opacity: 1.0,
		})
		if w, h := Dimensions(out); w != 100 || h != 100 {
			t.Fatalf("TextWatermarkCenter font=%s dims = %dx%d", name, w, h)
		}
	}
}

func TestTextWatermark_UnknownFontFallsBack(t *testing.T) {
	img := createTestImage()
	out := TextWatermarkCenter(img, "X", TextWatermarkOptions{
		Font: "nonexistent-font", FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("fallback TextWatermarkCenter dims = %dx%d", w, h)
	}
}

func TestTextWatermark_WithAngle(t *testing.T) {
	img := createTestImage()
	out := TextWatermark(img, "ROTATED", 10, 50, TextWatermarkOptions{
		FontSize: 16, Angle: 45, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("TextWatermark with angle dims = %dx%d", w, h)
	}
}

func TestTextWatermark_OpacityClamp(t *testing.T) {
	img := createTestImage()
	// Opacity > 1 should be clamped to 1.
	out := TextWatermarkCenter(img, "X", TextWatermarkOptions{
		FontSize: 16, Opacity: 5.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("opacity-clamped dims = %dx%d", w, h)
	}
	// Opacity < 0 should be clamped to 0.
	out2 := TextWatermarkCenter(img, "X", TextWatermarkOptions{
		FontSize: 16, Opacity: -1.0,
	})
	if w, h := Dimensions(out2); w != 100 || h != 100 {
		t.Fatalf("negative opacity dims = %dx%d", w, h)
	}
}

func TestLoadFontBytes(t *testing.T) {
	if err := LoadFontBytes("test-custom", goregular.TTF); err != nil {
		t.Fatalf("LoadFontBytes failed: %v", err)
	}
	img := createTestImage()
	out := TextWatermarkCenter(img, "Q", TextWatermarkOptions{
		Font: "test-custom", FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("custom font TextWatermarkCenter dims = %dx%d", w, h)
	}
}

func TestLoadFontBytes_InvalidData(t *testing.T) {
	err := LoadFontBytes("bad-font", []byte("not a font"))
	if err == nil {
		t.Fatal("LoadFontBytes with invalid data should fail")
	}
}

func TestLoadFont(t *testing.T) {
	// Write a valid font to a temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, gobold.TTF, 0644); err != nil {
		t.Fatalf("write temp font: %v", err)
	}
	if err := LoadFont("test-from-file", path); err != nil {
		t.Fatalf("LoadFont failed: %v", err)
	}
	img := createTestImage()
	out := TextWatermarkCenter(img, "F", TextWatermarkOptions{
		Font: "test-from-file", FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("file-loaded font dims = %dx%d", w, h)
	}
}

func TestLoadFont_NotExist(t *testing.T) {
	err := LoadFont("missing", "/nonexistent/font.ttf")
	if err == nil {
		t.Fatal("LoadFont with nonexistent file should fail")
	}
}

func TestLoadFont_InvalidData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ttf")
	if err := os.WriteFile(path, []byte("not a font"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := LoadFont("bad", path)
	if err == nil {
		t.Fatal("LoadFont with invalid font data should fail")
	}
}

func TestLoadFontTTC_NotTTC(t *testing.T) {
	// A regular TTF passed to LoadFontTTC should work (extractTTCFont returns as-is).
	dir := t.TempDir()
	path := filepath.Join(dir, "regular.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := LoadFontTTC("ttc-regular", path, 0); err != nil {
		t.Fatalf("LoadFontTTC on regular TTF failed: %v", err)
	}
}

func TestLoadFontTTC_NotExist(t *testing.T) {
	err := LoadFontTTC("missing", "/nonexistent/font.ttc", 0)
	if err == nil {
		t.Fatal("LoadFontTTC with nonexistent file should fail")
	}
}

func TestExtractTTCFont_NotTTC(t *testing.T) {
	data := []byte("not a ttc file at all")
	out, err := extractTTCFont(data, 0)
	if err != nil {
		t.Fatalf("extractTTCFont on non-TTC should not error: %v", err)
	}
	if string(out) != "not a ttc file at all" {
		t.Fatal("extractTTCFont on non-TTC should return data as-is")
	}
}

func TestExtractTTCFont_TooShort(t *testing.T) {
	data := []byte("tt") // too short to be a TTC
	out, err := extractTTCFont(data, 0)
	if err != nil {
		t.Fatalf("extractTTCFont too short should not error: %v", err)
	}
	if string(out) != "tt" {
		t.Fatal("extractTTCFont too short should return data as-is")
	}
}

func TestExtractTTCFont_IndexOutOfRange(t *testing.T) {
	// Build a minimal fake TTC header with 1 font.
	data := make([]byte, 20)
	copy(data[0:4], []byte("ttcf"))
	// numFonts = 1
	data[8] = 0
	data[9] = 0
	data[10] = 0
	data[11] = 1
	// offset to font 0 = 16
	data[12] = 0
	data[13] = 0
	data[14] = 0
	data[16] = 16

	_, err := extractTTCFont(data, 5) // index 5 out of range
	if err == nil {
		t.Fatal("extractTTCFont with out-of-range index should fail")
	}
}

func TestRegisterFont(t *testing.T) {
	// LoadFontBytes parses and registers; we can verify RegisterFont works
	// by using LoadFontBytes then checking the font is usable.
	if err := LoadFontBytes("manual-register", goregular.TTF); err != nil {
		t.Fatalf("LoadFontBytes: %v", err)
	}
	f := lookupFont("manual-register")
	if f == nil {
		t.Fatal("lookupFont returned nil for registered font")
	}
	RegisterFont("manual-register-2", f)
	// Verify it's usable.
	img := createTestImage()
	out := TextWatermarkCenter(img, "R", TextWatermarkOptions{
		Font: "manual-register-2", FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("manually registered font dims = %dx%d", w, h)
	}
}

func TestRotateImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := rotateImage(img, 0.5) // ~28.6 degrees
	if w, h := Dimensions(out); w == 0 || h == 0 {
		t.Fatalf("rotateImage produced zero-size image: %dx%d", w, h)
	}
}

func TestRotateImage_ZeroAngle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 20))
	out := rotateImage(img, 0)
	if w, h := Dimensions(out); w != 50 || h != 20 {
		t.Fatalf("rotateImage(0) = %dx%d, want 50x20", w, h)
	}
}
