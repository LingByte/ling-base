// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sqlite

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore(t *testing.T) {
	path := "/tmp/test_stats_sqlite2.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	oldDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")

	// Save various record types.
	require.NoError(t, store.Save(stats.ArchiveRecord{
		Key: "pv:" + oldDate + ":/home", Type: "counter",
		Value: int64(1000), Date: oldDate, Archived: time.Now().Format(time.RFC3339),
	}))
	require.NoError(t, store.Save(stats.ArchiveRecord{
		Key: "uv:" + oldDate, Type: "hll",
		Value: uint64(5000), Date: oldDate, Archived: time.Now().Format(time.RFC3339),
	}))
	require.NoError(t, store.Save(stats.ArchiveRecord{
		Key: "response_time:" + oldDate, Type: "timer",
		Value: stats.TimerSummary{Count: 1000, Mean: 50e6, P50: 45e6, P95: 95e6, P99: 99e6},
		Date: oldDate, Archived: time.Now().Format(time.RFC3339),
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

	// Upsert test — same key should replace.
	require.NoError(t, store.Save(stats.ArchiveRecord{
		Key: "pv:" + oldDate + ":/home", Type: "counter",
		Value: int64(2000), Date: oldDate, Archived: time.Now().Format(time.RFC3339),
	}))
	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), pv, "upserted value should be 2000")

	t.Logf("Archived %d records for date %s", len(records), oldDate)
}

func TestSQLiteTTLIntegration(t *testing.T) {
	path := "/tmp/test_stats_ttl_sqlite2.db"
	defer os.Remove(path)

	store, err := New(path)
	require.NoError(t, err)
	defer store.Close()

	// Use ArchiveAdapter to bridge TTL → ArchiveStore.
	c := memory.New(
		memory.WithReservoirTimer(4096),
		memory.WithTTL(memory.TTLConfig{
			RetentionDays: 1,
			OnExpire:      memory.ArchiveAdapter(store),
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

	// Verify in SQLite.
	pv, err := store.GetPV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, int64(60), pv)

	uv, err := store.GetUV(oldDate)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), uv)

	t.Logf("TTL → SQLite: %d keys expired, PV=%d, UV=%d", removed, pv, uv)
}

func TestSQLiteInterfaceCompliance(t *testing.T) {
	var _ stats.ArchiveStore = (*Store)(nil)
}

func TestArchiveRecordJSON(t *testing.T) {
	// Verify TimerSummary serializes correctly.
	r := stats.ArchiveRecord{
		Key:   "rt:2026-08-18",
		Type:  "timer",
		Value: stats.TimerSummary{Count: 100, Mean: 50, P50: 45, P95: 95, P99: 99},
	}
	data, err := json.Marshal(r.Value)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"p95":95`)
}
