# captcha

A unified CAPTCHA package with multiple challenge types. All comments and
documentation are in English. The package follows the same multi-module
architecture as `cache/` and `lock/` in `ling-base`.

## Supported challenge types

| Type | Constant | Description | User input |
|---|---|---|---|
| Image | `TypeImage` | Distorted text rendered as a PNG | Text code (case-insensitive) |
| Click | `TypeClick` | Click target characters in order | Ordered list of (x, y) points |
| Slider | `TypeSlider` | Drag slider to the end of the track | X offset (int) |
| Math | `TypeMath` | Arithmetic problem ("3 + 5 = ?") | Numeric answer (int) |
| Jigsaw | `TypeJigsaw` | Drag puzzle piece to fit the cutout | X offset (int) |
| Rotate | `TypeRotate` | Rotate image back to upright | Rotation angle (degrees) |

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/LingByte/ling-base/captcha"
)

func main() {
    // Initialize the global manager with defaults.
    captcha.InitGlobalManager(captcha.DefaultConfig())

    // Generate a random challenge (picks one of the six types).
    result, err := captcha.EnsureGlobalManager().GenerateRandom()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Captcha ID: %s, Type: %s\n", result.ID, result.Type)

    // Verify the user's answer (consumes the captcha on success).
    ok, err := captcha.VerifyPayload(captcha.Payload{
        ID:    result.ID,
        Type:  result.Type,
        Value: /* user's answer */,
    })
    fmt.Println("Valid:", ok, "Error:", err)
}
```

## Individual challenge types

### Image captcha

Renders a random alphanumeric code (excluding easily confused characters like
0/O/I/1/l) onto a PNG with interference lines and dots. The user types the
code they see.

```go
ic := captcha.NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
result, _ := ic.Generate()
// result.Data["image"] is a base64 PNG data URL.
// result.Data["length"] is the code length (4).

ok, _ := ic.Verify(result.ID, "ABCD")  // case-insensitive
```

### Click captcha

Displays a set of characters on a background image. The user must click the
target characters in the specified order. Includes decoy characters to
increase difficulty.

```go
cc := captcha.NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
result, _ := cc.Generate()
// result.Data["targets"] is the ordered list of target characters.
// result.Data["chars"] is all character positions (targets + decoys).
// result.Data["background"] is a base64 PNG data URL.
// result.Data["tolerance"] is the click tolerance in pixels.

// User clicks in order; each click is a Point{X, Y}.
ok, _ := cc.Verify(result.ID, []captcha.Point{
    {X: 50, Y: 80},
    {X: 120, Y: 90},
    {X: 200, Y: 75},
})
```

### Slider captcha

The user drags a slider to the end of a track. Verification passes when the
drag distance exceeds `passRatio * trackWidth` and does not exceed
`trackWidth`.

```go
sc := captcha.NewSliderCaptcha(300, 0.92, 5*time.Minute, nil)
result, _ := sc.Generate()
// result.Data["trackWidth"] is the track width in pixels.

ok, _ := sc.Verify(result.ID, 290)  // must be >= 0.92 * 300 = 276
```

### Math captcha

Generates a random arithmetic problem using addition, subtraction, or
multiplication. Results are always non-negative. The user submits the numeric
answer.

```go
mc := captcha.NewMathCaptcha(5*time.Minute, nil)
result, _ := mc.Generate()
// result.Data["question"] is e.g. "3 + 5 = ?"

ok, _ := mc.Verify(result.ID, 8)
// or: ok, _ := mc.VerifyString(result.ID, "8")
```

### Jigsaw captcha

A slider puzzle: a piece is cut from the background image and displayed
separately. The user drags the piece horizontally to fill the gap. The
server stores the target X position and checks it within a tolerance margin.

```go
jc := captcha.NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, nil)
result, _ := jc.Generate()
// result.Data["background"] is the background with the cutout masked.
// result.Data["piece"] is the puzzle piece image.
// result.Data["pieceSize"] is the piece size in pixels.
// result.Data["tolerance"] is the X tolerance in pixels.

ok, _ := jc.Verify(result.ID, 180)  // user's drag X position
```

### Rotate captcha

An image is rotated by a random angle. The user must rotate it back to
upright. The server stores the rotation angle and checks that the residual
(storedAngle - userAngle) mod 360 is within tolerance of 0 degrees.

```go
rc := captcha.NewRotateCaptcha(200, 15, 5*time.Minute, nil)
result, _ := rc.Generate()
// result.Data["image"] is the rotated image as a base64 PNG.
// result.Data["size"] is the image dimensions (square).
// result.Data["tolerance"] is the angular tolerance in degrees.

ok, _ := rc.Verify(result.ID, 90)  // user's rotation angle
```

## Configuration

The `Config` struct controls all challenge types at once:

```go
cfg := &captcha.Config{
    ImageWidth:       200,
    ImageHeight:      60,
    ImageLength:      4,
    ClickWidth:       300,
    ClickHeight:      200,
    ClickCount:       3,
    ClickTolerance:   30,
    SliderTrackWidth: 300,
    SliderPassRatio:  0.92,
    JigsawWidth:      300,
    JigsawHeight:     150,
    JigsawPieceSize:  40,
    JigsawTolerance:  5,
    RotateSize:       200,
    RotateTolerance:  15,
    Expiration:       5 * time.Minute,
    Store:            captcha.NewMemoryStore(),
}
m := captcha.NewManager(cfg)
```

Use `captcha.DefaultConfig()` for sensible defaults.

## Store interface

All challenge types store their state through the `Store` interface:

```go
type Store interface {
    Set(id string, data interface{}, expires time.Time) error
    Get(id string) (interface{}, error)
    Delete(id string) error
    VerifyWithFunc(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error)
    VerifyWithFuncWithoutDelete(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error)
}
```

`MemoryStore` is the default in-process implementation. For distributed
deployments, implement `Store` with Redis or another shared backend.

`VerifyWithFunc` consumes the captcha on success (one-shot verification).
`VerifyWithFuncWithoutDelete` checks without removing (for pre-verification
flows where the captcha should remain valid).

## Manager API

```go
// Generate a specific type.
result, err := m.Generate(captcha.TypeImage)

// Generate a random type.
result, err := m.GenerateRandom()

// Verify a payload (consumes on success).
ok, err := m.Verify(captcha.Payload{ID: id, Type: captcha.TypeMath, Value: 42})

// Validate via the global manager (convenience for HTTP handlers).
err := captcha.ValidatePayload(id, typeStr, value)
```

## Implementation notes

### Image captcha

- Uses `golang.org/x/image/font/gofont/goregular` for font rendering via
  `freetype`.
- Background is filled with a light color, then 5 interference lines and 50
  interference dots are drawn.
- Each character is rendered with a random dark color and slight position
  jitter.
- The code charset excludes `0`, `O`, `I`, `1`, `l` to avoid ambiguity.
- Verification is case-insensitive.

### Click captcha

- Generates `count + decoys` unique characters from a mixed pool of words and
  alphanumeric characters.
- Characters are placed at non-overlapping random positions on a gradient
  background.
- The first `count` characters (before shuffle) are the targets; the user
  must click them in order.
- Verification uses Euclidean distance with a squared tolerance check.

### Slider captcha

- Stores the track width and checks that the user's drag X is within
  `[passRatio * trackWidth, trackWidth]`.
- `passRatio` defaults to 0.92 (the slider must reach at least 92% of the
  track).

### Math captcha

- Randomly selects `+`, `-`, or `x` operations.
- Operands are in `[1, 20]` for addition/subtraction.
- Subtraction results are always non-negative (operands are swapped if
  needed).
- Multiplication operands are capped at 10 to keep answers manageable.
- `VerifyString` parses the input as an integer and delegates to `Verify`.

### Jigsaw captcha

- Generates a gradient background with interference lines.
- A square piece is cut from the right half of the image at a random X
  position.
- The cutout area in the background is darkened to show the gap.
- The piece image has a white border for visibility.
- Verification checks `abs(userX - targetX) <= tolerance`.

### Rotate captcha

- Generates a circular gradient image with a red arrow at the top so the user
  can identify the correct orientation.
- The image is rotated by a random angle (0-359 degrees) using inverse-pixel
  rotation.
- Verification computes `residual = (storedAngle - userAngle) mod 360` and
  accepts if `residual <= tolerance` or `residual >= 360 - tolerance` (to
  handle wrap-around at 0/360).

## Testing

```bash
go test ./... -v
go test ./... -cover
```

Current test coverage: 95.7% of statements. Tests cover all challenge types,
error paths, store operations, type mismatches, and edge cases (tolerance
boundaries, wrap-around, default values).
