package lru_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/cache"
	"github.com/LingByte/ling-base/cache/lru"
)

func TestLRUBasic(t *testing.T) {
	c, err := lru.New(2, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "a", []byte("1"), 0); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "b", []byte("2"), 0); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(ctx, "a")
	if err != nil || string(got) != "1" {
		t.Fatalf("Get(a) = %q, %v; want 1", got, err)
	}

	// Access a so b becomes LRU, then insert c to evict b.
	if err := c.Set(ctx, "c", []byte("3"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "b"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected b evicted, got %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d, want 2", c.Len())
	}
}

func TestLRUTTL(t *testing.T) {
	c, err := lru.New(10, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if ok, err := c.Exists(ctx, "k"); err != nil || !ok {
		t.Fatalf("Exists before expire = %v, %v", ok, err)
	}

	time.Sleep(60 * time.Millisecond)
	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestLRUPrefixAndDefaultTTL(t *testing.T) {
	c, err := lru.New(10,
		lru.WithPrefix("app:"),
		lru.WithDefaultTTL(time.Hour),
		lru.WithCleanupInterval(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "user", []byte("u1"), 0); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "user")
	if err != nil || string(got) != "u1" {
		t.Fatalf("Get = %q, %v", got, err)
	}
}

func TestLRUDeleteClearClose(t *testing.T) {
	c, err := lru.New(5, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"), 0)
	_ = c.Set(ctx, "b", []byte("2"), 0)

	if err := c.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "a"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}

	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 0 {
		t.Fatalf("Len after Clear = %d", c.Len())
	}

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "x", []byte("y"), 0); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestLRUEmptyKeyAndInvalidCapacity(t *testing.T) {
	if _, err := lru.New(0); !errors.Is(err, cache.ErrInvalidCapacity) {
		t.Fatalf("expected ErrInvalidCapacity, got %v", err)
	}

	c, err := lru.New(1, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := c.Get(context.Background(), ""); !errors.Is(err, cache.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey, got %v", err)
	}
}

func TestLRUConcurrent(t *testing.T) {
	c, err := lru.New(100, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + (i % 26)))
			_ = c.Set(ctx, key, []byte{byte(i)}, 0)
			_, _ = c.Get(ctx, key)
			_, _ = c.Exists(ctx, key)
		}(i)
	}
	wg.Wait()
}

func TestLRUDoubleCloseAndCap(t *testing.T) {
	c, err := lru.New(7, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	if c.Cap() != 7 {
		t.Fatalf("Cap = %d", c.Cap())
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLRUCancelledContext(t *testing.T) {
	c, err := lru.New(1, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Get(ctx, "k"); err != context.Canceled {
		t.Fatalf("Get cancelled = %v", err)
	}
	if err := c.Clear(ctx); err != context.Canceled {
		t.Fatalf("Clear cancelled = %v", err)
	}
}

func TestLRUUpdateExisting(t *testing.T) {
	c, err := lru.New(5, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	_ = c.Set(ctx, "k", []byte("1"), 0)
	_ = c.Set(ctx, "k", []byte("2"), time.Hour)
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "2" {
		t.Fatalf("updated = %q, %v", got, err)
	}
	ok, _ := c.Exists(ctx, "missing")
	if ok {
		t.Fatal("missing should not exist")
	}
}

func TestLRUCleanup(t *testing.T) {
	c, err := lru.New(10, lru.WithCleanupInterval(20*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	_ = c.Set(ctx, "temp", []byte("x"), 30*time.Millisecond)
	time.Sleep(80 * time.Millisecond)
	if c.Len() != 0 {
		t.Fatalf("expected cleanup to purge expired key, Len=%d", c.Len())
	}
}
