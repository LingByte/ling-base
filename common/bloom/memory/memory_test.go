package memory_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/LingByte/ling-base/common/bloom"
	"github.com/LingByte/ling-base/common/bloom/memory"
)

func newFilter(t *testing.T, opts ...memory.Option) *memory.Filter {
	t.Helper()
	f, err := memory.New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestMemoryAddTest(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(1000, 0.01))
	ctx := context.Background()

	if err := f.Add(ctx, "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := f.Add(ctx, "beta"); err != nil {
		t.Fatal(err)
	}

	if ok, err := f.Test(ctx, "alpha"); err != nil || !ok {
		t.Fatalf("alpha = %v, %v", ok, err)
	}
	if ok, err := f.Test(ctx, "beta"); err != nil || !ok {
		t.Fatalf("beta = %v, %v", ok, err)
	}
	// "gamma" was never added; must be absent (no false negatives).
	if ok, err := f.Test(ctx, "gamma"); err != nil || ok {
		t.Fatalf("gamma = %v, %v (want false)", ok, err)
	}
}

func TestMemoryNoFalseNegatives(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(10_000, 0.01))
	ctx := context.Background()

	const n = 5000
	for i := 0; i < n; i++ {
		if err := f.Add(ctx, key(i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < n; i++ {
		if ok, err := f.Test(ctx, key(i)); err != nil || !ok {
			t.Fatalf("false negative for %s: ok=%v err=%v", key(i), ok, err)
		}
	}
}

func TestMemoryFalsePositiveRate(t *testing.T) {
	// With n=1000 and p=0.01, the observed FP rate over unseen keys should be
	// well below, say, 5% (we leave headroom for test stability).
	f := newFilter(t, memory.WithCapacity(1000, 0.01))
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		_ = f.Add(ctx, key(i))
	}

	fp := 0
	const probe = 10_000
	for i := 0; i < probe; i++ {
		// Use a disjoint key space.
		if ok, _ := f.Test(ctx, "absent-"+key(i)); ok {
			fp++
		}
	}
	if rate := float64(fp) / probe; rate > 0.05 {
		t.Fatalf("observed FP rate too high: %.4f", rate)
	}
}

func TestMemoryReset(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(1000, 0.01))
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if ok, _ := f.Test(ctx, "x"); !ok {
		t.Fatal("x should be present before reset")
	}
	if err := f.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be absent after reset")
	}
}

func TestMemoryEmptyKey(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(100, 0.01))
	ctx := context.Background()
	if err := f.Add(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Add empty = %v, want ErrEmptyKey", err)
	}
	if _, err := f.Test(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Test empty = %v, want ErrEmptyKey", err)
	}
}

func TestMemoryNewValidation(t *testing.T) {
	if _, err := memory.New(); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("no options = %v, want ErrInvalidCapacity", err)
	}
	if _, err := memory.New(memory.WithParams(bloom.Params{M: 0, K: 7})); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("m=0 = %v, want ErrInvalidCapacity", err)
	}
	if _, err := memory.New(memory.WithParams(bloom.Params{M: 100, K: 0})); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("k=0 = %v, want ErrInvalidFalsePositiveRate", err)
	}
}

func TestMemoryClosed(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(100, 0.01))
	ctx := context.Background()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err) // idempotent
	}
	if err := f.Add(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Add after close = %v, want ErrClosed", err)
	}
	if _, err := f.Test(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Test after close = %v, want ErrClosed", err)
	}
	if err := f.Reset(ctx); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Reset after close = %v, want ErrClosed", err)
	}
}

func TestMemoryConcurrent(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(10_000, 0.01))
	ctx := context.Background()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(off int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				_ = f.Add(ctx, key(off*500+i))
			}
		}(g)
	}
	wg.Wait()

	for g := 0; g < 8; g++ {
		for i := 0; i < 500; i++ {
			if ok, _ := f.Test(ctx, key(g*500+i)); !ok {
				t.Fatalf("missing %s", key(g*500+i))
			}
		}
	}
}

func TestMemoryCtxCancelled(t *testing.T) {
	f := newFilter(t, memory.WithCapacity(100, 0.01))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.Add(ctx, "x"); err != context.Canceled {
		t.Fatalf("Add cancelled = %v, want context.Canceled", err)
	}
}

func TestMemoryMK(t *testing.T) {
	f := newFilter(t, memory.WithParams(bloom.Params{M: 1000, K: 5}))
	if f.M() != 1000 || f.K() != 5 {
		t.Fatalf("M,K = %d,%d", f.M(), f.K())
	}
}

func key(i int) string {
	return "key-" + strconv.Itoa(i)
}
