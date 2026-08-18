// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sqlite

import (
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore(t *testing.T) {
	path := "/tmp/test_stats_sqlite3.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	now := time.Now().Format(time.RFC3339)

	// Save various record types — Save is a stats.ExpireFunc.
	require.NoError(t, store.Save(stats.ExpiredKey{
		Key: "pv:" + oldDate + ":/home", Type: "counter",
		Value: int64(1000), Date: oldDate, ExpiredAt: now,
	}))
	require.NoError(t, store.Save(stats.ExpiredKey{
		Key: "uv:" + oldDate, Type: "hll",
		Value: uint64(5000), Date: oldDate, ExpiredAt: now,
	}))
	require.NoError(t, store.Save(stats.ExpiredKey{
		Key: "response_time:" + oldDate, Type: "timer",
		Value: stats.TimerSummary{Count: 1000, Mean: 50e6, P50: 45e6, P95: 95e6, P99: 99e6},
		Date: oldDate, ExpiredAt: now,
	}))

	// Query by date range.
	records, err := store.Query(oldDate, oldDate)
	require.NoError(t, err)
	assert.Equal(t, 3, len(records))

	// Query by type.
	counterRecords, err := store.QueryByType("counter", oldDate, oldDate)
	require.NoError(t, err)
	assert.Equal(t, 1, len(counterRecords))
	assert.Equal(t, "pv:"+oldDate+":/home", counterRecords[0].Key)

	// Upsert test.
	require.NoError(t, store.Save(stats.ExpiredKey{
		Key: "pv:" + oldDate + ":/home", Type: "counter",
		Value: int64(2000), Date: oldDate, ExpiredAt: now,
	}))
	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), pv)

	t.Logf("Archived %d records for date %s", len(records), oldDate)
}

func TestSQLiteTTLIntegration(t *testing.T) {
	path := "/tmp/test_stats_ttl_sqlite3.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	// OnExpire: store.Save — directly, no adapter needed!
	c := memory.New(
		memory.WithReservoirTimer(4096),
		memory.WithTTL(memory.TTLConfig{
			RetentionDays: 1,
			OnExpire:      store.Save, // ← direct
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(42)
	c.Counter("pv:" + oldDate + ":/about").IncrBy(18)
	c.HLL("uv:" + oldDate).Add("user1")
	c.HLL("uv:" + oldDate).Add("user2")
	c.HLL("uv:" + oldDate).Add("user3")

	removed := c.CleanupNow()
	assert.Equal(t, 3, removed)

	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(60), pv)

	uv, err := store.GetUV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), uv)

	t.Logf("TTL → SQLite: %d keys expired, PV=%d, UV=%d", removed, pv, uv)
}

func TestCustomCallback(t *testing.T) {
	// Demonstrate that OnExpire is a pure callback — no store needed.
	var saved []stats.ExpiredKey

	c := memory.New(
		memory.WithTTL(memory.TTLConfig{
			RetentionDays: 1,
			OnExpire: func(ek stats.ExpiredKey) error {
				saved = append(saved, ek)
				return nil
			},
		}),
	)
	defer c.Close()

	oldDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(99)

	c.CleanupNow()

	require.Equal(t, 1, len(saved))
	assert.Equal(t, "pv:"+oldDate+":/home", saved[0].Key)
	assert.Equal(t, "counter", saved[0].Type)
	assert.Equal(t, int64(99), saved[0].Value)
	assert.Equal(t, oldDate, saved[0].Date)

	t.Logf("Custom callback received: key=%s type=%s value=%v date=%s",
		saved[0].Key, saved[0].Type, saved[0].Value, saved[0].Date)
}
