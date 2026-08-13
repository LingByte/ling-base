package redlock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/lock"
	lockredis "github.com/LingByte/ling-base/lock/redis"
	"github.com/LingByte/ling-base/lock/redlock"
)

func redisClients(t *testing.T, n int) []lockredis.Client {
	t.Helper()
	clients := make([]lockredis.Client, n)
	for i := 0; i < n; i++ {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(mr.Close)
		client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		clients[i] = client
	}
	return clients
}

func TestNewMutexValidation(t *testing.T) {
	clients := redisClients(t, 1)
	if _, err := redlock.NewMutex(nil, "k"); err == nil {
		t.Fatal("expected empty clients error")
	}
	if _, err := redlock.NewMutex([]lockredis.Client{nil}, "k"); err == nil {
		t.Fatal("expected nil client error")
	}
	if _, err := redlock.NewMutex(clients, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := redlock.NewMutex(clients, "k", lock.WithTTL(0)); !errors.Is(err, lock.ErrInvalidTTL) {
		t.Fatalf("invalid ttl: %v", err)
	}
}

func TestTryLockQuorumSuccess(t *testing.T) {
	clients := redisClients(t, 3)
	ctx := context.Background()
	m, err := redlock.NewMutex(clients, "resource", lock.WithTTL(30*time.Second), lock.WithValue("v1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTryLockQuorumFail(t *testing.T) {
	clients := redisClients(t, 3)
	ctx := context.Background()

	// Hold lock on all nodes with same key so redlock cannot obtain quorum on conflicting value
	for _, c := range clients {
		mx, _ := lockredis.NewMutex(c, "resource", lock.WithTTL(time.Minute), lock.WithValue("other"))
		if err := mx.TryLock(ctx); err != nil {
			t.Fatal(err)
		}
	}

	m, _ := redlock.NewMutex(clients, "resource", lock.WithTTL(30*time.Second), lock.WithValue("new"))
	if err := m.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestLockRetryCancel(t *testing.T) {
	clients := redisClients(t, 1)
	ctx := context.Background()
	holder, _ := lockredis.NewMutex(clients[0], "r", lock.WithTTL(time.Minute), lock.WithValue("h"))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	m, _ := redlock.NewMutex(clients, "r", lock.WithRetryDelay(5*time.Millisecond))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Lock(cancelCtx) }()
	time.Sleep(25 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	_ = holder.Unlock(ctx)
}

func TestRefreshNotHeld(t *testing.T) {
	clients := redisClients(t, 3)
	ctx := context.Background()
	m, _ := redlock.NewMutex(clients, "r2", lock.WithTTL(30*time.Second))
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestContextCancelledTryLock(t *testing.T) {
	clients := redisClients(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := redlock.NewMutex(clients, "k", lock.WithTTL(time.Minute))
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestTwoNodeQuorum(t *testing.T) {
	clients := redisClients(t, 2)
	ctx := context.Background()
	m, _ := redlock.NewMutex(clients, "two", lock.WithTTL(30*time.Second), lock.WithValue("x"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m.Unlock(ctx)
}
