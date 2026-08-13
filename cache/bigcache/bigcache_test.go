package bigcache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bc "github.com/allegro/bigcache/v3"

	"github.com/LingByte/ling-base/cache"
	cachebig "github.com/LingByte/ling-base/cache/bigcache"
)

func TestBigcacheNewValidation(t *testing.T) {
	if _, err := cachebig.New(0); err == nil {
		t.Fatal("expected lifeWindow error")
	}
	if _, err := cachebig.NewWithCache(nil); err == nil {
		t.Fatal("expected nil cache error")
	}
}

func TestBigcacheBasic(t *testing.T) {
	c, err := cachebig.New(time.Minute,
		cachebig.WithShards(4),
		cachebig.WithMaxEntriesInWindow(100),
		cachebig.WithMaxEntrySize(512),
		cachebig.WithHardMaxCacheSize(1),
		cachebig.WithDefaultTTL(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "v" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	ok, err := c.Exists(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	ok, err = c.Exists(ctx, "missing")
	if err != nil || ok {
		t.Fatalf("Exists miss = %v, %v", ok, err)
	}
	if err := c.Delete(ctx, "missing"); err != nil {
		t.Fatal(err)
	}
}

func TestBigcachePrefix(t *testing.T) {
	c, err := cachebig.New(time.Minute, cachebig.WithPrefix("app:"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	_ = c.Set(ctx, "k", []byte("v"), 0)
	if _, err := c.Get(ctx, "k"); err != nil {
		t.Fatal(err)
	}
}

func TestBigcacheNewWithCache(t *testing.T) {
	raw, err := bc.New(context.Background(), bc.DefaultConfig(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	c, err := cachebig.NewWithCache(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"), 0)
	if _, err := c.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
}

func TestBigcacheClosedAndErrors(t *testing.T) {
	c, err := cachebig.New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := c.Get(ctx, ""); !errors.Is(err, cache.ErrEmptyKey) {
		t.Fatalf("empty key = %v", err)
	}
	ctxC, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.Get(ctxC, "k"); err != context.Canceled {
		t.Fatalf("cancelled = %v", err)
	}
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("closed = %v", err)
	}
	if err := c.Clear(ctx); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("clear closed = %v", err)
	}
	if err := c.Set(ctx, "k", []byte("v"), 0); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("set closed = %v", err)
	}
	if err := c.Delete(ctx, "k"); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("delete closed = %v", err)
	}
	ok, err := c.Exists(ctx, "k")
	if err == nil || ok {
		t.Fatalf("exists closed = %v, %v", ok, err)
	}
}

func TestBigcacheSetCopiesValue(t *testing.T) {
	c, err := cachebig.New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	src := []byte("hello")
	if err := c.Set(ctx, "k", src, 0); err != nil {
		t.Fatal(err)
	}
	src[0] = 'H'
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "hello" {
		t.Fatalf("Get = %q, %v; want hello (Set must copy)", got, err)
	}
	if c.Len() != 1 {
		t.Fatalf("Len = %d", c.Len())
	}
	if c.LifeWindow() != time.Minute {
		t.Fatalf("LifeWindow = %v", c.LifeWindow())
	}
}

func TestBigcacheStrictTTL(t *testing.T) {
	c, err := cachebig.New(time.Minute, cachebig.WithStrictTTL())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), time.Second); !errors.Is(err, cachebig.ErrPerKeyTTLUnsupported) {
		t.Fatalf("strict ttl = %v", err)
	}
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
}

func TestBigcacheNewWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cachebig.NewWithContext(ctx, time.Minute); err == nil {
		t.Fatal("expected canceled context error")
	}
}

