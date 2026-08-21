# barcode

One-dimensional and two-dimensional barcode generation using github.com/boombuler/barcode.

## Supported types

Code128, Code39, Code93, Codabar, EAN-13, EAN-8, UPC-A, 2-of-5, PDF417, DataMatrix, Aztec.

## Key types

- `BarcodeType` -- symbology identifier (`TypeCode128`, `TypeEAN13`, `TypePDF417`, ...)
- `Metadata` -- barcode metadata (`CodeKind`, `Dimensions`, `Content`)

## Key functions

- `Generate(typ, content)` -- raw barcode image (1 module per pixel)
- `GenerateScaled(typ, content, width, height)` -- scaled to pixel dimensions
- `GeneratePNG(typ, content, width, height)` -- PNG-encoded bytes
- `Save(typ, content, path, width, height)` -- write PNG to file
- Convenience wrappers: `Code128`, `EAN13`, `PDF417`, `DataMatrix`, `Aztec`, ...
- `GetMetadata(img)` -- extract metadata from an unscaled barcode

## Quick start

```go
import "github.com/LingByte/ling-base/common/barcode"

// Generate a Code128 barcode as PNG bytes
pngBytes, err := barcode.GeneratePNG(barcode.TypeCode128, "HELLO123", 300, 80)
if err != nil {
    log.Fatal(err)
}
_ = os.WriteFile("barcode.png", pngBytes, 0644)
```

## License

MIT
