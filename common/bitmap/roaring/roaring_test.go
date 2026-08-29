package roaring_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/LingByte/ling-base/common/bitmap"
	bmroaring "github.com/LingByte/ling-base/common/bitmap/roaring"
)

func TestRoaringSetGetClear(t *testing.T) {
	b := bmroaring.New()
	t.Cleanup(func() { _ = b.Close() })
	ctx := context.Background()

	_ = b.Set(ctx, 1)
	_ = b.Set(ctx, 1_000_000)
	if ok, _ := b.Get(ctx, 1_000_000); !ok {
		t.Fatal("sparse bit missing")
	}
	if n, _ := b.Count(ctx); n != 2 {
		t.Fatalf("count=%d", n)
	}
	_ = b.Clear(ctx, 1)
	if ok, _ := b.Get(ctx, 1); ok {
		t.Fatal("should be cleared")
	}
}

func TestRoaringBatchAndOr(t *testing.T) {
	a := bmroaring.New()
	c := bmroaring.New()
	t.Cleanup(func() {
		_ = a.Close()
		_ = c.Close()
	})
	ctx := context.Background()
	_ = a.SetBatch(ctx, []uint64{1, 2, 3})
	_ = c.SetBatch(ctx, []uint64{2, 3, 4})

	if err := a.AndInPlace(c); err != nil {
		t.Fatal(err)
	}
	got, _ := a.GetBatch(ctx, []uint64{1, 2, 3, 4})
	want := []bool{false, true, true, false}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("and got[%d]=%v", i, got[i])
		}
	}
}

func TestRoaringSnapshot(t *testing.T) {
	b := bmroaring.New()
	t.Cleanup(func() { _ = b.Close() })
	ctx := context.Background()
	_ = b.Set(ctx, 42)
	_ = b.Set(ctx, 99)

	var buf bytes.Buffer
	if _, err := b.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	b2 := bmroaring.New()
	t.Cleanup(func() { _ = b2.Close() })
	if _, err := b2.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b2.Get(ctx, 42); !ok {
		t.Fatal("restore failed")
	}
}

func TestRoaringClosed(t *testing.T) {
	b := bmroaring.New()
	ctx := context.Background()
	_ = b.Close()
	if err := b.Set(ctx, 1); !errors.Is(err, bitmap.ErrClosed) {
		t.Fatalf("got %v", err)
	}
}

func TestRoaringOffsetTooLarge(t *testing.T) {
	b := bmroaring.New()
	t.Cleanup(func() { _ = b.Close() })
	err := b.Set(context.Background(), uint64(1)<<33)
	if !errors.Is(err, bitmap.ErrOffsetOutOfRange) {
		t.Fatalf("got %v", err)
	}
}
