// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestResizeWithPadding(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 128, B: 64, A: 255})
		}
	}
	out := ResizeWithPadding(img, 100, 100, color.Black)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("ResizeWithPadding dimensions = %dx%d, want 100x100", w, h)
	}
	r, g, b, _ := out.At(50, 0).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("ResizeWithPadding top padding = %d,%d,%d, want black", r, g, b)
	}
	r, g, b, _ = out.At(50, 50).RGBA()
	if r == 0 && g == 0 && b == 0 {
		t.Fatal("ResizeWithPadding center is black (image not drawn?)")
	}
}

func TestResizeWithPadding_ZeroTarget(t *testing.T) {
	img := createTestImage()
	if out := ResizeWithPadding(img, 0, 100, color.Black); out != img {
		t.Fatal("ResizeWithPadding(0, 100) should return original")
	}
	if out := ResizeWithPadding(img, 100, 0, color.Black); out != img {
		t.Fatal("ResizeWithPadding(100, 0) should return original")
	}
}

func TestResizeWithPadding_SameSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	out := ResizeWithPadding(img, 50, 50, color.Black)
	// Source is same size as target, but still goes through the canvas path.
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("ResizeWithPadding same size = %dx%d", w, h)
	}
}

func TestResizeWithPadding_SquareSource(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := ResizeWithPadding(img, 200, 100, color.Black)
	if w, h := Dimensions(out); w != 200 || h != 100 {
		t.Fatalf("ResizeWithPadding square source = %dx%d", w, h)
	}
}

func TestThumbnailWithPadding_NoUpscale(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := ThumbnailWithPadding(img, 100, 100, color.Black)
	r, _, _, _ := out.At(50, 50).RGBA()
	if r/256 != 255 {
		t.Fatalf("ThumbnailWithPadding center R = %d, want 255", r/256)
	}
	r, _, _, _ = out.At(0, 0).RGBA()
	if r != 0 {
		t.Fatalf("ThumbnailWithPadding corner R = %d, want 0 (padding)", r)
	}
}

func TestThumbnailWithPadding_Downscale(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := ThumbnailWithPadding(img, 50, 50, color.Black)
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("ThumbnailWithPadding downscale = %dx%d", w, h)
	}
	// Center should have image content (non-black).
	r, _, _, _ := out.At(25, 25).RGBA()
	if r == 0 {
		t.Fatal("ThumbnailWithPadding downscale center is black")
	}
}

func TestThumbnailWithPadding_ZeroTarget(t *testing.T) {
	img := createTestImage()
	if out := ThumbnailWithPadding(img, 0, 100, color.Black); out != img {
		t.Fatal("ThumbnailWithPadding(0, 100) should return original")
	}
}

func TestMinFloat(t *testing.T) {
	if minFloat(1.0, 2.0) != 1.0 {
		t.Fatal("minFloat(1, 2) should be 1")
	}
	if minFloat(3.0, 2.0) != 2.0 {
		t.Fatal("minFloat(3, 2) should be 2")
	}
	if minFloat(1.5, 1.5) != 1.5 {
		t.Fatal("minFloat(1.5, 1.5) should be 1.5")
	}
}
