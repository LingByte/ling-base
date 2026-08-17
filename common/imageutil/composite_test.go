// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestBlend_Multiply(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}
	out := Blend(base, top, BlendMultiply)
	r, _, _, _ := out.At(5, 5).RGBA()
	want := uint32(200 * 100 / 255)
	if r/256 < want-2 || r/256 > want+2 {
		t.Fatalf("BlendMultiply = %d, want ~%d", r/256, want)
	}
}

func TestBlend_TransparentTop(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 100, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 255, A: 0})
		}
	}
	out := Blend(base, top, BlendNormal)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 100 {
		t.Fatalf("Blend with transparent top changed base: got %d, want 100", r/256)
	}
}

func TestBlend_Screen(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}
	out := Blend(base, top, BlendScreen)
	r, _, _, _ := out.At(5, 5).RGBA()
	want := uint32(255 - (255-100)*(255-100)/255)
	if r/256 < want-2 || r/256 > want+2 {
		t.Fatalf("BlendScreen = %d, want ~%d", r/256, want)
	}
}

func TestBlend_Overlay(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255}) // < 128
			top.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out := Blend(base, top, BlendOverlay)
	r, _, _, _ := out.At(5, 5).RGBA()
	want := uint32(2 * 50 * 200 / 255)
	if r/256 < want-2 || r/256 > want+2 {
		t.Fatalf("BlendOverlay (base<128) = %d, want ~%d", r/256, want)
	}
}

func TestBlend_Add(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}
	out := Blend(base, top, BlendAdd)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 200 {
		t.Fatalf("BlendAdd = %d, want 200", r/256)
	}
}

func TestBlend_AddClamp(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out := Blend(base, top, BlendAdd)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 255 {
		t.Fatalf("BlendAdd clamp = %d, want 255", r/256)
	}
}

func TestBlend_Subtract(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}
	out := Blend(base, top, BlendSubtract)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 100 {
		t.Fatalf("BlendSubtract = %d, want 100", r/256)
	}
}

func TestBlend_Difference(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out := Blend(base, top, BlendDifference)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 150 {
		t.Fatalf("BlendDifference = %d, want 150", r/256)
	}
}

func TestBlend_Darken(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}
	out := Blend(base, top, BlendDarken)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 100 {
		t.Fatalf("BlendDarken = %d, want 100", r/256)
	}
}

func TestBlend_Lighten(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out := Blend(base, top, BlendLighten)
	r, _, _, _ := out.At(5, 5).RGBA()
	if r/256 != 200 {
		t.Fatalf("BlendLighten = %d, want 200", r/256)
	}
}

func TestBlend_UnknownMode(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 10, 10))
	top := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			base.SetRGBA(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
			top.SetRGBA(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	out := Blend(base, top, BlendMode(999))
	r, _, _, _ := out.At(5, 5).RGBA()
	// Unknown mode falls back to "top" (BlendNormal behavior).
	if r/256 != 200 {
		t.Fatalf("Blend unknown mode = %d, want 200 (fallback to top)", r/256)
	}
}

func TestAddBorder(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	out := AddBorder(img, 5, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	w, h := Dimensions(out)
	if w != 60 || h != 60 {
		t.Fatalf("AddBorder dimensions = %dx%d, want 60x60", w, h)
	}
	r, g, b, _ := out.At(0, 0).RGBA()
	if r/256 != 255 || g/256 != 0 || b/256 != 0 {
		t.Fatalf("AddBorder corner color = %d,%d,%d, want red", r/256, g/256, b/256)
	}
}

func TestAddBorder_Zero(t *testing.T) {
	img := createTestImage()
	if out := AddBorder(img, 0, color.Black); out != img {
		t.Fatal("AddBorder(img, 0, _) should return the original image")
	}
}

func TestAddBorder_Negative(t *testing.T) {
	img := createTestImage()
	if out := AddBorder(img, -5, color.Black); out != img {
		t.Fatal("AddBorder(img, -5, _) should return the original image")
	}
}

func TestAddPadding(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	out := AddPadding(img, 100, 100, color.Black)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("AddPadding dimensions = %dx%d, want 100x100", w, h)
	}
	r, g, b, _ := out.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("AddPadding corner = %d,%d,%d, want black", r, g, b)
	}
}

func TestAddPadding_NoChange(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	if out := AddPadding(img, 50, 50, color.Black); out != img {
		t.Fatal("AddPadding with same size should return original")
	}
}

func TestAddPadding_SmallerTarget(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	out := AddPadding(img, 50, 50, color.Black)
	// Target smaller than source -> target is clamped to source size.
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("AddPadding smaller target = %dx%d, want 100x100", w, h)
	}
}

func TestTileWatermark(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 100, 100))
	wm := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			wm.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out := TileWatermark(base, wm, 10, 10, 0.5)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("TileWatermark dimensions = %dx%d", w, h)
	}
	found := false
	for y := 0; y < 100 && !found; y++ {
		for x := 0; x < 100 && !found; x++ {
			r, _, _, _ := out.At(x, y).RGBA()
			if r > 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("TileWatermark produced an all-black result")
	}
}

func TestTileWatermark_NegativeOpacity(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 50, 50))
	wm := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			wm.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := TileWatermark(base, wm, 5, 5, -1)
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("TileWatermark negative opacity dims = %dx%d", w, h)
	}
}

func TestTileWatermark_OverOneOpacity(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 50, 50))
	wm := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			wm.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := TileWatermark(base, wm, 5, 5, 2)
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("TileWatermark over-1 opacity dims = %dx%d", w, h)
	}
}

func TestTileWatermark_ZeroSpacing(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 50, 50))
	wm := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			wm.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := TileWatermark(base, wm, 0, 0, 0.5)
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("TileWatermark zero spacing dims = %dx%d", w, h)
	}
}
