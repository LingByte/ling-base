// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command app-demo is a full application example demonstrating bootstrap +
// config (file + DB store) + eventbus + idgen + limiter + logger + gin + gorm
// + redis cache + full-text search + DB schema migrations working together as
// a realistic ling-base application.
//
// Migration strategy:
//   - dev/test:  GORM AutoMigrate (schema auto-syncs from struct definitions)
//   - prod:      Versioned SQL migrations (embedded .sql files, reviewable)
//
// Usage:
//
//	# Run with default (dev) config from ./config/config.yaml
//	go run ./cmd/app-demo
//
//	# Run with a specific environment
//	APP_ENV=prod go run ./cmd/app-demo
//
//	# Run with SQLite-backed config store + request persistence
//	APP_DB_DSN=sqlite:./app.db go run ./cmd/app-demo
//
//	# Run with Redis cache
//	REDIS_ADDR=localhost:6379 go run ./cmd/app-demo
//
//	# Run with full-text search (in-memory)
//	SEARCH_INDEX_PATH=memory go run ./cmd/app-demo
//
//	# Override any config via env var
//	SERVER_PORT=9090 APP_ENV=prod go run ./cmd/app-demo
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LingByte/ling-base/bootstrap"
	"github.com/LingByte/ling-base/common/migration"
	gormmigrator "github.com/LingByte/ling-base/common/migration/gormmigrator"
	"github.com/LingByte/ling-base/common/constants"
	"github.com/LingByte/ling-base/common/eventbus"
	"github.com/LingByte/ling-base/example/internal/config"
	"github.com/LingByte/ling-base/example/internal/handlers"
	"github.com/LingByte/ling-base/example/internal/listeners"
	"github.com/LingByte/ling-base/example/internal/models"
	"github.com/LingByte/ling-base/common/logger"
	"go.uber.org/zap"
)

// migrationFS embeds the SQL migration files for production use.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
	envFlag := flag.String("env", "", "environment name (dev/prod/test, default: $APP_ENV or dev)")
	flag.Parse()

	env := *envFlag
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	if env == "" {
		env = "dev"
	}

	// 1. Initialize logger.
	logger.InitTimezone(constants.DefaultTimezone)
	logCfg := &logger.LogConfig{
		Level:  "debug",
		Daily:  false,
		MaxAge: 7,
	}
	if err := logger.Init(logCfg, env); err != nil {
		panic(err)
	}
	defer logger.Sync()

	// 2. Load configuration (YAML + env + optional DB store).
	store, appCfg, db, err := config.Load(env)
	if err != nil {
		logger.Fatal("load config failed", zap.Error(err))
	}
	defer store.Close()

	// 3. Create the bootstrap application.
	shutdownTimeout := appCfg.Server.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	appOpts := []bootstrap.Option{
		bootstrap.WithProfile(env),
		bootstrap.WithShutdownTimeout(shutdownTimeout),
	}

	// Configure migration strategy based on profile.
	//   - prod:      Versioned SQL migrations (embedded .sql files)
	//   - dev/test:  GORM AutoMigrate (auto-sync from struct definitions)
	if db != nil {
		if env == bootstrap.ProfileProd || env == bootstrap.ProfileStaging {
			// Production: use versioned SQL migrations for controlled,
			// reviewable schema changes.
			src := migration.NewEmbedSource(migrationFS, "migrations")
			migrator, err := gormmigrator.New(db, src)
			if err != nil {
				logger.Fatal("create migrator failed", zap.Error(err))
			}
			appOpts = append(appOpts, bootstrap.WithMigration(migrator))
			logger.Info("migration strategy: versioned SQL (prod/staging)")
		} else {
			// Dev/test: use GORM AutoMigrate for rapid iteration.
			appOpts = append(appOpts, bootstrap.WithAutoMigrate(db, &models.RequestLog{}))
			logger.Info("migration strategy: auto-migrate (dev/test)")
		}
	}

	app := bootstrap.New("app-demo", appOpts...)

	// 4. Subscribe to lifecycle events.
	app.OnEvent(bootstrap.EventAppStarting, func(ctx context.Context, e *eventbus.Event) error {
		logger.Info("event: app starting", zap.String("eventId", e.ID))
		return nil
	})
	app.OnEvent(bootstrap.EventAppStarted, func(ctx context.Context, e *eventbus.Event) error {
		logger.Info("event: app started", zap.String("eventId", e.ID))
		return nil
	})
	app.OnEvent(bootstrap.EventAppReady, func(ctx context.Context, e *eventbus.Event) error {
		logger.Info("event: app ready", zap.String("eventId", e.ID))
		return nil
	})
	app.OnEvent(bootstrap.EventAppStopping, func(ctx context.Context, e *eventbus.Event) error {
		logger.Info("event: app stopping", zap.String("eventId", e.ID))
		return nil
	})

	// 5. Initialize optional dependencies: Redis cache + search engine.
	redisCache := handlers.NewRedisCache(appCfg.Redis.Addr, appCfg.Redis.Password, appCfg.Redis.DB)
	searchEng := handlers.NewSearchEngine(appCfg.Search.IndexPath, appCfg.Search.BatchSize, appCfg.Search.QueryTimeout)

	// 6. Wire listeners — subscribes to event bus signals for async DB
	// persistence and other side-effects. Must be called before handlers
	// start emitting signals.
	listeners.InitRequestListeners(db, app.Events())

	// 7. Create handlers — holds db + cache + search + config + limiters.
	h := handlers.NewHandlers(db, redisCache, searchEng, store, app.Events())

	// 7. Register handlers as a lifecycle component (starts/stops the gin server).
	if err := app.Register("api", h); err != nil {
		logger.Fatal("register api failed", zap.Error(err))
	}

	// 8. Init hook: print config summary.
	app.AddInitHook("print-config", func(ctx context.Context) error {
		logger.Info("configuration loaded",
			zap.Int("server.port", store.GetIntValue("SERVER_PORT", 0)),
			zap.String("server.host", store.GetStringWithDefault("SERVER_HOST", "")),
			zap.Float64("rate_limit.rps", store.GetFloatValue("RATE_LIMIT_RPS", 0)),
			zap.Int("rate_limit.max_conns", store.GetIntValue("RATE_LIMIT_MAX_CONNS", 0)),
			zap.String("log.level", store.GetStringWithDefault("LOG_LEVEL", "info")),
			zap.Bool("db.enabled", db != nil),
			zap.Bool("redis.enabled", redisCache != nil),
			zap.Bool("search.enabled", searchEng != nil),
		)

		if store.HasDB() {
			publics, _ := store.LoadPublicConfigs()
			logger.Info("public configs loaded", zap.Int("count", len(publics)))
		}
		return nil
	})

	// 9. Shutdown hook.
	app.AddShutdownHook("cleanup", func(ctx context.Context) error {
		logger.Info("shutdown: purging caches and cleaning up...")
		store.PurgeAllCache()
		return nil
	})

	// 10. After app is ready, simulate traffic + index demo docs.
	app.OnEvent(bootstrap.EventAppReady, func(ctx context.Context, e *eventbus.Event) error {
		go simulateTraffic(h)
		return nil
	})

	// 11. Run.
	errCh := app.RunAsync()
	if err := <-errCh; err != nil {
		logger.Fatal("app run failed", zap.Error(err))
	}

	// 12. Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutdown signal received")
	app.Stop()

	// 13. Print event bus metrics.
	m := app.Events().Metrics()
	logger.Info("event bus metrics",
		zap.Int64("published", m.Published),
		zap.Int64("delivered", m.Delivered),
		zap.Int64("failed", m.Failed),
		zap.Int64("subscribers", m.Subscribers),
	)

	logger.Info("application stopped gracefully")
}

// simulateTraffic sends a few demo HTTP requests to the running gin server.
func simulateTraffic(h *handlers.Handlers) {
	time.Sleep(300 * time.Millisecond)

	baseURL := fmt.Sprintf("http://%s:%d", h.Host(), h.Port())
	client := &http.Client{Timeout: 5 * time.Second}

	// Send echo requests.
	for i := 1; i <= 3; i++ {
		resp, err := client.Get(baseURL + "/api/v1/echo")
		if err != nil {
			logger.Warn("simulated request failed", zap.Int("seq", i), zap.Error(err))
		} else {
			logger.Info("simulated request ok", zap.Int("seq", i), zap.Int("status", resp.StatusCode))
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
}
