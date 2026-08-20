package ristretto_test

import (
	"context"
	"errors"
	"testing"
	"time"

	rist "github.com/dgraph-io/ristretto/v2"

	"github.com/LingByte/ling-base/common/cache"
	cacherist "github.com/LingByte/ling-base/common/cache/ristretto"
)

func TestRistrettoNewValidation(t *testing.T) {
	if _, err := cacherist.New(0, 10); !errors.Is(err, cache.ErrInvalidCapacity) {
		t.Fatalf("numCounters = %v", err)
	}
	if _, err := cacherist.New(10, 0); !errors.Is(err, cache.ErrInvalidCapacity) {
		t.Fatalf("maxCost = %v", err)
	}
	if _, err := cacherist.NewWithCache(nil); err == nil {
		t.Fatal("expected nil cache error")
	}
}

func TestRistrettoBasic(t *testing.T) {
	c, err := cacherist.New(1<<16, 1<<20,
		cacherist.WithPrefix("r:"),
		cacherist.WithDefaultTTL(time.Hour),
		cacherist.WithCost(2),
	)
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
	if err := c.Set(ctx, "k2", []byte("v2"), 0); err != nil {
		t.Fatal(err)
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
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRistrettoNewWithCache(t *testing.T) {
	rc, err := rist.NewCache(&rist.Config[string, []byte]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := cacherist.NewWithCache(rc)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()
	if err := c.Set(ctx, "a", []byte("1"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	}
}

func TestRistrettoClosedAndErrors(t *testing.T) {
	c, err := cacherist.New(1<<10, 1<<16)
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
	ctxC2, cancel2 := context.WithCancel(ctx)
	cancel2()
	if err := c.Clear(ctxC2); err != context.Canceled {
		t.Fatalf("clear cancelled = %v", err)
	}
}
