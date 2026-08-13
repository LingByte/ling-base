// Package lru provides a thread-safe in-memory LRU cache with optional TTL.
package lru

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/LingByte/ling-base/cache"
)

// Cache is an in-memory LRU cache that implements cache.Cache.
type Cache struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[string]*list.Element
	opts     cache.Options
	closed   bool
	stopGC   chan struct{}
	wg       sync.WaitGroup
}

type entry struct {
	key       string
	value     []byte
	expiresAt time.Time // zero means no expiration
}

// New creates an LRU cache with the given capacity (number of entries).
// capacity must be greater than zero.
//
// If WithCleanupInterval is provided (or defaults when DefaultTTL / entries
// may expire), a background goroutine periodically purges expired entries.
func New(capacity int, opts ...Option) (*Cache, error) {
	if capacity <= 0 {
		return nil, cache.ErrInvalidCapacity
	}

	cfg := applyOptions(opts...)
	c := &Cache{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[string]*list.Element, capacity),
		opts:     cfg.Options,
		stopGC:   make(chan struct{}),
	}

	if cfg.cleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop(cfg.cleanupInterval)
	}

	return c, nil
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := c.check(ctx, key); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, cache.ErrClosed
	}

	elem, ok := c.items[c.opts.Key(key)]
	if !ok {
		return nil, cache.ErrNotFound
	}

	ent := elem.Value.(*entry)
	if isExpired(ent) {
		c.removeElement(elem)
		return nil, cache.ErrNotFound
	}

	c.ll.MoveToFront(elem)
	out := make([]byte, len(ent.value))
	copy(out, ent.value)
	return out, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.check(ctx, key); err != nil {
		return err
	}

	ttl = c.opts.ResolveTTL(ttl)
	fullKey := c.opts.Key(key)

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

	if elem, ok := c.items[fullKey]; ok {
		ent := elem.Value.(*entry)
		ent.value = val
		ent.expiresAt = expiresAt
		c.ll.MoveToFront(elem)
		return nil
	}

	elem := c.ll.PushFront(&entry{
		key:       fullKey,
		value:     val,
		expiresAt: expiresAt,
	})
	c.items[fullKey] = elem

	if c.ll.Len() > c.capacity {
		c.removeElement(c.ll.Back())
	}
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

	if elem, ok := c.items[c.opts.Key(key)]; ok {
		c.removeElement(elem)
	}
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

	elem, ok := c.items[c.opts.Key(key)]
	if !ok {
		return false, nil
	}
	ent := elem.Value.(*entry)
	if isExpired(ent) {
		c.removeElement(elem)
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

	c.ll.Init()
	c.items = make(map[string]*list.Element, c.capacity)
	return nil
}

// Len returns the number of entries currently stored (including expired ones
// that have not yet been purged).
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Cap returns the maximum number of entries.
func (c *Cache) Cap() int {
	return c.capacity
}

func (c *Cache) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.stopGC)
	c.ll.Init()
	c.items = nil
	c.mu.Unlock()

	c.wg.Wait()
	return nil
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

func (c *Cache) removeElement(elem *list.Element) {
	if elem == nil {
		return
	}
	ent := elem.Value.(*entry)
	delete(c.items, ent.key)
	c.ll.Remove(elem)
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	defer c.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopGC:
			return
		case <-ticker.C:
			c.purgeExpired()
		}
	}
}

func (c *Cache) purgeExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	for elem := c.ll.Back(); elem != nil; {
		prev := elem.Prev()
		ent := elem.Value.(*entry)
		if !ent.expiresAt.IsZero() && !ent.expiresAt.After(now) {
			c.removeElement(elem)
		}
		elem = prev
	}
}

func isExpired(ent *entry) bool {
	return !ent.expiresAt.IsZero() && !ent.expiresAt.After(time.Now())
}

// Ensure Cache implements cache.Cache.
var _ cache.Cache = (*Cache)(nil)
