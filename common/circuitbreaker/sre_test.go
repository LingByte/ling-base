package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestSREBreaker_AllowsBeforeRequestThreshold(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(3), WithSREWindow(time.Second), WithSREBucket(2))
	b.MarkFailed()
	b.MarkFailed()

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil", err)
	}
}

func TestSREBreaker_RejectsAfterFailures(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(1), WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 0 }
	b.MarkFailed()

	if err := b.Allow(); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("Allow() error = %v, want %v", err, ErrNotAllowed)
	}
}

func TestSREBreaker_DefaultRequestThreshold(t *testing.T) {
	b := NewSREBreaker(WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 0 }

	for range 19 {
		b.MarkFailed()
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil before default request threshold", err)
	}

	b.MarkFailed()
	if err := b.Allow(); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("Allow() error = %v, want %v at default request threshold", err, ErrNotAllowed)
	}
}

func TestSREBreaker_DefaultFailureRatio(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(20), WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 0 }

	for range 10 {
		b.MarkSuccess()
		b.MarkFailed()
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil at default failure ratio threshold", err)
	}

	b.MarkFailed()
	if err := b.Allow(); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("Allow() error = %v, want %v above default failure ratio threshold", err, ErrNotAllowed)
	}
}

func TestSREBreaker_ClosesAfterSuccesses(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(1), WithSREFailureRatio(0.5), WithSREWindow(time.Second), WithSREBucket(2))
	b.MarkSuccess()
	b.MarkSuccess()

	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil", err)
	}
}

func TestSREBreaker_StateTransitions(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(1), WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 0 }

	if b.State() != SREStateClosed {
		t.Fatalf("initial state = %d, want %d", b.State(), SREStateClosed)
	}

	b.MarkFailed()
	_ = b.Allow() // triggers open
	if b.State() != SREStateOpen {
		t.Fatalf("state after failures = %d, want %d", b.State(), SREStateOpen)
	}

	// Recover with successes
	b.random = func() float64 { return 1 } // never drop
	b.MarkSuccess()
	b.MarkSuccess()
	_ = b.Allow()
	if b.State() != SREStateClosed {
		t.Fatalf("state after recovery = %d, want %d", b.State(), SREStateClosed)
	}
}

func TestSREBreaker_InvalidOptions(t *testing.T) {
	// Negative failure ratio → default 0.5
	b := NewSREBreaker(WithSREFailureRatio(-1))
	if b.k != 1/(1-0.5) {
		t.Fatalf("k = %v, want %v for default ratio", b.k, 1/(1-0.5))
	}

	// Failure ratio >= 1 → default 0.5
	b2 := NewSREBreaker(WithSREFailureRatio(1.5))
	if b2.k != 1/(1-0.5) {
		t.Fatalf("k = %v, want %v for default ratio", b2.k, 1/(1-0.5))
	}

	// Zero request → default 1
	b3 := NewSREBreaker(WithSRERequest(0))
	if b3.request != 1 {
		t.Fatalf("request = %d, want 1", b3.request)
	}

	// Zero bucket → default 1
	b4 := NewSREBreaker(WithSREBucket(0))
	if len(b4.stat.buckets) != 1 {
		t.Fatalf("buckets = %d, want 1", len(b4.stat.buckets))
	}

	// Zero window → default 3s (with default 10 buckets → 300ms each)
	b5 := NewSREBreaker(WithSREWindow(0), WithSREBucket(1))
	if b5.stat.bucketDuration != 3*time.Second {
		t.Fatalf("bucketDuration = %v, want 3s", b5.stat.bucketDuration)
	}
}

func TestSREBreaker_AllowPassesWhenRandomHigh(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(1), WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 1 } // always > dropRatio, never drop

	b.MarkFailed()
	b.MarkFailed()
	// Even with failures, random=1 means never dropped
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil (random=1 never drops)", err)
	}
}

func TestSREBreaker_MarkSuccessIncreasesAllowance(t *testing.T) {
	b := NewSREBreaker(WithSRERequest(2), WithSREFailureRatio(0.5), WithSREWindow(time.Second), WithSREBucket(2))
	b.random = func() float64 { return 0 }

	// 2 failures → should reject
	b.MarkFailed()
	b.MarkFailed()
	if err := b.Allow(); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("Allow() error = %v, want %v", err, ErrNotAllowed)
	}

	// Add successes → should allow
	b.MarkSuccess()
	b.MarkSuccess()
	b.MarkSuccess()
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow() error = %v, want nil after successes", err)
	}
}
