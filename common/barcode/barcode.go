// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package barcode provides one-dimensional and two-dimensional barcode
// generation using github.com/boombuler/barcode. Supported types include
// Code128, Code39, Code93, Codabar, EAN-13, EAN-8, UPC-A, 2-of-5,
// PDF417, DataMatrix, and Aztec Code.
package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/codabar"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/code93"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
	"github.com/boombuler/barcode/twooffive"
)

// BarcodeType identifies the barcode symbology.
type BarcodeType string

const (
	TypeCode128    BarcodeType = "code128"
	TypeCode39     BarcodeType = "code39"
	TypeCode93     BarcodeType = "code93"
	TypeCodabar    BarcodeType = "codabar"
	TypeEAN13      BarcodeType = "ean13"
	TypeEAN8       BarcodeType = "ean8"
	TypeUPCA       BarcodeType = "upca"
	TypeTwoOfFive  BarcodeType = "2of5"
	TypePDF417     BarcodeType = "pdf417"
	TypeDataMatrix BarcodeType = "datamatrix"
	TypeAztec      BarcodeType = "aztec"
)

// ──────────────────────────────────────────────
// Generic generate
// ──────────────────────────────────────────────

// Generate creates a barcode of the specified type from the given content.
// The returned image is the raw barcode at 1 module per pixel; use Scale
// or GenerateScaled for a pixel-sized output. Returns an error for
// invalid content (e.g. wrong length for EAN).
func Generate(typ BarcodeType, content string) (image.Image, error) {
	if content == "" {
		return nil, fmt.Errorf("barcode: content is empty")
	}

	var (
		img barcode.Barcode
		err error
	)

	switch typ {
	case TypeCode128:
		img, err = code128.Encode(content)
	case TypeCode39:
		img, err = code39.Encode(content, false, false)
	case TypeCode93:
		img, err = code93.Encode(content, false, false)
	case TypeCodabar:
		img, err = codabar.Encode(content)
	case TypeEAN13:
		img, err = ean.Encode(content)
	case TypeEAN8:
		img, err = ean.Encode(content)
	case TypeUPCA:
		// UPC-A is a subset of EAN-13 with a leading 0. If the user
		// provides 11 or 12 digits, prepend/normalize to 12 digits
		// and let EAN-13 handle it (12 digits → EAN-13 adds checksum).
		if len(content) == 11 {
			content = "0" + content
		}
		img, err = ean.Encode(content)
	case TypeTwoOfFive:
		img, err = twooffive.Encode(content, false)
	case TypePDF417:
		img, err = pdf417.Encode(content, 8)
	case TypeDataMatrix:
		img, err = datamatrix.Encode(content)
	case TypeAztec:
		img, err = aztec.Encode([]byte(content), aztec.DEFAULT_EC_PERCENT, 0)
	default:
		return nil, fmt.Errorf("barcode: unknown type %q", typ)
	}

	if err != nil {
		return nil, fmt.Errorf("barcode: encode %s: %w", typ, err)
	}
	return img, nil
}

// GenerateScaled creates a barcode and scales it to the given pixel
// dimensions. For 1D barcodes, width should be much larger than height.
func GenerateScaled(typ BarcodeType, content string, width, height int) (image.Image, error) {
	img, err := Generate(typ, content)
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		return img, nil
	}
	bc, ok := img.(barcode.Barcode)
	if !ok {
		return img, nil // can't scale, return as-is
	}
	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, fmt.Errorf("barcode: scale: %w", err)
	}
	return scaled, nil
}

// GeneratePNG creates a barcode and encodes it as PNG bytes.
func GeneratePNG(typ BarcodeType, content string, width, height int) ([]byte, error) {
	img, err := GenerateScaled(typ, content, width, height)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("barcode: png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// Save generates a barcode and writes it to a PNG file.
func Save(typ BarcodeType, content, path string, width, height int) error {
	img, err := GenerateScaled(typ, content, width, height)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("barcode: create %s: %w", path, err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ──────────────────────────────────────────────
// Convenience functions for common types
// ──────────────────────────────────────────────

// Code128 generates a Code128 barcode (supports full ASCII).
func Code128(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeCode128, content, width, height)
}

// Code39 generates a Code39 barcode (uppercase A-Z, 0-9, and - . $ / + % SPACE).
func Code39(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeCode39, content, width, height)
}

// Code93 generates a Code93 barcode (full ASCII via shift characters).
func Code93(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeCode93, content, width, height)
}

// Codabar generates a Codabar barcode (digits and -$:/.+ABCD).
func Codabar(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeCodabar, content, width, height)
}

// EAN13 generates an EAN-13 barcode (12 or 13 digits; checksum is
// auto-calculated if 12 digits are provided).
func EAN13(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeEAN13, content, width, height)
}

// EAN8 generates an EAN-8 barcode (7 or 8 digits; checksum is
// auto-calculated if 7 digits are provided).
func EAN8(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeEAN8, content, width, height)
}

// UPC_A generates a UPC-A barcode (11 or 12 digits; checksum is
// auto-calculated if 11 digits are provided).
func UPC_A(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeUPCA, content, width, height)
}

// TwoOfFive generates an Industrial 2-of-5 barcode (digits only).
func TwoOfFive(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeTwoOfFive, content, width, height)
}

// PDF417 generates a PDF417 stacked barcode (high capacity, binary-safe).
func PDF417(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypePDF417, content, width, height)
}

// DataMatrix generates a DataMatrix 2D barcode.
func DataMatrix(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeDataMatrix, content, width, height)
}

// Aztec generates an Aztec 2D barcode.
func Aztec(content string, width, height int) (image.Image, error) {
	return GenerateScaled(TypeAztec, content, width, height)
}

// ──────────────────────────────────────────────
// Metadata
// ──────────────────────────────────────────────

// Metadata holds information about a generated barcode.
type Metadata struct {
	CodeKind   string // barcode type name (e.g. "Code 128")
	Dimensions byte   // 1 for 1D, 2 for 2D
	Content    string // encoded content
}

// GetMetadata extracts metadata from a generated barcode image. Only
// works on unscaled barcodes (before barcode.Scale is applied).
func GetMetadata(img image.Image) (*Metadata, error) {
	bc, ok := img.(barcode.Barcode)
	if !ok {
		return nil, fmt.Errorf("barcode: image is not a barcode (type %T)", img)
	}
	meta := bc.Metadata()
	return &Metadata{
		CodeKind:   meta.CodeKind,
		Dimensions: meta.Dimensions,
		Content:    bc.Content(),
	}, nil
}
