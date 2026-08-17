// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command barcode-demo demonstrates the ling-base common/barcode package.
//
// It generates one PNG file for each supported barcode symbology and
// writes them to an output directory. It also prints metadata for each
// generated barcode.
//
// Usage:
//
//	go run ./cmd/barcode-demo
//	go run ./cmd/barcode-demo -out ./out/barcode
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LingByte/ling-base/common/barcode"
)

func main() {
	outDir := flag.String("out", "out/barcode", "output directory for barcode PNGs")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create out dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Barcode Demo ===")
	fmt.Printf("Output directory: %s\n\n", *outDir)

	// Each entry: type, content, filename, width, height.
	demos := []struct {
		name    string
		typ     barcode.BarcodeType
		content string
		w, h    int
	}{
		{"Code128", barcode.TypeCode128, "LingBase-2026-PROD-0001", 400, 120},
		{"Code39", barcode.TypeCode39, "LINGBASE2026", 400, 120},
		{"Code93", barcode.TypeCode93, "LINGBASE93", 400, 120},
		{"Codabar", barcode.TypeCodabar, "A12345678B", 400, 120},
		{"EAN13", barcode.TypeEAN13, "590123412345", 300, 150},
		{"EAN8", barcode.TypeEAN8, "9638501", 250, 150},
		{"UPC-A", barcode.TypeUPCA, "36000291451", 300, 150},
		{"2of5", barcode.TypeTwoOfFive, "1234567890", 400, 120},
		{"PDF417", barcode.TypePDF417, "PDF417: LingBase anti-counterfeit label with longer data payload", 500, 200},
		{"DataMatrix", barcode.TypeDataMatrix, "DM:LingBase-2026", 300, 300},
		{"Aztec", barcode.TypeAztec, "Aztec:LingBase-2026", 300, 300},
	}

	for _, d := range demos {
		path := filepath.Join(*outDir, fmt.Sprintf("%s.png", d.name))

		img, err := barcode.GenerateScaled(d.typ, d.content, d.w, d.h)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", d.name, err)
			continue
		}

		if err := savePNG(path, img); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR save %s: %v\n", d.name, err)
			continue
		}

		// Try to get metadata (only works on unscaled barcodes).
		rawImg, _ := barcode.Generate(d.typ, d.content)
		meta, _ := barcode.GetMetadata(rawImg)

		info := ""
		if meta != nil {
			info = fmt.Sprintf(" [kind=%s, dim=%d, content=%q]",
				meta.CodeKind, meta.Dimensions, meta.Content)
		}
		fmt.Printf("  %-12s %dx%d%s\n", d.name+".png", d.w, d.h, info)
	}

	// ──────────────────────────────────────────────
	// Anti-counterfeit: signed Code128
	// ──────────────────────────────────────────────
	fmt.Println("\n[Anti-counterfeit] Signed Code128 (HMAC embedded in content)")

	// In a real anti-counterfeit scenario, the barcode content itself
	// can carry a signed token (similar to the QR code security module).
	// Here we demonstrate a simple HMAC-appended Code128.
	hmacSecret := []byte("ling-base-barcode-secret")
	productCode := "LB-PROD-2026-0001"
	signedContent := productCode + "|" + simpleHMAC(productCode, hmacSecret)

	signedPath := filepath.Join(*outDir, "Code128_signed.png")
	img, err := barcode.GenerateScaled(barcode.TypeCode128, signedContent, 600, 120)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR signed: %v\n", err)
	} else {
		savePNG(signedPath, img)
		fmt.Printf("  Code128_signed.png  content=%s\n", signedContent)
		fmt.Printf("  → Scan and verify: split on '|', recompute HMAC, compare.\n")
	}

	fmt.Printf("\n=== Done! All barcodes saved to %s/ ===\n", *outDir)
}
