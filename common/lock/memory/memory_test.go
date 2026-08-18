package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/lock"
	"github.com/LingByte/ling-base/common/lock/memory"
)

func TestNewMutexValidation(t *testing.T) {
	mgr := memory.NewManager()
	if _, err := mgr.NewMutex(""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := mgr.NewMutex("k", lock.WithTTL(0)); !errors.Is(err, lock.ErrInvalidTTL) {
		t.Fatalf("invalid ttl: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	mgr := memory.NewManager()
	ctx := context.Background()

	m1, err := mgr.NewMutex("resource", lock.WithTTL(time.Minute), lock.WithValue("a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m1.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	m2, _ := mgr.NewMutex("resource", lock.WithTTL(time.Minute), lock.WithValue("b"))
	if err := m2.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("second TryLock = %v", err)
	}
	if err := m1.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m1.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m2.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m2.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestUnlockNotHeld(t *testing.T) {
	mgr := memory.NewManager()
	ctx := context.Background()
	m, _ := mgr.NewMutex("k", lock.WithTTL(time.Minute), lock.WithValue("v"))
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock without hold = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh without hold = %v", err)
	}
}

func TestLockRetryAndCancel(t *testing.T) {
	mgr := memory.NewManager()
	ctx := context.Background()

	holder, _ := mgr.NewMutex("k", lock.WithTTL(time.Minute), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := mgr.NewMutex("k", lock.WithTTL(time.Minute), lock.WithValue("w"), lock.WithRetryDelay(5*time.Millisecond))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- waiter.Lock(cancelCtx) }()

	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Lock cancel = %v", err)
	}

	if err := holder.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := waiter.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := waiter.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExpiredLockReacquired(t *testing.T) {
	mgr := memory.NewManager()
	ctx := context.Background()
	m1, _ := mgr.NewMutex("k", lock.WithTTL(20*time.Millisecond), lock.WithValue("a"))
	if err := m1.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	m2, _ := mgr.NewMutex("k", lock.WithTTL(time.Minute), lock.WithValue("b"))
	if err := m2.TryLock(ctx); err != nil {
		t.Fatalf("after expiry: %v", err)
	}
	if err := m1.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("stale unlock = %v", err)
	}
	_ = m2.Unlock(ctx)
}

func TestContextCancelledTryLock(t *testing.T) {
	mgr := memory.NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := mgr.NewMutex("k", lock.WithTTL(time.Minute))
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}
