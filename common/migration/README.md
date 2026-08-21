# migration

Database schema migration framework with versioned SQL files, up/down directions, and pluggable migrator backends.

## Features

- Versioned SQL migrations with `.up.sql` / `.down.sql` file pairs
- Sources: embedded filesystem (`EmbedSource`), OS directory (`FileSource`), programmatic (`StaticSource`)
- Migrator interface for database-specific execution (e.g. `gormmigrator` subpackage)
- Up, down, step, and version-targeted migration
- Status inspection (applied, pending, current version)

## Key types

- `Migration` -- single migration (Version, Description, UpSQL, DownSQL)
- `MigrationStatus` -- migration with applied state and timestamp
- `Source` -- interface providing migration files
- `EmbedSource` / `FileSource` / `StaticSource` -- source implementations
- `Migrator` -- interface for executing migrations (Up, Down, Step, Version, Status)

## Key functions

- `NewEmbedSource(fsys, root)` -- source from `go:embed` filesystem
- `NewFileSource(dir)` -- source from OS directory
- `NewStaticSource(migrations...)` -- source from explicit list
- `Migrator.Up(ctx)`, `Migrator.Down(ctx)`, `Migrator.Step(ctx, n)`
- `Migrator.Version()` -- current applied version
- `Migrator.Status(ctx)` -- list all migrations with applied state

## Quick start

```go
import (
    "embed"
    "github.com/LingByte/ling-base/common/migration"
    "github.com/LingByte/ling-base/common/migration/gormmigrator"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

src := migration.NewEmbedSource(migrationFS, "migrations")
migrator := gormmigrator.New(db, src)

if err := migrator.Up(context.Background()); err != nil {
    log.Fatal(err)
}
```

## License

MIT
