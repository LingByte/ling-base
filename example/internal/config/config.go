// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package config handles application configuration loading from YAML files,
// environment variables, and optional database-backed config store.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LingByte/ling-base/common"
	appconfig "github.com/LingByte/ling-base/common/config"
	"github.com/LingByte/ling-base/constants"
	"github.com/LingByte/ling-base/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AppConfig is the top-level application configuration struct.
// It is loaded from config/config.<env>.yaml and overridden by env vars.
type AppConfig struct {
	Server struct {
		Port            int           `yaml:"port" env:"SERVER_PORT"`
		Host            string        `yaml:"host" env:"SERVER_HOST"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT"`
	} `yaml:"server"`

	DB struct {
		Driver string `yaml:"driver" env:"DB_DRIVER"`
		DSN    string `yaml:"dsn" env:"APP_DB_DSN"`
	} `yaml:"db"`

	Redis struct {
		Addr     string `yaml:"addr" env:"REDIS_ADDR"`
		Password string `yaml:"password" env:"REDIS_PASSWORD"`
		DB       int    `yaml:"db" env:"REDIS_DB"`
	} `yaml:"redis"`

	Search struct {
		IndexPath    string        `yaml:"index_path" env:"SEARCH_INDEX_PATH"`
		BatchSize    int           `yaml:"batch_size"`
		QueryTimeout time.Duration `yaml:"query_timeout"`
	} `yaml:"search"`

	RateLimit struct {
		RequestsPerSecond float64 `yaml:"rps" env:"RATE_LIMIT_RPS"`
		MaxConnections    int     `yaml:"max_conns" env:"RATE_LIMIT_MAX_CONNS"`
	} `yaml:"rate_limit"`

	Log struct {
		Level  string `yaml:"level" env:"LOG_LEVEL"`
		File   string `yaml:"file" env:"LOG_FILE"`
		Mode   string `yaml:"mode" env:"MODE"`
		Daily  bool   `yaml:"daily"`
		MaxAge int    `yaml:"max_age"`
	} `yaml:"log"`
}

// Load loads YAML config files and returns a config Store + AppConfig + *gorm.DB.
// If db.DSN is set, it creates a DB-backed store via common.InitDatabase and
// returns the *gorm.DB for use by other components (e.g. request persistence).
// Otherwise, it uses env-only mode and returns a nil *gorm.DB.
func Load(env string) (*appconfig.Store, *AppConfig, *gorm.DB, error) {
	var appCfg AppConfig
	cfgDir := "config"
	if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
		cfgDir = "example/config"
	}

	if err := appconfig.New().
		Dir(cfgDir).
		Env(env).
		Load(&appCfg); err != nil {
		logger.Warn("config file load failed, continuing with defaults", zap.Error(err))
	}

	var store *appconfig.Store
	var db *gorm.DB
	var err error

	if appCfg.DB.DSN != "" {
		driver := appCfg.DB.Driver
		dsn := appCfg.DB.DSN

		// Support "sqlite:path" shorthand.
		if strings.HasPrefix(dsn, "sqlite:") {
			driver = constants.DBDriverSQLite
			dsn = strings.TrimPrefix(dsn, "sqlite:")
		}

		db, err = common.InitDatabase(nil, driver, dsn)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("init database: %w", err)
		}

		store, err = appconfig.NewStoreWithDB(db)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("create config store: %w", err)
		}
		if err := store.AutoMigrate(); err != nil {
			return nil, nil, nil, fmt.Errorf("auto-migrate configs table: %w", err)
		}
		logger.Info("config store: DB-backed", zap.String("driver", driver), zap.String("dsn", dsn))
	} else {
		store, err = appconfig.NewEnvOnlyStore()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("create env store: %w", err)
		}
		logger.Info("config store: env-only")
	}

	seedDefaults(store, &appCfg)
	if err := store.LoadAutoloads(); err != nil {
		logger.Warn("loadAutoloads failed", zap.Error(err))
	}

	return store, &appCfg, db, nil
}

// seedDefaults inserts default config values if they don't exist.
func seedDefaults(store *appconfig.Store, cfg *AppConfig) {
	defaults := []struct {
		key, val, format string
		autoload, public bool
	}{
		{"SERVER_PORT", fmt.Sprintf("%d", cfg.Server.Port), appconfig.ConfigFormatInt, true, true},
		{"SERVER_HOST", cfg.Server.Host, appconfig.ConfigFormatText, true, true},
		{"SERVER_SHUTDOWN_TIMEOUT", cfg.Server.ShutdownTimeout.String(), appconfig.ConfigFormatText, true, false},
		{"RATE_LIMIT_RPS", fmt.Sprintf("%.0f", cfg.RateLimit.RequestsPerSecond), appconfig.ConfigFormatFloat, true, false},
		{"RATE_LIMIT_MAX_CONNS", fmt.Sprintf("%d", cfg.RateLimit.MaxConnections), appconfig.ConfigFormatInt, true, false},
		{"LOG_LEVEL", cfg.Log.Level, appconfig.ConfigFormatText, true, false},
	}

	for _, d := range defaults {
		if err := store.CheckValue(d.key, d.val, d.format, d.autoload, d.public); err != nil {
			logger.Warn("seed config failed", zap.String("key", d.key), zap.Error(err))
		}
	}
}
