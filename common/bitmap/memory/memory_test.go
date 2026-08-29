package memory_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/LingByte/ling-base/common/bitmap"
	"github.com/LingByte/ling-base/common/bitmap/memory"
)

func newBM(t *testing.T, opts ...memory.Option) *memory.Bitmap {
	t.Helper()
	b, err := memory.New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func TestMemorySetGetClear(t *testing.T) {
	b := newBM(t)
	ctx := context.Background()

	if ok, _ := b.Get(ctx, 7); ok {
		t.Fatal("expected unset")
	}
	if err := b.Set(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b.Get(ctx, 7); !ok {
		t.Fatal("expected set")
	}
	if n, _ := b.Count(ctx); n != 1 {
		t.Fatalf("count=%d", n)
	}
	_ = b.Clear(ctx, 7)
	if ok, _ := b.Get(ctx, 7); ok {
		t.Fatal("expected cleared")
	}
}

func TestMemoryGrow(t *testing.T) {
	b := newBM(t)
	ctx := context.Background()
	if err := b.Set(ctx, 1000); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b.Get(ctx, 1000); !ok {
		t.Fatal("missing after grow")
	}
}

func TestMemoryFixed(t *testing.T) {
	b := newBM(t, memory.WithFixed(64))
	ctx := context.Background()
	if err := b.Set(ctx, 63); err != nil {
		t.Fatal(err)
	}
	if err := b.Set(ctx, 64); !errors.Is(err, bitmap.ErrOffsetOutOfRange) {
		t.Fatalf("want ErrOffsetOutOfRange, got %v", err)
	}
}

func TestMemoryBatch(t *testing.T) {
	b := newBM(t)
	ctx := context.Background()
	offs := []uint64{1, 2, 100}
	if err := b.SetBatch(ctx, offs); err != nil {
		t.Fatal(err)
	}
	got, err := b.GetBatch(ctx, []uint64{1, 2, 3, 100})
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, true, false, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%v want %v", i, got[i], want[i])
		}
	}
}

func TestMemorySnapshot(t *testing.T) {
	b := newBM(t, memory.WithFixed(128))
	ctx := context.Background()
	_ = b.Set(ctx, 3)
	_ = b.Set(ctx, 99)

	var buf bytes.Buffer
	if _, err := b.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	b2 := newBM(t)
	if _, err := b2.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b2.Get(ctx, 3); !ok {
		t.Fatal("3 missing after restore")
	}
	if ok, _ := b2.Get(ctx, 99); !ok {
		t.Fatal("99 missing after restore")
	}
	if n, _ := b2.Count(ctx); n != 2 {
		t.Fatalf("count=%d", n)
	}
}

func TestMemoryResetClose(t *testing.T) {
	b := newBM(t)
	ctx := context.Background()
	_ = b.Set(ctx, 1)
	_ = b.Reset(ctx)
	if n, _ := b.Count(ctx); n != 0 {
		t.Fatal("not empty after reset")
	}
	_ = b.Close()
	if err := b.Set(ctx, 1); !errors.Is(err, bitmap.ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
}

func TestMemoryConcurrent(t *testing.T) {
	b := newBM(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = b.Set(ctx, uint64(i))
			_, _ = b.Get(ctx, uint64(i))
		}(i)
	}
	wg.Wait()
	if n, _ := b.Count(ctx); n != 32 {
		t.Fatalf("count=%d", n)
	}
}
