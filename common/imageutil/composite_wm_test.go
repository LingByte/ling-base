// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"image"
	"image/color"
	"testing"
)

// makeLogo creates a simple 30x30 solid-color logo for testing.
func makeLogo(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 30, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 30; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestCompositeWatermark_LogoLeftTextRight(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	out := CompositeWatermarkBottomRight(base, CompositeWatermarkOptions{
		Logo: logo, Text: "Brand", FontSize: 16, Opacity: 1.0, Padding: 5, Spacing: 4,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("CompositeWatermarkBottomRight dims = %dx%d", w, h)
	}
	foundRed, foundWhite := false, false
	for y := 70; y < 100; y++ {
		for x := 0; x < 100; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			if r/256 > 200 && g/256 < 50 && b/256 < 50 {
				foundRed = true
			}
			if r/256 > 200 && g/256 > 200 && b/256 > 200 {
				foundWhite = true
			}
		}
	}
	if !foundRed {
		t.Fatal("CompositeWatermark did not render the logo (no red pixels found)")
	}
	if !foundWhite {
		t.Fatal("CompositeWatermark did not render the text (no white pixels found)")
	}
}

func TestCompositeWatermark_LogoRightTextLeft(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 0, G: 255, B: 0, A: 255})
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Logo: logo, Text: "Test", FontSize: 16, Layout: LayoutLogoRightTextLeft, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("CompositeWatermarkCenter dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_LogoTopTextBottom(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 0, G: 0, B: 255, A: 255})
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Logo: logo, Text: "Stacked", FontSize: 14, Layout: LayoutLogoTopTextBottom, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("vertical layout dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_LogoBottomTextTop(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 255, A: 255})
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Logo: logo, Text: "TopText", FontSize: 14, Layout: LayoutLogoBottomTextTop, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("logo-bottom-text-top dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_LogoOnly(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 255, B: 0, A: 255})
	out := CompositeWatermarkBottomRight(base, CompositeWatermarkOptions{
		Logo: logo, Opacity: 1.0, Padding: 5,
	})
	found := false
	for y := 80; y < 100 && !found; y++ {
		for x := 80; x < 100 && !found; x++ {
			r, g, b, _ := out.At(x, y).RGBA()
			if r/256 > 200 && g/256 > 200 && b/256 < 50 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("CompositeWatermark with logo only did not render the logo")
	}
}

func TestCompositeWatermark_TextOnly(t *testing.T) {
	base := createTestImage()
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Text: "NoLogo", FontSize: 16, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("text-only dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_LogoHeightScaling(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	out := CompositeWatermarkBottomRight(base, CompositeWatermarkOptions{
		Logo: logo, Text: "Scaled", FontSize: 16, LogoHeight: 16, Opacity: 1.0, Padding: 5,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("scaled logo dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_AllCorners(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	for _, fn := range []struct {
		name string
		f    func(image.Image, CompositeWatermarkOptions) image.Image
	}{
		{"TopLeft", CompositeWatermarkTopLeft},
		{"TopRight", CompositeWatermarkTopRight},
		{"BottomLeft", CompositeWatermarkBottomLeft},
		{"BottomRight", CompositeWatermarkBottomRight},
		{"Center", CompositeWatermarkCenter},
	} {
		out := fn.f(base, CompositeWatermarkOptions{
			Logo: logo, Text: "X", FontSize: 12, Opacity: 1.0, Padding: 5,
		})
		if w, h := Dimensions(out); w != 100 || h != 100 {
			t.Fatalf("%s dims = %dx%d", fn.name, w, h)
		}
	}
}

func TestCompositeWatermark_AtPosition(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	out := CompositeWatermark(base, 10, 10, CompositeWatermarkOptions{
		Logo: logo, Text: "Pos", FontSize: 12, Opacity: 1.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("CompositeWatermark at position dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_Defaults(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	// Zero-value options should use defaults.
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Logo: logo, Text: "D",
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("defaults dims = %dx%d", w, h)
	}
}

func TestCompositeWatermark_OpacityClamp(t *testing.T) {
	base := createTestImage()
	logo := makeLogo(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	out := CompositeWatermarkCenter(base, CompositeWatermarkOptions{
		Logo: logo, Text: "C", FontSize: 12, Opacity: 5.0,
	})
	if w, h := Dimensions(out); w != 100 || h != 100 {
		t.Fatalf("opacity-clamped dims = %dx%d", w, h)
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(3, 5) != 5 {
		t.Fatal("maxInt(3, 5) should be 5")
	}
	if maxInt(10, 2) != 10 {
		t.Fatal("maxInt(10, 2) should be 10")
	}
}
