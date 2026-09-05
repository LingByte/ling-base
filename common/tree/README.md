# tree

Service-tree (hierarchical resource organization) library for
[ling-base](https://github.com/LingByte/ling-base).

A service tree organizes resources into a fixed three-level hierarchy
modeled as **group.project.app** (g.p.a), the pattern popularized by
large-scale DevOps platforms. Each leaf node (app) is the attachment
point for concrete resources (hosts, databases, load balancers, …).

## On-demand modules

| Module path | Third-party deps |
|-------------|------------------|
| `github.com/LingByte/ling-base/common/tree` | **none** (interface + core logic) |
| `.../common/tree/memory` | **none** (in-process store) |
| `.../common/tree/gormstore` | gorm.io/gorm |
| `.../common/tree/mysql` | gorm.io/driver/mysql |
| `.../common/tree/postgres` | gorm.io/driver/postgres |
| `.../common/tree/sqlite` | gorm.io/driver/sqlite |

```bash
go get github.com/LingByte/ling-base/common/tree          # core
go get github.com/LingByte/ling-base/common/tree/memory    # in-memory store
go get github.com/LingByte/ling-base/common/tree/sqlite    # SQLite store
```

## Features

- Fixed three-level hierarchy: **group → project → app**
- Materialized-path storage for O(1) subtree queries
- Auto-creation of intermediate nodes on `Add`
- Cascade delete with optional `force` flag
- Storage-agnostic `Store` interface with multiple backends
- Transactional safety via `Store.Tx`
- Zero external dependencies in the core package

## Storage model

Nodes use a materialized-path scheme:

```
Level 1 (group)   path = "0"            e.g. inf
Level 2 (project) path = "/{gid}"       e.g. inf.monitor
Level 3 (app)     path = "/{gid}/{pid}"  e.g. inf.monitor.prometheus
```

This makes subtree queries a single prefix scan (`path LIKE '/{gid}/%'`)
and avoids recursive CTEs.

## Quick start

### In-memory

```go
import (
    "context"
    "github.com/LingByte/ling-base/common/tree"
    "github.com/LingByte/ling-base/common/tree/memory"
)

store := memory.New()
tr, _ := tree.New(store)
ctx := context.Background()

// Add a full g.p.a path; missing intermediate nodes are created.
leaf, _ := tr.Add(ctx, "inf.monitor.prometheus")

// Query all projects under a group
projects, _ := tr.Query(ctx, "inf", tree.QueryChildren)
// → ["cicd", "monitor"]

// Query all apps under a group (recursive)
apps, _ := tr.Query(ctx, "inf", tree.QueryLeaves)
// → ["inf.cicd.jenkins", "inf.monitor.prometheus", "inf.monitor.thanos"]

// Check existence
ok, _ := tr.Exists(ctx, "inf.monitor.prometheus")

// Delete with cascade
n, _ := tr.Delete(ctx, "inf", true) // force=true removes subtree
```

### SQLite

```go
import (
    "github.com/LingByte/ling-base/common/tree/sqlite"
)

store, err := sqlite.New("devops.db")
tr, _ := tree.New(store)
```

### MySQL

```go
store, err := mysql.New("user:pass@tcp(127.0.0.1:3306)/devops?charset=utf8mb4&parseTime=true")
```

### PostgreSQL

```go
store, err := postgres.New("host=127.0.0.1 user=devops password=pass dbname=devops sslmode=disable")
```

## Interface

### Tree

```go
type Tree interface {
    Add(ctx context.Context, path string) (*Node, error)
    Get(ctx context.Context, path string) (*Node, error)
    Query(ctx context.Context, path string, qt QueryType) ([]string, error)
    Exists(ctx context.Context, path string) (bool, error)
    Delete(ctx context.Context, path string, force bool) (int64, error)
    ListChildren(ctx context.Context, path string) ([]*Node, error)
    SubTree(ctx context.Context, path string) ([]*Node, error)
    Close() error
}
```

### Store

```go
type Store interface {
    Insert(ctx context.Context, n *Node) error
    Get(ctx context.Context, level int, path, name string) (*Node, error)
    GetByID(ctx context.Context, id int64) (*Node, error)
    DeleteByID(ctx context.Context, id int64) error
    ListChildren(ctx context.Context, level int, path string) ([]*Node, error)
    ListByPathPrefix(ctx context.Context, level int, pathPrefix string) ([]*Node, error)
    Tx(ctx context.Context, fn func(ctx context.Context) error) error
    Close() error
}
```

## Query types

| QueryType | Input | Output |
|-----------|-------|--------|
| `QueryChildren` | `g` or `g.p` | direct child names: `["p1", "p2"]` |
| `QueryLeaves` | `g` or `g.p` | full leaf paths: `["g.p1.a1", "g.p1.a2"]` |
| `QueryExists` | `g.p.a` | `["g.p.a"]` if exists, `[]` if not |

## Errors

| Error | Meaning |
|-------|---------|
| `ErrEmptyPath` | path string is empty |
| `ErrInvalidPath` | path doesn't match g / g.p / g.p.a format |
| `ErrNodeNotFound` | node doesn't exist |
| `ErrNodeExists` | node already exists (on Add) |
| `ErrHasChildren` | delete blocked by children, use `force=true` |
| `ErrInvalidQueryType` | unknown QueryType |

## Schema (SQL backends)

```sql
CREATE TABLE service_tree_node (
    id         BIGINT PRIMARY KEY AUTOINCREMENT,
    level      INTEGER NOT NULL,
    path       VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    meta       TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX idx_tree_level_path_name
    ON service_tree_node (level, path, name);
CREATE INDEX idx_tree_path ON service_tree_node (path);
```

The schema is auto-migrated by `gormstore.New` / `mysql.New` / etc.
