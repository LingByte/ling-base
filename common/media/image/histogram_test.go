// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestCalcHistogram(t *testing.T) {
	img := createTestImage()
	h := CalcHistogram(img)
	if h.Total != 100*100 {
		t.Fatalf("Histogram Total = %d, want 10000", h.Total)
	}
	if h.Width != 100 || h.Height != 100 {
		t.Fatalf("Histogram dims = %dx%d", h.Width, h.Height)
	}
	if h.R[0] != 100 {
		t.Fatalf("R[0] = %d, want 100", h.R[0])
	}
	if h.R[2] != 100 {
		t.Fatalf("R[2] = %d, want 100", h.R[2])
	}
	if h.B[128] != 10000 {
		t.Fatalf("B[128] = %d, want 10000", h.B[128])
	}
}

func TestHistogram_MeanRGB(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 100, G: 200, B: 50, A: 255})
		}
	}
	h := CalcHistogram(img)
	r, g, b := h.MeanRGB()
	if r != 100 || g != 200 || b != 50 {
		t.Fatalf("MeanRGB = %.1f,%.1f,%.1f, want 100,200,50", r, g, b)
	}
}

func TestHistogram_Luminance(t *testing.T) {
	img := createTestImage()
	h := CalcHistogram(img)
	lum := h.Luminance()
	total := uint32(0)
	for _, v := range lum {
		total += v
	}
	if total != uint32(h.Total) {
		t.Fatalf("Luminance total = %d, want %d", total, h.Total)
	}
}

func TestHistogram_Contrast(t *testing.T) {
	img := createTestImage()
	h := CalcHistogram(img)
	c := h.Contrast()
	if c != h.StdDev {
		t.Fatalf("Contrast = %v, want StdDev = %v", c, h.StdDev)
	}
	if c < 0 {
		t.Fatal("Contrast should be non-negative")
	}
}

func TestHistogram_UniformImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	h := CalcHistogram(img)
	// Uniform image: contrast (stddev) should be 0.
	if h.Contrast() > 0.01 {
		t.Fatalf("Uniform image Contrast = %.2f, want ~0", h.Contrast())
	}
	if h.Mean < 127.5 || h.Mean > 128.5 {
		t.Fatalf("Uniform image Mean = %.2f, want ~128", h.Mean)
	}
}

func TestHistogram_BlankImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	h := CalcHistogram(img)
	if h.Total != 25 {
		t.Fatalf("blank image Total = %d, want 25", h.Total)
	}
	if h.R[0] != 25 {
		t.Fatalf("blank image R[0] = %d, want 25", h.R[0])
	}
}

func TestHistogram_NRGBAImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	h := CalcHistogram(img)
	if h.R[255] != 25 {
		t.Fatalf("NRGBA R[255] = %d, want 25", h.R[255])
	}
}

func TestHistogram_GrayImage(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.SetGray(x, y, color.Gray{Y: 200})
		}
	}
	h := CalcHistogram(img)
	if h.Total != 25 {
		t.Fatalf("Gray Total = %d, want 25", h.Total)
	}
}
