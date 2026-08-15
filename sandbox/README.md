# sandbox

Isolated execution environments for running untrusted scripts. Supports
Docker containers and local process isolation with security validation.

## Sandbox Types

| Type | Description |
|------|-------------|
| `SandboxTypeDocker` | Docker containers with resource limits |
| `SandboxTypeLocal` | Local process with restricted environment |
| `SandboxTypeDisabled` | Script execution disabled |

## Interfaces

```go
type Sandbox interface {
    Execute(ctx context.Context, config *ExecuteConfig) (*ExecuteResult, error)
    Cleanup(ctx context.Context) error
    Type() SandboxType
    IsAvailable(ctx context.Context) bool
}

type Manager interface {
    Execute(ctx context.Context, config *ExecuteConfig) (*ExecuteResult, error)
    Cleanup(ctx context.Context) error
    GetSandbox() Sandbox
    GetType() SandboxType
}
```

## Usage

```go
import "github.com/LingByte/ling-base/sandbox"

// Create a Docker sandbox manager
mgr, err := sandbox.NewManager(&sandbox.Config{
    Type:        sandbox.SandboxTypeDocker,
    DockerImage: "python:3.12-slim",
    Timeout:     30 * time.Second,
    MemoryLimit: 256 * 1024 * 1024,
    CPULimit:    1.0,
})
if err != nil {
    log.Fatal(err)
}

// Execute a script
result, err := mgr.Execute(ctx, &sandbox.ExecuteConfig{
    Script: "/path/to/script.py",
    Args:   []string{"--input", "data"},
})
```

## Security

The sandbox includes a `ScriptValidator` that detects:

- Network access attempts (curl, wget, sockets, etc.)
- Reverse shell patterns
- Embedded shell command substitution
- Dangerous shell operators
- Argument injection
- Stdin injection

Path traversal protection ensures scripts can only access files under
the configured base directory. Dangerous environment variables are filtered.

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `Timeout` | 60s | Maximum execution time |
| `MemoryLimit` | 256MB | Memory limit (Docker) |
| `CPULimit` | 1.0 | CPU core limit (Docker) |
| `DockerImage` | `wechatopenai/weknora-sandbox:latest` | Docker image |
| `AllowNetwork` | false | Network access (Docker) |
| `ReadOnlyRootfs` | false | Read-only root filesystem (Docker) |

## License

MIT
