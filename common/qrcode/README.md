# qrcode

QR code generation (with optional logo overlay and styled modules), decoding, and anti-counterfeiting signing/encryption.

## Features

- **Generation**: `Generate`, `GeneratePNG`, `Save` with configurable error-correction level
- **Logo overlay**: composite a logo onto the QR center with a safe zone
- **Fancy/styled**: custom module shapes (circle, rounded, liquid, stripes, diamond), gradients, transparent background, halftone QR
- **Templates**: named style presets (`simple` / `classic` / `creative` / `custom`) via `GenerateFromTemplate`
- **Decoding**: `Decode`, `DecodeFile`, `DecodeBytes` via ZXing (gozxing)
- **Anti-counterfeiting**: `Sign`/`Verify` (HMAC), `Encrypt`/`Decrypt` (AES-GCM), `Secure`/`Unseal` (encrypt + sign)

## Key types

- `ErrorCorrectionLevel` — `ECLLow`, `ECLMedium`, `ECLQuartile`, `ECLHigh`
- `ModuleShape` — styled module shapes for fancy QR codes
- `Template` / `TemplateCategory` — gallery presets for fancy QR

## Quick start

```go
import "github.com/LingByte/ling-base/common/qrcode"

pngBytes, _ := qrcode.GeneratePNG("https://example.com", qrcode.ECLMedium, 256)
_ = qrcode.Save("https://example.com", "qr.png", qrcode.ECLHigh, 400)

// Named style template (e.g. simple-dots, classic-ocean, creative-neon).
fancy, _ := qrcode.GenerateFromTemplate("https://example.com", "classic-ocean")
_ = fancy

// List gallery tabs: simple | classic | creative | custom
for _, t := range qrcode.ListTemplates(qrcode.CategorySimple) {
    _ = t.ID // "simple-dots", ...
}

text, _ := qrcode.DecodeFile("qr.png")

// Anti-counterfeit: sign then verify.
token, _ := qrcode.Sign("product-id:12345", secretKey)
payload, err := qrcode.Verify(token, secretKey, 0)
```

## Template scope

Built-in templates cover **parametric** styling (module shape, finder shape, solid/gradient colors) — the same class as product UIs under 黑白简约 / 经典绚丽 / 创意样式.

**AI artistic QR** (ControlNet / Stable Diffusion scenes that paint the code into photos) is **out of scope** for this package: that needs an external image model. You can still feed a stylized bitmap into `FancyOptions.Halftone` or overlay a logo after generation.
