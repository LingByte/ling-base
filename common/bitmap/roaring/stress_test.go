package roaring_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"

	bmroaring "github.com/LingByte/ling-base/common/bitmap/roaring"
)

func TestRoaringStressSparseExact(t *testing.T) {
	const n = 100_000
	b := bmroaring.New()
	t.Cleanup(func() { _ = b.Close() })
	ctx := context.Background()

	// Sparse: large gaps between offsets.
	for i := uint64(0); i < n; i++ {
		off := i * 1000
		if err := b.Set(ctx, off); err != nil {
			t.Fatalf("Set(%d): %v", off, err)
		}
	}
	cnt, err := b.Count(ctx)
	if err != nil || cnt != n {
		t.Fatalf("Count=%d err=%v want %d", cnt, err, n)
	}
	for i := uint64(0); i < n; i++ {
		off := i * 1000
		ok, err := b.Get(ctx, off)
		if err != nil || !ok {
			t.Fatalf("Get(%d)=%v err=%v", off, ok, err)
		}
		// neighbor must be unset
		ok, err = b.Get(ctx, off+1)
		if err != nil || ok {
			t.Fatalf("neighbor %d unexpectedly set", off+1)
		}
	}
}

func TestRoaringStressConcurrent(t *testing.T) {
	const (
		workers = 64
		per     = 2000
	)
	b := bmroaring.New()
	t.Cleanup(func() { _ = b.Close() })
	ctx := context.Background()
	var wg sync.WaitGroup
	var fails atomic.Int64

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			base := uint64(w * per * 10)
			for i := uint64(0); i < per; i++ {
				off := base + i*10
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

func TestRoaringStressSnapshotAndOps(t *testing.T) {
	const n = 20_000
	a := bmroaring.New()
	c := bmroaring.New()
	t.Cleanup(func() {
		_ = a.Close()
		_ = c.Close()
	})
	ctx := context.Background()

	offsA := make([]uint64, n)
	offsC := make([]uint64, n)
	for i := 0; i < n; i++ {
		offsA[i] = uint64(i * 2)
		offsC[i] = uint64(i*2 + 1)
	}
	if err := a.SetBatch(ctx, offsA); err != nil {
		t.Fatal(err)
	}
	if err := c.SetBatch(ctx, offsC); err != nil {
		t.Fatal(err)
	}
	if err := a.OrInPlace(c); err != nil {
		t.Fatal(err)
	}
	if cnt, _ := a.Count(ctx); cnt != uint64(2*n) {
		t.Fatalf("union Count=%d want %d", cnt, 2*n)
	}

	var buf bytes.Buffer
	if _, err := a.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	restored := bmroaring.New()
	t.Cleanup(func() { _ = restored.Close() })
	if _, err := restored.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if cnt, _ := restored.Count(ctx); cnt != uint64(2*n) {
		t.Fatalf("restored Count=%d", cnt)
	}
}

func BenchmarkRoaringSetSparse(b *testing.B) {
	bm := bmroaring.New()
	defer bm.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Keep offsets in uint32 range; stride keeps the set sparse.
		_ = bm.Set(ctx, uint64(i%200_000)*100)
	}
}

func BenchmarkRoaringSetDense(b *testing.B) {
	bm := bmroaring.New()
	defer bm.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.Set(ctx, uint64(i%1_000_000))
	}
}

func BenchmarkRoaringGet(b *testing.B) {
	bm := bmroaring.New()
	defer bm.Close()
	ctx := context.Background()
	for i := uint64(0); i < 100_000; i++ {
		_ = bm.Set(ctx, i*10)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bm.Get(ctx, uint64(i%100_000)*10)
	}
}

func BenchmarkRoaringSetBatch1k(b *testing.B) {
	bm := bmroaring.New()
	defer bm.Close()
	ctx := context.Background()
	offs := make([]uint64, 1000)
	for i := range offs {
		offs[i] = uint64(i) * 17
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.SetBatch(ctx, offs)
	}
}

func BenchmarkRoaringParallelSet(b *testing.B) {
	bm := bmroaring.New()
	defer bm.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i uint64
		for pb.Next() {
			_ = bm.Set(ctx, (i%100_000)*13)
			i++
		}
	})
}
