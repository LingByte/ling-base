// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command image-demo is a CLI tool that demonstrates the ling-base
// common/imageutil package on a source image.
//
// It produces a set of processed variants (resize, crop, filters, color
// adjustments, composites, watermarks, rounded corners, etc.) and writes them
// to an output directory, plus prints the source image's histogram summary.
//
// Usage:
//
//	# Process the bundled sample.png into ./out
//	go run ./cmd/image-demo
//
//	# Use a custom input and output dir
//	go run ./cmd/image-demo -in path/to/photo.jpg -out ./processed
//
//	# Only run a subset of operations
//	go run ./cmd/image-demo -ops resize,filters,watermark
//
// All output files are PNG (lossless) so quality differences are visible.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/ling-base/common/imageutil"
)

func main() {
	inFlag := flag.String("in", "sample.png", "input image path (relative to -indir)")
	inDirFlag := flag.String("indir", ".", "directory containing the input image")
	outFlag := flag.String("out", "out", "output directory")
	opsFlag := flag.String("ops", "all", "comma-separated list of operation groups to run (all|resize|crop|filters|color|composite|watermark|histogram)")
	flag.Parse()

	inPath := filepath.Join(*inDirFlag, *inFlag)
	if _, err := os.Stat(inPath); err != nil {
		fmt.Fprintf(os.Stderr, "input image not found: %s (%v)\n", inPath, err)
		fmt.Fprintf(os.Stderr, "tip: place a PNG/JPEG at %s or use -in/-indir\n", inPath)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outFlag, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create out dir: %v\n", err)
		os.Exit(1)
	}

	img, format, err := imageutil.DecodeFile(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode %s: %v\n", inPath, err)
		os.Exit(1)
	}
	w, h := imageutil.Dimensions(img)
	fmt.Printf("Loaded %s  format=%s  size=%dx%d\n", inPath, format, w, h)

	ops := parseOps(*opsFlag)

	var n int
	save := func(name string, out image.Image) {
		path := filepath.Join(*outFlag, name+".png")
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ! %s: %v\n", name, err)
			return
		}
		defer f.Close()
		if err := png.Encode(f, out); err != nil {
			fmt.Fprintf(os.Stderr, "  ! %s: %v\n", name, err)
			return
		}
		n++
		fmt.Printf("  -> %s\n", path)
	}

	// ── Resize ──────────────────────────────────────
	if ops["resize"] {
		fmt.Println("\n[resize]")
		save("resize_bilinear_200x0", imageutil.ResizeBilinear(img, 200, 0))
		save("resize_nearest_200x0", imageutil.ResizeNearest(img, 200, 0))
		save("resize_catmullrom_200x0", imageutil.ResizeCatmullRom(img, 200, 0)) // high-quality cubic
		save("thumbnail_128x128", imageutil.Thumbnail(img, 128, 128))
		save("resize_pad_200x200_black", imageutil.ResizeWithPadding(img, 200, 200, color.Black))
		save("resize_pad_200x200_white", imageutil.ResizeWithPadding(img, 200, 200, color.White))
		save("thumb_pad_200x200_white", imageutil.ThumbnailWithPadding(img, 200, 200, color.White))
	}

	// ── Crop ────────────────────────────────────────
	if ops["crop"] {
		fmt.Println("\n[crop]")
		save("crop_center_100x100", imageutil.CropCenter(img, 100))
		save("crop_smart_200x200", imageutil.CropSmart(img, 200, 200))
		save("crop_aspect_1x1_topleft", imageutil.CropAspectRatio(img, 1, 1, imageutil.AnchorTopLeft))
		save("crop_aspect_1x1_center", imageutil.CropAspectRatio(img, 1, 1, imageutil.AnchorMiddleCenter))
		save("crop_aspect_16x9_center", imageutil.CropAspectRatio(img, 16, 9, imageutil.AnchorMiddleCenter))
		save("crop_aspect_1x1_resize_128x128", imageutil.CropAspectRatioResize(img, 1, 1, 128, 128, imageutil.AnchorMiddleCenter))
		save("crop_circle", imageutil.CropCircle(img))
		save("round_corners_24", imageutil.RoundCorners(img, 24))
		save("round_corners_48", imageutil.RoundCorners(img, 48))
	}

	// ── Filters ─────────────────────────────────────
	if ops["filters"] {
		fmt.Println("\n[filters]")
		save("box_blur_r2", imageutil.BoxBlur(img, 2))
		save("gaussian_blur_r3_s1.5", imageutil.GaussianBlur(img, 3, 1.5))
		save("sharpen_1.0", imageutil.Sharpen(img, 1.0))
		save("sharpen_2.0", imageutil.Sharpen(img, 2.0))
		save("edge_detect", imageutil.EdgeDetect(img))
		save("emboss", imageutil.Emboss(img))
	}

	// ── Color adjustments ───────────────────────────
	if ops["color"] {
		fmt.Println("\n[color]")
		save("grayscale", imageutil.Grayscale(img))
		save("invert", imageutil.Invert(img))
		save("sepia", imageutil.Sepia(img))
		save("brightness_+40", imageutil.AdjustBrightness(img, 40))
		save("brightness_-40", imageutil.AdjustBrightness(img, -40))
		save("contrast_1.5", imageutil.AdjustContrast(img, 1.5))
		save("gamma_0.7", imageutil.AdjustGamma(img, 0.7))
		save("gamma_1.5", imageutil.AdjustGamma(img, 1.5))
		save("saturation_1.5", imageutil.AdjustSaturation(img, 1.5))
		save("hue_+120", imageutil.HueRotate(img, 120))
		save("hue_+180", imageutil.HueRotate(img, 180))
		save("temp_warm_+50", imageutil.AdjustTemperature(img, 50))
		save("temp_cool_-50", imageutil.AdjustTemperature(img, -50))
		save("tint_red_0.4", imageutil.Tint(img, color.RGBA{R: 255, A: 255}, 0.4))
		save("tint_blue_0.4", imageutil.Tint(img, color.RGBA{B: 255, A: 255}, 0.4))
		save("posterize_4", imageutil.Posterize(img, 4))
		save("posterize_8", imageutil.Posterize(img, 8))
		save("threshold_128", imageutil.Threshold(img, 128))
	}

	// ── Composite ───────────────────────────────────
	if ops["composite"] {
		fmt.Println("\n[composite]")
		save("border_10_black", imageutil.AddBorder(img, 10, color.Black))
		save("border_10_white", imageutil.AddBorder(img, 10, color.White))
		save("pad_400x400_black", imageutil.AddPadding(img, 400, 400, color.Black))

		// Build a solid red overlay for blend-mode demos.
		overlay := image.NewRGBA(img.Bounds())
		for y := 0; y < img.Bounds().Dy(); y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				overlay.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 128})
			}
		}
		save("blend_multiply", imageutil.Blend(img, overlay, imageutil.BlendMultiply))
		save("blend_screen", imageutil.Blend(img, overlay, imageutil.BlendScreen))
		save("blend_overlay", imageutil.Blend(img, overlay, imageutil.BlendOverlay))
		save("blend_add", imageutil.Blend(img, overlay, imageutil.BlendAdd))
		save("blend_difference", imageutil.Blend(img, overlay, imageutil.BlendDifference))
	}

	// ── Watermark ───────────────────────────────────
	if ops["watermark"] {
		fmt.Println("\n[watermark]")
		// Image-based watermark: a 60x20 white block.
		wm := image.NewRGBA(image.Rect(0, 0, 60, 20))
		for y := 0; y < 20; y++ {
			for x := 0; x < 60; x++ {
				wm.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 200})
			}
		}
		save("watermark_center_0.5", imageutil.WatermarkCenter(img, wm, 0.5))
		save("watermark_bottomright_0.7_pad10", imageutil.WatermarkBottomRight(img, wm, 0.7, 10))
		save("watermark_tiled_30x30_0.3", imageutil.TileWatermark(img, wm, 30, 30, 0.3))

		// Text watermarks — built-in Go fonts (no .ttf file needed).
		// Use larger font sizes for crisp rendering; the built-in fonts
		// support common symbols including ©.
		save("text_wm_center_bold", imageutil.TextWatermarkCenter(img, "LingByte", imageutil.TextWatermarkOptions{
			Font: imageutil.FontGoBold, FontSize: 72, Color: color.White, Opacity: 0.6,
		}))
		save("text_wm_bottomright_copyright", imageutil.TextWatermarkBottomRight(img, "© 2026 LingByte", imageutil.TextWatermarkOptions{
			Font: imageutil.FontGoMedium, FontSize: 32, Color: color.White, Opacity: 0.85, Padding: 24,
		}))
		save("text_wm_tiled_diagonal", imageutil.TextWatermarkTiled(img, "CONFIDENTIAL", imageutil.TextWatermarkOptions{
			Font: imageutil.FontGoBold, FontSize: 48, Color: color.RGBA{R: 255, G: 80, B: 80, A: 255}, Opacity: 0.35, Angle: -30, Padding: 100,
		}))
		save("text_wm_center_mono", imageutil.TextWatermarkCenter(img, "DRAFT v0.1", imageutil.TextWatermarkOptions{
			Font: imageutil.FontGoMono, FontSize: 56, Color: color.RGBA{R: 255, G: 200, B: 0, A: 255}, Opacity: 0.7,
		}))
	}

	// ── Histogram ───────────────────────────────────
	if ops["histogram"] {
		fmt.Println("\n[histogram]")
		h := imageutil.CalcHistogram(img)
		mr, mg, mb := h.MeanRGB()
		fmt.Printf("  total pixels : %d\n", h.Total)
		fmt.Printf("  mean RGB     : %.2f, %.2f, %.2f\n", mr, mg, mb)
		fmt.Printf("  luminance    : mean=%.2f  stddev=%.2f  min=%d  max=%d\n",
			h.Mean, h.StdDev, h.Min, h.Max)
		fmt.Printf("  contrast     : %.2f\n", h.Contrast())

		// Write a tiny ASCII luminance histogram (32 buckets).
		lum := h.Luminance()
		var maxB uint32
		for _, v := range lum {
			if v > maxB {
				maxB = v
			}
		}
		if maxB == 0 {
			maxB = 1
		}
		const cols = 32
		fmt.Println("  luminance histogram (32 buckets):")
		for i := 0; i < cols; i++ {
			lo := i * 256 / cols
			hi := (i + 1) * 256 / cols
			var sum uint32
			for j := lo; j < hi; j++ {
				sum += lum[j]
			}
			barLen := int(float64(sum) / float64(maxB) * 40)
			fmt.Printf("  %3d-%-3d | %s\n", lo, hi-1, strings.Repeat("#", barLen))
		}
	}

	fmt.Printf("\nDone. Wrote %d files to %s/\n", n, *outFlag)
}

// parseOps parses the -ops flag into a set lookup. "all" enables every group.
func parseOps(s string) map[string]bool {
	all := map[string]bool{
		"resize":    true,
		"crop":      true,
		"filters":   true,
		"color":     true,
		"composite": true,
		"watermark": true,
		"histogram": true,
	}
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "all" {
		return all
	}
	out := make(map[string]bool)
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if _, ok := all[name]; ok {
			out[name] = true
		} else {
			fmt.Fprintf(os.Stderr, "ignoring unknown op group: %q\n", name)
		}
	}
	return out
}
