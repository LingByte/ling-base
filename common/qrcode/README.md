# qrcode

QR code generation (with optional logo overlay and styled modules), decoding, and anti-counterfeiting signing/encryption.

## Features

- **Generation**: `Generate`, `GeneratePNG`, `Save` with configurable error-correction level
- **Logo overlay**: composite a logo onto the QR center with a safe zone
- **Fancy/styled**: custom module shapes (circle, rounded, liquid, stripes, diamond), gradients, transparent background, halftone QR
- **Decoding**: `Decode`, `DecodeFile`, `DecodeBytes` via ZXing (gozxing)
- **Anti-counterfeiting**: `Sign`/`Verify` (HMAC), `Encrypt`/`Decrypt` (AES-GCM), `Secure`/`Unseal` (encrypt + sign)

## Key types

- `ErrorCorrectionLevel` — `ECLLow`, `ECLMedium`, `ECLQuartile`, `ECLHigh`
- `ModuleShape` — styled module shapes for fancy QR codes

## Quick start

```go
import "github.com/LingByte/ling-base/common/qrcode"

pngBytes, _ := qrcode.GeneratePNG("https://example.com", qrcode.ECLMedium, 256)
_ = qrcode.Save("https://example.com", "qr.png", qrcode.ECLHigh, 400)

text, _ := qrcode.DecodeFile("qr.png")

// Anti-counterfeit: sign then verify.
token, _ := qrcode.Sign("product-id:12345", secretKey)
payload, err := qrcode.Verify(token, secretKey, 0)
```
