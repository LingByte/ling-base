package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/cache"
	cacheredis "github.com/LingByte/ling-base/cache/redis"
)

func newTestCache(t *testing.T, opts ...cacheredis.Option) (*cacheredis.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c, err := cacheredis.NewWithClient(client, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, mr
}

func TestRedisBasic(t *testing.T) {
	c, _ := newTestCache(t)
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
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRedisTTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Second); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(2 * time.Second)
	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected expired, got %v", err)
	}
}

func TestRedisPrefixClear(t *testing.T) {
	c, _ := newTestCache(t, cacheredis.WithPrefix("app:"))
	ctx := context.Background()

	_ = c.Set(ctx, "a", []byte("1"), 0)
	_ = c.Set(ctx, "b", []byte("2"), 0)

	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "a"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected cleared, got %v", err)
	}
}

func TestRedisEmptyKey(t *testing.T) {
	c, _ := newTestCache(t)
	if _, err := c.Get(context.Background(), ""); !errors.Is(err, cache.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey, got %v", err)
	}
}

func TestRedisNewValidation(t *testing.T) {
	if _, err := cacheredis.New(nil); err == nil {
		t.Fatal("expected nil options error")
	}
	if _, err := cacheredis.NewWithClient(nil); err == nil {
		t.Fatal("expected nil client error")
	}
	if _, err := cacheredis.New(&goredis.Options{Addr: "127.0.0.1:59999"}); err == nil {
		t.Fatal("expected ping failure")
	}
}

func TestRedisNewWithMiniredis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	c, err := cacheredis.New(&goredis.Options{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Client() == nil {
		t.Fatal("Client() nil")
	}
}

func TestRedisDefaultTTL(t *testing.T) {
	c, _ := newTestCache(t, cacheredis.WithDefaultTTL(time.Hour))
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "k"); err != nil {
		t.Fatal(err)
	}
}

func TestRedisFlushDBClear(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"), 0)
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "a"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected flushed, got %v", err)
	}
}

func TestRedisClosedAndCancelled(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()
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
		t.Fatalf("closed Get = %v", err)
	}
	if err := c.Clear(ctx); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("clear closed = %v", err)
	}
}

func TestRedisClearCancelled(t *testing.T) {
	c, _ := newTestCache(t)
	ctxC, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Clear(ctxC); err != context.Canceled {
		t.Fatalf("clear cancelled = %v", err)
	}
}

func TestRedisCloseWithoutCloser(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	wrapped := struct{ goredis.Cmdable }{Cmdable: client}
	c, err := cacheredis.NewWithClient(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisExistsFalse(t *testing.T) {
	c, _ := newTestCache(t)
	ok, err := c.Exists(context.Background(), "missing")
	if err != nil || ok {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
}
