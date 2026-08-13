package redisbloom_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/LingByte/ling-base/bloom"
	"github.com/LingByte/ling-base/bloom/redisbloom"
)

func newFilter(t *testing.T, backend redisbloom.Backend, opts ...redisbloom.Option) *redisbloom.Filter {
	t.Helper()
	allOpts := append([]redisbloom.Option{
		redisbloom.WithKey("bf"),
		redisbloom.WithCapacity(1000, 0.01),
	}, opts...)
	f, err := redisbloom.NewWithBackend(backend, allOpts...)
	if err != nil {
		t.Fatalf("NewWithBackend: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestRedisBloomAddTest(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()

	if err := f.Add(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := f.Add(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Test(ctx, "a"); !ok {
		t.Fatal("a missing")
	}
	if ok, _ := f.Test(ctx, "b"); !ok {
		t.Fatal("b missing")
	}
	if ok, _ := f.Test(ctx, "c"); ok {
		t.Fatal("c should be absent")
	}
	if !f.IsReserved() {
		t.Fatal("filter should be reserved after Add")
	}
	if !bk.reserved {
		t.Fatal("backend should have received Reserve call")
	}
}

func TestRedisBloomReserveParams(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk,
		redisbloom.WithKey("mybf"),
		redisbloom.WithCapacity(5000, 0.001),
		redisbloom.WithExpansion(3),
		redisbloom.WithNonScaling(),
	)
	ctx := context.Background()
	_ = f.Add(ctx, "x")

	if bk.reserveKey != "mybf" {
		t.Fatalf("reserveKey = %q, want mybf", bk.reserveKey)
	}
	if bk.reserveCapacity != 5000 {
		t.Fatalf("reserveCapacity = %d, want 5000", bk.reserveCapacity)
	}
	if bk.reserveErrorRate != 0.001 {
		t.Fatalf("reserveErrorRate = %v, want 0.001", bk.reserveErrorRate)
	}
	if bk.reserveExpansion != 3 {
		t.Fatalf("reserveExpansion = %d, want 3", bk.reserveExpansion)
	}
	if !bk.reserveNonScaling {
		t.Fatal("reserveNonScaling = false, want true")
	}
}

func TestRedisBloomNoCreate(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk, redisbloom.WithNoCreate())
	ctx := context.Background()
	if err := f.Add(ctx, "x"); err != nil {
		t.Fatal(err)
	}
	if bk.reserved {
		t.Fatal("backend should NOT have received Reserve with NoCreate")
	}
}

func TestRedisBloomReserveExistingKey(t *testing.T) {
	bk := newFakeBackend()
	bk.reserveErr = itemExistsError{} // simulate "item exists" from BF.RESERVE
	f := newFilter(t, bk)
	ctx := context.Background()
	if err := f.Add(ctx, "x"); err != nil {
		t.Fatalf("Add should succeed despite item exists error: %v", err)
	}
	if !f.IsReserved() {
		t.Fatal("filter should be marked reserved after item-exists error")
	}
}

func TestRedisBloomReserveFailure(t *testing.T) {
	bk := newFakeBackend()
	bk.reserveErr = stringError("connection refused")
	f := newFilter(t, bk)
	ctx := context.Background()
	if err := f.Add(ctx, "x"); err == nil {
		t.Fatal("Add should fail on non-item-exists Reserve error")
	}
}

func TestRedisBloomBatch(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
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

func TestRedisBloomBatchEmpty(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	if err := f.AddBatch(ctx, nil); err != nil {
		t.Fatal(err)
	}
	res, err := f.TestBatch(ctx, nil)
	if err != nil || len(res) != 0 {
		t.Fatalf("TestBatch nil = %v, %v", res, err)
	}
}

func TestRedisBloomBatchEmptyKey(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	if err := f.AddBatch(ctx, []string{"ok", ""}); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("AddBatch with empty = %v, want ErrEmptyKey", err)
	}
	if _, err := f.TestBatch(ctx, []string{"ok", ""}); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("TestBatch with empty = %v, want ErrEmptyKey", err)
	}
}

func TestRedisBloomReset(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if err := f.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if !bk.delCalled {
		t.Fatal("backend Del should have been called")
	}
	if ok, _ := f.Test(ctx, "x"); ok {
		t.Fatal("x should be absent after reset")
	}
	if f.IsReserved() {
		t.Fatal("reserved flag should be cleared after reset")
	}
	// Adding after reset should re-reserve.
	_ = f.Add(ctx, "y")
	if !f.IsReserved() {
		t.Fatal("should re-reserve after reset + Add")
	}
}

func TestRedisBloomTTL(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk, redisbloom.WithTTL(2*time.Second))
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if bk.expireKey != "bf" {
		t.Fatalf("expireKey = %q, want bf", bk.expireKey)
	}
	if bk.expireTTL != 2*time.Second {
		t.Fatalf("expireTTL = %v, want 2s", bk.expireTTL)
	}
}

func TestRedisBloomNoTTLByDefault(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	_ = f.Add(ctx, "x")
	if bk.expireKey != "" {
		t.Fatal("Expire should not be called without TTL")
	}
}

func TestRedisBloomEmptyKey(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	if err := f.Add(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Add empty = %v, want ErrEmptyKey", err)
	}
	if _, err := f.Test(ctx, ""); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("Test empty = %v, want ErrEmptyKey", err)
	}
}

func TestRedisBloomNewValidation(t *testing.T) {
	bk := newFakeBackend()
	// Missing key.
	if _, err := redisbloom.NewWithBackend(bk, redisbloom.WithCapacity(100, 0.01)); !errors.Is(err, bloom.ErrEmptyKey) {
		t.Fatalf("no key = %v, want ErrEmptyKey", err)
	}
	// Invalid capacity (0).
	if _, err := redisbloom.NewWithBackend(bk,
		redisbloom.WithKey("k"),
		redisbloom.WithExpectedCapacity(0),
	); !errors.Is(err, bloom.ErrInvalidCapacity) {
		t.Fatalf("capacity=0 = %v, want ErrInvalidCapacity", err)
	}
	// Invalid error rate (0).
	if _, err := redisbloom.NewWithBackend(bk,
		redisbloom.WithKey("k"),
		redisbloom.WithErrorRate(0),
	); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("errorRate=0 = %v, want ErrInvalidFalsePositiveRate", err)
	}
	// Invalid error rate (>= 1).
	if _, err := redisbloom.NewWithBackend(bk,
		redisbloom.WithKey("k"),
		redisbloom.WithErrorRate(1),
	); !errors.Is(err, bloom.ErrInvalidFalsePositiveRate) {
		t.Fatalf("errorRate=1 = %v, want ErrInvalidFalsePositiveRate", err)
	}
	// Nil backend.
	if _, err := redisbloom.NewWithBackend(nil, redisbloom.WithKey("k"), redisbloom.WithCapacity(100, 0.01)); err == nil {
		t.Fatal("nil backend should error")
	}
}

func TestRedisBloomClosed(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx := context.Background()
	_ = f.Close()
	if err := f.Add(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Add closed = %v, want ErrClosed", err)
	}
	if _, err := f.Test(ctx, "x"); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Test closed = %v, want ErrClosed", err)
	}
	if _, err := f.TestBatch(ctx, []string{"x"}); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("TestBatch closed = %v, want ErrClosed", err)
	}
	if err := f.Reset(ctx); !errors.Is(err, bloom.ErrClosed) {
		t.Fatalf("Reset closed = %v, want ErrClosed", err)
	}
}

func TestRedisBloomCtxCancelled(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := f.Add(ctx, "x"); err != context.Canceled {
		t.Fatalf("Add cancelled = %v, want context.Canceled", err)
	}
}

func TestRedisBloomCloseIdempotent(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisBloomGetters(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk,
		redisbloom.WithKey("mybf"),
		redisbloom.WithCapacity(5000, 0.001),
	)
	if f.Key() != "mybf" {
		t.Fatalf("Key = %q", f.Key())
	}
	if f.Capacity() != 5000 {
		t.Fatalf("Capacity = %d", f.Capacity())
	}
	if f.ErrorRate() != 0.001 {
		t.Fatalf("ErrorRate = %v", f.ErrorRate())
	}
}

func TestRedisBloomLargeBatch(t *testing.T) {
	bk := newFakeBackend()
	f := newFilter(t, bk, redisbloom.WithCapacity(10_000, 0.01))
	ctx := context.Background()

	const n = 1000
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = key(i)
	}
	if err := f.AddBatch(ctx, keys); err != nil {
		t.Fatal(err)
	}
	results, err := f.TestBatch(ctx, keys)
	if err != nil {
		t.Fatal(err)
	}
	for i, ok := range results {
		if !ok {
			t.Fatalf("false negative for %s", key(i))
		}
	}
}

func key(i int) string {
	return "key-" + strconv.Itoa(i)
}
