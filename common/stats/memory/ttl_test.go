// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTLExpiration(t *testing.T) {
	// Create collector with 1-day retention and fast check interval.
	var expiredKeys []string
	var expiredEntries []SnapshotEntry

	c := New(
		WithReservoirTimer(4096),
		WithTTL(TTLConfig{
			RetentionDays:  1,
			CheckInterval:  100 * time.Millisecond,
			OnExpire: func(key string, entry SnapshotEntry) error {
				expiredKeys = append(expiredKeys, key)
				expiredEntries = append(expiredEntries, entry)
				return nil
			},
		}),
	)
	defer c.Close()

	// Record data with an OLD date (2 days ago — should expire).
	oldDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(100)
	c.HLL("uv:" + oldDate).Add("user1")
	c.HLL("uv:" + oldDate).Add("user2")
	c.Set("daily_users:" + oldDate).Add("user1")
	c.Timer("response_time:" + oldDate).Record(50000000)

	// Record data with TODAY's date — should NOT expire.
	today := time.Now().Format("2006-01-02")
	c.Counter("pv:" + today + ":/home").IncrBy(50)
	c.HLL("uv:" + today).Add("user3")

	// Wait for TTL cleanup to run.
	time.Sleep(500 * time.Millisecond)

	// Verify old keys were expired.
	assert.Contains(t, expiredKeys, "pv:"+oldDate+":/home")
	assert.Contains(t, expiredKeys, "uv:"+oldDate)

	// Verify old keys are gone from memory.
	assert.Equal(t, int64(0), c.Counter("pv:"+oldDate+":/home").Get())
	assert.Equal(t, uint64(0), c.HLL("uv:"+oldDate).Estimate())

	// Verify today's keys are still present.
	assert.Equal(t, int64(50), c.Counter("pv:"+today+":/home").Get())
	assert.Equal(t, uint64(1), c.HLL("uv:"+today).Estimate())

	t.Logf("Expired %d keys: %v", len(expiredKeys), expiredKeys)
}

func TestTTLManualCleanup(t *testing.T) {
	c := New(
		WithTTL(TTLConfig{
			RetentionDays: 1,
			OnExpire:      func(key string, entry SnapshotEntry) error { return nil },
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

	assert.Equal(t, 2, removed, "2 old keys should be removed")
	assert.Equal(t, before-2, after, "key count should decrease by 2")
	t.Logf("KeyCount: before=%d, removed=%d, after=%d", before, removed, after)
}

func TestTTLWithSQLite(t *testing.T) {
	// This test verifies the OnExpire callback pattern works with SQLite.
	// We simulate the SQLite store with an in-memory map.
	path := "/tmp/test_stats_ttl_sqlite.db"
	defer os.Remove(path)

	// Use a simple map as a mock store.
	store := &mockStore{data: make(map[string]string)}

	c := New(
		WithReservoirTimer(4096),
		WithTTL(TTLConfig{
			RetentionDays: 1,
			OnExpire:      store.save,
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(42)
	c.HLL("uv:" + oldDate).Add("user1")
	c.HLL("uv:" + oldDate).Add("user2")

	// Trigger cleanup.
	removed := c.CleanupNow()
	assert.Equal(t, 2, removed, "2 keys should be expired and saved")

	// Verify data was saved to store.
	assert.Contains(t, store.data, "pv:"+oldDate+":/home")
	assert.Contains(t, store.data, "uv:"+oldDate)

	t.Logf("Saved to store: %v", store.data)
}

func TestDefaultKeyDateExtractor(t *testing.T) {
	tests := []struct {
		key     string
		date    string
		ok      bool
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

// mockStore simulates a database store for testing.
type mockStore struct {
	data map[string]string
}

func (m *mockStore) save(key string, entry SnapshotEntry) error {
	m.data[key] = entry.Type + ":" + formatValue(entry.Value)
	return nil
}

func formatValue(v any) string {
	switch val := v.(type) {
	case int64:
		return string(rune(val)) // simplified
	case uint64:
		_ = val
		return "uint64"
	case int:
		_ = val
		return "int"
	default:
		return "other"
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
