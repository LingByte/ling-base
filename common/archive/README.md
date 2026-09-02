# archive

Archive creation and extraction utilities for zip, tar, and tar.gz formats,
plus single-file gzip compression. Uses only the Go standard library.

## Features

- `Archiver` interface with `ZipArchiver`, `TarArchiver`, `TarGzArchiver`
- Convenience functions: `Zip/Unzip`, `Tar/Untar`, `TarGz/UntarGz`
- Single-file `CompressFile`/`DecompressFile` (gzip)
- `DetectFormat` auto-detects zip / tar / tar.gz / gz by magic bytes
- `ListArchive` lists entries in any supported format
- Path-traversal (zip-slip / tar-slip) protection on every extraction

## Key functions

- `(*ZipArchiver) Archive/Unarchive`, `(*TarArchiver) ...`, `(*TarGzArchiver) ...`
- `Zip/Unzip`, `Tar/Untar`, `TarGz/UntarGz`
- `CompressFile/DecompressFile`
- `DetectFormat(path)`, `ListArchive(path)`

## Quick start

```go
import "github.com/LingByte/ling-base/common/archive"

archive.Zip("mydir", "mydir.zip")
archive.Unzip("mydir.zip", "out")

archive.TarGz("mydir", "mydir.tar.gz")
archive.UntarGz("mydir.tar.gz", "out")

format, _ := archive.DetectFormat("mydir.zip") // "zip"
names, _ := archive.ListArchive("mydir.zip")
```

## License

MIT
