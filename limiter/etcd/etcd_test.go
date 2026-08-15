// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package etcd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/limiter"
)

// mockEtcdClient is an in-memory etcd client for testing.
type mockEtcdClient struct {
	mu       sync.Mutex
	kvs      map[string]string
	leases   map[string]int64 // key → expiry unix timestamp
	leaseSeq int64
}

func newMockEtcdClient() *mockEtcdClient {
	return &mockEtcdClient{
		kvs:    make(map[string]string),
		leases: make(map[string]int64),
	}
}

func (m *mockEtcdClient) cleanup() {
	now := time.Now().Unix()
	for k, exp := range m.leases {
		if exp <= now {
			delete(m.kvs, k)
			delete(m.leases, k)
		}
	}
}

func (m *mockEtcdClient) Grant(ctx context.Context, ttl int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaseSeq++
	return m.leaseSeq, nil
}

func (m *mockEtcdClient) PutWithLease(ctx context.Context, key, val string, leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()
	m.kvs[key] = val
	// We don't know the TTL from leaseID in mock; use a default.
	m.leases[key] = time.Now().Unix() + 3600
	return nil
}

func (m *mockEtcdClient) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()
	delete(m.kvs, key)
	delete(m.leases, key)
	return nil
}

func (m *mockEtcdClient) GetPrefix(ctx context.Context, prefix string) ([]KeyValue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()
	var result []KeyValue
	for k, v := range m.kvs {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, KeyValue{Key: k, Value: v})
		}
	}
	return result, nil
}

// ===== Concurrency =====

func TestConcurrency_Basic(t *testing.T) {
	client := newMockEtcdClient()
	l := NewConcurrency(client, "/limiter/test/", 3, 10)
	ctx := context.Background()

	assert.NoError(t, l.Acquire(ctx, []byte("a")))
	assert.NoError(t, l.Acquire(ctx, []byte("b")))
	assert.NoError(t, l.Acquire(ctx, []byte("c")))
	err := l.Acquire(ctx, []byte("d"))
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestConcurrency_Release(t *testing.T) {
	client := newMockEtcdClient()
	l := NewConcurrency(client, "/limiter/test2/", 2, 10)
	ctx := context.Background()

	assert.NoError(t, l.Acquire(ctx, []byte("a")))
	assert.NoError(t, l.Acquire(ctx, []byte("b")))
	assert.Equal(t, limiter.ErrLimitExceeded, l.Acquire(ctx, []byte("c")))

	l.Release([]byte("a"))
	assert.NoError(t, l.Acquire(ctx, []byte("c")))
}

func TestConcurrency_ReleaseUnknownKey(t *testing.T) {
	client := newMockEtcdClient()
	l := NewConcurrency(client, "/limiter/test3/", 2, 10)

	assert.NotPanics(t, func() {
		l.Release([]byte("nonexistent"))
	})
}

func TestConcurrency_Running(t *testing.T) {
	client := newMockEtcdClient()
	l := NewConcurrency(client, "/limiter/test4/", 10, 10)
	ctx := context.Background()

	l.Acquire(ctx, []byte("a"))
	l.Acquire(ctx, []byte("b"))
	assert.Equal(t, 2, l.Running())
}

// ===== RateLimit =====

func TestRateLimit_Basic(t *testing.T) {
	client := newMockEtcdClient()
	l := NewRateLimit(client, "/rate/test/", 3, time.Second)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		assert.NoError(t, l.Acquire(ctx, []byte("req")))
	}
	err := l.Acquire(ctx, []byte("req"))
	assert.Equal(t, limiter.ErrLimitExceeded, err)
}

func TestRateLimit_Running(t *testing.T) {
	client := newMockEtcdClient()
	l := NewRateLimit(client, "/rate/test2/", 10, time.Second)
	ctx := context.Background()

	l.Acquire(ctx, []byte("a"))
	l.Acquire(ctx, []byte("b"))
	assert.Equal(t, 2, l.Running())
}

func TestRateLimit_ReleaseNoOp(t *testing.T) {
	client := newMockEtcdClient()
	l := NewRateLimit(client, "/rate/test3/", 10, time.Second)
	assert.NotPanics(t, func() {
		l.Release(nil)
	})
}

func TestNewConcurrency_NotNil(t *testing.T) {
	client := newMockEtcdClient()
	l := NewConcurrency(client, "/limiter/created/", 5, 30)
	require.NotNil(t, l)
}

func TestNewRateLimit_NotNil(t *testing.T) {
	client := newMockEtcdClient()
	l := NewRateLimit(client, "/rate/created/", 100, time.Minute)
	require.NotNil(t, l)
}
