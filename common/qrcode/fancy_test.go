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

func TestGenerateFancy_Default(t *testing.T) {
	data, err := GenerateFancy("https://ling-base.dev", ECLHigh, FancyOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
}

func TestGenerateFancy_AllModuleShapes(t *testing.T) {
	shapes := []ModuleShape{
		ShapeRectangle,
		ShapeCircle,
		ShapeRounded,
		ShapeLiquid,
		ShapeHStripe,
		ShapeVStripe,
		ShapeDiamond,
	}
	for _, s := range shapes {
		opts := FancyOptions{
			Module:      s,
			FgColor:     color.RGBA{R: 0, G: 100, B: 200, A: 255},
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		}
		data, err := GenerateFancy("shape test", ECLHigh, opts)
		require.NoError(t, err, "module shape %d", s)
		assert.NotEmpty(t, data)

		img, err := png.Decode(bytes.NewReader(data))
		require.NoError(t, err)
		assert.True(t, img.Bounds().Dx() > 0)
	}
}

func TestGenerateFancy_FinderShapes(t *testing.T) {
	for _, f := range []FinderShape{FinderRounded, FinderSquare} {
		opts := FancyOptions{
			Module: ShapeRectangle,
			Finder: f,
		}
		data, err := GenerateFancy("finder test", ECLHigh, opts)
		require.NoError(t, err, "finder %d", f)
		assert.NotEmpty(t, data)
	}
}

func TestGenerateFancy_RoundedFinderWithCircle(t *testing.T) {
	opts := FancyOptions{
		Module: ShapeCircle,
		Finder: FinderRounded,
	}
	data, err := GenerateFancy("circle + rounded finder", ECLHigh, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGenerateFancy_Gradient(t *testing.T) {
	grad := NewLinearGradient(45,
		ColorStop{Color: color.RGBA{255, 0, 0, 255}, T: 0.0},
		ColorStop{Color: color.RGBA{0, 0, 255, 255}, T: 1.0},
	)
	opts := FancyOptions{
		Module:      ShapeCircle,
		FgGradient:  grad,
		BgColor:     color.White,
		ModuleWidth: 21,
		BorderWidth: 20,
	}
	data, err := GenerateFancy("gradient test", ECLHigh, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGenerateFancy_TransparentBg(t *testing.T) {
	opts := FancyOptions{
		Module:        ShapeCircle,
		FgColor:       color.Black,
		BgTransparent: true,
		ModuleWidth:   21,
	}
	data, err := GenerateFancy("transparent test", ECLHigh, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGenerateFancy_WithLogo(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			logo.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	opts := FancyOptions{
		Module:             ShapeCircle,
		Finder:             FinderRounded,
		Logo:               logo,
		LogoSizeMultiplier: 5,
		LogoSafeZone:       true,
		ModuleWidth:        21,
		BorderWidth:        20,
	}
	data, err := GenerateFancy("logo test", ECLHigh, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGenerateFancy_WithHalftone(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			src.SetRGBA(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: 128,
				A: 255,
			})
		}
	}
	opts := FancyOptions{
		Module:      ShapeRectangle,
		FgColor:     color.Black,
		BgColor:     color.White,
		Halftone:    src,
		ModuleWidth: 21,
		BorderWidth: 20,
	}
	data, err := GenerateFancy("halftone test", ECLHigh, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGenerateFancy_EmptyText(t *testing.T) {
	_, err := GenerateFancy("", ECLHigh, FancyOptions{})
	assert.Error(t, err)
}

func TestGenerateFancy_AllLevels(t *testing.T) {
	for _, level := range []ErrorCorrectionLevel{ECLLow, ECLMedium, ECLQuartile, ECLHigh} {
		data, err := GenerateFancy("level test", level, FancyOptions{Module: ShapeCircle})
		require.NoError(t, err, "level %d", level)
		assert.NotEmpty(t, data)
	}
}

func TestSaveFancy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fancy.png")
	opts := FancyCirclePreset(color.RGBA{R: 0, G: 100, B: 200, A: 255}, color.White)
	err := SaveFancy("save fancy test", path, ECLHigh, opts)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	f, _ := os.Open(path)
	defer f.Close()
	_, err = png.Decode(f)
	require.NoError(t, err)
}

func TestSaveFancy_Error(t *testing.T) {
	err := SaveFancy("test", "/nonexistent/dir/fancy.png", ECLHigh, FancyOptions{})
	assert.Error(t, err)
}

func TestGenerateFancyImage(t *testing.T) {
	opts := FancyCirclePreset(color.Black, color.White)
	img, err := GenerateFancyImage("image test", ECLHigh, opts)
	require.NoError(t, err)
	assert.True(t, img.Bounds().Dx() > 0)
	assert.True(t, img.Bounds().Dy() > 0)
}

func TestGenerateFancyImage_EmptyText(t *testing.T) {
	_, err := GenerateFancyImage("", ECLHigh, FancyOptions{})
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Presets
// ──────────────────────────────────────────────

func TestFancyCirclePreset(t *testing.T) {
	opts := FancyCirclePreset(color.Black, color.White)
	assert.Equal(t, ShapeCircle, opts.Module)
	assert.NotNil(t, opts.FgColor)
	assert.NotNil(t, opts.BgColor)
}

func TestFancyGradientPreset(t *testing.T) {
	grad := NewLinearGradient(90, ColorStop{Color: color.White, T: 0})
	opts := FancyGradientPreset(grad)
	assert.Equal(t, ShapeCircle, opts.Module)
	assert.NotNil(t, opts.FgGradient)
}

func TestFancyLogoPreset(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 10, 10))
	opts := FancyLogoPreset(logo)
	assert.Equal(t, ShapeCircle, opts.Module)
	assert.Equal(t, FinderRounded, opts.Finder)
	assert.NotNil(t, opts.Logo)
	assert.True(t, opts.LogoSafeZone)
}

func TestFancyHalftonePreset(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	opts := FancyHalftonePreset(src, color.Black)
	assert.Equal(t, ShapeRectangle, opts.Module)
	assert.NotNil(t, opts.Halftone)
}

func TestNewLinearGradient(t *testing.T) {
	g := NewLinearGradient(45, ColorStop{Color: color.Black, T: 0}, ColorStop{Color: color.White, T: 1})
	assert.Equal(t, 45.0, g.Angle)
	assert.Len(t, g.Stops, 2)
}

func TestToRGBA(t *testing.T) {
	rgba := toRGBA(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	assert.Equal(t, uint8(255), rgba.R)
	assert.Equal(t, uint8(0), rgba.G)

	rgba = toRGBA(nil)
	assert.Equal(t, uint8(0), rgba.R)
	assert.Equal(t, uint8(255), rgba.A)
}

func TestNopWriteCloser(t *testing.T) {
	var buf bytes.Buffer
	w := nopWriteCloser{&buf}
	_, err := w.Write([]byte("test"))
	assert.NoError(t, err)
	err = w.Close()
	assert.NoError(t, err)
	assert.Equal(t, "test", buf.String())
}
