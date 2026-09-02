# backup

A pluggable backup/restore manager that copies data from a `Source` to a
`Destination`, computing a SHA-256 checksum and optionally applying gzip
compression. File-based source/destination implementations are included.

## Design

Both the backup source and destination are abstracted as interfaces so
that file systems, object stores, databases, or any other backend can be
plugged in:

```go
type Source interface {
    Read() (io.ReadCloser, error)
}
type Destination interface {
    Write(name string, r io.Reader) (string, error)
}
```

File-based implementations are provided:

- `FileSource{Path}` — reads from a local file
- `FileDestination{Dir}` — writes artifacts into a local directory

## Features

- SHA-256 checksum of the stored (possibly compressed) bytes
- Optional gzip compression via `WithCompression()`
- In-memory registry of completed backups (`List`, `Delete`)
- Single-pass streaming: data is compressed, checksummed, and written
  concurrently without buffering the whole stream

## Quick start

```go
import "github.com/LingByte/ling-base/common/backup"

mgr := backup.NewManager(
    &backup.FileSource{Path: "/var/data/app.db"},
    &backup.FileDestination{Dir: "/var/backups"},
    backup.WithCompression(),
)

bp, err := mgr.Backup(ctx, "app-2026-01-01")
if err != nil { /* ... */ }
fmt.Printf("backed up to %s (%d bytes, sha256=%s)\n", bp.Path, bp.Size, bp.Checksum)

// Later, restore it
err = mgr.Restore(ctx, bp)

// List / delete
backups, _ := mgr.List()
_ = mgr.Delete("app-2026-01-01")
```

## API

### Manager

| Method | Description |
| --- | --- |
| `NewManager(src, dst, opts...)` | Create a manager |
| `Backup(ctx, name)` | Back up source → destination |
| `Restore(ctx, *Backup)` | Restore an artifact back to the source |
| `List()` | List recorded backups (sorted by name) |
| `Delete(name)` | Delete a backup record + artifact |

### Options

| Option | Description |
| --- | --- |
| `WithCompression()` | Enable gzip compression |

### Backup fields

`Name`, `Timestamp`, `Size`, `Checksum` (SHA-256 hex), `Path`, `Compressed`.

## Checksum semantics

The `Checksum` is the SHA-256 of the **stored** bytes — i.e. after
compression when `WithCompression()` is enabled. This lets receivers
verify the artifact on disk directly without decompressing first.
