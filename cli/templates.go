// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"strings"
	"time"
)

// ──────────────────────────────────────────────
// Core module template
// ──────────────────────────────────────────────

// generateCore generates files for a core interface module.
// Produces: go.mod, <name>.go (interface + errors + options), <name>_test.go
func generateCore(spec *ModuleSpec) []FileToGenerate {
	pkg := spec.PackageName()
	mod := spec.FullModulePath()
	desc := spec.Description
	if desc == "" {
		desc = fmt.Sprintf("Package %s provides a core abstraction.", pkg)
	}

	goMod := fmt.Sprintf(`module %s

go 1.26.2

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
`, mod)

	coreGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// %s
package %s

import (
	"context"
	"errors"
	"time"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	// ErrNotFound is returned when a key/item is not found.
	ErrNotFound = errors.New("%s: not found")
	// ErrClosed is returned when operating on a closed instance.
	ErrClosed = errors.New("%s: closed")
	// ErrInvalidConfig is returned when configuration is invalid.
	ErrInvalidConfig = errors.New("%s: invalid config")
)

// ──────────────────────────────────────────────
// Options
// ──────────────────────────────────────────────

// Options holds configuration for a %s instance.
type Options struct {
	Prefix     string
	DefaultTTL time.Duration
}

// Option is a functional option for configuring Options.
type Option func(*Options)

// WithPrefix sets a key prefix.
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithDefaultTTL sets the default time-to-live.
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// ApplyOptions builds Options from a list of Option functions.
func ApplyOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// ──────────────────────────────────────────────
// Core interface
// ──────────────────────────────────────────────

// %s is the core interface for %s operations.
// Implementations should be safe for concurrent use.
type %s interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (any, error)
	// Set stores a value with the given TTL.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Delete removes a value by key.
	Delete(ctx context.Context, key string) error
	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)
	// Close releases all resources.
	Close() error
}
`, time.Now().Year(), desc, pkg, pkg, pkg, pkg, pkg, strings.Title(pkg), pkg, strings.Title(pkg))

	testGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package %s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyOptions(t *testing.T) {
	opts := ApplyOptions(
		WithPrefix("test:"),
		WithDefaultTTL(60),
	)
	assert.Equal(t, "test:", opts.Prefix)
	assert.Equal(t, time.Duration(60), opts.DefaultTTL)
}

func TestApplyOptions_Empty(t *testing.T) {
	opts := ApplyOptions()
	assert.Equal(t, "", opts.Prefix)
	assert.Equal(t, time.Duration(0), opts.DefaultTTL)
}

func TestApplyOptions_Nil(t *testing.T) {
	opts := ApplyOptions(nil, nil)
	assert.Equal(t, "", opts.Prefix)
}

func TestErrors(t *testing.T) {
	assert.Equal(t, "%s: not found", ErrNotFound.Error())
	assert.Equal(t, "%s: closed", ErrClosed.Error())
	assert.Equal(t, "%s: invalid config", ErrInvalidConfig.Error())
}
`, time.Now().Year(), pkg, pkg, pkg, pkg)

	return []FileToGenerate{
		{Path: spec.ModulePath() + "/go.mod", Content: goMod},
		{Path: spec.ModulePath() + "/" + pkg + ".go", Content: coreGo},
		{Path: spec.ModulePath() + "/" + pkg + "_test.go", Content: testGo},
	}
}

// ──────────────────────────────────────────────
// Backend module template
// ──────────────────────────────────────────────

// generateBackend generates files for a backend implementation module.
// Produces: go.mod, <name>.go (config + struct + constructor + methods), <name>_test.go
func generateBackend(spec *ModuleSpec) []FileToGenerate {
	pkg := spec.PackageName()
	mod := spec.FullModulePath()
	parentMod := spec.ParentModulePath()
	desc := spec.Description
	if desc == "" {
		desc = fmt.Sprintf("Package %s provides a %s backend implementation.", pkg, spec.Parent)
	}

	goMod := fmt.Sprintf(`module %s

go 1.26.2

require (
	%s v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace %s => ../
`, mod, parentMod, parentMod)

	implGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// %s
package %s

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	// %s — uncomment and use when implementing the interface.
	// "%s"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	ErrNotFound = errors.New("%s: not found")
	ErrClosed   = errors.New("%s: closed")
)

// ──────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────

// Config holds the configuration for a %s backend.
type Config struct {
	// Address is the backend server address.
	Address string
	// Username for authentication (optional).
	Username string
	// Password for authentication (optional).
	Password string
	// Database or namespace index.
	Database int
	// DialTimeout is the connection timeout.
	DialTimeout time.Duration
	// MaxRetries is the maximum number of retries on failure.
	MaxRetries int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Address:     "localhost:6379",
		DialTimeout: 5 * time.Second,
		MaxRetries:  3,
	}
}

// Validate checks the config for required fields.
func (c *Config) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("%s: address is required")
	}
	return nil
}

// ──────────────────────────────────────────────
// Backend implementation
// ──────────────────────────────────────────────

// Backend implements the %s interface using %s.
//
// TODO: Replace the placeholder methods with the actual interface
// from the %s package. Uncomment the import above and add:
//
//	var _ %s.Interface = (*Backend)(nil)
type Backend struct {
	mu     sync.RWMutex
	cfg    Config
	closed bool

	// TODO: Add backend client connection here.
	// client *xxx.Client
}

// Option is a functional option for the Backend.
type Option func(*Config)

// New creates a new %s Backend with the given config and options.
//
//	cfg := DefaultConfig()
//	cfg.Address = "localhost:6379"
//	backend, err := New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer backend.Close()
func New(cfg Config, opts ...Option) (*Backend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: invalid config: %%w", err)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	b := &Backend{cfg: cfg}

	// TODO: Initialize backend client connection.
	// client, err := xxx.Connect(cfg.Address)
	// if err != nil {
	//     return nil, fmt.Errorf("connect: %%w", err)
	// }
	// b.client = client

	return b, nil
}

// Get retrieves a value by key.
func (b *Backend) Get(ctx context.Context, key string) (any, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return nil, ErrClosed
	}
	// TODO: Implement Get.
	_ = ctx
	_ = key
	return nil, ErrNotFound
}

// Set stores a value with the given TTL.
func (b *Backend) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return ErrClosed
	}
	// TODO: Implement Set.
	_ = ctx
	_ = key
	_ = value
	_ = ttl
	return nil
}

// Delete removes a value by key.
func (b *Backend) Delete(ctx context.Context, key string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return ErrClosed
	}
	// TODO: Implement Delete.
	_ = ctx
	_ = key
	return nil
}

// Exists checks if a key exists.
func (b *Backend) Exists(ctx context.Context, key string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return false, ErrClosed
	}
	// TODO: Implement Exists.
	_ = ctx
	_ = key
	return false, nil
}

// Close releases all resources.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	// TODO: Close backend client connection.
	// return b.client.Close()
	return nil
}
`, time.Now().Year(), desc, pkg, parentMod, parentMod, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg)

	testGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package %s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.Address)
	assert.True(t, cfg.DialTimeout > 0)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", DefaultConfig(), false},
		{"empty address", Config{Address: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	_, err := New(Config{Address: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestBackend_Close(t *testing.T) {
	b, err := New(DefaultConfig())
	require.NoError(t, err)

	// Double close should not error.
	err = b.Close()
	assert.NoError(t, err)
	err = b.Close()
	assert.NoError(t, err)
}

func TestBackend_OperationsAfterClose(t *testing.T) {
	b, err := New(DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, b.Close())

	err = b.Set(nil, "key", "val", 0)
	assert.Error(t, err)
}
`, time.Now().Year(), pkg)

	return []FileToGenerate{
		{Path: spec.ModulePath() + "/go.mod", Content: goMod},
		{Path: spec.ModulePath() + "/" + pkg + ".go", Content: implGo},
		{Path: spec.ModulePath() + "/" + pkg + "_test.go", Content: testGo},
	}
}

// ──────────────────────────────────────────────
// Utility module template
// ──────────────────────────────────────────────

// generateUtil generates files for a utility module.
// Produces: go.mod, <name>.go (utility functions), <name>_test.go
func generateUtil(spec *ModuleSpec) []FileToGenerate {
	pkg := spec.PackageName()
	mod := spec.FullModulePath()
	desc := spec.Description
	if desc == "" {
		desc = fmt.Sprintf("Package %s provides utility functions.", pkg)
	}

	goMod := fmt.Sprintf(`module %s

go 1.26.2

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
`, mod)

	utilGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// %s
package %s

// ──────────────────────────────────────────────
// Utility functions
// ──────────────────────────────────────────────

// DoSomething is a placeholder utility function.
// Replace this with your actual utility implementation.
func DoSomething(input string) string {
	// TODO: Implement your utility logic here.
	return input
}

// IsEmpty checks if a string is empty or whitespace-only.
func IsEmpty(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// DefaultIfEmpty returns defaultVal if s is empty, otherwise s.
func DefaultIfEmpty(s, defaultVal string) string {
	if IsEmpty(s) {
		return defaultVal
	}
	return s
}
`, time.Now().Year(), desc, pkg)

	testGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package %s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoSomething(t *testing.T) {
	assert.Equal(t, "hello", DoSomething("hello"))
	assert.Equal(t, "", DoSomething(""))
}

func TestIsEmpty(t *testing.T) {
	assert.True(t, IsEmpty(""))
	assert.True(t, IsEmpty("   "))
	assert.True(t, IsEmpty("\t\n"))
	assert.False(t, IsEmpty("a"))
	assert.False(t, IsEmpty(" a "))
}

func TestDefaultIfEmpty(t *testing.T) {
	assert.Equal(t, "default", DefaultIfEmpty("", "default"))
	assert.Equal(t, "default", DefaultIfEmpty("  ", "default"))
	assert.Equal(t, "value", DefaultIfEmpty("value", "default"))
}
`, time.Now().Year(), pkg)

	return []FileToGenerate{
		{Path: spec.ModulePath() + "/go.mod", Content: goMod},
		{Path: spec.ModulePath() + "/" + pkg + ".go", Content: utilGo},
		{Path: spec.ModulePath() + "/" + pkg + "_test.go", Content: testGo},
	}
}

// ──────────────────────────────────────────────
// Provider adapter template
// ──────────────────────────────────────────────

// generateProvider generates files for a provider adapter module.
// Produces: go.mod, <name>.go (config + adapter + factory registration), <name>_test.go
func generateProvider(spec *ModuleSpec) []FileToGenerate {
	pkg := spec.PackageName()
	mod := spec.FullModulePath()
	parentMod := spec.ParentModulePath()
	desc := spec.Description
	if desc == "" {
		desc = fmt.Sprintf("Package %s provides a %s provider adapter.", pkg, spec.Parent)
	}

	goMod := fmt.Sprintf(`module %s

go 1.26.2

require (
	%s v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace %s => ../
`, mod, parentMod, parentMod)

	providerGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// %s
package %s

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	// %s — uncomment and use when implementing the interface.
	// "%s"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	ErrInvalidConfig = errors.New("%s: invalid config")
	ErrClosed        = errors.New("%s: closed")
)

// ──────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────

// Config holds the configuration for the %s provider.
type Config struct {
	// APIKey is the provider API key.
	APIKey string
	// APISecret is the provider API secret.
	APISecret string
	// Endpoint is the provider API endpoint (optional, uses default if empty).
	Endpoint string
	// Region is the provider region (optional).
	Region string
	// Timeout is the request timeout (optional, default 30s).
	Timeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
	}
}

// Validate checks the config for required fields.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("%%w: APIKey is required", ErrInvalidConfig)
	}
	return nil
}

// ──────────────────────────────────────────────
// Provider
// ──────────────────────────────────────────────

// Provider implements the %s interface for the %s provider.
//
// TODO: Replace the placeholder methods with the actual interface
// from the %s package. Uncomment the import above and add:
//
//	var _ %s.Interface = (*Provider)(nil)
type Provider struct {
	mu     sync.RWMutex
	cfg    Config
	closed bool

	// TODO: Add provider client here.
	// client *xxx.Client
}

// New creates a new %s Provider with the given config.
//
//	cfg := DefaultConfig()
//	cfg.APIKey = "your-api-key"
//	provider, err := New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
func New(cfg Config) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %%w", err)
	}

	p := &Provider{cfg: cfg}

	// TODO: Initialize provider client.
	// client, err := xxx.NewClient(cfg.APIKey, cfg.Endpoint)
	// if err != nil {
	//     return nil, fmt.Errorf("create client: %%w", err)
	// }
	// p.client = client

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return %q
}

// Send performs an operation via the provider.
// TODO: Replace with the actual interface method(s) from the parent module.
func (p *Provider) Send(ctx context.Context, data map[string]any) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return ErrClosed
	}
	// TODO: Implement provider API call.
	_ = ctx
	_ = data
	return nil
}

// Close releases all resources.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	// TODO: Close provider client.
	return nil
}
`, time.Now().Year(), desc, pkg, parentMod, parentMod, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg, pkg)

	testGo := fmt.Sprintf(`// Copyright (c) %d LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package %s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.True(t, cfg.Timeout > 0)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.APIKey = "test-key"
		assert.NoError(t, cfg.Validate())
	})

	t.Run("missing APIKey", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.Error(t, cfg.Validate())
	})
}

func TestNew_InvalidConfig(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestNew_Valid(t *testing.T) {
	p, err := New(Config{APIKey: "test-key"})
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "%s", p.Name())
}

func TestProvider_Close(t *testing.T) {
	p, err := New(Config{APIKey: "test-key"})
	require.NoError(t, err)

	err = p.Close()
	assert.NoError(t, err)

	// Double close.
	err = p.Close()
	assert.NoError(t, err)
}
`, time.Now().Year(), pkg, pkg)

	return []FileToGenerate{
		{Path: spec.ModulePath() + "/go.mod", Content: goMod},
		{Path: spec.ModulePath() + "/" + pkg + ".go", Content: providerGo},
		{Path: spec.ModulePath() + "/" + pkg + "_test.go", Content: testGo},
	}
}
