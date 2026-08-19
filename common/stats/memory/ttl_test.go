// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/stretchr/testify/assert"
)

func TestTTLExpiration(t *testing.T) {
	var mu sync.Mutex
	var expiredKeys []stats.ExpiredKey

	c := New(
		WithReservoirTimer(4096),
		WithTTL(TTLConfig{
			RetentionDays: 1,
			CheckInterval: 100 * time.Millisecond,
			OnExpire: func(ek stats.ExpiredKey) error {
				mu.Lock()
				expiredKeys = append(expiredKeys, ek)
				mu.Unlock()
				return nil
			},
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(100)
	c.HLL("uv:" + oldDate).Add("user1")
	c.HLL("uv:" + oldDate).Add("user2")
	c.Set("daily_users:" + oldDate).Add("user1")
	c.Timer("response_time:" + oldDate).Record(50000000)

	today := time.Now().Format("2006-01-02")
	c.Counter("pv:" + today + ":/home").IncrBy(50)
	c.HLL("uv:" + today).Add("user3")

	time.Sleep(500 * time.Millisecond)

	// Verify old keys expired (lock-protected read).
	mu.Lock()
	keyNames := make([]string, len(expiredKeys))
	for i, ek := range expiredKeys {
		keyNames[i] = ek.Key
	}
	mu.Unlock()
	assert.Contains(t, keyNames, "pv:"+oldDate+":/home")
	assert.Contains(t, keyNames, "uv:"+oldDate)

	// Old keys gone from memory.
	assert.Equal(t, int64(0), c.Counter("pv:"+oldDate+":/home").Get())
	assert.Equal(t, uint64(0), c.HLL("uv:"+oldDate).Estimate())

	// Today's keys still present.
	assert.Equal(t, int64(50), c.Counter("pv:"+today+":/home").Get())
	assert.Equal(t, uint64(1), c.HLL("uv:"+today).Estimate())

	t.Logf("Expired %d keys: %v", len(expiredKeys), keyNames)
}

func TestTTLManualCleanup(t *testing.T) {
	c := New(
		WithTTL(TTLConfig{
			RetentionDays: 1,
			OnExpire:      func(ek stats.ExpiredKey) error { return nil },
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	c.Counter("pv:" + oldDate + ":/home").Incr()
	c.Counter("pv:" + oldDate + ":/about").Incr()
	c.Counter("pv:" + today + ":/home").Incr()

	before := c.KeyCount()
	removed := c.CleanupNow()
	after := c.KeyCount()

	assert.Equal(t, 2, removed)
	assert.Equal(t, before-2, after)
	t.Logf("KeyCount: before=%d, removed=%d, after=%d", before, removed, after)
}

func TestTTLCallbackReceivesValue(t *testing.T) {
	var received []stats.ExpiredKey

	c := New(
		WithTTL(TTLConfig{
			RetentionDays: 1,
			OnExpire: func(ek stats.ExpiredKey) error {
				received = append(received, ek)
				return nil
			},
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(42)
	c.HLL("uv:" + oldDate).Add("u1")
	c.HLL("uv:" + oldDate).Add("u2")
	c.Timer("rt:" + oldDate).Record(1000000)

	c.CleanupNow()

	assert.Equal(t, 3, len(received))

	// Verify each entry has correct type and value.
	for _, ek := range received {
		switch ek.Type {
		case "counter":
			assert.Equal(t, int64(42), ek.Value)
		case "hll":
			assert.Equal(t, uint64(2), ek.Value)
		case "timer":
			ts, ok := ek.Value.(stats.TimerSummary)
			assert.True(t, ok)
			assert.Equal(t, int64(1), ts.Count)
		}
		assert.Equal(t, oldDate, ek.Date)
		assert.NotEmpty(t, ek.ExpiredAt)
	}

	t.Logf("Received %d expired keys with values", len(received))
}

func TestTTLCallbackErrorRetries(t *testing.T) {
	var attempts int

	c := New(
		WithTTL(TTLConfig{
			RetentionDays: 1,
			OnExpire: func(ek stats.ExpiredKey) error {
				attempts++
				if attempts == 1 {
					return errFake // first attempt fails
				}
				return nil
			},
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").Incr()

	// First cleanup — callback returns error, key should NOT be removed.
	c.CleanupNow()
	assert.Equal(t, 1, c.KeyCount(), "key should remain after callback error")

	// Second cleanup — callback succeeds, key should be removed.
	c.CleanupNow()
	assert.Equal(t, 0, c.KeyCount(), "key should be removed after callback success")

	t.Logf("attempts=%d (1 error + 1 success)", attempts)
}

var errFake = newFakeError()

type fakeError struct{}

func (e *fakeError) Error() string { return "fake error" }

func newFakeError() error { return &fakeError{} }

func TestDefaultKeyDateExtractor(t *testing.T) {
	tests := []struct {
		key  string
		date string
		ok   bool
	}{
		{"pv:2026-08-18:/home", "2026-08-18", true},
		{"uv:2026-08-18", "2026-08-18", true},
		{"response_time:2026-08-18", "2026-08-18", true},
		{"daily_users:2026-08-17", "2026-08-17", true},
		{"pv:2026-08-18:/api/v1/users/123", "2026-08-18", true},
		{"all_users", "", false},
		{"pv_total:2026-08-18", "2026-08-18", true},
		{"custom_metric", "", false},
	}

	for _, tt := range tests {
		date, ok := DefaultKeyDateExtractor(tt.key)
		assert.Equal(t, tt.ok, ok, "ok mismatch for key %q", tt.key)
		if ok {
			assert.Equal(t, tt.date, date, "date mismatch for key %q", tt.key)
		}
	}
}

func TestKeyCount(t *testing.T) {
	c := New()
	defer c.Close()

	c.Counter("c1")
	c.Counter("c2")
	c.Gauge("g1")
	c.Set("s1")
	c.HLL("h1")
	c.Timer("t1")

	assert.Equal(t, 6, c.KeyCount())
}
