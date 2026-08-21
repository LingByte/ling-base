# bootstrap

Go application bootstrap framework inspired by Java Spring Boot: banner
printing, lifecycle management, component registry, profile-based
configuration, event publishing, database migrations, and graceful shutdown.

## Structure

```
bootstrap/
├── application.go    # Application orchestrator (SpringApplication equivalent)
├── banner.go         # ASCII art banner generation (Doom font, ANSI gradient)
├── font_embed.go     # Embedded Doom font data for offline banner rendering
├── event.go          # Application event names + async event bus setup
├── lifecycle.go      # Lifecycle interface, init/start/stop hooks
├── migration.go      # MigrationRunner / AutoMigrator interfaces
├── profile.go        # ProfileManager (dev / test / staging / prod)
├── properties.go     # Properties: typed config from env / file / map
├── registry.go       # Component registry (name + type based lookup)
└── shutdown.go       # ShutdownManager: signal handling + graceful hooks
```

## Key Types

```go
// Application is the central orchestrator.
type Application struct { ... }

// Lifecycle is implemented by components needing start/stop callbacks.
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
}

// AppStatus tracks the application state machine.
type AppStatus int32  // Created → Initializing → Starting → Running → Stopping → Stopped

// Properties stores typed config values (string / int / bool / duration / slice).
type Properties struct { ... }

// ProfileManager manages active profiles (dev, test, staging, prod).
type ProfileManager struct { ... }

// Registry is a named component container with type-based lookup.
type Registry struct { ... }

// ShutdownManager handles OS signals and runs hooks in priority order.
type ShutdownManager struct { ... }
```

## Application Events

| Event                  | When emitted                  |
|------------------------|-------------------------------|
| `app.starting`         | Before init phase             |
| `app.started`          | After all components started  |
| `app.ready`            | Application is ready to serve |
| `app.stopping`         | Shutdown signal received      |
| `app.stopped`          | All components stopped        |
| `app.failed`           | Startup failed                |
| `component.init/start/stop` | Per-component lifecycle  |

## Quick Start

```go
import "github.com/LingByte/ling-base/bootstrap"

app := bootstrap.New("myapp",
    bootstrap.WithProfile(bootstrap.ProfileProd),
    bootstrap.WithEnvProperties("APP"),       // APP_DB_HOST -> db.host
    bootstrap.WithShutdownTimeout(60*time.Second),
    bootstrap.WithBannerText("MyApp"),
)

// Register components (Lifecycle impls are auto-wired).
app.MustRegister("db", &DatabaseComponent{})
app.MustRegister("cache", &CacheComponent{})

// Init / shutdown hooks (analogous to @PostConstruct / @PreDestroy).
app.AddInitHook("warm-cache", func(ctx context.Context) error { ... })
app.AddShutdownHook("flush-logs", func(ctx context.Context) error { ... })

// Subscribe to events.
app.OnEvent(bootstrap.EventAppReady, func(e eventbus.Event) { ... })

if err := app.Run(); err != nil {  // blocks until SIGINT/SIGTERM
    log.Fatal(err)
}
```

### Database Migrations

```go
// Production: versioned SQL migrations
app := bootstrap.New("myapp",
    bootstrap.WithProfile(bootstrap.ProfileProd),
    bootstrap.WithMigration(migrator),  // implements MigrationRunner
)

// Dev/test: GORM auto-migrate from struct definitions
app := bootstrap.New("myapp",
    bootstrap.WithProfile(bootstrap.ProfileDev),
    bootstrap.WithAutoMigrate(db, &User{}, &Order{}),
)
```
