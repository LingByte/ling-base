package counting_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/LingByte/ling-base/bloom"
	"github.com/LingByte/ling-base/bloom/counting"
)

func newFilter(t *testing.T, opts ...counting.Option) *counting.Filter {
	t.Helper()
	f, err := counting.New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestCountingAddTestRemove(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(1000, 0.01))
	ctx := context.Background()

	_ = f.Add(ctx, "a")
	_ = f.Add(ctx, "b")
	if ok, _ := f.Test(ctx, "a"); !ok {
		t.Fatal("a should be present")
	}

	if err := f.Remove(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "a"); ok {
		t.Fatal("a should be absent after remove")
	}
	if ok, _ := f.Test(ctx, "b"); !ok {
		t.Fatal("b should still be present")
	}
}

func TestCountingNoFalseNegatives(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(10_000, 0.01))
	ctx := context.Background()
	const n = 3000
	for i := 0; i < n; i++ {
		_ = f.Add(ctx, key(i))
	}
	for i := 0; i < n; i++ {
		if ok, _ := f.Test(ctx, key(i)); !ok {
			t.Fatalf("false negative for %s", key(i))
		}
	}
}

func TestCountingRemoveRestores(t *testing.T) {
	// Removing all added elements should leave the filter effectively empty
	// for those keys (no false negatives are possible, but after removal the
	// keys should report absent).
	f := newFilter(t, counting.WithCapacity(5000, 0.01))
	ctx := context.Background()
	const n = 1000
	for i := 0; i < n; i++ {
		_ = f.Add(ctx, key(i))
	}
	for i := 0; i < n; i++ {
		_ = f.Remove(ctx, key(i))
	}
	absent := 0
	for i := 0; i < n; i++ {
		if ok, _ := f.Test(ctx, key(i)); !ok {
			absent++
		}
	}
	// Most (ideally all) should be absent after balanced add/remove.
	if absent < n*9/10 {
		t.Fatalf("only %d/%d absent after balanced remove", absent, n)
	}
}

func TestCountingRemoveUnseenNoPanic(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(100, 0.01))
	ctx := context.Background()
	// Removing a never-added key must not panic and must not go negative.
	if err := f.Remove(ctx, "ghost"); err != nil {
		t.Fatal(err)
	}
	// Counters should remain zero.
	if ok, _ := f.Test(ctx, "ghost"); ok {
		t.Fatal("ghost should be absent")
	}
}

func TestCountingSaturates(t *testing.T) {
	f := newFilter(t, counting.WithParams(bloom.Params{M: 64, K: 3}))
	ctx := context.Background()
	// Add the same key many times; counters must saturate at MaxCounter.
	for i := 0; i < 100; i++ {
		_ = f.Add(ctx, "sat")
	}
	if ok, _ := f.Test(ctx, "sat"); !ok {
		t.Fatal("sat should be present")
	}
	// Remove once; still present because counters are saturated high.
	_ = f.Remove(ctx, "sat")
	if ok, _ := f.Test(ctx, "sat"); !ok {
		t.Fatal("sat should still be present after one remove (saturated)")
	}
}

func TestCountingReset(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(100, 0.01))
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if err := f.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be absent after reset")
	}
}

func TestCountingEmptyKey(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(100, 0.01))
	ctx := context.Background()
	if err := f.Add(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Add empty = %v", err)
	}
	if err := f.Remove(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Remove empty = %v", err)
	}
	if _, err := f.Test(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Test empty = %v", err)
	}
}

func TestCountingNewValidation(t *testing.T) {
	if _, err := counting.New(); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("no options = %v", err)
	}
	if _, err := counting.New(counting.WithParams(bloom.Params{M: 100, K: 0})); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("k=0 = %v", err)
	}
}

func TestCountingClosed(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(100, 0.01))
	ctx := context.Background()
	_ = f.Close()
	if err := f.Add(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Add closed = %v", err)
	}
	if err := f.Remove(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Remove closed = %v", err)
	}
	if _, err := f.Test(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Test closed = %v", err)
	}
}

func TestCountingConcurrent(t *testing.T) {
	f := newFilter(t, counting.WithCapacity(10_000, 0.01))
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

func TestCountingRemoverInterface(t *testing.T) {
	var r bloom.Remover = newFilter(t, counting.WithCapacity(100, 0.01))
	ctx := context.Background()
	_ = r.Remove(ctx, "x") // compiles + runs
}

func key(i int) string {
	return "key-" + strconv.Itoa(i)
}
