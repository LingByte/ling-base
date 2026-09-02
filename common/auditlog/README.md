# auditlog

A pluggable audit-logging facility that records who did what to which
resource, when, and from where, with query support through a swappable
storage backend.

## Design

The storage backend is abstracted by the `Storage` interface so concrete
implementations (SQL, NoSQL, file, etc.) can be plugged in. An in-memory
implementation (`MemoryStorage`) is included for testing and simple
in-process use.

The `Logger` auto-generates a unique ID (crypto/rand based) and timestamp
when they are not provided.

## Quick start

```go
import (
    "context"
    "github.com/LingByte/ling-base/common/auditlog"
)

store := auditlog.NewMemoryStorage()
logger := auditlog.NewLogger(store)

err := logger.LogAction(ctx, "user-1", "login", "session", "sess-9",
    auditlog.WithIP("10.0.0.1"),
    auditlog.WithUserAgent("curl/8"),
    auditlog.WithStatus("success"),
    auditlog.WithDetail(map[string]any{"method": "password"}),
    auditlog.WithRequestID("req-1"),
)

// Query recent successful logins for a user
start := time.Now().Add(-24 * time.Hour)
entries, _ := store.Query(ctx, &auditlog.Filter{
    UserID:    "user-1",
    Action:    "login",
    StartTime: &start,
    Limit:     100,
})
```

## API

### Logger

| Method | Description |
| --- | --- |
| `NewLogger(storage)` | Create a logger backed by a storage |
| `Log(ctx, *Entry)` | Record a raw entry (auto-fills ID/Timestamp) |
| `LogAction(ctx, userID, action, resource, resourceID, opts...)` | Convenience wrapper |

### Options

| Option | Field set |
| --- | --- |
| `WithIP(ip)` | `Entry.IP` |
| `WithUserAgent(ua)` | `Entry.UserAgent` |
| `WithDetail(m)` | `Entry.Detail` |
| `WithStatus(s)` | `Entry.Status` |
| `WithRequestID(id)` | `Entry.RequestID` |

### Storage

| Method | Description |
| --- | --- |
| `Save(ctx, *Entry)` | Persist one entry |
| `Query(ctx, *Filter)` | Return matching entries (newest-first) |

### Filter fields

`UserID`, `Action`, `Resource`, `StartTime`, `EndTime`, `Limit`.

## Implementing a custom backend

```go
type PostgresStorage struct{ db *sql.DB }

func (p *PostgresStorage) Save(ctx context.Context, e *auditlog.Entry) error { /* ... */ }
func (p *PostgresStorage) Query(ctx context.Context, f *auditlog.Filter) ([]*auditlog.Entry, error) { /* ... */ }
```
