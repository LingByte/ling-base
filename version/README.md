# version

Build-time and runtime version information for the ling-base library.

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `Version` | `"1.0.0"` | Semantic version string |
| `GitCommit` | `"unknown"` | Short Git commit hash |
| `BuildTime` | `"unknown"` | Build timestamp (RFC3339) |
| `GoVersion` | `"unknown"` | Go toolchain version |

## Functions

- `GetVersion()` — returns the version string
- `GetVersionInfo()` — returns a human-readable summary
- `GetGitCommit()` — returns the Git commit hash
- `GetBuildTime()` — returns the build timestamp
- `GetGoVersion()` — returns the Go version

## Link-time override

Override at build time with `-ldflags`:

```sh
go build -ldflags "\
  -X github.com/LingByte/ling-base/version.Version=1.2.0 \
  -X github.com/LingByte/ling-base/version.GitCommit=$(git rev-parse --short HEAD) \
  -X github.com/LingByte/ling-base/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/LingByte/ling-base/version.GoVersion=$(go version)" \
  ./...
```

## Automatic resolution

When not overridden via `-ldflags`, `GitCommit` and `BuildTime` are resolved
automatically from:

1. VCS build info (Go 1.18+ `debug.ReadBuildInfo`)
2. The local git repository (via `git` command)

## Usage

```go
import "github.com/LingByte/ling-base/version"

fmt.Println(version.GetVersionInfo())
// Output: 1.0.0 (commit: abc123def456, built at: 2026-01-15T10:30:00Z, go: go1.26.2)
```

## License

MIT
