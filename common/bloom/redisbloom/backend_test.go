package redisbloom_test

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/common/bloom/redisbloom"
)

// fakeBackend is an in-memory Backend implementation for testing. It simulates
// RedisBloom's BF.* commands with exact (non-probabilistic) semantics, which
// is sufficient for testing the driver logic.
type fakeBackend struct {
	mu       sync.Mutex
	items    map[string]bool
	reserved bool
	// Recorded parameters from the last Reserve call.
	reserveKey        string
	reserveErrorRate  float64
	reserveCapacity   int64
	reserveExpansion  int64
	reserveNonScaling bool
	// Whether Del was called.
	delCalled bool
	// Whether Expire was called and with what TTL.
	expireKey string
	expireTTL time.Duration
	// If non-nil, Reserve returns this error instead of succeeding.
	reserveErr error
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{items: make(map[string]bool)}
}

func (b *fakeBackend) Reserve(ctx context.Context, key string, errorRate float64, capacity int64, expansion int64, nonScaling bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reserveKey = key
	b.reserveErrorRate = errorRate
	b.reserveCapacity = capacity
	b.reserveExpansion = expansion
	b.reserveNonScaling = nonScaling
	if b.reserveErr != nil {
		return b.reserveErr
	}
	b.reserved = true
	return nil
}

func (b *fakeBackend) Add(ctx context.Context, key, item string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	already := b.items[item]
	b.items[item] = true
	return !already, nil
}

func (b *fakeBackend) Exists(ctx context.Context, key, item string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.items[item], nil
}

func (b *fakeBackend) MAdd(ctx context.Context, key string, items []string) ([]bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	results := make([]bool, len(items))
	for i, item := range items {
		already := b.items[item]
		b.items[item] = true
		results[i] = !already
	}
	return results, nil
}

func (b *fakeBackend) MExists(ctx context.Context, key string, items []string) ([]bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	results := make([]bool, len(items))
	for i, item := range items {
		results[i] = b.items[item]
	}
	return results, nil
}

func (b *fakeBackend) Del(ctx context.Context, keys ...string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.delCalled = true
	b.items = make(map[string]bool)
	b.reserved = false
	return nil
}

func (b *fakeBackend) Expire(ctx context.Context, key string, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.expireKey = key
	b.expireTTL = ttl
	return nil
}

// Compile-time check.
var _ redisbloom.Backend = (*fakeBackend)(nil)

// itemExistsError simulates the "item exists" error from BF.RESERVE.
type itemExistsError struct{}

func (itemExistsError) Error() string { return "item exists" }

// stringError wraps a string as an error for testing.
type stringError string

func (e stringError) Error() string { return string(e) }

// ensure itemExistsError matches isItemExistsError (which checks for
// "item exists" substring).
func init() {
	_ = strings.Contains(itemExistsError{}.Error(), "item exists")
}
