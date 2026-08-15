package multilevel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/cache"
	"github.com/LingByte/ling-base/cache/lru"
	"github.com/LingByte/ling-base/cache/memory"
	"github.com/LingByte/ling-base/cache/multilevel"
)

func newLevels(t *testing.T) (cache.Cache[string, []byte], cache.Cache[string, []byte]) {
	t.Helper()
	l1, err := lru.New[string, []byte](10, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	l2 := memory.New[string, []byte](memory.WithCleanupInterval(0))
	t.Cleanup(func() { _ = l1.Close(); _ = l2.Close() })
	return l1, l2
}

func TestMultilevelNewValidation(t *testing.T) {
	l1, l2 := newLevels(t)
	if _, err := multilevel.New[string, []byte](nil, l2); err == nil {
		t.Fatal("expected error for nil l1")
	}
	if _, err := multilevel.New[string, []byte](l1, nil); err == nil {
		t.Fatal("expected error for nil l2")
	}
}

func TestMultilevelL2FillL1(t *testing.T) {
	l1, l2 := newLevels(t)
	c, err := multilevel.New[string, []byte](l1, l2)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := l2.Set(ctx, "k", []byte("from-l2"), 0); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "from-l2" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	got2, err := l1.Get(ctx, "k")
	if err != nil || string(got2) != "from-l2" {
		t.Fatalf("L1 populated = %q, %v", got2, err)
	}
}

func TestMultilevelL1Hit(t *testing.T) {
	l1, l2 := newLevels(t)
	c, err := multilevel.New[string, []byte](l1, l2)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	_ = c.Set(ctx, "k", []byte("v"), time.Minute)
	_ = l2.Delete(ctx, "k")
	got, err := c.Get(ctx, "k")
	if err != nil || string(got) != "v" {
		t.Fatalf("L1 hit = %q, %v", got, err)
	}
}

func TestMultilevelSetDeleteExistsClear(t *testing.T) {
	l1, l2 := newLevels(t)
	c, err := multilevel.New[string, []byte](l1, l2)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "a", []byte("1"), 0); err != nil {
		t.Fatal(err)
	}
	ok, err := c.Exists(ctx, "a")
	if err != nil || !ok {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
	if err := c.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "a"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	_ = l2.Set(ctx, "b", []byte("2"), 0)
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMultilevelClosedAndErrors(t *testing.T) {
	l1, l2 := newLevels(t)
	c, err := multilevel.New[string, []byte](l1, l2)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ctxC, cancel := context.WithCancel(context.Background())
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
	if err := c.Clear(context.Background()); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("clear closed = %v", err)
	}
}

type stubCache struct {
	get    func(context.Context, string) ([]byte, error)
	set    func(context.Context, string, []byte, time.Duration) error
	del    func(context.Context, string) error
	exists func(context.Context, string) (bool, error)
	clear  func(context.Context) error
	close  func() error
}

func (s stubCache) Get(ctx context.Context, key string) ([]byte, error) {
	if s.get != nil {
		return s.get(ctx, key)
	}
	return nil, cache.ErrNotFound
}
func (s stubCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s.set != nil {
		return s.set(ctx, key, value, ttl)
	}
	return nil
}
func (s stubCache) Delete(ctx context.Context, key string) error {
	if s.del != nil {
		return s.del(ctx, key)
	}
	return nil
}
func (s stubCache) Exists(ctx context.Context, key string) (bool, error) {
	if s.exists != nil {
		return s.exists(ctx, key)
	}
	return false, nil
}
func (s stubCache) Clear(ctx context.Context) error {
	if s.clear != nil {
		return s.clear(ctx)
	}
	return nil
}
func (s stubCache) Close() error {
	if s.close != nil {
		return s.close()
	}
	return nil
}

func TestMultilevelErrorPaths(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("boom")

	l1Err := stubCache{get: func(context.Context, string) ([]byte, error) {
		return nil, boom
	}}
	l2, _ := lru.New[string, []byte](1, lru.WithCleanupInterval(0))
	defer l2.Close()
	c, err := multilevel.New[string, []byte](l1Err, l2)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Get(ctx, "k"); !errors.Is(err, boom) {
		t.Fatalf("L1 error = %v", err)
	}

	l1Miss := stubCache{}
	l2Miss, _ := lru.New[string, []byte](1, lru.WithCleanupInterval(0))
	defer l2Miss.Close()
	c2, _ := multilevel.New[string, []byte](l1Miss, l2Miss)
	defer c2.Close()
	if _, err := c2.Get(ctx, "k"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("L2 miss = %v", err)
	}

	l1OK := stubCache{}
	l2Fail := stubCache{set: func(context.Context, string, []byte, time.Duration) error { return boom }}
	c3, _ := multilevel.New[string, []byte](l1OK, l2Fail)
	defer c3.Close()
	if err := c3.Set(ctx, "k", []byte("v"), 0); !errors.Is(err, boom) {
		t.Fatalf("Set L2 fail = %v", err)
	}

	l1ExistsErr := stubCache{exists: func(context.Context, string) (bool, error) { return false, boom }}
	c4, _ := multilevel.New[string, []byte](l1ExistsErr, l2Miss)
	defer c4.Close()
	if _, err := c4.Exists(ctx, "k"); !errors.Is(err, boom) {
		t.Fatalf("Exists L1 err = %v", err)
	}

	ctxC, cancel := context.WithCancel(ctx)
	cancel()
	c5, _ := multilevel.New[string, []byte](l1OK, l2Miss)
	defer c5.Close()
	if err := c5.Clear(ctxC); err != context.Canceled {
		t.Fatalf("clear cancelled = %v", err)
	}
}
