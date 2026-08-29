package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/common/bitmap"
	bmredis "github.com/LingByte/ling-base/common/bitmap/redis"
)

func newBM(t *testing.T, opts ...bmredis.Option) (*bmredis.Bitmap, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	all := append([]bmredis.Option{bmredis.WithKey("bm")}, opts...)
	b, err := bmredis.NewWithClient(client, all...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b, mr
}

func TestRedisSetGetClear(t *testing.T) {
	b, _ := newBM(t)
	ctx := context.Background()
	_ = b.Set(ctx, 10)
	if ok, _ := b.Get(ctx, 10); !ok {
		t.Fatal("missing")
	}
	if n, _ := b.Count(ctx); n != 1 {
		t.Fatalf("count=%d", n)
	}
	_ = b.Clear(ctx, 10)
	if ok, _ := b.Get(ctx, 10); ok {
		t.Fatal("still set")
	}
}

func TestRedisBatch(t *testing.T) {
	b, _ := newBM(t)
	ctx := context.Background()
	_ = b.SetBatch(ctx, []uint64{1, 5, 9})
	got, err := b.GetBatch(ctx, []uint64{1, 2, 5, 9})
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%v", i, got[i])
		}
	}
}

func TestRedisTTL(t *testing.T) {
	b, mr := newBM(t, bmredis.WithTTL(time.Second))
	ctx := context.Background()
	_ = b.Set(ctx, 1)
	if mr.TTL("bm") <= 0 {
		t.Fatal("expected TTL")
	}
}

func TestRedisReset(t *testing.T) {
	b, _ := newBM(t)
	ctx := context.Background()
	_ = b.Set(ctx, 3)
	_ = b.Reset(ctx)
	if n, _ := b.Count(ctx); n != 0 {
		t.Fatalf("count=%d", n)
	}
}

func TestRedisRequireKey(t *testing.T) {
	_, err := bmredis.NewWithClient(goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"}))
	if !errors.Is(err, bitmap.ErrEmptyKey) {
		t.Fatalf("got %v", err)
	}
}

func TestRedisBitOpOr(t *testing.T) {
	b, mr := newBM(t)
	ctx := context.Background()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	a, err := bmredis.NewWithClient(client, bmredis.WithKey("a"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := bmredis.NewWithClient(client, bmredis.WithKey("c"))
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Set(ctx, 1)
	_ = c.Set(ctx, 2)
	if err := b.BitOpOr(ctx, "dest", "a", "c"); err != nil {
		t.Fatal(err)
	}
	dest, err := bmredis.NewWithClient(client, bmredis.WithKey("dest"))
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := dest.Get(ctx, 1); !ok {
		t.Fatal("bit 1 missing")
	}
	if ok, _ := dest.Get(ctx, 2); !ok {
		t.Fatal("bit 2 missing")
	}
}
