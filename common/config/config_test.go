// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to create a temp config dir with files.
func writeConfigDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	// Purge env store between tests to avoid cross-test pollution.
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })
	return dir
}

// Test config struct.
type testAppConfig struct {
	Server struct {
		Port    int           `yaml:"port" env:"SERVER_PORT"`
		Host    string        `yaml:"host" env:"SERVER_HOST"`
		Timeout time.Duration `yaml:"timeout" env:"SERVER_TIMEOUT"`
	} `yaml:"server"`
	DB struct {
		DSN    string `yaml:"dsn" env:"DSN"`
		MaxCon int    `yaml:"max_conns" env:"DB_MAX_CONNS"`
	} `yaml:"db"`
	Debug   bool     `yaml:"debug" env:"DEBUG"`
	Tags    []string `yaml:"tags" env:"TAGS"`
	NoTag   string   `yaml:"no_tag"`
	private string   `yaml:"private"`
}

// ===== YAML loading =====

func TestLoadYAML_Basic(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `
server:
  port: 8080
  host: localhost
  timeout: 30s
db:
  dsn: "root:pass@tcp(localhost:3306)/mydb"
  max_conns: 100
debug: true
tags:
  - api
  - v2
`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 30*time.Second, cfg.Server.Timeout)
	assert.Equal(t, "root:pass@tcp(localhost:3306)/mydb", cfg.DB.DSN)
	assert.Equal(t, 100, cfg.DB.MaxCon)
	assert.True(t, cfg.Debug)
	assert.Equal(t, []string{"api", "v2"}, cfg.Tags)
}

func TestLoadYAML_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	var cfg testAppConfig
	err := New().Dir(dir).Load(&cfg)
	assert.NoError(t, err) // missing base file is not an error
}

// ===== Multi-environment =====

func TestLoad_MultiEnv_Override(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `
server:
  port: 8080
  host: localhost
db:
  dsn: "base-dsn"
  max_conns: 10
debug: false
`,
		"config.dev.yaml": `
server:
  port: 9090
db:
  dsn: "dev-dsn"
debug: true
`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Env("dev").Load(&cfg)
	require.NoError(t, err)
	// Overridden by dev.
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "dev-dsn", cfg.DB.DSN)
	assert.True(t, cfg.Debug)
	// Not overridden — should keep base value.
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 10, cfg.DB.MaxCon)
}

func TestLoad_MultiEnv_NoEnvFile(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `
server:
  port: 8080
debug: false
`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Env("prod").Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoad_NoEnv(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `server: { port: 3000 }`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 3000, cfg.Server.Port)
}

// ===== ENV var overrides =====

func TestLoad_EnvVarOverride(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `
server:
  port: 8080
db:
  dsn: "file-dsn"
debug: false
`,
	})

	t.Setenv("SERVER_PORT", "7070")
	t.Setenv("DSN", "env-dsn")
	t.Setenv("DEBUG", "true")

	var cfg testAppConfig
	err := New().Dir(dir).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 7070, cfg.Server.Port)
	assert.Equal(t, "env-dsn", cfg.DB.DSN)
	assert.True(t, cfg.Debug)
}

func TestLoad_EnvVarRequired(t *testing.T) {
	type cfgReq struct {
		APIKey string `env:"API_KEY,required"`
	}
	dir := t.TempDir()
	err := New().Dir(dir).Load(&cfgReq{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY")
}

func TestLoad_EnvVarDuration(t *testing.T) {
	type cfgDur struct {
		Timeout time.Duration `env:"TIMEOUT"`
	}
	dir := t.TempDir()
	t.Setenv("TIMEOUT", "5m30s")
	var c cfgDur
	err := New().Dir(dir).Load(&c)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute+30*time.Second, c.Timeout)
}

func TestLoad_EnvVarSlice(t *testing.T) {
	type cfgSlice struct {
		Tags []string `env:"TAGS"`
	}
	dir := t.TempDir()
	t.Setenv("TAGS", "a, b, c")
	var c cfgSlice
	err := New().Dir(dir).Load(&c)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, c.Tags)
}

func TestLoad_EnvVarDisabled(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `server: { port: 8080 }`,
	})
	t.Setenv("SERVER_PORT", "9999")

	var cfg testAppConfig
	err := New().Dir(dir).WithEnvVars(false).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port) // not overridden
}

// ===== .env format =====

func TestLoad_ENVFormat(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.env": `
# Base config
SERVER_PORT=8080
SERVER_HOST=localhost
DSN=root:pass@tcp(localhost:3306)/db
DEBUG=true
`,
		"config.dev.env": `
SERVER_PORT=9090
DSN=dev-dsn
`,
	})

	t.Setenv("SERVER_PORT", "") // clear to test .env file values

	var cfg testAppConfig
	// .env format doesn't unmarshal into structs directly; it sets env vars.
	// We need OverwriteEnvVars(true) to make them visible.
	err := New().Dir(dir).Env("dev").OverwriteEnvVars(true).Load(&cfg)
	require.NoError(t, err)

	// Values come from env overrides (applied via struct tags).
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "dev-dsn", cfg.DB.DSN)
	assert.True(t, cfg.Debug)
}

func TestLoad_ENVFormat_QuotesAndComments(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.env": `
SERVER_HOST="myhost"
DSN='root:pass##word@tcp(127.0.0.1:3306)/db'
DEBUG=true  # inline comment
`,
	})

	t.Setenv("SERVER_HOST", "")
	t.Setenv("DSN", "")
	t.Setenv("DEBUG", "")

	var cfg testAppConfig
	err := New().Dir(dir).OverwriteEnvVars(true).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "myhost", cfg.Server.Host)
	assert.Equal(t, "root:pass##word@tcp(127.0.0.1:3306)/db", cfg.DB.DSN)
	assert.True(t, cfg.Debug)
}

func TestLoad_ENVFormat_ExportPrefix(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.env": `export SERVER_PORT=4040`,
	})

	t.Setenv("SERVER_PORT", "")
	var cfg testAppConfig
	err := New().Dir(dir).OverwriteEnvVars(true).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 4040, cfg.Server.Port)
}

// ===== Convenience functions =====

func TestLoad_ConvenienceFunc(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `server: { port: 3000 }`,
	})

	var cfg testAppConfig
	err := Load(dir, "", &cfg)
	require.NoError(t, err)
	assert.Equal(t, 3000, cfg.Server.Port)
}

func TestLoadYAML_ConvenienceFunc(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"myconfig.yaml": `server: { port: 5555 }`,
	})

	var cfg testAppConfig
	err := LoadYAML(filepath.Join(dir, "myconfig.yaml"), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 5555, cfg.Server.Port)
}

func TestLoadENV_ConvenienceFunc(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"test.env": `MY_TEST_KEY=my_test_value`,
	})

	t.Setenv("MY_TEST_KEY", "")
	err := LoadENV(filepath.Join(dir, "test.env"))
	require.NoError(t, err)
	assert.Equal(t, "my_test_value", os.Getenv("MY_TEST_KEY"))
}

// ===== Builder API =====

func TestBuilder_BaseName(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"app.yaml": `server: { port: 1234 }`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).BaseName("app").Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 1234, cfg.Server.Port)
}

func TestBuilder_FormatYAML(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": `server: { port: 2222 }`,
		"config.env":  `SERVER_PORT=3333`,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Format(FormatYAML).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, 2222, cfg.Server.Port)
}

// ===== Edge cases =====

func TestLoad_NilOutput(t *testing.T) {
	dir := t.TempDir()
	err := New().Dir(dir).Load(nil)
	assert.Error(t, err)
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := writeConfigDir(t, map[string]string{
		"config.yaml": ``,
	})

	var cfg testAppConfig
	err := New().Dir(dir).Load(&cfg)
	require.NoError(t, err)
	// All zero values.
	assert.Equal(t, 0, cfg.Server.Port)
}

func TestLoad_NestedStructNoEnvTag(t *testing.T) {
	type inner struct {
		Val string `env:"INNER_VAL"`
	}
	type outer struct {
		Inner inner `yaml:"inner"`
	}

	dir := t.TempDir()
	t.Setenv("INNER_VAL", "hello")

	var cfg outer
	err := New().Dir(dir).Load(&cfg)
	require.NoError(t, err)
	assert.Equal(t, "hello", cfg.Inner.Val)
}

func TestFormat_String(t *testing.T) {
	assert.Equal(t, "yaml", FormatYAML.String())
	assert.Equal(t, "env", FormatENV.String())
	assert.Equal(t, "auto", FormatAuto.String())
}

// ===== Concurrent =====

func TestEnvStore_Concurrent(t *testing.T) {
	envStore.Purge()
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			envStore.Set("KEY", "val")
			_, _ = envStore.Get("KEY")
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
