// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestHueRotate(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	out := HueRotate(img, 120)
	c := out.At(5, 5)
	r, g, b, _ := c.RGBA()
	if g <= r || g <= b {
		t.Fatalf("HueRotate(120) on red: r=%d g=%d b=%d, expected green dominant", r, g, b)
	}
}

func TestHueRotate_360NoChange(t *testing.T) {
	img := createTestImage()
	out := HueRotate(img, 360)
	o := out.At(20, 20)
	r, g, b, _ := o.RGBA()
	orig := img.At(20, 20)
	or, og, ob, _ := orig.RGBA()
	if abs(int(r/256)-int(or/256)) > 2 || abs(int(g/256)-int(og/256)) > 2 || abs(int(b/256)-int(ob/256)) > 2 {
		t.Fatalf("HueRotate(360) changed pixel too much: got %d,%d,%d want ~%d,%d,%d",
			r/256, g/256, b/256, or/256, og/256, ob/256)
	}
}

func TestHueRotate_Zero(t *testing.T) {
	img := createTestImage()
	out := HueRotate(img, 0)
	o := out.At(20, 20)
	r, g, b, _ := o.RGBA()
	or, og, ob, _ := img.At(20, 20).RGBA()
	if abs(int(r/256)-int(or/256)) > 2 || abs(int(g/256)-int(og/256)) > 2 || abs(int(b/256)-int(ob/256)) > 2 {
		t.Fatal("HueRotate(0) changed pixels")
	}
}

func TestAdjustTemperature(t *testing.T) {
	img := createTestImage()
	warm := AdjustTemperature(img, 50)
	cool := AdjustTemperature(img, -50)
	wr, _, wb, _ := warm.At(50, 50).RGBA()
	cr, _, cb, _ := cool.At(50, 50).RGBA()
	if wr <= cr {
		t.Fatalf("warm R (%d) should be > cool R (%d)", wr, cr)
	}
	if wb >= cb {
		t.Fatalf("warm B (%d) should be < cool B (%d)", wb, cb)
	}
}

func TestAdjustTemperature_Zero(t *testing.T) {
	img := createTestImage()
	if out := AdjustTemperature(img, 0); out != img {
		t.Fatal("AdjustTemperature(img, 0) should return the original image")
	}
}

func TestAdjustTemperature_Clamp(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 250, G: 250, B: 250, A: 255})
		}
	}
	out := AdjustTemperature(img, 100)
	r, _, _, _ := out.At(2, 2).RGBA()
	if r/256 > 255 {
		t.Fatal("AdjustTemperature should clamp to 255")
	}
}

func TestTint(t *testing.T) {
	img := createTestImage()
	out := Tint(img, color.RGBA{R: 255, G: 0, B: 0, A: 255}, 0.5)
	r, g, b, _ := out.At(50, 50).RGBA()
	if r <= g || r <= b {
		t.Fatalf("Tint red 0.5: r=%d g=%d b=%d, expected red dominant", r, g, b)
	}
}

func TestTint_Zero(t *testing.T) {
	img := createTestImage()
	if out := Tint(img, color.RGBA{R: 255, A: 255}, 0); out != img {
		t.Fatal("Tint(..., 0) should return the original image")
	}
}

func TestTint_FullAmount(t *testing.T) {
	img := createTestImage()
	out := Tint(img, color.RGBA{R: 0, G: 255, B: 0, A: 255}, 1.0)
	_, g, b, _ := out.At(50, 50).RGBA()
	if g/256 <= b/256 {
		t.Fatal("Tint green 1.0 should make green dominant")
	}
}

func TestPosterize(t *testing.T) {
	img := createTestImage()
	out := Posterize(img, 4)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("Posterize dimensions = %dx%d", w, h)
	}
	seen := map[uint8]bool{}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, _, _, _ := out.At(x, y).RGBA()
			seen[uint8(r/256)] = true
		}
	}
	if len(seen) > 4 {
		t.Fatalf("Posterize(4) produced %d distinct R values, want <= 4", len(seen))
	}
}

func TestPosterize_OneLevel(t *testing.T) {
	img := createTestImage()
	out := Posterize(img, 1)
	// levels < 2 is clamped to 2, so we expect at most 2 distinct values.
	seen := map[uint8]bool{}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, _, _, _ := out.At(x, y).RGBA()
			seen[uint8(r/256)] = true
		}
	}
	if len(seen) > 2 {
		t.Fatalf("Posterize(1) clamped to 2 produced %d distinct R values, want <= 2", len(seen))
	}
}

func TestThreshold(t *testing.T) {
	img := createTestImage()
	out := Threshold(img, 128)
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			v := uint8(r / 256)
			if v != 0 && v != 255 {
				t.Fatalf("Threshold produced non-binary value %d at (%d,%d)", v, x, y)
			}
			if uint8(g/256) != v || uint8(b/256) != v {
				t.Fatalf("Threshold channels differ at (%d,%d)", x, y)
			}
		}
	}
}

func TestThreshold_AllBlack(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	out := Threshold(img, 128)
	r, _, _, _ := out.At(2, 2).RGBA()
	if r/256 != 0 {
		t.Fatal("Threshold on all-black image should produce all-black output")
	}
}

func TestThreshold_AllWhite(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out := Threshold(img, 128)
	r, _, _, _ := out.At(2, 2).RGBA()
	if r/256 != 255 {
		t.Fatal("Threshold on all-white image should produce all-white output")
	}
}

func TestRGBToHSV_KnownValues(t *testing.T) {
	// h is in sector units [0,6), so green=2, blue=4 (not 120/240 degrees).
	tests := []struct {
		r, g, b     uint8
		h, s, v     float64
	}{
		{255, 0, 0, 0, 1, 1},       // red
		{0, 255, 0, 2, 1, 1},       // green (sector 2)
		{0, 0, 255, 4, 1, 1},       // blue (sector 4)
		{0, 0, 0, 0, 0, 0},         // black
		{255, 255, 255, 0, 0, 1},   // white
	}
	for _, tt := range tests {
		h, s, v := rgbToHSV(tt.r, tt.g, tt.b)
		if abs(int(h-tt.h)) > 1 || s-tt.s > 0.01 || s-tt.s < -0.01 || v-tt.v > 0.01 || v-tt.v < -0.01 {
			t.Fatalf("rgbToHSV(%d,%d,%d) = (%.1f,%.2f,%.2f), want (~%.0f,~%.2f,~%.2f)",
				tt.r, tt.g, tt.b, h, s, v, tt.h, tt.s, tt.v)
		}
	}
}

func TestHSVToRGB_KnownValues(t *testing.T) {
	// Red (sector 0)
	r, g, b := hsvToRGB(0, 1, 1)
	if r != 255 || g != 0 || b != 0 {
		t.Fatalf("hsvToRGB(0,1,1) = %d,%d,%d, want 255,0,0", r, g, b)
	}
	// Green (sector 2)
	r, g, b = hsvToRGB(2, 1, 1)
	if r != 0 || g != 255 || b != 0 {
		t.Fatalf("hsvToRGB(2,1,1) = %d,%d,%d, want 0,255,0", r, g, b)
	}
	// Blue (sector 4)
	r, g, b = hsvToRGB(4, 1, 1)
	if r != 0 || g != 0 || b != 255 {
		t.Fatalf("hsvToRGB(4,1,1) = %d,%d,%d, want 0,0,255", r, g, b)
	}
	// Black
	r, g, b = hsvToRGB(0, 0, 0)
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("hsvToRGB(0,0,0) = %d,%d,%d, want 0,0,0", r, g, b)
	}
}
