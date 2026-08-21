# random

Cryptographically secure random utilities built on `crypto/rand`.

## Features

- Secure random numbers: `Int`, `Intn`, `IntRange`, `Float64`, `Bytes`
- Random strings with custom charsets: `String`, `StringWithCharset`
- Sampling and shuffling: `Shuffle`, `Sample`, `Choice`, `Permutation`
- Random colors: `HexColor`, `RGBColor`, `HSLColor`
- UUID v4 (RFC 4122)

All functions use `crypto/rand` and are safe for concurrent use.

## Character sets

`CharsetAlphaLower`, `CharsetAlphaUpper`, `CharsetAlpha`, `CharsetNumeric`, `CharsetAlphaNum`, `CharsetHexLower`, `CharsetHexUpper`, `CharsetBase62`, `CharsetBase64URL`, `CharsetSymbol`.

## Quick start

```go
import "github.com/LingByte/ling-base/common/random"

n := random.IntRange(1, 100)
s := random.String(16, random.CharsetAlphaNum)
b := random.Bytes(32)
id := random.UUID() // e.g. "550e8400-e29b-41d4-a716-446655440000"
```
