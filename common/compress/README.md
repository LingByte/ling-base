# compress

Compression and decompression utilities supporting multiple algorithms.

## Supported algorithms

Gzip, Zlib, Flate/Deflate, Zstd, Snappy (block format), LZ4 (frame format).

## Key types

- `Algorithm` -- algorithm identifier (`AlgGzip`, `AlgZstd`, `AlgSnappy`, `AlgLZ4`, ...)
- Compression level constants: `LevelNone`, `LevelBest`, `LevelFastest`, `LevelDefault`

## Key functions

- `GzipCompress(data, level)` / `GzipDecompress(data)`
- `ZlibCompress(data, level)` / `ZlibDecompress(data)`
- `FlateCompress(data, level)` / `FlateDecompress(data)`
- `ZstdCompress(data)` / `ZstdDecompress(data)`
- `SnappyCompress(data)` / `SnappyDecompress(data)`
- `LZ4Compress(data)` / `LZ4Decompress(data)`
- `NewGzipWriter(w, level)` / `NewGzipReader(r)` -- streaming helpers

## Quick start

```go
import "github.com/LingByte/ling-base/common/compress"

data := []byte("hello world hello world hello world")

compressed, err := compress.GzipCompress(data, compress.LevelBest)
if err != nil {
    log.Fatal(err)
}

decompressed, err := compress.GzipDecompress(compressed)
if err != nil {
    log.Fatal(err)
}
```

## License

MIT
