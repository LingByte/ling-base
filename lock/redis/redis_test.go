package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/lock"
	lockredis "github.com/LingByte/ling-base/lock/redis"
)

func newRedisClient(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}

func TestNewMutexValidation(t *testing.T) {
	client, _ := newRedisClient(t)
	if _, err := lockredis.NewMutex(nil, "k"); err == nil {
		t.Fatal("expected nil client error")
	}
	if _, err := lockredis.NewMutex(client, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := lockredis.NewMutex(client, "k", lock.WithTTL(0)); !errors.Is(err, lock.ErrInvalidTTL) {
		t.Fatalf("invalid ttl: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	client, _ := newRedisClient(t)
	ctx := context.Background()

	m1, err := lockredis.NewMutex(client, "lock:1", lock.WithTTL(time.Minute), lock.WithValue("tok-a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m1.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	m2, _ := lockredis.NewMutex(client, "lock:1", lock.WithTTL(time.Minute), lock.WithValue("tok-b"))
	if err := m2.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("contention: %v", err)
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

func TestUnlockRefreshNotHeld(t *testing.T) {
	client, _ := newRedisClient(t)
	ctx := context.Background()

	m, _ := lockredis.NewMutex(client, "lock:2", lock.WithValue("wrong"))
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}

	other, _ := lockredis.NewMutex(client, "lock:2", lock.WithValue("owner"))
	_ = other.TryLock(ctx)
	mHeld, _ := lockredis.NewMutex(client, "lock:2", lock.WithValue("not-owner"))
	if err := mHeld.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("wrong token unlock = %v", err)
	}
	_ = other.Unlock(ctx)
}

func TestLockRetryCancel(t *testing.T) {
	client, _ := newRedisClient(t)
	ctx := context.Background()

	holder, _ := lockredis.NewMutex(client, "lock:3", lock.WithTTL(time.Minute), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := lockredis.NewMutex(client, "lock:3", lock.WithTTL(time.Minute), lock.WithRetryDelay(5*time.Millisecond))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- waiter.Lock(cancelCtx) }()
	time.Sleep(25 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	_ = holder.Unlock(ctx)
	if err := waiter.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = waiter.Unlock(ctx)
}

func TestContextCancelled(t *testing.T) {
	client, _ := newRedisClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := lockredis.NewMutex(client, "k")
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}

type brokenClient struct {
	*goredis.Client
	failSetNX bool
	failEval  bool
}

func (b *brokenClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.BoolCmd {
	if b.failSetNX {
		return goredis.NewBoolResult(false, errors.New("setnx fail"))
	}
	return b.Client.SetNX(ctx, key, value, expiration)
}

func (b *brokenClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *goredis.Cmd {
	if b.failEval {
		return goredis.NewCmdResult(nil, errors.New("eval fail"))
	}
	return b.Client.Eval(ctx, script, keys, args...)
}

func TestClientErrors(t *testing.T) {
	client, _ := newRedisClient(t)
	ctx := context.Background()

	bc := &brokenClient{Client: client, failSetNX: true}
	m, _ := lockredis.NewMutex(bc, "err-key")
	if err := m.TryLock(ctx); err == nil {
		t.Fatal("expected SetNX error")
	}

	bc2 := &brokenClient{Client: client}
	m2, _ := lockredis.NewMutex(bc2, "err-key2", lock.WithValue("v"))
	if err := m2.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	bc2.failEval = true
	if err := m2.Unlock(ctx); err == nil {
		t.Fatal("expected Eval error on Unlock")
	}
	bc2.failEval = true
	if err := m2.Refresh(ctx); err == nil {
		t.Fatal("expected Eval error on Refresh")
	}
}
