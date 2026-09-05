// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/cache"
	"github.com/LingByte/ling-base/common/cache/lru"
)

// stubLocker is a simple in-process lock for testing.
type stubLocker struct {
	mu     sync.Mutex
	locked bool
	waits  int32
}

func newStubLocker() *stubLocker {
	return &stubLocker{}
}

func (s *stubLocker) TryLock(ctx context.Context) error {
	if s.mu.TryLock() {
		if s.locked {
			s.mu.Unlock()
			atomic.AddInt32(&s.waits, 1)
			return cache.ErrLockNotObtained
		}
		s.locked = true
		s.mu.Unlock()
		return nil
	}
	atomic.AddInt32(&s.waits, 1)
	return cache.ErrLockNotObtained
}

func (s *stubLocker) Unlock(ctx context.Context) error {
	s.mu.Lock()
	s.locked = false
	s.mu.Unlock()
	return nil
}

func newTestCache(t *testing.T) cache.ByteCache {
	t.Helper()
	c, err := lru.New[string, []byte](100, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestGetOrLoad_CacheHit(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "key1", []byte("value1"), time.Minute)

	val, err := cache.GetOrLoad(ctx, c, "key1", func(ctx context.Context) ([]byte, error) {
		t.Error("loader should not be called on cache hit")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("GetOrLoad: %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("val = %q, want value1", val)
	}
}

func TestGetOrLoad_CacheMiss_LoadSuccess(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	var loadCalls int32
	val, err := cache.GetOrLoad(ctx, c, "key2", func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&loadCalls, 1)
		return []byte("loaded"), nil
	}, cache.WithCacheTTL(time.Minute))
	if err != nil {
		t.Fatalf("GetOrLoad: %v", err)
	}
	if string(val) != "loaded" {
		t.Errorf("val = %q, want loaded", val)
	}
	if atomic.LoadInt32(&loadCalls) != 1 {
		t.Errorf("loadCalls = %d, want 1", loadCalls)
	}

	// Second call should hit cache.
	val, err = cache.GetOrLoad(ctx, c, "key2", func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&loadCalls, 1)
		return []byte("loaded-again"), nil
	})
	if err != nil {
		t.Fatalf("second GetOrLoad: %v", err)
	}
	if string(val) != "loaded" {
		t.Errorf("val = %q, want loaded (from cache)", val)
	}
	if atomic.LoadInt32(&loadCalls) != 1 {
		t.Errorf("loadCalls = %d, want 1 (second should hit cache)", loadCalls)
	}
}

func TestGetOrLoad_CacheMiss_LoadNotFound_NullMarker(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	// First call: loader returns ErrNotFound → null marker written.
	_, err := cache.GetOrLoad(ctx, c, "key3", func(ctx context.Context) ([]byte, error) {
		return nil, cache.ErrNotFound
	}, cache.WithNullTTL(time.Minute))
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	// Second call: should hit null marker, NOT call loader.
	var loadCalls int32
	_, err = cache.GetOrLoad(ctx, c, "key3", func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&loadCalls, 1)
		return []byte("should-not-reach"), nil
	})
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound (from null marker)", err)
	}
	if atomic.LoadInt32(&loadCalls) != 0 {
		t.Errorf("loadCalls = %d, want 0 (null marker should prevent load)", loadCalls)
	}
}

func TestGetOrLoad_NullMarkerDisabled(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	_, err := cache.GetOrLoad(ctx, c, "key4", func(ctx context.Context) ([]byte, error) {
		return nil, cache.ErrNotFound
	}, cache.WithNullMarker(false))
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	// No null marker should have been written.
	exists, _ := c.Exists(ctx, "key4")
	if exists {
		t.Error("null marker should not be written when disabled")
	}
}

func TestGetOrLoad_LoadError_NoCache(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	loadErr := errors.New("db connection failed")
	_, err := cache.GetOrLoad(ctx, c, "key5", func(ctx context.Context) ([]byte, error) {
		return nil, loadErr
	})
	if !errors.Is(err, loadErr) {
		t.Errorf("err = %v, want loadErr", err)
	}

	// Non-ErrNotFound errors should NOT be cached.
	exists, _ := c.Exists(ctx, "key5")
	if exists {
		t.Error("error result should not be cached")
	}
}

func TestGetOrLoad_WithLocker_DoubleCheck(t *testing.T) {
	c := newTestCache(t)
	locker := newStubLocker()
	ctx := context.Background()

	var loadCalls int32
	loader := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&loadCalls, 1)
		return []byte("from-db"), nil
	}

	val, err := cache.GetOrLoad(ctx, c, "key6", loader, cache.WithLocker(locker), cache.WithCacheTTL(time.Minute))
	if err != nil {
		t.Fatalf("GetOrLoad: %v", err)
	}
	if string(val) != "from-db" {
		t.Errorf("val = %q", val)
	}
	if atomic.LoadInt32(&loadCalls) != 1 {
		t.Errorf("loadCalls = %d, want 1", loadCalls)
	}
}

func TestGetOrLoad_WithLocker_LockNotObtained(t *testing.T) {
	c := newTestCache(t)
	locker := newStubLocker()
	ctx := context.Background()

	// Pre-lock to simulate contention.
	_ = locker.TryLock(ctx)

	_, err := cache.GetOrLoad(ctx, c, "key7", func(ctx context.Context) ([]byte, error) {
		return []byte("x"), nil
	}, cache.WithLocker(locker))
	if !errors.Is(err, cache.ErrLockNotObtained) {
		t.Errorf("err = %v, want ErrLockNotObtained", err)
	}
}

func TestGetOrLoadJSON(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	var dest User
	err := cache.GetOrLoadJSON(ctx, c, "user:1", &dest, func(ctx context.Context) (any, error) {
		return &User{Name: "Alice", Age: 30}, nil
	}, cache.WithCacheTTL(time.Minute))
	if err != nil {
		t.Fatalf("GetOrLoadJSON: %v", err)
	}
	if dest.Name != "Alice" || dest.Age != 30 {
		t.Errorf("dest = %+v", dest)
	}

	// Second call should hit cache.
	var dest2 User
	err = cache.GetOrLoadJSON(ctx, c, "user:1", &dest2, func(ctx context.Context) (any, error) {
		t.Error("loader should not be called on cache hit")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("second GetOrLoadJSON: %v", err)
	}
	if dest2.Name != "Alice" {
		t.Errorf("dest2 = %+v", dest2)
	}
}

func TestInvalidate(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "key8", []byte("val"), time.Minute)

	if err := cache.Invalidate(ctx, c, "key8"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	exists, _ := c.Exists(ctx, "key8")
	if exists {
		t.Error("key should be invalidated")
	}
}

func TestInvalidateBatch(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"), time.Minute)
	_ = c.Set(ctx, "b", []byte("2"), time.Minute)

	if err := cache.InvalidateBatch(ctx, c, "a", "b", "nonexistent"); err != nil {
		t.Fatalf("InvalidateBatch: %v", err)
	}
	for _, k := range []string{"a", "b"} {
		exists, _ := c.Exists(ctx, k)
		if exists {
			t.Errorf("key %q should be invalidated", k)
		}
	}
}

func TestGetOrLoad_NilLoader(t *testing.T) {
	c := newTestCache(t)
	_, err := cache.GetOrLoad(context.Background(), c, "key", nil)
	if !errors.Is(err, cache.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
