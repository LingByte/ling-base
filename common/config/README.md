# config

Multi-format, multi-environment configuration loader with YAML and .env support, env-specific overrides, and a database-backed key/value store.

## Features

- Layered loading: base file -> env-specific file -> OS env var overrides
- YAML and .env file formats (auto-detected by extension)
- Struct tag-based env var binding (`env:"KEY"` / `env:"KEY,required"`)
- `Store` -- database-backed config with LRU cache and env fallback

## Key types

- `Loader` -- builder-style config loader
- `Format` -- file format (`FormatAuto`, `FormatYAML`, `FormatENV`)
- `Store` -- system config store with DB persistence and cache
- `ConfigItem` -- GORM model for the `configs` table
- `StoreOptions` -- store configuration (DB, cache, TTL)

## Key functions

- `Load(dir, env, out)` -- convenience loader
- `New()` -- builder API (`Dir`, `Env`, `BaseName`, `Format`, `WithEnvVars`)
- `LoadYAML(path, out)` / `LoadENV(path)` -- single-file helpers
- `NewStore(opts)` / `NewStoreWithDB(db)` / `NewEnvOnlyStore()`
- `Store.GetValue`, `Store.SetIntValue`, `Store.SetBoolValue`, `Store.SetValue`

## Quick start

```go
import "github.com/LingByte/ling-base/common/config"

type AppCfg struct {
    Server struct {
        Port int    `yaml:"port" env:"SERVER_PORT"`
        Host string `yaml:"host" env:"SERVER_HOST"`
    } `yaml:"server"`
}

var cfg AppCfg
if err := config.Load("config/", "dev", &cfg); err != nil {
    log.Fatal(err)
}
```

## License

MIT
