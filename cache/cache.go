// Package cache defines a unified caching interface and shared errors
// for in-memory and distributed cache backends.
package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a key does not exist or has expired.
	ErrNotFound = errors.New("cache: key not found")

	// ErrClosed is returned when an operation is attempted on a closed cache.
	ErrClosed = errors.New("cache: closed")

	// ErrInvalidCapacity is returned when capacity is less than or equal to zero.
	ErrInvalidCapacity = errors.New("cache: capacity must be greater than zero")

	// ErrEmptyKey is returned when a key is empty.
	ErrEmptyKey = errors.New("cache: key must not be empty")
)

// Cache is the common interface implemented by all cache backends.
type Cache interface {
	// Get returns the value for key. Returns ErrNotFound if missing or expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores value under key. A ttl of 0 means no expiration (backend permitting).
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes key. It is not an error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Exists reports whether key is present and not expired.
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error

	// Close releases resources held by the cache.
	Close() error
}

// Getter is a read-only view of a cache.
type Getter interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// Setter is a write-only view of a cache.
type Setter interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
