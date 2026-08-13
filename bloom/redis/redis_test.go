package redis_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/bloom"
	bloomredis "github.com/LingByte/ling-base/bloom/redis"
)

func newFilter(t *testing.T, opts ...bloomredis.Option) (*bloomredis.Filter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	allOpts := append([]bloomredis.Option{bloomredis.WithKey("bf")}, opts...)
	f, err := bloomredis.NewWithClient(client, allOpts...)
	if err != nil {
		t.Fatalf("NewWithClient: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f, mr
}

func TestRedisAddTest(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(1000, 0.01))
	ctx := context.Background()

	_ = f.Add(ctx, "a")
	_ = f.Add(ctx, "b")
	if ok, _ := f.Test(ctx, "a"); !ok {
		t.Fatal("a missing")
	}
	if ok, _ := f.Test(ctx, "b"); !ok {
		t.Fatal("b missing")
	}
	if ok, _ := f.Test(ctx, "c"); ok {
		t.Fatal("c should be absent")
	}
}

func TestRedisNoFalseNegatives(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(10_000, 0.01))
	ctx := context.Background()
	const n = 3000
	for i := 0; i < n; i++ {
		if err := f.Add(ctx, key(i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < n; i++ {
		if ok, _ := f.Test(ctx, key(i)); !ok {
			t.Fatalf("false negative for %s", key(i))
		}
	}
}

func TestRedisBatch(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(5000, 0.01))
	ctx := context.Background()

	keys := []string{"x1", "x2", "x3"}
	if err := f.AddBatch(ctx, keys); err != nil {
		t.Fatal(err)
	}
	results, err := f.TestBatch(ctx, []string{"x1", "x2", "x3", "absent"})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0] || !results[1] || !results[2] {
		t.Fatalf("added keys missing: %v", results[:3])
	}
	if results[3] {
		t.Fatal("absent should be false")
	}
}

func TestRedisBatchEmpty(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(100, 0.01))
	ctx := context.Background()
	if err := f.AddBatch(ctx, nil); err != nil {
		t.Fatal(err)
	}
	res, err := f.TestBatch(ctx, nil)
	if err != nil || len(res) != 0 {
		t.Fatalf("TestBatch nil = %v, %v", res, err)
	}
}

func TestRedisReset(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(100, 0.01))
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if err := f.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be absent after reset")
	}
}

func TestRedisTTL(t *testing.T) {
	f, mr := newFilter(t,
		bloomredis.WithCapacity(100, 0.01),
		bloomredis.WithTTL(2*time.Second),
	)
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if ok, _ := f.Test(ctx, "x"); !ok {
		t.Fatal("x should be present")
	}
	mr.FastForward(3 * time.Second)
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be gone after TTL")
	}
}

func TestRedisSharedFilter(t *testing.T) {
	// Two filters on the same Redis key + geometry share state.
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	addr := mr.Addr()

	c1 := goredis.NewClient(&goredis.Options{Addr: addr})
	defer c1.Close()
	c2 := goredis.NewClient(&goredis.Options{Addr: addr})
	defer c2.Close()

	opts := []bloomredis.Option{
		bloomredis.WithKey("shared"),
		bloomredis.WithCapacity(1000, 0.01),
	}
	f1, err := bloomredis.NewWithClient(c1, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()
	f2, err := bloomredis.NewWithClient(c2, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	ctx := context.Background()
	_ = f1.Add(ctx, "shared-key")
	if ok, _ := f2.Test(ctx, "shared-key"); !ok {
		t.Fatal("f2 should see key added by f1")
	}
}

func TestRedisEmptyKey(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(100, 0.01))
	ctx := context.Background()
	if err := f.Add(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Add empty = %v", err)
	}
	if _, err := f.Test(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Test empty = %v", err)
	}
	if err := f.AddBatch(ctx, []string{"ok", ""}); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("AddBatch with empty = %v", err)
	}
}

func TestRedisNewValidation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()

	if _, err := bloomredis.NewWithClient(nil, bloomredis.WithKey("k"), bloomredis.WithCapacity(100, 0.01)); err == nil {
		t.Fatal("nil client should error")
	}
	if _, err := bloomredis.NewWithClient(client, bloomredis.WithCapacity(100, 0.01)); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("no key = %v", err)
	}
	if _, err := bloomredis.NewWithClient(client, bloomredis.WithKey("k")); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("no capacity = %v", err)
	}
	if _, err := bloomredis.NewWithClient(client, bloomredis.WithKey("k"), bloomredis.WithParams(bloom.Params{M: 100, K: 0})); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("k=0 = %v", err)
	}
	if _, err := bloomredis.New(nil, bloomredis.WithKey("k"), bloomredis.WithCapacity(100, 0.01)); err == nil {
		t.Fatal("nil redis opts should error")
	}
}

func TestRedisClosed(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(100, 0.01))
	ctx := context.Background()
	_ = f.Close()
	if err := f.Add(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Add closed = %v", err)
	}
	if _, err := f.Test(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Test closed = %v", err)
	}
	if _, err := f.TestBatch(ctx, []string{"x"}); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("TestBatch closed = %v", err)
	}
	if err := f.Reset(ctx); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Reset closed = %v", err)
	}
}

func TestRedisCtxCancelled(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithCapacity(100, 0.01))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.Add(ctx, "x"); err != context.Canceled {
		t.Fatalf("Add cancelled = %v, want context.Canceled", err)
	}
}

func TestRedisMK(t *testing.T) {
	f, _ := newFilter(t, bloomredis.WithParams(bloom.Params{M: 1000, K: 5}))
	if f.M() != 1000 || f.K() != 5 {
		t.Fatalf("M,K = %d,%d", f.M(), f.K())
	}
	if f.Key() != "bf" {
		t.Fatalf("Key = %q", f.Key())
	}
}

func key(i int) string {
	return "key-" + strconv.Itoa(i)
}
