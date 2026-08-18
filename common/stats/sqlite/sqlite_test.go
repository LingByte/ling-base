// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sqlite

import (
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore(t *testing.T) {
	path := "/tmp/test_stats_sqlite.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	// Simulate expiration of various key types.
	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")

	err = store.OnExpire("pv:"+oldDate+":/home", memory.SnapshotEntry{
		Type:  "counter",
		Value: int64(1000),
	})
	require.NoError(t, err)

	err = store.OnExpire("uv:"+oldDate, memory.SnapshotEntry{
		Type:  "hll",
		Value: uint64(5000),
	})
	require.NoError(t, err)

	err = store.OnExpire("response_time:"+oldDate, memory.SnapshotEntry{
		Type: "timer",
		Value: memory.TimerSnapshot{
			Count: 1000,
			Mean:  50000000,
			P50:   45000000,
			P95:   95000000,
			P99:   99000000,
		},
	})
	require.NoError(t, err)

	// Query by date range.
	records, err := store.Query(oldDate, oldDate)
	require.NoError(t, err)
	assert.Equal(t, 3, len(records), "3 records should be archived")

	// Query by type.
	counterRecords, err := store.QueryByType("counter", oldDate, oldDate)
	require.NoError(t, err)
	assert.Equal(t, 1, len(counterRecords))
	assert.Equal(t, "pv:"+oldDate+":/home", counterRecords[0].Key)

	// GetPV helper.
	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), pv)

	// GetUV helper.
	uv, err := store.GetUV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, uint64(5000), uv)

	t.Logf("Archived %d records for date %s", len(records), oldDate)
	for _, r := range records {
		t.Logf("  %s (%s): %s", r.Key, r.Type, r.Value)
	}
}

func TestSQLiteTTLIntegration(t *testing.T) {
	path := "/tmp/test_stats_ttl_integration.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	// Create memory collector with TTL that saves to SQLite on expire.
	c := memory.New(
		memory.WithReservoirTimer(4096),
		memory.WithTTL(memory.TTLConfig{
			RetentionDays: 1,
			OnExpire:      store.OnExpire,
		}),
	)
	defer c.Close()

	// Record data with old date.
	oldDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	c.Counter("pv:" + oldDate + ":/home").IncrBy(42)
	c.Counter("pv:" + oldDate + ":/about").IncrBy(18)
	c.HLL("uv:" + oldDate).Add("user1")
	c.HLL("uv:" + oldDate).Add("user2")
	c.HLL("uv:" + oldDate).Add("user3")

	// Trigger cleanup.
	removed := c.CleanupNow()
	assert.Equal(t, 3, removed, "3 keys should expire")

	// Verify data is in SQLite.
	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(60), pv) // 42 + 18 = 60 (sum across all paths)

	uv, err := store.GetUV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), uv)

	t.Logf("TTL → SQLite integration: %d keys expired, PV=%d, UV=%d", removed, pv, uv)
}
