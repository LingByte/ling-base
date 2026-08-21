# imageutil

Image processing utilities supporting decode/encode, resize, crop, rotate, flip, filters, effects, watermark, composite, and histogram analysis.

## Features

- Decode/Encode: JPEG, PNG, GIF, BMP, TIFF, WebP (decode only)
- Resize: nearest-neighbor, bilinear, CatmullRom, Lanczos, fit-with-padding
- Crop: rectangular, center, smart, aspect-ratio (9-grid anchor), circular, rounded corners
- Rotate 90/180/270, flip horizontal/vertical
- Adjust: brightness, contrast, gamma, saturation, hue, color temperature, tint
- Filters: box blur, gaussian blur, sharpen, edge detect (Sobel), emboss
- Effects: invert, sepia, posterize, threshold
- Watermark: image overlay (center/bottom-right/tiled) and text watermark
- Composite: blend modes, border, padding
- Info: dimensions, format detection, histogram and statistics

## Key types

- `Format` -- image format (`FormatJPEG`, `FormatPNG`, `FormatGIF`, `FormatBMP`, `FormatTIFF`, `FormatWebP`)

## Key functions

- `DecodeFile(path)`, `Decode(r)`, `FromBytes(data)` -- load images
- `SaveJPEG`, `SavePNG`, `SaveByExtension` -- write images
- `ResizeBilinear`, `ResizeLanczos`, `ResizeNearest`, `Thumbnail`, `Fit`
- `Crop`, `CropCenter`, `CropSmart`, `CropToRatio`, `CircleCrop`, `RoundCorners`
- `Rotate90`, `FlipH`, `FlipV`, `Grayscale`
- `GaussianBlur`, `Sharpen`, `EdgeDetect`, `Invert`, `Sepia`
- `AddWatermark`, `AddTextWatermark`
- `FormatFromExtension(ext)`, `ImageInfo(img)`

## Quick start

```go
import (
    "image"
    "github.com/LingByte/ling-base/common/imageutil"
)

img, _, err := imageutil.DecodeFile("input.png")
if err != nil {
    log.Fatal(err)
}
resized := imageutil.ResizeLanczos(img, 200, 0) // width=200, auto height
rounded := imageutil.RoundCorners(resized, 24)
_ = imageutil.SaveJPEG(rounded, "output.jpg", 85)
```

## License

MIT
