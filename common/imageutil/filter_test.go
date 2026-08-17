// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestBoxBlur(t *testing.T) {
	img := createTestImage()
	out := BoxBlur(img, 2)
	w, h := Dimensions(out)
	if w != 100 || h != 100 {
		t.Fatalf("BoxBlur dimensions = %dx%d, want 100x100", w, h)
	}
	c := out.At(50, 50)
	r, g, b, _ := c.RGBA()
	if r == 100 && g == 100 && b == 128 {
		t.Fatal("BoxBlur did not change center pixel")
	}
}

func TestBoxBlur_ZeroRadius(t *testing.T) {
	img := createTestImage()
	out := BoxBlur(img, 0)
	if out != img {
		t.Fatal("BoxBlur(img, 0) should return the original image")
	}
}

func TestBoxBlur_NegativeRadius(t *testing.T) {
	img := createTestImage()
	out := BoxBlur(img, -5)
	if out != img {
		t.Fatal("BoxBlur(img, -5) should return the original image")
	}
}

func TestGaussianBlur(t *testing.T) {
	img := createTestImage()
	out := GaussianBlur(img, 3, 1.5)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("GaussianBlur dimensions = %dx%d", w, h)
	}
}

func TestGaussianBlur_DefaultSigma(t *testing.T) {
	img := createTestImage()
	out := GaussianBlur(img, 2, 0)
	if out == nil {
		t.Fatal("GaussianBlur with default sigma returned nil")
	}
}

func TestGaussianBlur_ZeroRadiusZeroSigma(t *testing.T) {
	img := createTestImage()
	out := GaussianBlur(img, 0, 0)
	if out != img {
		t.Fatal("GaussianBlur(0, 0) should return the original image")
	}
}

func TestGaussianBlur_NegativeSigma(t *testing.T) {
	img := createTestImage()
	out := GaussianBlur(img, 4, -1)
	if out == nil {
		t.Fatal("GaussianBlur with negative sigma returned nil")
	}
}

func TestSharpen(t *testing.T) {
	img := createTestImage()
	out := Sharpen(img, 1.0)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("Sharpen dimensions = %dx%d", w, h)
	}
}

func TestSharpen_ZeroAmount(t *testing.T) {
	img := createTestImage()
	if out := Sharpen(img, 0); out != img {
		t.Fatal("Sharpen(img, 0) should return the original image")
	}
}

func TestSharpen_NegativeAmount(t *testing.T) {
	img := createTestImage()
	if out := Sharpen(img, -1); out != img {
		t.Fatal("Sharpen(img, -1) should return the original image")
	}
}

func TestEdgeDetect(t *testing.T) {
	img := createTestImage()
	out := EdgeDetect(img)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("EdgeDetect dimensions = %dx%d", w, h)
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
		t.Fatal("EdgeDetect produced an all-black image on a gradient")
	}
}

func TestEmboss(t *testing.T) {
	img := createTestImage()
	out := Emboss(img)
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("Emboss dimensions = %dx%d", w, h)
	}
	c := out.At(50, 50)
	r, _, _, _ := c.RGBA()
	if r == 0 {
		t.Fatal("Emboss center pixel is pure black (bias not applied?)")
	}
}

func TestConvolve_SmallImage(t *testing.T) {
	// 1x1 image should not crash convolve.
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	kernel := [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}
	out := convolve(img, kernel, 1, 128)
	if w, h := Dimensions(out); w != 1 || h != 1 {
		t.Fatalf("convolve 1x1 dims = %dx%d", w, h)
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct{ in, want int }{
		{-10, 0}, {0, 0}, {128, 128}, {255, 255}, {300, 255},
	}
	for _, tt := range tests {
		if got := clampInt(tt.in, 0, 255); got != tt.want {
			t.Fatalf("clampInt(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestToRGBA(t *testing.T) {
	// NRGBA input
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	nrgba.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 200, B: 50, A: 255})
	out := toRGBA(nrgba)
	if out == nil {
		t.Fatal("toRGBA(NRGBA) returned nil")
	}
	// Gray input
	gray := image.NewGray(image.Rect(0, 0, 2, 2))
	out = toRGBA(gray)
	if out == nil {
		t.Fatal("toRGBA(Gray) returned nil")
	}
	// Already RGBA
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	out = toRGBA(rgba)
	if out != rgba {
		t.Fatal("toRGBA(RGBA) should return the same image")
	}
}
