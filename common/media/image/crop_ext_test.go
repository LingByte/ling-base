// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestCropAspectRatio(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	out := CropAspectRatio(img, 1, 1, AnchorMiddleCenter)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("CropAspectRatio 1:1 = %dx%d, want 100x100", w, h)
	}
}

func TestCropAspectRatio_NoChange(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	if out := CropAspectRatio(img, 1, 1, AnchorMiddleCenter); out != img {
		t.Fatal("CropAspectRatio on already-matching ratio should return original")
	}
}

func TestCropAspectRatio_Anchors(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: 0, B: 0, A: 255})
		}
	}
	left := CropAspectRatio(img, 1, 1, AnchorTopLeft)
	right := CropAspectRatio(img, 1, 1, AnchorTopRight)
	lr, _, _, _ := left.At(0, 0).RGBA()
	rr, _, _, _ := right.At(0, 0).RGBA()
	if lr/256 != 0 {
		t.Fatalf("TopLeft anchor x=0 should be 0, got %d", lr/256)
	}
	if rr/256 != 100 {
		t.Fatalf("TopRight anchor x=0 should be 100, got %d", rr/256)
	}
}

func TestCropAspectRatio_TallSource(t *testing.T) {
	// 100x200 source, crop to 1:1 -> trim height.
	img := image.NewRGBA(image.Rect(0, 0, 100, 200))
	out := CropAspectRatio(img, 1, 1, AnchorMiddleCenter)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("CropAspectRatio tall source = %dx%d, want 100x100", w, h)
	}
}

func TestCropAspectRatio_InvalidRatio(t *testing.T) {
	img := createTestImage()
	if out := CropAspectRatio(img, 0, 1, AnchorMiddleCenter); out != img {
		t.Fatal("CropAspectRatio with ratioW=0 should return original")
	}
	if out := CropAspectRatio(img, 1, 0, AnchorMiddleCenter); out != img {
		t.Fatal("CropAspectRatio with ratioH=0 should return original")
	}
}

func TestCropAspectRatioResize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	out := CropAspectRatioResize(img, 16, 9, 160, 90, AnchorMiddleCenter)
	w, h := Dimensions(out)
	if w != 160 || h != 90 {
		t.Fatalf("CropAspectRatioResize = %dx%d, want 160x90", w, h)
	}
}

func TestCropAspectRatioResize_NoCropNeeded(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	out := CropAspectRatioResize(img, 1, 1, 50, 50, AnchorMiddleCenter)
	w, h := Dimensions(out)
	if w != 50 || h != 50 {
		t.Fatalf("CropAspectRatioResize no-crop = %dx%d, want 50x50", w, h)
	}
}

func TestAnchorOffset_AllAnchors(t *testing.T) {
	w, h, cw, ch := 200, 100, 100, 50
	tests := []struct {
		anchor Anchor
		wantX  int
		wantY  int
	}{
		{AnchorTopLeft, 0, 0},
		{AnchorTopCenter, (w - cw) / 2, 0},
		{AnchorTopRight, w - cw, 0},
		{AnchorMiddleLeft, 0, (h - ch) / 2},
		{AnchorMiddleCenter, (w - cw) / 2, (h - ch) / 2},
		{AnchorBottomLeft, 0, h - ch},
		{AnchorBottomCenter, (w - cw) / 2, h - ch},
		{AnchorBottomRight, w - cw, h - ch},
	}
	for _, tt := range tests {
		x, y := anchorOffset(tt.anchor, w, h, cw, ch)
		if x != tt.wantX || y != tt.wantY {
			t.Fatalf("anchorOffset(%v) = (%d,%d), want (%d,%d)", tt.anchor, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestAnchorOffset_UnknownAnchor(t *testing.T) {
	x, y := anchorOffset(Anchor(999), 200, 100, 50, 50)
	// Unknown anchor defaults to center.
	if x != 75 || y != 25 {
		t.Fatalf("anchorOffset(unknown) = (%d,%d), want (75,25)", x, y)
	}
}

func TestCropCircle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := CropCircle(img)
	_, _, _, a := out.At(0, 0).RGBA()
	if a != 0 {
		t.Fatalf("CropCircle corner alpha = %d, want 0", a)
	}
	_, _, _, a = out.At(50, 50).RGBA()
	if a == 0 {
		t.Fatal("CropCircle center should be opaque")
	}
}

func TestCropCircle_NonSquare(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	out := CropCircle(img)
	if w, h := Dimensions(out); w != 200 || h != 100 {
		t.Fatalf("CropCircle non-square dims = %dx%d", w, h)
	}
}

func TestRoundCorners(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	out := RoundCorners(img, 20)
	_, _, _, a := out.At(0, 0).RGBA()
	if a != 0 {
		t.Fatalf("RoundCorners corner alpha = %d, want 0", a)
	}
	_, _, _, a = out.At(50, 50).RGBA()
	if a == 0 {
		t.Fatal("RoundCorners center should be opaque")
	}
}

func TestRoundCorners_ZeroRadius(t *testing.T) {
	img := createTestImage()
	if out := RoundCorners(img, 0); out != img {
		t.Fatal("RoundCorners(img, 0) should return the original image")
	}
}

func TestRoundCorners_NegativeRadius(t *testing.T) {
	img := createTestImage()
	if out := RoundCorners(img, -5); out != img {
		t.Fatal("RoundCorners(img, -5) should return the original image")
	}
}

func TestRoundCorners_RadiusTooLarge(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	// Radius larger than half the image should be clamped.
	out := RoundCorners(img, 100)
	if w, h := Dimensions(out); w != 50 || h != 50 {
		t.Fatalf("RoundCorners too-large radius dims = %dx%d", w, h)
	}
}
