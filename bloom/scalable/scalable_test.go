package scalable_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/LingByte/ling-base/bloom"
	"github.com/LingByte/ling-base/bloom/scalable"
)

func newFilter(t *testing.T, opts ...scalable.Option) *scalable.Filter {
	t.Helper()
	f, err := scalable.New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestScalableBasic(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	_ = f.Add(ctx, "a")
	_ = f.Add(ctx, "b")
	if ok, _ := f.Test(ctx, "a"); !ok {
		t.Fatal("a missing")
	}
	if ok, _ := f.Test(ctx, "c"); ok {
		t.Fatal("c should be absent")
	}
}

func TestScalableGrows(t *testing.T) {
	// Tiny initial capacity forces growth.
	f := newFilter(t,
		scalable.WithInitialCapacity(50),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	if f.NumSubFilters() != 1 {
		t.Fatalf("start = %d, want 1", f.NumSubFilters())
	}
	const n = 5000
	for i := 0; i < n; i++ {
		if err := f.Add(ctx, key(i)); err != nil {
			t.Fatal(err)
		}
	}
	if f.NumSubFilters() < 2 {
		t.Fatalf("did not grow: %d sub-filters", f.NumSubFilters())
	}
	// No false negatives.
	for i := 0; i < n; i++ {
		if ok, _ := f.Test(ctx, key(i)); !ok {
			t.Fatalf("false negative for %s", key(i))
		}
	}
}

func TestScalableNoFalseNegatives(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(1000),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	const n = 20_000
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

func TestScalableReset(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if err := f.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be absent after reset")
	}
	if f.NumSubFilters() != 1 {
		t.Fatalf("after reset = %d, want 1", f.NumSubFilters())
	}
}

func TestScalableEmptyKey(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	if err := f.Add(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Add empty = %v", err)
	}
	if _, err := f.Test(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Test empty = %v", err)
	}
}

func TestScalableNewValidation(t *testing.T) {
	if _, err := scalable.New(scalable.WithFalsePositiveRate(0.01)); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("no capacity = %v", err)
	}
	if _, err := scalable.New(scalable.WithInitialCapacity(100)); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("no p = %v", err)
	}
	if _, err := scalable.New(
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
		scalable.WithGrowthRatio(1),
	); err == nil {
		t.Fatal("growth=1 should error")
	}
	if _, err := scalable.New(
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
		scalable.WithFPRRatio(1),
	); err == nil {
		t.Fatal("fpRatio=1 should error")
	}
}

func TestScalableClosed(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	_ = f.Close()
	if err := f.Add(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Add closed = %v", err)
	}
	if _, err := f.Test(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Test closed = %v", err)
	}
}

func TestScalableApproximateCount(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_ = f.Add(ctx, key(i))
	}
	// ApproximateCount counts Add calls that flipped at least one new bit, so
	// it is a lower bound on distinct elements and may be slightly less than
	// the number of Add calls when a key collides fully with existing bits.
	if c := f.ApproximateCount(); c > 100 || c < 95 {
		t.Fatalf("count = %d, want in [95, 100]", c)
	}
}

func TestScalableCtxCancelled(t *testing.T) {
	f := newFilter(t,
		scalable.WithInitialCapacity(100),
		scalable.WithFalsePositiveRate(0.01),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.Add(ctx, "x"); err != context.Canceled {
		t.Fatalf("Add cancelled = %v", err)
	}
}

func key(i int) string {
	return "key-" + strconv.Itoa(i)
}
