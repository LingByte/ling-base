// Package memory provides a simple concurrent in-memory cache with TTL.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/ling-base/cache"
)

// Cache is a plain map-backed cache (no eviction besides TTL).
type Cache struct {
	mu     sync.RWMutex
	items  map[string]entry
	opts   cache.Options
	closed bool
	stopGC chan struct{}
	wg     sync.WaitGroup
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

// New creates a memory cache.
func New(opts ...Option) *Cache {
	cfg := applyOptions(opts...)
	c := &Cache{
		items:  make(map[string]entry),
		opts:   cfg.Options,
		stopGC: make(chan struct{}),
	}
	if cfg.cleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop(cfg.cleanupInterval)
	}
	return c
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := c.check(ctx, key); err != nil {
		return nil, err
	}
	full := c.opts.Key(key)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, cache.ErrClosed
	}
	ent, ok := c.items[full]
	if !ok {
		return nil, cache.ErrNotFound
	}
	if expired(ent) {
		delete(c.items, full)
		return nil, cache.ErrNotFound
	}
	out := make([]byte, len(ent.value))
	copy(out, ent.value)
	return out, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.check(ctx, key); err != nil {
		return err
	}
	ttl = c.opts.ResolveTTL(ttl)
	val := make([]byte, len(value))
	copy(val, value)
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return cache.ErrClosed
	}
	c.items[c.opts.Key(key)] = entry{value: val, expiresAt: expiresAt}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.check(ctx, key); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return cache.ErrClosed
	}
	delete(c.items, c.opts.Key(key))
	return nil
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.check(ctx, key); err != nil {
		return false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false, cache.ErrClosed
	}
	ent, ok := c.items[c.opts.Key(key)]
	if !ok {
		return false, nil
	}
	if expired(ent) {
		delete(c.items, c.opts.Key(key))
		return false, nil
	}
	return true, nil
}

func (c *Cache) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return cache.ErrClosed
	}
	c.items = make(map[string]entry)
	return nil
}

func (c *Cache) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.stopGC)
	c.items = nil
	c.mu.Unlock()
	c.wg.Wait()
	return nil
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *Cache) check(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key == "" {
		return cache.ErrEmptyKey
	}
	return nil
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	defer c.wg.Done()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-c.stopGC:
			return
		case <-t.C:
			c.purge()
		}
	}
}

func (c *Cache) purge() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	for k, ent := range c.items {
		if !ent.expiresAt.IsZero() && !ent.expiresAt.After(now) {
			delete(c.items, k)
		}
	}
}

func expired(ent entry) bool {
	return !ent.expiresAt.IsZero() && !ent.expiresAt.After(time.Now())
}

var _ cache.Cache = (*Cache)(nil)
