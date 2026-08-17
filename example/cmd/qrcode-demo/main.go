// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command qrcode-demo demonstrates the ling-base common/qrcode package.
//
// It generates a variety of QR codes (plain, with logo, signed,
// encrypted+signed) and writes them as PNG files to an output directory.
// It also decodes each generated QR code to verify round-trip
// correctness, including signature verification for the anti-counterfeit
// tokens.
//
// Usage:
//
//	go run ./cmd/qrcode-demo
//	go run ./cmd/qrcode-demo -out ./out/qr -logo ./logo.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/ling-base/common/qrcode"
)

func main() {
	outDir := flag.String("out", "out/qr", "output directory for QR code PNGs")
	logoPath := flag.String("logo", "logo.png", "logo image file for branded QR codes")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create out dir: %v\n", err)
		os.Exit(1)
	}

	// Anti-counterfeit keys (in production, load from a secret manager).
	hmacSecret := []byte("ling-base-demo-hmac-secret-2026")
	aesKey := []byte("0123456789abcdef0123456789abcdef") // 32 bytes (AES-256)

	fmt.Println("=== QR Code Demo ===")
	fmt.Printf("Output directory: %s\n\n", *outDir)

	// ──────────────────────────────────────────────
	// 1. Plain QR codes at all error-correction levels
	// ──────────────────────────────────────────────
	fmt.Println("[1] Plain QR codes (all error-correction levels)")
	levels := []struct {
		name  string
		level qrcode.ErrorCorrectionLevel
	}{
		{"L", qrcode.ECLLow},
		{"M", qrcode.ECLMedium},
		{"Q", qrcode.ECLQuartile},
		{"H", qrcode.ECLHigh},
	}
	for _, lv := range levels {
		path := filepath.Join(*outDir, fmt.Sprintf("plain_%s.png", lv.name))
		text := fmt.Sprintf("https://ling-base.dev/qr?level=%s", lv.name)
		if err := qrcode.Save(text, path, lv.level, 400); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", lv.name, err)
			continue
		}
		// Round-trip decode.
		decoded, err := qrcode.DecodeFile(path)
		status := "OK"
		if err != nil || decoded != text {
			status = fmt.Sprintf("DECODE FAIL (got %q, err %v)", decoded, err)
		}
		fmt.Printf("  plain_%s.png  → %s\n", lv.name, status)
	}

	// ──────────────────────────────────────────────
	// 2. QR code with logo (branded)
	// ──────────────────────────────────────────────
	fmt.Println("\n[2] QR code with logo (branded)")
	logoSizes := []struct {
		name string
		size int
	}{
		{"small", 60},
		{"medium", 80},
		{"large", 100},
	}
	for _, ls := range logoSizes {
		path := filepath.Join(*outDir, fmt.Sprintf("logo_%s.png", ls.name))
		text := "https://ling-base.dev/branded"
		if err := qrcode.SaveWithLogoFile(text, path, qrcode.ECLHigh, 400, *logoPath, ls.size); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR logo_%s: %v\n", ls.name, err)
			continue
		}
		fmt.Printf("  logo_%s.png (logo=%dpx) → generated\n", ls.name, ls.size)
	}

	// ──────────────────────────────────────────────
	// 3. WiFi QR code
	// ──────────────────────────────────────────────
	fmt.Println("\n[3] WiFi QR code")
	wifiText := "WIFI:T:WPA;S:LingByte-Guest;P:welcome2026;;"
	wifiPath := filepath.Join(*outDir, "wifi.png")
	if err := qrcode.Save(wifiText, wifiPath, qrcode.ECLMedium, 400); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR wifi: %v\n", err)
	} else {
		decoded, _ := qrcode.DecodeFile(wifiPath)
		fmt.Printf("  wifi.png → %s\n", decoded)
	}

	// ──────────────────────────────────────────────
	// 4. vCard QR code
	// ──────────────────────────────────────────────
	fmt.Println("\n[4] vCard QR code")
	vcard := "BEGIN:VCARD\nVERSION:3.0\nFN:Ling Byte\nORG:LingByte Inc.\nTEL:+86-138-0000-0000\nEMAIL:hello@ling-base.dev\nURL:https://ling-base.dev\nEND:VCARD"
	vcardPath := filepath.Join(*outDir, "vcard.png")
	if err := qrcode.Save(vcard, vcardPath, qrcode.ECLHigh, 400); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR vcard: %v\n", err)
	} else {
		decoded, _ := qrcode.DecodeFile(vcardPath)
		fmt.Printf("  vcard.png → decoded %d chars\n", len(decoded))
	}

	// ──────────────────────────────────────────────
	// 5. Anti-counterfeit: HMAC-signed QR
	// ──────────────────────────────────────────────
	fmt.Println("\n[5] Anti-counterfeit: HMAC-signed QR")
	productID := "SN-2026-0001-Auth"
	signedPath := filepath.Join(*outDir, "signed.png")
	token, err := qrcode.Sign(productID, hmacSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  sign error: %v\n", err)
	} else {
		if err := qrcode.Save(token, signedPath, qrcode.ECLHigh, 400); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\n", err)
		} else {
			// Verify round-trip.
			decoded, _ := qrcode.DecodeFile(signedPath)
			payload, err := qrcode.Verify(decoded, hmacSecret, 24*time.Hour)
			if err != nil {
				fmt.Printf("  signed.png → VERIFY FAIL: %v\n", err)
			} else {
				fmt.Printf("  signed.png → verified payload: %q\n", payload)
			}
		}
	}

	// ──────────────────────────────────────────────
	// 6. Anti-counterfeit: Encrypted QR (confidential)
	// ──────────────────────────────────────────────
	fmt.Println("\n[6] Anti-counterfeit: AES-GCM encrypted QR")
	secretData := "confidential-batch:B-2026-0817;price:999.00"
	encPath := filepath.Join(*outDir, "encrypted.png")
	encToken, err := qrcode.Encrypt(secretData, aesKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  encrypt error: %v\n", err)
	} else {
		if err := qrcode.Save(encToken, encPath, qrcode.ECLHigh, 400); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\n", err)
		} else {
			decoded, _ := qrcode.DecodeFile(encPath)
			plaintext, err := qrcode.Decrypt(decoded, aesKey)
			if err != nil {
				fmt.Printf("  encrypted.png → DECRYPT FAIL: %v\n", err)
			} else {
				fmt.Printf("  encrypted.png → decrypted: %q\n", plaintext)
			}
		}
	}

	// ──────────────────────────────────────────────
	// 7. Anti-counterfeit: Secure QR (encrypt + sign)
	// ──────────────────────────────────────────────
	fmt.Println("\n[7] Anti-counterfeit: Secure QR (encrypt + HMAC sign)")
	secureData := "premium-product:SN-2026-GOLD-0042;warranty:5y"
	securePath := filepath.Join(*outDir, "secure.png")
	secureToken, err := qrcode.Secure(secureData, aesKey, hmacSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  secure error: %v\n", err)
	} else {
		if err := qrcode.Save(secureToken, securePath, qrcode.ECLHigh, 400); err != nil {
			fmt.Fprintf(os.Stderr, "  save error: %v\n", err)
		} else {
			decoded, _ := qrcode.DecodeFile(securePath)
			payload, err := qrcode.Unseal(decoded, aesKey, hmacSecret, 0)
			if err != nil {
				fmt.Printf("  secure.png → UNSEAL FAIL: %v\n", err)
			} else {
				fmt.Printf("  secure.png → unsealed: %q\n", payload)
			}
		}
	}

	// ──────────────────────────────────────────────
	// 8. Secure QR with logo (anti-counterfeit + branded)
	// ──────────────────────────────────────────────
	fmt.Println("\n[8] Secure QR with logo (anti-counterfeit + branded)")
	secureLogoPath := filepath.Join(*outDir, "secure_logo.png")
	secureToken2, _ := qrcode.Secure("luxury-item:SN-2026-DIAMOND-0001", aesKey, hmacSecret)
	if err := qrcode.SaveWithLogoFile(secureToken2, secureLogoPath, qrcode.ECLHigh, 500, *logoPath, 80); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR secure_logo: %v\n", err)
	} else {
		decoded, _ := qrcode.DecodeFile(secureLogoPath)
		if qrcode.IsSecureToken(decoded) {
			payload, err := qrcode.Unseal(decoded, aesKey, hmacSecret, 0)
			if err != nil {
				fmt.Printf("  secure_logo.png → UNSEAL FAIL: %v\n", err)
			} else {
				fmt.Printf("  secure_logo.png → unsealed: %q\n", payload)
			}
		} else {
			fmt.Printf("  secure_logo.png → decoded but not a secure token (logo may obscure data)\n")
		}
	}

	// ──────────────────────────────────────────────
	// 9. Tamper detection demo
	// ──────────────────────────────────────────────
	fmt.Println("\n[9] Tamper detection demo")
	tamperedToken := token[:len(token)-2] + "XX" // corrupt signature
	_, err = qrcode.Verify(tamperedToken, hmacSecret, 0)
	fmt.Printf("  tampered token verification: %v (expected: signature mismatch)\n", err)

	wrongKeyPayload, err := qrcode.Verify(token, []byte("wrong-secret"), 0)
	fmt.Printf("  wrong-key verification: got %q, err %v (expected: mismatch)\n", wrongKeyPayload, err)

	// ──────────────────────────────────────────────
	// 10. Fancy QR codes (花式二维码)
	// ──────────────────────────────────────────────
	fmt.Println("\n[10] Fancy QR codes (花式二维码)")

	// 10a. Circle modules with blue foreground
	circlePath := filepath.Join(*outDir, "fancy_circle_blue.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/circle",
		circlePath, qrcode.ECLHigh,
		qrcode.FancyCirclePreset(
			color.RGBA{R: 30, G: 100, B: 220, A: 255}, // blue
			color.White,
		))
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_circle_blue: %v\n", err)
	} else {
		fmt.Println("  fancy_circle_blue.png  → circle modules, blue fg")
	}

	// 10b. Circle modules with gradient (red → purple → blue)
	grad := qrcode.NewLinearGradient(135,
		qrcode.ColorStop{Color: color.RGBA{255, 50, 50, 255}, T: 0.0},
		qrcode.ColorStop{Color: color.RGBA{180, 50, 200, 255}, T: 0.5},
		qrcode.ColorStop{Color: color.RGBA{50, 100, 255, 255}, T: 1.0},
	)
	gradPath := filepath.Join(*outDir, "fancy_gradient.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/gradient",
		gradPath, qrcode.ECLHigh,
		qrcode.FancyGradientPreset(grad))
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_gradient: %v\n", err)
	} else {
		fmt.Println("  fancy_gradient.png     → circle modules, red→purple→blue gradient")
	}

	// 10c. Rounded modules with rounded finder
	roundedPath := filepath.Join(*outDir, "fancy_rounded.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/rounded",
		roundedPath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:      qrcode.ShapeRounded,
			Finder:      qrcode.FinderRounded,
			FgColor:     color.RGBA{R: 20, G: 20, B: 20, A: 255},
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_rounded: %v\n", err)
	} else {
		fmt.Println("  fancy_rounded.png      → rounded modules + rounded finder")
	}

	// 10d. Liquid (blob) style
	liquidPath := filepath.Join(*outDir, "fancy_liquid.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/liquid",
		liquidPath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:      qrcode.ShapeLiquid,
			Finder:      qrcode.FinderRounded,
			FgColor:     color.RGBA{R: 0, G: 180, B: 120, A: 255}, // teal
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_liquid: %v\n", err)
	} else {
		fmt.Println("  fancy_liquid.png       → liquid blob style, teal")
	}

	// 10e. Diamond modules
	diamondPath := filepath.Join(*outDir, "fancy_diamond.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/diamond",
		diamondPath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:      qrcode.ShapeDiamond,
			Finder:      qrcode.FinderSquare,
			FgColor:     color.RGBA{R: 200, G: 50, B: 50, A: 255}, // red
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_diamond: %v\n", err)
	} else {
		fmt.Println("  fancy_diamond.png      → diamond modules, red")
	}

	// 10f. Horizontal stripe modules
	hstripePath := filepath.Join(*outDir, "fancy_hstripe.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/hstripe",
		hstripePath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:      qrcode.ShapeHStripe,
			FgColor:     color.RGBA{R: 80, G: 80, B: 80, A: 255},
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_hstripe: %v\n", err)
	} else {
		fmt.Println("  fancy_hstripe.png      → horizontal stripe modules")
	}

	// 10g. Vertical stripe modules
	vstripePath := filepath.Join(*outDir, "fancy_vstripe.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/vstripe",
		vstripePath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:      qrcode.ShapeVStripe,
			FgColor:     color.RGBA{R: 60, G: 60, B: 60, A: 255},
			BgColor:     color.White,
			ModuleWidth: 21,
			BorderWidth: 20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_vstripe: %v\n", err)
	} else {
		fmt.Println("  fancy_vstripe.png      → vertical stripe modules")
	}

	// 10h. Fancy QR with logo (circle modules + rounded finder + logo)
	if _, lerr := os.Stat(*logoPath); lerr == nil {
		logoImg, _, derr := image.Decode(mustOpen(*logoPath))
		if derr == nil {
			fancyLogoPath := filepath.Join(*outDir, "fancy_logo.png")
			err = qrcode.SaveFancy("https://ling-base.dev/fancy/logo",
				fancyLogoPath, qrcode.ECLHigh,
				qrcode.FancyLogoPreset(logoImg))
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR fancy_logo: %v\n", err)
			} else {
				fmt.Println("  fancy_logo.png         → circle + rounded finder + logo")
			}
		}
	}

	// 10i. Halftone QR (using demo.png as the source image)
	if _, herr := os.Stat("demo.png"); herr == nil {
		srcImg, _, derr := image.Decode(mustOpen("demo.png"))
		if derr == nil {
			halftonePath := filepath.Join(*outDir, "fancy_halftone.png")
			err = qrcode.SaveFancy("https://ling-base.dev/fancy/halftone",
				halftonePath, qrcode.ECLHigh,
				qrcode.FancyHalftonePreset(srcImg, color.Black))
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ERROR fancy_halftone: %v\n", err)
			} else {
				fmt.Println("  fancy_halftone.png     → halftone QR from demo.png")
			}
		}
	}

	// 10j. Transparent background
	transparentPath := filepath.Join(*outDir, "fancy_transparent.png")
	err = qrcode.SaveFancy("https://ling-base.dev/fancy/transparent",
		transparentPath, qrcode.ECLHigh, qrcode.FancyOptions{
			Module:        qrcode.ShapeCircle,
			FgColor:       color.RGBA{R: 200, G: 50, B: 100, A: 255}, // pink
			BgTransparent: true,
			ModuleWidth:   21,
			BorderWidth:   20,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR fancy_transparent: %v\n", err)
	} else {
		fmt.Println("  fancy_transparent.png  → circle modules, transparent bg, pink fg")
	}

	fmt.Printf("\n=== Done! All QR codes saved to %s/ ===\n", *outDir)
}


// mustOpen opens a file for reading, panicking on error. Used for
// loading logo/source images in the demo.
func mustOpen(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	return f
}
