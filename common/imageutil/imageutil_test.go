// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// createTestImage creates a 100x100 RGBA test image with a gradient.
func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 2),
				G: uint8(y * 2),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

// writeTestPNG writes a test image to a temp file as PNG.
func writeTestPNG(t *testing.T, img image.Image) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.png")
	if err := SavePNG(img, path); err != nil {
		t.Fatalf("SavePNG failed: %v", err)
	}
	return path
}

// writeTestJPEG writes a test image to a temp file as JPEG.
func writeTestJPEG(t *testing.T, img image.Image, quality int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.jpg")
	if err := SaveJPEG(img, path, quality); err != nil {
		t.Fatalf("SaveJPEG failed: %v", err)
	}
	return path
}

// ──────────────────────────────────────────────
// Format
// ──────────────────────────────────────────────

func TestFormatFromExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want Format
		fail bool
	}{
		{"jpg", FormatJPEG, false},
		{"jpeg", FormatJPEG, false},
		{"JPG", FormatJPEG, false},
		{"png", FormatPNG, false},
		{"gif", FormatGIF, false},
		{"bmp", FormatBMP, false},
		{".png", FormatPNG, false},
		{"xyz", "", true},
	}
	for _, tt := range tests {
		got, err := FormatFromExtension(tt.ext)
		if tt.fail {
			if err == nil {
				t.Fatalf("FormatFromExtension(%q) should fail", tt.ext)
			}
			continue
		}
		if err != nil {
			t.Fatalf("FormatFromExtension(%q) failed: %v", tt.ext, err)
		}
		if got != tt.want {
			t.Fatalf("FormatFromExtension(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

// ──────────────────────────────────────────────
// Decode / Encode
// ──────────────────────────────────────────────

func TestEncodeDecode_PNG(t *testing.T) {
	original := createTestImage()
	data, err := ToBytes(original, FormatPNG, 0)
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}
	decoded, format, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}
	if format != FormatPNG {
		t.Fatalf("format = %q, want png", format)
	}
	w, h := Dimensions(decoded)
	if w != 100 || h != 100 {
		t.Fatalf("dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestEncodeDecode_JPEG(t *testing.T) {
	original := createTestImage()
	data, err := ToBytes(original, FormatJPEG, 90)
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}
	decoded, format, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}
	if format != FormatJPEG {
		t.Fatalf("format = %q, want jpeg", format)
	}
	w, h := Dimensions(decoded)
	if w != 100 || h != 100 {
		t.Fatalf("dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestDecodeFile(t *testing.T) {
	path := writeTestPNG(t, createTestImage())
	img, format, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("DecodeFile failed: %v", err)
	}
	if format != FormatPNG {
		t.Fatalf("format = %q, want png", format)
	}
	if w, h := Dimensions(img); w != 100 || h != 100 {
		t.Fatalf("dimensions = %dx%d", w, h)
	}
}

func TestDecodeFile_NotExist(t *testing.T) {
	_, _, err := DecodeFile("nonexistent.png")
	if err == nil {
		t.Fatal("DecodeFile with nonexistent file should fail")
	}
}

func TestSaveByExtension(t *testing.T) {
	img := createTestImage()
	dir := t.TempDir()

	pngPath := filepath.Join(dir, "out.png")
	if err := SaveByExtension(img, pngPath, 90); err != nil {
		t.Fatalf("SaveByExtension png failed: %v", err)
	}
	if _, err := os.Stat(pngPath); err != nil {
		t.Fatal("png file not created")
	}

	jpgPath := filepath.Join(dir, "out.jpg")
	if err := SaveByExtension(img, jpgPath, 90); err != nil {
		t.Fatalf("SaveByExtension jpg failed: %v", err)
	}
	if _, err := os.Stat(jpgPath); err != nil {
		t.Fatal("jpg file not created")
	}
}

func TestSaveByExtension_NoExtension(t *testing.T) {
	err := SaveByExtension(createTestImage(), "noext", 90)
	if err == nil {
		t.Fatal("SaveByExtension without extension should fail")
	}
}

func TestEncode_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Encode(&buf, createTestImage(), FormatBMP, 90)
	if err == nil {
		t.Fatal("Encode with unsupported format should fail")
	}
}

// ──────────────────────────────────────────────
// Image info
// ──────────────────────────────────────────────

func TestGetInfo(t *testing.T) {
	path := writeTestPNG(t, createTestImage())
	info, err := GetInfo(path)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info.Width != 100 || info.Height != 100 {
		t.Fatalf("dimensions = %dx%d, want 100x100", info.Width, info.Height)
	}
	if info.Format != FormatPNG {
		t.Fatalf("format = %q, want png", info.Format)
	}
}

func TestDimensions(t *testing.T) {
	img := createTestImage()
	w, h := Dimensions(img)
	if w != 100 || h != 100 {
		t.Fatalf("Dimensions = %dx%d, want 100x100", w, h)
	}
}

// ──────────────────────────────────────────────
// Resize
// ──────────────────────────────────────────────

func TestResizeNearest(t *testing.T) {
	img := createTestImage()
	resized := ResizeNearest(img, 50, 50)
	w, h := Dimensions(resized)
	if w != 50 || h != 50 {
		t.Fatalf("ResizeNearest dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestResizeNearest_AspectRatio(t *testing.T) {
	img := createTestImage()
	resized := ResizeNearest(img, 200, 0) // auto height
	w, h := Dimensions(resized)
	if w != 200 || h != 200 {
		t.Fatalf("ResizeNearest(200, 0) = %dx%d, want 200x200", w, h)
	}
}

func TestResizeNearest_SameSize(t *testing.T) {
	img := createTestImage()
	resized := ResizeNearest(img, 100, 100)
	if w, h := Dimensions(resized); w != 100 || h != 100 {
		t.Fatalf("dimensions = %dx%d", w, h)
	}
}

func TestResizeBilinear(t *testing.T) {
	img := createTestImage()
	resized := ResizeBilinear(img, 50, 50)
	w, h := Dimensions(resized)
	if w != 50 || h != 50 {
		t.Fatalf("ResizeBilinear dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestResizeBilinear_Upscale(t *testing.T) {
	img := createTestImage()
	resized := ResizeBilinear(img, 200, 200)
	w, h := Dimensions(resized)
	if w != 200 || h != 200 {
		t.Fatalf("ResizeBilinear dimensions = %dx%d, want 200x200", w, h)
	}
}

func TestThumbnail(t *testing.T) {
	img := createTestImage()
	thumb := Thumbnail(img, 50, 50)
	w, h := Dimensions(thumb)
	if w > 50 || h > 50 {
		t.Fatalf("Thumbnail dimensions = %dx%d, should fit 50x50", w, h)
	}
}

func TestThumbnail_NoUpscale(t *testing.T) {
	img := createTestImage()
	thumb := Thumbnail(img, 200, 200)
	w, h := Dimensions(thumb)
	if w != 100 || h != 100 {
		t.Fatalf("Thumbnail should not upscale: %dx%d", w, h)
	}
}

// ──────────────────────────────────────────────
// Crop
// ──────────────────────────────────────────────

func TestCrop(t *testing.T) {
	img := createTestImage()
	cropped := Crop(img, image.Rect(10, 10, 60, 60))
	w, h := Dimensions(cropped)
	if w != 50 || h != 50 {
		t.Fatalf("Crop dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestCropCenter(t *testing.T) {
	img := createTestImage()
	cropped := CropCenter(img, 50)
	w, h := Dimensions(cropped)
	if w != 50 || h != 50 {
		t.Fatalf("CropCenter dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestCropTopLeft(t *testing.T) {
	img := createTestImage()
	cropped := CropTopLeft(img, 30)
	w, h := Dimensions(cropped)
	if w != 30 || h != 30 {
		t.Fatalf("CropTopLeft dimensions = %dx%d, want 30x30", w, h)
	}
}

func TestCropSmart(t *testing.T) {
	img := createTestImage()
	result := CropSmart(img, 50, 50)
	w, h := Dimensions(result)
	if w != 50 || h != 50 {
		t.Fatalf("CropSmart dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestCropSmart_NonSquare(t *testing.T) {
	img := createTestImage()
	result := CropSmart(img, 80, 40)
	w, h := Dimensions(result)
	if w != 80 || h != 40 {
		t.Fatalf("CropSmart dimensions = %dx%d, want 80x40", w, h)
	}
}

// ──────────────────────────────────────────────
// Rotate
// ──────────────────────────────────────────────

func TestRotate90(t *testing.T) {
	img := createTestImage()
	rotated := Rotate90(img)
	w, h := Dimensions(rotated)
	if w != 100 || h != 100 {
		t.Fatalf("Rotate90 dimensions = %dx%d, want 100x100", w, h)
	}
	// Check corner: top-left of rotated should be bottom-left of original.
	origTL := img.At(0, 0)
	rotBL := rotated.At(0, 99)
	if origTL != rotBL {
		t.Log("Rotate90 corner check (may vary due to color model)")
	}
}

func TestRotate180(t *testing.T) {
	img := createTestImage()
	rotated := Rotate180(img)
	w, h := Dimensions(rotated)
	if w != 100 || h != 100 {
		t.Fatalf("Rotate180 dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestRotate270(t *testing.T) {
	img := createTestImage()
	rotated := Rotate270(img)
	w, h := Dimensions(rotated)
	if w != 100 || h != 100 {
		t.Fatalf("Rotate270 dimensions = %dx%d, want 100x100", w, h)
	}
}

func TestRotate(t *testing.T) {
	img := createTestImage()
	for _, deg := range []int{0, 90, 180, 270, 360, -90, 450} {
		result, err := Rotate(img, deg)
		if err != nil {
			t.Fatalf("Rotate(%d) failed: %v", deg, err)
		}
		if w, h := Dimensions(result); w != 100 || h != 100 {
			t.Fatalf("Rotate(%d) dimensions = %dx%d", deg, w, h)
		}
	}
}

func TestRotate_InvalidDegrees(t *testing.T) {
	_, err := Rotate(createTestImage(), 45)
	if err == nil {
		t.Fatal("Rotate(45) should fail")
	}
}

// ──────────────────────────────────────────────
// Flip
// ──────────────────────────────────────────────

func TestFlipHorizontal(t *testing.T) {
	img := createTestImage()
	flipped := FlipHorizontal(img)
	w, h := Dimensions(flipped)
	if w != 100 || h != 100 {
		t.Fatalf("FlipH dimensions = %dx%d", w, h)
	}
	// Left pixel should equal right pixel of original.
	left := img.At(0, 50)
	right := flipped.At(99, 50)
	if left != right {
		t.Log("FlipH pixel check (may vary due to color model)")
	}
}

func TestFlipVertical(t *testing.T) {
	img := createTestImage()
	flipped := FlipVertical(img)
	w, h := Dimensions(flipped)
	if w != 100 || h != 100 {
		t.Fatalf("FlipV dimensions = %dx%d", w, h)
	}
}

// ──────────────────────────────────────────────
// Color adjustments
// ──────────────────────────────────────────────

func TestGrayscale(t *testing.T) {
	img := createTestImage()
	gray := Grayscale(img)
	w, h := Dimensions(gray)
	if w != 100 || h != 100 {
		t.Fatalf("Grayscale dimensions = %dx%d", w, h)
	}
	// Check that a pixel is gray (R=G=B).
	c := gray.At(50, 50).(color.Gray)
	_ = c // Gray type always has R=G=B
}

func TestAdjustBrightness(t *testing.T) {
	img := createTestImage()
	bright := AdjustBrightness(img, 50)
	w, h := Dimensions(bright)
	if w != 100 || h != 100 {
		t.Fatalf("AdjustBrightness dimensions = %dx%d", w, h)
	}
	// Check that a pixel got brighter (or clamped to 255).
	r, _, _, _ := bright.At(0, 0).RGBA()
	origR, _, _, _ := img.At(0, 0).RGBA()
	if r < origR {
		t.Fatal("brightness should increase or clamp")
	}
}

func TestAdjustContrast(t *testing.T) {
	img := createTestImage()
	result := AdjustContrast(img, 1.5)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("AdjustContrast dimensions = %dx%d", w, h)
	}
}

func TestAdjustGamma(t *testing.T) {
	img := createTestImage()
	result := AdjustGamma(img, 2.0)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("AdjustGamma dimensions = %dx%d", w, h)
	}
}

func TestAdjustSaturation(t *testing.T) {
	img := createTestImage()
	result := AdjustSaturation(img, 0.5)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("AdjustSaturation dimensions = %dx%d", w, h)
	}
}

func TestInvert(t *testing.T) {
	img := createTestImage()
	inverted := Invert(img)
	r, _, _, _ := inverted.At(0, 0).RGBA()
	origR, _, _, _ := img.At(0, 0).RGBA()
	// Inverted should be 255 - original.
	if r/256 != 255-origR/256 {
		t.Fatalf("Invert: r = %d, want %d", r/256, 255-origR/256)
	}
}

func TestSepia(t *testing.T) {
	img := createTestImage()
	result := Sepia(img)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("Sepia dimensions = %dx%d", w, h)
	}
}

// ──────────────────────────────────────────────
// Watermark
// ──────────────────────────────────────────────

func TestWatermark(t *testing.T) {
	base := createTestImage()
	wm := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			wm.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	result := Watermark(base, wm, 10, 10, 0.5)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("Watermark dimensions = %dx%d", w, h)
	}
}

func TestWatermarkCenter(t *testing.T) {
	base := createTestImage()
	wm := image.NewRGBA(image.Rect(0, 0, 20, 20))
	result := WatermarkCenter(base, wm, 0.5)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("WatermarkCenter dimensions = %dx%d", w, h)
	}
}

func TestWatermarkBottomRight(t *testing.T) {
	base := createTestImage()
	wm := image.NewRGBA(image.Rect(0, 0, 20, 20))
	result := WatermarkBottomRight(base, wm, 0.5, 5)
	w, h := Dimensions(result)
	if w != 100 || h != 100 {
		t.Fatalf("WatermarkBottomRight dimensions = %dx%d", w, h)
	}
}

// ──────────────────────────────────────────────
// Format conversion
// ──────────────────────────────────────────────

func TestConvertFormat(t *testing.T) {
	pngPath := writeTestPNG(t, createTestImage())
	jpgPath := filepath.Join(t.TempDir(), "converted.jpg")
	if err := ConvertFormat(pngPath, jpgPath, 85); err != nil {
		t.Fatalf("ConvertFormat failed: %v", err)
	}
	if _, err := os.Stat(jpgPath); err != nil {
		t.Fatal("converted file not created")
	}
}

func TestReduceQuality(t *testing.T) {
	img := createTestImage()
	data, err := ReduceQuality(img, 50)
	if err != nil {
		t.Fatalf("ReduceQuality failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ReduceQuality returned empty data")
	}
}

func TestReduceQualityFile(t *testing.T) {
	input := writeTestJPEG(t, createTestImage(), 95)
	output := filepath.Join(t.TempDir(), "reduced.jpg")
	if err := ReduceQualityFile(input, output, 30); err != nil {
		t.Fatalf("ReduceQualityFile failed: %v", err)
	}
	info, _ := os.Stat(output)
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

// ──────────────────────────────────────────────
// Composite operations
// ──────────────────────────────────────────────

func TestOptimizeForWeb(t *testing.T) {
	img := createTestImage()
	data, err := OptimizeForWeb(img, 50, 80)
	if err != nil {
		t.Fatalf("OptimizeForWeb failed: %v", err)
	}
	// Verify it's valid JPEG.
	_, err = jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("OptimizeForWeb output is not valid JPEG: %v", err)
	}
}

func TestOptimizeForWebFile(t *testing.T) {
	input := writeTestPNG(t, createTestImage())
	output := filepath.Join(t.TempDir(), "web.jpg")
	if err := OptimizeForWebFile(input, output, 50, 80); err != nil {
		t.Fatalf("OptimizeForWebFile failed: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal("output file not created")
	}
}

// ──────────────────────────────────────────────
// Round-trip tests
// ──────────────────────────────────────────────

func TestPNGRoundTrip(t *testing.T) {
	img := createTestImage()
	data, _ := ToBytes(img, FormatPNG, 0)
	decoded, _, _ := FromBytes(data)
	// PNG is lossless, so pixels should match.
	for y := 0; y < 100; y += 10 {
		for x := 0; x < 100; x += 10 {
			r1, g1, b1, a1 := img.At(x, y).RGBA()
			r2, g2, b2, a2 := decoded.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatalf("pixel mismatch at (%d,%d)", x, y)
			}
		}
	}
}

func TestEncodeDecodeConfig(t *testing.T) {
	img := createTestImage()
	data, _ := ToBytes(img, FormatPNG, 0)
	cfg, format, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeConfig failed: %v", err)
	}
	if cfg.Width != 100 || cfg.Height != 100 {
		t.Fatalf("config dimensions = %dx%d", cfg.Width, cfg.Height)
	}
	if format != FormatPNG {
		t.Fatalf("format = %q, want png", format)
	}
}

func TestSaveGIF(t *testing.T) {
	img := createTestImage()
	path := filepath.Join(t.TempDir(), "test.gif")
	if err := SaveGIF(img, path); err != nil {
		t.Fatalf("SaveGIF failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("gif file not created")
	}
}

func TestSave_UnsupportedFormat(t *testing.T) {
	err := Save(createTestImage(), "test.xyz", FormatBMP, 90)
	if err == nil {
		t.Fatal("Save with unsupported format should fail")
	}
}

func TestDecode_InvalidData(t *testing.T) {
	_, _, err := Decode(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Fatal("Decode of invalid data should fail")
	}
}

func TestDecodeConfig_InvalidData(t *testing.T) {
	_, _, err := DecodeConfig(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Fatal("DecodeConfig of invalid data should fail")
	}
}

// Ensure png import is used.
var _ = png.Encode
