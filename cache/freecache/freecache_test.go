package freecache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	fc "github.com/coocood/freecache"

	"github.com/LingByte/ling-base/cache"
	cachefree "github.com/LingByte/ling-base/cache/freecache"
)

func TestFreecacheNewValidation(t *testing.T) {
	if _, err := cachefree.New(0); !errors.Is(err, cache.ErrInvalidCapacity) {
		t.Fatalf("capacity = %v", err)
	}
	if _, err := cachefree.NewWithCache(nil); err == nil {
		t.Fatal("expected nil cache error")
	}
}

func TestFreecacheBasic(t *testing.T) {
	c, err := cachefree.New(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
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
}

func TestFreecacheShortTTL(t *testing.T) {
	c, err := cachefree.New(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "v" {
		t.Fatalf("Get short TTL = %q, %v", got, err)
	}
}

func TestFreecachePrefixDefaultTTL(t *testing.T) {
	c, err := cachefree.New(1<<20,
		cachefree.WithPrefix("p:"),
		cachefree.WithDefaultTTL(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	ok, _ := c.Exists(ctx, "k")
	if !ok {
		t.Fatal("expected key")
	}
}

func TestFreecacheNewWithCache(t *testing.T) {
	raw := fc.NewCache(1 << 20)
	c, err := cachefree.NewWithCache(raw, cachefree.WithPrefix("x:"))
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

func TestFreecacheClosedAndErrors(t *testing.T) {
	c, err := cachefree.New(1 << 20)
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

	_ = c.Close()
	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("get closed = %v", err)
	}
	if err := c.Set(ctx, "k", []byte("v"), 0); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("set closed = %v", err)
	}
	if err := c.Delete(ctx, "k"); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("delete closed = %v", err)
	}

	c2, _ := cachefree.New(1 << 20)
	if err := c2.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	_ = c2.Close()
	if err := c2.Clear(ctx); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("clear closed = %v", err)
	}
	ctxC2, cancel2 := context.WithCancel(ctx)
	cancel2()
	if err := c2.Clear(ctxC2); err != context.Canceled {
		t.Fatalf("clear cancelled = %v", err)
	}
}
