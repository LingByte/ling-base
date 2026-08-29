package memory_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LingByte/ling-base/common/bitmap/memory"
)

// Stress: 100k sequential sets must be exact (no false negatives / extras).
func TestMemoryStressDenseExact(t *testing.T) {
	const n = 100_000
	b := newBM(t, memory.WithCapacity(n))
	ctx := context.Background()

	for i := uint64(0); i < n; i++ {
		if err := b.Set(ctx, i); err != nil {
			t.Fatalf("Set(%d): %v", i, err)
		}
	}
	cnt, err := b.Count(ctx)
	if err != nil || cnt != n {
		t.Fatalf("Count=%d err=%v want %d", cnt, err, n)
	}
	for i := uint64(0); i < n; i++ {
		ok, err := b.Get(ctx, i)
		if err != nil || !ok {
			t.Fatalf("Get(%d)=%v err=%v", i, ok, err)
		}
	}
	// unset range must stay false
	for i := uint64(n); i < n+1000; i++ {
		ok, err := b.Get(ctx, i)
		if err != nil || ok {
			t.Fatalf("Get(%d) unexpectedly set", i)
		}
	}
}

// Stress: concurrent writers/readers on overlapping offsets.
func TestMemoryStressConcurrent(t *testing.T) {
	const (
		workers = 64
		per     = 2000
	)
	b := newBM(t)
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

// Stress: batch path under load + snapshot round-trip of large bitmap.
func TestMemoryStressBatchAndSnapshot(t *testing.T) {
	const n = 50_000
	b := newBM(t)
	ctx := context.Background()

	offs := make([]uint64, n)
	for i := range offs {
		offs[i] = uint64(i * 3) // slightly sparse
	}
	if err := b.SetBatch(ctx, offs); err != nil {
		t.Fatal(err)
	}
	got, err := b.GetBatch(ctx, offs)
	if err != nil {
		t.Fatal(err)
	}
	for i, ok := range got {
		if !ok {
			t.Fatalf("batch miss at %d", offs[i])
		}
	}

	var buf bytes.Buffer
	if _, err := b.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	b2 := newBM(t)
	if _, err := b2.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	cnt, _ := b2.Count(ctx)
	if cnt != n {
		t.Fatalf("restored Count=%d want %d", cnt, n)
	}
}

func BenchmarkMemorySet(b *testing.B) {
	bm, err := memory.New(memory.WithFixed(1_000_000))
	if err != nil {
		b.Fatal(err)
	}
	defer bm.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.Set(ctx, uint64(i%1_000_000))
	}
}

func BenchmarkMemoryGet(b *testing.B) {
	bm, err := memory.New(memory.WithFixed(1_000_000))
	if err != nil {
		b.Fatal(err)
	}
	defer bm.Close()
	ctx := context.Background()
	for i := uint64(0); i < 100_000; i++ {
		_ = bm.Set(ctx, i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bm.Get(ctx, uint64(i%100_000))
	}
}

func BenchmarkMemorySetBatch1k(b *testing.B) {
	bm, err := memory.New()
	if err != nil {
		b.Fatal(err)
	}
	defer bm.Close()
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

func BenchmarkMemoryParallelSet(b *testing.B) {
	bm, err := memory.New(memory.WithFixed(1_000_000))
	if err != nil {
		b.Fatal(err)
	}
	defer bm.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i uint64
		for pb.Next() {
			_ = bm.Set(ctx, i%1_000_000)
			i++
		}
	})
}
