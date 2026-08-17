// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"os"
)

// savePNG writes an image.Image to a PNG file.
func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// simpleHMAC computes a hex-encoded HMAC-SHA256 of the given message.
// This is used for the barcode anti-counterfeit demo where the HMAC is
// appended to the product code and encoded in a 1D barcode.
func simpleHMAC(message string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))[:16] // first 16 hex chars for compactness
}
