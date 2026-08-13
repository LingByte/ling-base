// Package noop provides a no-op cache useful for tests and feature flags.
package noop

import (
	"context"
	"time"

	"github.com/LingByte/ling-base/cache"
)

// Cache discards all writes and always misses on reads.
type Cache struct{}

// New returns a no-op cache.
func New() *Cache { return &Cache{} }

func (Cache) Get(context.Context, string) ([]byte, error) { return nil, cache.ErrNotFound }

func (Cache) Set(context.Context, string, []byte, time.Duration) error { return nil }

func (Cache) Delete(context.Context, string) error { return nil }

func (Cache) Exists(context.Context, string) (bool, error) { return false, nil }

func (Cache) Clear(context.Context) error { return nil }

func (Cache) Close() error { return nil }

var _ cache.Cache = (*Cache)(nil)
