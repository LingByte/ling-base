package etcd_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/lock"
	locketcd "github.com/LingByte/ling-base/lock/etcd"
)

type fakeBackend struct {
	mu           sync.Mutex
	keys         map[string]string
	leases       map[int64]int64 // leaseID -> ttl seconds
	nextID       int64
	grantErr     error
	acquireErr   error
	acquireOK    bool
	revokeErr    error
	deleteErr    error
	keepAliveErr error
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		keys:   make(map[string]string),
		leases: make(map[int64]int64),
	}
}

func (f *fakeBackend) Grant(_ context.Context, ttlSeconds int64) (int64, error) {
	if f.grantErr != nil {
		return 0, f.grantErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := f.nextID
	f.leases[id] = ttlSeconds
	return id, nil
}

func (f *fakeBackend) Revoke(_ context.Context, leaseID int64) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.leases, leaseID)
	return nil
}

func (f *fakeBackend) KeepAliveOnce(_ context.Context, leaseID int64) error {
	if f.keepAliveErr != nil {
		return f.keepAliveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.leases[leaseID]; !ok {
		return errors.New("unknown lease")
	}
	return nil
}

func (f *fakeBackend) Acquire(_ context.Context, key, value string, leaseID int64) (bool, error) {
	if f.acquireErr != nil {
		return false, f.acquireErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.acquireOK {
		f.keys[key] = value
		return true, nil
	}
	if _, exists := f.keys[key]; exists {
		return false, nil
	}
	f.keys[key] = value
	return true, nil
}

func (f *fakeBackend) Delete(_ context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.keys, key)
	return nil
}

func TestNewMutexValidation(t *testing.T) {
	if _, err := locketcd.NewMutex(nil, "k"); err == nil {
		t.Fatal("nil client")
	}
	if _, err := locketcd.NewMutexWithBackend(nil, "k"); err == nil {
		t.Fatal("nil backend")
	}
	b := newFakeBackend()
	if _, err := locketcd.NewMutexWithBackend(b, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(500*time.Millisecond)); !errors.Is(err, lock.ErrInvalidTTL) {
		t.Fatalf("invalid ttl: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, err := locketcd.NewMutexWithBackend(b, "/lock/1", lock.WithTTL(10*time.Second), lock.WithValue("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	m2, _ := locketcd.NewMutexWithBackend(b, "/lock/1", lock.WithTTL(10*time.Second), lock.WithValue("other"))
	if err := m2.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("contention: %v", err)
	}

	if err := m.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m2.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m2.Unlock(ctx)
}

func TestTryLockGrantAndAcquireErrors(t *testing.T) {
	ctx := context.Background()

	b := newFakeBackend()
	b.grantErr = errors.New("grant fail")
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err == nil {
		t.Fatal("expected grant error")
	}

	b2 := newFakeBackend()
	b2.acquireErr = errors.New("acquire fail")
	m2, _ := locketcd.NewMutexWithBackend(b2, "k", lock.WithTTL(5*time.Second))
	if err := m2.TryLock(ctx); err == nil {
		t.Fatal("expected acquire error")
	}
}

func TestUnlockRefreshNotHeld(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestUnlockDeleteError(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	b.deleteErr = errors.New("delete fail")
	if err := m.Unlock(ctx); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestLockRetryCancel(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	holder, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(10*time.Second), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithRetryDelay(5*time.Millisecond), lock.WithTTL(10*time.Second))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- waiter.Lock(cancelCtx) }()
	time.Sleep(25 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	_ = holder.Unlock(ctx)
}

func TestRefreshKeepAliveError(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	b.keepAliveErr = errors.New("keepalive fail")
	if err := m.Refresh(ctx); err == nil {
		t.Fatal("expected keepalive error")
	}
}

func TestUnlockContextCancelled(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Unlock(cancelCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unlock = %v", err)
	}
}

func TestRefreshContextCancelled(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Refresh(cancelCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestLockSuccessAfterRetry(t *testing.T) {
	b := newFakeBackend()
	ctx := context.Background()
	holder, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(10*time.Second), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	waiter, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithRetryDelay(5*time.Millisecond), lock.WithTTL(10*time.Second))
	done := make(chan error, 1)
	go func() { done <- waiter.Lock(ctx) }()
	time.Sleep(15 * time.Millisecond)
	if err := holder.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Lock = %v", err)
	}
	_ = waiter.Unlock(ctx)
}

func TestContextCancelled(t *testing.T) {
	b := newFakeBackend()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := locketcd.NewMutexWithBackend(b, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}
