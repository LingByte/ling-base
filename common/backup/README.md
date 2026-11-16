# backup

Pluggable **logical** backup/restore orchestrator for ling-base.

Streams data from a `Source` to a `Destination` with optional gzip and SHA-256
checksums. Restore reads the artifact back and writes it to a `Sink`.

## Platforms & scenarios

| Scenario | Helper |
|----------|--------|
| Local file | `FileSource` / `FileSink` / `FileDestination` |
| Directory tree (uploads, configs) | `DirectorySource` / `DirectorySink` (tar.gz) |
| MySQL / MariaDB (self-hosted **or cloud**) | `MySQLDump` / `MySQLLoad` |
| PostgreSQL | `PostgresDump` / `PostgresLoad` |
| MongoDB | `MongoDump` / `MongoRestore` (`--archive`) |
| SQLite | `SQLiteDump` / `SQLiteLoad` |
| Redis (RDB stream) | `RedisDump` |
| ClickHouse | `ClickHouseDump` |
| SQL Server / Azure SQL | `MSSQLDump` (`sqlcmd`) |
| Arbitrary CLI | `CommandSource` / `CommandSink` |
| In-memory / custom io | `BytesSource`, `ReaderSource`, `WriterSink`, `BufferSink` |

Cloud databases (RDS, PolarDB, Cloud SQL, Azure SQL, ElastiCache, …) use the
**same helpers**: point `Host` / `URI` at the cloud endpoint with a backup
account. Vendor physical snapshots / PITR APIs are **out of scope** (heavy
SDKs); implement `Source`/`Destination` in your app if you need them.

Object storage: implement `Destination` yourself (do not pull `common/stores`
into this module).

## Quick start — file

```go
src := &backup.FileSource{Path: "/var/data/app.db"}
dst := &backup.FileDestination{Dir: "/var/backups"}
sink := &backup.FileSink{Path: "/var/data/app.db"}

mgr := backup.New(src, dst,
    backup.WithCompression(),
    backup.WithCatalog(&backup.FileCatalog{Path: "/var/backups/catalog.json"}),
    backup.WithRetention(backup.KeepLastN(7)),
    backup.WithSink(sink),
)

bp, err := mgr.Backup(ctx, "app-2026-01-01")
_ = mgr.Verify(ctx, bp)
_ = mgr.Restore(ctx, bp, nil)
_, _ = mgr.Prune(ctx, nil)
```

## Quick start — cloud MySQL

```go
cfg := backup.MySQLConfig{
    Host:     "rm-xxx.mysql.rds.aliyuncs.com",
    Port:     3306,
    User:     "backup",
    Password: "...",
    Database: "app",
}
mgr := backup.New(
    backup.MySQLDump(cfg),
    &backup.FileDestination{Dir: "/var/backups"},
    backup.WithCompression(),
    backup.WithCatalog(&backup.FileCatalog{Path: "/var/backups/catalog.json"}),
)
bp, err := mgr.Backup(ctx, "mysql-app")
err = mgr.Restore(ctx, bp, backup.MySQLLoad(cfg))
```

## Quick start — directory

```go
mgr := backup.New(
    &backup.DirectorySource{Root: "/var/lib/uploads"},
    &backup.FileDestination{Dir: "/var/backups"},
)
bp, _ := mgr.Backup(ctx, "uploads")
_ = mgr.Restore(ctx, bp, &backup.DirectorySink{Dest: "/var/lib/uploads-restored"})
```

## API

| Method | Description |
|--------|-------------|
| `Backup` | Source → Destination (+ catalog) |
| `Restore` | Destination → Sink |
| `Verify` | Re-hash artifact vs recorded checksum |
| `List` / `Get` / `Delete` | Catalog + artifact lifecycle |
| `Prune` | Retention (`KeepLastN`, `KeepNewerThan`) |

## License

MIT
