package consul_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"

	"github.com/LingByte/ling-base/common/lock"
	lockconsul "github.com/LingByte/ling-base/common/lock/consul"
)

type fakeSession struct {
	mu         sync.Mutex
	sessions   map[string]*api.SessionEntry
	next       int
	createErr  error
	destroyErr error
	renewErr   error
}

func newFakeSession() *fakeSession {
	return &fakeSession{sessions: make(map[string]*api.SessionEntry)}
}

func (f *fakeSession) Create(se *api.SessionEntry, _ *api.WriteOptions) (string, *api.WriteMeta, error) {
	if f.createErr != nil {
		return "", nil, f.createErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	id := fmt.Sprintf("sess-%d", f.next)
	f.sessions[id] = se
	return id, &api.WriteMeta{}, nil
}

func (f *fakeSession) Destroy(id string, _ *api.WriteOptions) (*api.WriteMeta, error) {
	if f.destroyErr != nil {
		return nil, f.destroyErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, id)
	return &api.WriteMeta{}, nil
}

func (f *fakeSession) Renew(id string, _ *api.WriteOptions) (*api.SessionEntry, *api.WriteMeta, error) {
	if f.renewErr != nil {
		return nil, nil, f.renewErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	se, ok := f.sessions[id]
	if !ok {
		return nil, nil, errors.New("unknown session")
	}
	return se, &api.WriteMeta{}, nil
}

type fakeKV struct {
	mu         sync.Mutex
	locks      map[string]string // key -> session id
	acquireErr error
	releaseErr error
}

func newFakeKV() *fakeKV {
	return &fakeKV{locks: make(map[string]string)}
}

func (f *fakeKV) Acquire(p *api.KVPair, _ *api.WriteOptions) (bool, *api.WriteMeta, error) {
	if f.acquireErr != nil {
		return false, nil, f.acquireErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if sid, ok := f.locks[p.Key]; ok && sid != p.Session {
		return false, &api.WriteMeta{}, nil
	}
	f.locks[p.Key] = p.Session
	return true, &api.WriteMeta{}, nil
}

func (f *fakeKV) Release(p *api.KVPair, _ *api.WriteOptions) (bool, *api.WriteMeta, error) {
	if f.releaseErr != nil {
		return false, nil, f.releaseErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locks[p.Key] != p.Session {
		return false, &api.WriteMeta{}, nil
	}
	delete(f.locks, p.Key)
	return true, &api.WriteMeta{}, nil
}

func TestNewMutexValidation(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	if _, err := lockconsul.NewMutex(nil, "k"); err == nil {
		t.Fatal("nil client")
	}
	if _, err := lockconsul.NewMutexWithAPI(nil, sess, "k"); err == nil {
		t.Fatal("nil kv")
	}
	if _, err := lockconsul.NewMutexWithAPI(kv, nil, "k"); err == nil {
		t.Fatal("nil session")
	}
	if _, err := lockconsul.NewMutexWithAPI(kv, sess, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(500*time.Millisecond)); !errors.Is(err, lock.ErrInvalidTTL) {
		t.Fatalf("invalid ttl: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	ctx := context.Background()
	m, err := lockconsul.NewMutexWithAPI(kv, sess, "service/lock", lock.WithTTL(30*time.Second), lock.WithValue("v"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	m2, _ := lockconsul.NewMutexWithAPI(kv, sess, "service/lock", lock.WithTTL(30*time.Second), lock.WithValue("other"))
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

func TestTryLockSessionCreateError(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	sess.createErr = errors.New("create fail")
	m, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected create error")
	}
}

func TestTryLockAcquireError(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	kv.acquireErr = errors.New("acquire fail")
	m, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected acquire error")
	}
}

func TestUnlockNotHeldAndReleaseError(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	ctx := context.Background()
	m, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(5*time.Second))
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}

	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	kv.releaseErr = errors.New("release fail")
	if err := m.Unlock(ctx); err == nil {
		t.Fatal("expected release error")
	}
}

func TestRefreshRenewError(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	ctx := context.Background()
	m, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	sess.renewErr = errors.New("renew fail")
	if err := m.Refresh(ctx); err == nil {
		t.Fatal("expected renew error")
	}
}

func TestLockRetryCancel(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	ctx := context.Background()
	holder, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(30*time.Second), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithRetryDelay(5*time.Millisecond), lock.WithTTL(30*time.Second))
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

func TestLockSuccessAfterRetry(t *testing.T) {
	kv, sess := newFakeKV(), newFakeSession()
	ctx := context.Background()
	holder, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(30*time.Second), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithRetryDelay(5*time.Millisecond), lock.WithTTL(30*time.Second))
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
	kv, sess := newFakeKV(), newFakeSession()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := lockconsul.NewMutexWithAPI(kv, sess, "k", lock.WithTTL(5*time.Second))
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}
