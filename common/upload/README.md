# upload

File upload utilities for HTTP servers, using only the Go standard library.

## Features

- `Handler` -- single and multiple file uploads from multipart requests
- MIME-type validation (extension-based pre-check + content-based detection)
- Safe filename generation (path-traversal prevention, unique names)
- Configurable max file size, allowed types, and custom filename generators
- `ChunkUploader` -- chunked (resumable) uploads with save, merge, cleanup, and status

## Key types

- `Handler` -- upload processor with `MaxSize`, `AllowedTypes`, `DestDir`
- `HandlerOption` -- functional option for `Handler`
- `FileInfo` -- saved file metadata (`OriginalName`, `SavedName`, `Path`, `Size`, `MIMEType`, `Extension`)
- `ChunkUploader` -- chunked upload manager

## Key functions

- `NewHandler(destDir, opts...)` -- create a handler
- `WithMaxSize(n)`, `WithAllowedTypes(types)`, `WithFilenameGenerator(fn)` -- options
- `(*Handler) Save(file, header)`, `SaveFromRequest(r, field)`, `SaveMultipleFromRequest(r, field)`
- `ValidateFile(header, allowedTypes, maxSize)` -- validate a file header
- `GenerateSafeFilename(original)` -- path-traversal-safe filename
- `IsAllowedType(mimeType, allowed)`, `DetectMIME(data)`
- `NewChunkUploader(destDir, chunkSize)`
- `(*ChunkUploader) SaveChunk(uploadID, index, r)`, `Merge(uploadID, total, filename)`, `Cleanup(uploadID)`, `Status(uploadID)`

## Quick start

```go
import "github.com/LingByte/ling-base/common/upload"

h := upload.NewHandler("/var/uploads",
    upload.WithMaxSize(10<<20),
    upload.WithAllowedTypes([]string{"image/png", "image/jpeg"}),
)

// Single file
info, err := h.SaveFromRequest(r, "file")
if err != nil { log.Fatal(err) }
log.Println(info.Path, info.MIMEType)

// Multiple files
infos, err := h.SaveMultipleFromRequest(r, "files")

// Chunked upload
cu := upload.NewChunkUploader("/var/uploads", 1<<20)
cu.SaveChunk("upload-123", 0, chunkReader)
cu.SaveChunk("upload-123", 1, chunkReader)
info, err := cu.Merge("upload-123", 2, "video.mp4")
defer cu.Cleanup("upload-123")
```

## License

MIT
