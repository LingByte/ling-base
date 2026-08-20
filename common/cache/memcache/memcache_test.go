package memcache_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gomen "github.com/bradfitz/gomemcache/memcache"

	"github.com/LingByte/ling-base/common/cache"
	"github.com/LingByte/ling-base/common/cache/memcache"
)

type fakeClient struct {
	mu     sync.Mutex
	items  map[string]*gomen.Item
	getErr error
	setErr error
	delErr error
}

func newFake() *fakeClient {
	return &fakeClient{items: make(map[string]*gomen.Item)}
}

func (f *fakeClient) Get(key string) (*gomen.Item, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	it, ok := f.items[key]
	if !ok {
		return nil, gomen.ErrCacheMiss
	}
	return &gomen.Item{Key: it.Key, Value: append([]byte(nil), it.Value...), Expiration: it.Expiration}, nil
}

func (f *fakeClient) Set(item *gomen.Item) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.items[item.Key] = &gomen.Item{
		Key:        item.Key,
		Value:      append([]byte(nil), item.Value...),
		Expiration: item.Expiration,
	}
	return nil
}

func (f *fakeClient) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.delErr != nil {
		return f.delErr
	}
	if _, ok := f.items[key]; !ok {
		return gomen.ErrCacheMiss
	}
	delete(f.items, key)
	return nil
}

func newTestCache(t *testing.T, fc *fakeClient, opts ...memcache.Option) *memcache.Cache {
	t.Helper()
	c, err := memcache.NewWithClient(fc, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestMemcacheNewValidation(t *testing.T) {
	if _, err := memcache.New(nil); err == nil {
		t.Fatal("expected error for empty servers")
	}
	if _, err := memcache.NewWithClient(nil); err == nil {
		t.Fatal("expected error for nil client")
	}
	c, err := memcache.New([]string{"127.0.0.1:11211"},
		memcache.WithTimeout(time.Second),
		memcache.WithMaxIdleConns(5),
		memcache.WithPrefix("p:"),
		memcache.WithDefaultTTL(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
}

func TestMemcacheBasic(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc, memcache.WithPrefix("app:"))
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
	if err := c.Delete(ctx, "missing"); err != nil {
		t.Fatal(err)
	}
}

func TestMemcacheShortTTL(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc)
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	fc.mu.Lock()
	exp := fc.items["k"].Expiration
	fc.mu.Unlock()
	if exp != 1 {
		t.Fatalf("short TTL expiration = %d, want 1", exp)
	}
}

func TestMemcacheDefaultTTL(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc, memcache.WithDefaultTTL(2*time.Minute))
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	fc.mu.Lock()
	exp := fc.items["k"].Expiration
	fc.mu.Unlock()
	if exp != 120 {
		t.Fatalf("default TTL expiration = %d", exp)
	}
}

func TestMemcacheClient(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc)
	if c.Client() != fc {
		t.Fatal("Client() should return underlying client")
	}
}

func TestMemcacheErrors(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc)
	ctx := context.Background()

	if _, err := c.Get(ctx, ""); !errors.Is(err, cache.ErrEmptyKey) {
		t.Fatalf("empty key = %v", err)
	}
	if _, err := c.Get(ctx, "missing"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}

	fc.getErr = errors.New("get boom")
	if _, err := c.Get(ctx, "k"); err == nil || err.Error() != "get boom" {
		t.Fatalf("get err = %v", err)
	}
	fc.getErr = nil

	fc.setErr = errors.New("set boom")
	if err := c.Set(ctx, "k", []byte("v"), 0); err == nil {
		t.Fatal("expected set error")
	}
	fc.setErr = nil

	fc.delErr = errors.New("del boom")
	if err := c.Delete(ctx, "k"); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestMemcacheClosedAndClear(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc)
	ctx := context.Background()

	if err := c.Clear(ctx); err == nil {
		t.Fatal("expected Clear unsupported")
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
	ctxC, cancel := context.WithCancel(ctx)
	cancel()
	if err := c.Clear(ctxC); err != context.Canceled {
		t.Fatalf("clear cancelled = %v", err)
	}
}

func TestMemcacheExistsMiss(t *testing.T) {
	fc := newFake()
	c := newTestCache(t, fc)
	ok, err := c.Exists(context.Background(), "nope")
	if err != nil || ok {
		t.Fatalf("Exists miss = %v, %v", ok, err)
	}
}
