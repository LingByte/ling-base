package lock_test

import (
	"testing"
	"time"

	"github.com/LingByte/ling-base/lock"
)

func TestApplyOptionsDefaults(t *testing.T) {
	o := lock.ApplyOptions()
	if o.TTL != 10*time.Second {
		t.Fatalf("TTL = %v", o.TTL)
	}
	if o.RetryDelay != 50*time.Millisecond {
		t.Fatalf("RetryDelay = %v", o.RetryDelay)
	}
	if o.Value != "" {
		t.Fatalf("Value = %q", o.Value)
	}
}

func TestApplyOptionsWithTTL(t *testing.T) {
	o := lock.ApplyOptions(lock.WithTTL(3 * time.Second))
	if o.TTL != 3*time.Second {
		t.Fatalf("TTL = %v", o.TTL)
	}
}

func TestApplyOptionsWithRetryDelay(t *testing.T) {
	o := lock.ApplyOptions(lock.WithRetryDelay(100 * time.Millisecond))
	if o.RetryDelay != 100*time.Millisecond {
		t.Fatalf("RetryDelay = %v", o.RetryDelay)
	}
}

func TestApplyOptionsWithValue(t *testing.T) {
	o := lock.ApplyOptions(lock.WithValue("owner-1"))
	if o.Value != "owner-1" {
		t.Fatalf("Value = %q", o.Value)
	}
}

func TestApplyOptionsNilOptionSkipped(t *testing.T) {
	o := lock.ApplyOptions(nil, lock.WithTTL(time.Minute))
	if o.TTL != time.Minute {
		t.Fatalf("TTL = %v", o.TTL)
	}
}

func TestApplyOptionsCombined(t *testing.T) {
	o := lock.ApplyOptions(
		lock.WithTTL(5*time.Second),
		lock.WithRetryDelay(10*time.Millisecond),
		lock.WithValue("x"),
	)
	if o.TTL != 5*time.Second || o.RetryDelay != 10*time.Millisecond || o.Value != "x" {
		t.Fatalf("opts = %+v", o)
	}
}

func TestErrorsSentinel(t *testing.T) {
	if lock.ErrNotObtained.Error() != "lock: not obtained" {
		t.Fatal(lock.ErrNotObtained)
	}
	if lock.ErrNotHeld.Error() != "lock: not held" {
		t.Fatal(lock.ErrNotHeld)
	}
	if lock.ErrInvalidTTL.Error() != "lock: ttl must be greater than zero" {
		t.Fatal(lock.ErrInvalidTTL)
	}
	if lock.ErrEmptyKey.Error() != "lock: key must not be empty" {
		t.Fatal(lock.ErrEmptyKey)
	}
}
