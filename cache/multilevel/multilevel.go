// Package multilevel composes an L1 (local) and L2 (remote) cache.
package multilevel

import (
	"context"
	"errors"
	"time"

	"github.com/LingByte/ling-base/cache"
)

// Cache reads L1 first, falls back to L2, and writes through to both.
type Cache struct {
	l1     cache.Cache
	l2     cache.Cache
	closed bool
}

// New creates a multilevel cache. Both l1 and l2 are required.
func New(l1, l2 cache.Cache) (*Cache, error) {
	if l1 == nil || l2 == nil {
		return nil, errors.New("multilevel: l1 and l2 are required")
	}
	return &Cache{l1: l1, l2: l2}, nil
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := c.check(ctx, key); err != nil {
		return nil, err
	}
	if val, err := c.l1.Get(ctx, key); err == nil {
		return val, nil
	} else if !errors.Is(err, cache.ErrNotFound) {
		return nil, err
	}

	val, err := c.l2.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	_ = c.l1.Set(ctx, key, val, 0)
	return val, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.check(ctx, key); err != nil {
		return err
	}
	if err := c.l2.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	return c.l1.Set(ctx, key, value, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.check(ctx, key); err != nil {
		return err
	}
	_ = c.l1.Delete(ctx, key)
	return c.l2.Delete(ctx, key)
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.check(ctx, key); err != nil {
		return false, err
	}
	if ok, err := c.l1.Exists(ctx, key); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return c.l2.Exists(ctx, key)
}

func (c *Cache) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.closed {
		return cache.ErrClosed
	}
	_ = c.l1.Clear(ctx)
	return c.l2.Clear(ctx)
}

func (c *Cache) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	err1 := c.l1.Close()
	err2 := c.l2.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *Cache) check(ctx context.Context, key string) error {
	if c.closed {
		return cache.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if key == "" {
		return cache.ErrEmptyKey
	}
	return nil
}

var _ cache.Cache = (*Cache)(nil)
