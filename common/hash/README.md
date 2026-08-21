# hash

Hashing utilities: standard cryptographic hashes, HMAC, CRC, MurmurHash3, xxHash, and file/stream helpers.

## Features

- Standard hashes: MD5, SHA-1, SHA-256, SHA-512
- HMAC: HMAC-SHA256, HMAC-SHA512 with constant-time comparison
- CRC: CRC16-CCITT, CRC32 (IEEE), CRC64 (ISO), Adler-32, FNV-1a
- Non-cryptographic: MurmurHash3 (32-bit and 128-bit x64), xxHash64, xxHash32
- File and stream hashing helpers

## Key functions

- `MD5` / `MD5Hex` / `MD5String`, `SHA1` / `SHA1Hex`, `SHA256` / `SHA256Hex`, `SHA512` / `SHA512Hex`
- `HMACSHA256` / `HMACSHA256Hex`, `HMACSHA512` / `HMACSHA512Hex`, `HMACEqual`
- `CRC16` / `CRC16Hex`, `CRC32` / `CRC32Hex`, `CRC64` / `CRC64Hex`
- `Adler32`, `FNV1a32`, `FNV1a64`
- `Murmur3_32` / `Murmur3_32Hex`, `Murmur3_128` / `Murmur3_128Hex`
- `XXHash64` / `XXHash64Hex`, `XXHash32` / `XXHash32Hex`
- `MD5File(path)`, `SHA256File(path)` -- hash file contents

## Quick start

```go
import "github.com/LingByte/ling-base/common/hash"

hash.MD5Hex([]byte("hello"))              // "5d41402abc4b2a76b9719d911017c592"
hash.SHA256Hex([]byte("hello"))
hash.HMACSHA256Hex([]byte("data"), key)
hash.Murmur3_32Hex([]byte("hello"), 0)
hash.XXHash64Hex([]byte("hello"), 0)
```

## License

MIT
