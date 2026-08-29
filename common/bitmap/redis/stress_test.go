package redis_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	bmredis "github.com/LingByte/ling-base/common/bitmap/redis"
)

func TestRedisStressExact(t *testing.T) {
	const n = 10_000
	b, _ := newBM(t)
	ctx := context.Background()

	offs := make([]uint64, n)
	for i := range offs {
		offs[i] = uint64(i)
	}
	if err := b.SetBatch(ctx, offs); err != nil {
		t.Fatal(err)
	}
	cnt, err := b.Count(ctx)
	if err != nil || cnt != n {
		t.Fatalf("Count=%d err=%v want %d", cnt, err, n)
	}
	got, err := b.GetBatch(ctx, offs)
	if err != nil {
		t.Fatal(err)
	}
	for i, ok := range got {
		if !ok {
			t.Fatalf("missing offset %d", i)
		}
	}
	// clear half
	half := offs[: n/2]
	if err := b.ClearBatch(ctx, half); err != nil {
		t.Fatal(err)
	}
	if cnt, _ := b.Count(ctx); cnt != n/2 {
		t.Fatalf("after clear Count=%d want %d", cnt, n/2)
	}
}

func TestRedisStressConcurrent(t *testing.T) {
	const (
		workers = 16
		per     = 500
	)
	b, _ := newBM(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	var fails atomic.Int64

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			base := uint64(w * per)
			for i := uint64(0); i < per; i++ {
				off := base + i
				if err := b.Set(ctx, off); err != nil {
					fails.Add(1)
					return
				}
				ok, err := b.Get(ctx, off)
				if err != nil || !ok {
					fails.Add(1)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("concurrent failures: %d", fails.Load())
	}
	want := uint64(workers * per)
	if cnt, _ := b.Count(ctx); cnt != want {
		t.Fatalf("Count=%d want %d", cnt, want)
	}
}

func TestRedisStressBitOp(t *testing.T) {
	b, mr := newBM(t)
	ctx := context.Background()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	a, err := bmredis.NewWithClient(client, bmredis.WithKey("stress:a"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := bmredis.NewWithClient(client, bmredis.WithKey("stress:c"))
	if err != nil {
		t.Fatal(err)
	}
	const n = 2000
	offsA := make([]uint64, n)
	offsC := make([]uint64, n)
	for i := 0; i < n; i++ {
		offsA[i] = uint64(i)
		offsC[i] = uint64(i + n/2) // overlap half
	}
	_ = a.SetBatch(ctx, offsA)
	_ = c.SetBatch(ctx, offsC)
	if err := b.BitOpAnd(ctx, "stress:and", "stress:a", "stress:c"); err != nil {
		t.Fatal(err)
	}
	andBM, err := bmredis.NewWithClient(client, bmredis.WithKey("stress:and"))
	if err != nil {
		t.Fatal(err)
	}
	cnt, _ := andBM.Count(ctx)
	if cnt != uint64(n/2) {
		t.Fatalf("AND Count=%d want %d", cnt, n/2)
	}
}

func newBenchRedis(b *testing.B) (*bmredis.Bitmap, func()) {
	b.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	bm, err := bmredis.NewWithClient(client, bmredis.WithKey("bench"))
	if err != nil {
		b.Fatal(err)
	}
	return bm, func() {
		_ = bm.Close()
		_ = client.Close()
		mr.Close()
	}
}

func BenchmarkRedisSet(b *testing.B) {
	bm, cleanup := newBenchRedis(b)
	defer cleanup()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.Set(ctx, uint64(i%100_000))
	}
}

func BenchmarkRedisGet(b *testing.B) {
	bm, cleanup := newBenchRedis(b)
	defer cleanup()
	ctx := context.Background()
	for i := uint64(0); i < 10_000; i++ {
		_ = bm.Set(ctx, i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bm.Get(ctx, uint64(i%10_000))
	}
}

func BenchmarkRedisSetBatch1k(b *testing.B) {
	bm, cleanup := newBenchRedis(b)
	defer cleanup()
	ctx := context.Background()
	offs := make([]uint64, 1000)
	for i := range offs {
		offs[i] = uint64(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.SetBatch(ctx, offs)
	}
}
