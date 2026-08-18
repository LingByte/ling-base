// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redis_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	redisstats "github.com/LingByte/ling-base/common/stats/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getRedisClient returns a Redis client for testing, or skips the test
// if Redis is not available.
func getRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := client.Ping(t.Context()).Err(); err != nil {
		client.Close()
		t.Skipf("Redis not available: %v", err)
	}
	// Flush test keys before each test.
	client.FlushDB(t.Context())
	return client
}

// TestRedisCounter tests the Redis-backed counter.
func TestRedisCounter(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	ctr := c.Counter("pv:home")

	assert.Equal(t, int64(0), ctr.Get())
	ctr.Incr()
	assert.Equal(t, int64(1), ctr.Get())
	ctr.IncrBy(10)
	assert.Equal(t, int64(11), ctr.Get())

	// Same key returns same logical counter.
	ctr2 := c.Counter("pv:home")
	assert.Equal(t, int64(11), ctr2.Get())

	require.NoError(t, ctr.Reset())
	assert.Equal(t, int64(0), ctr.Get())
}

// TestRedisGauge tests the Redis-backed gauge.
func TestRedisGauge(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	g := c.Gauge("connections")

	g.Set(100)
	assert.Equal(t, int64(100), g.Get())
	g.Incr()
	assert.Equal(t, int64(101), g.Get())
	g.Decr()
	assert.Equal(t, int64(100), g.Get())
}

// TestRedisSet tests the Redis-backed set.
func TestRedisSet(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	s := c.Set("daily_users")

	assert.True(t, s.Add("user1"))
	assert.False(t, s.Add("user1"))
	assert.True(t, s.Add("user2"))
	assert.True(t, s.Has("user1"))
	assert.False(t, s.Has("user3"))
	assert.Equal(t, 2, s.Count())

	s2 := c.Set("daily_users_next")
	s2.Add("user1")
	s2.Add("user3")
	assert.Equal(t, 1, s.Intersect(s2))
}

// TestRedisHLL tests the Redis-backed HyperLogLog.
func TestRedisHLL(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	h := c.HLL("uv:2026-08-18")

	for i := 0; i < 10000; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
	est := h.Estimate()
	// PFADD/PFCOUNT has ~0.81% error.
	assert.True(t, est > 9500 && est < 10500, "expected ~10000, got %d", est)
}

// TestRedisHLLMerge tests merging two HLLs.
func TestRedisHLLMerge(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	h1 := c.HLL("uv:day1")
	h2 := c.HLL("uv:day2")

	for i := 0; i < 1000; i++ {
		h1.Add(fmt.Sprintf("user-%d", i))
		h2.Add(fmt.Sprintf("user-%d", i+500)) // 500 overlap
	}

	// Merge h2 into h1.
	require.NoError(t, h1.Merge(h2))
	merged := h1.Estimate()
	// 1000 + 1000 - 500 overlap = 1500 unique.
	assert.True(t, merged > 1400 && merged < 1600, "expected ~1500, got %d", merged)
}

// TestRedisTimer tests the Redis-backed timer.
func TestRedisTimer(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	timer := c.Timer("response_time")

	samples := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, s := range samples {
		timer.Record(s)
	}

	assert.Equal(t, int64(10), timer.Count())
	assert.InDelta(t, 55.0, timer.Mean(), 0.1)
	assert.InDelta(t, 55.0, timer.Percentile(50), 0.1)
	assert.InDelta(t, 95.5, timer.Percentile(95), 0.5)
}

// TestRedisWebsiteMetrics tests the WebsiteMetrics convenience layer with Redis.
func TestRedisWebsiteMetrics(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	c := redisstats.New(client, redisstats.WithKeyPrefix("test:"))
	defer c.Close()

	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"

	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/about")
	assert.Equal(t, int64(2), wm.GetPV(date, "/home"))

	wm.RecordUV(date, "user1")
	wm.RecordUV(date, "user2")
	wm.RecordUV(date, "user1")
	assert.Equal(t, uint64(2), wm.GetUV(date))

	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "5.6.7.8")
	assert.Equal(t, uint64(2), wm.GetIP(date))

	wm.RecordVV(date)
	wm.RecordVV(date)
	assert.Equal(t, int64(2), wm.GetVV(date))

	wm.RecordBounce(date)
	assert.InDelta(t, 0.5, wm.GetBounceRate(date), 0.01)

	wm.RecordImpression(date, "banner")
	wm.RecordImpression(date, "banner")
	wm.RecordClick(date, "banner")
	assert.InDelta(t, 0.5, wm.GetCTR(date, "banner"), 0.01)

	wm.RecordConversion(date, "signup")
	assert.InDelta(t, 0.5, wm.GetCVR(date, "signup"), 0.01)

	wm.RecordDAU(date, "user1")
	wm.RecordDAU(date, "user2")
	assert.Equal(t, uint64(2), wm.GetDAU(date))

	wm.RecordMAU("2026-08", "user1")
	wm.RecordMAU("2026-08", "user2")
	assert.Equal(t, uint64(2), wm.GetMAU("2026-08"))

	wm.RecordDailyUserSet("2026-08-17", "user1")
	wm.RecordDailyUserSet("2026-08-17", "user2")
	wm.RecordDailyUserSet("2026-08-18", "user1")
	wm.RecordDailyUserSet("2026-08-18", "user3")
	assert.InDelta(t, 0.5, wm.GetRetention("2026-08-17", "2026-08-18"), 0.01)

	wm.RecordResponseTimeMs(date, 50)
	wm.RecordResponseTimeMs(date, 100)
	wm.RecordResponseTimeMs(date, 200)
	assert.Greater(t, wm.GetResponseTimeP95(date), 0.0)

	wm.RecordRequest(date)
	wm.RecordRequest(date)
	wm.RecordError(date)
	assert.InDelta(t, 0.5, wm.GetErrorRate(date), 0.01)
}

// TestDualBackendConsistency verifies that memory and redis produce
// similar results for the same operations.
func TestDualBackendConsistency(t *testing.T) {
	client := getRedisClient(t)
	defer client.Close()

	memCollector := memory.New()
	redisCollector := redisstats.New(client, redisstats.WithKeyPrefix("dual:"))
	defer redisCollector.Close()

	memWM := stats.NewWebsiteMetrics(memCollector)
	redisWM := stats.NewWebsiteMetrics(redisCollector)
	date := "2026-08-18"

	// Record the same data on both backends.
	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("user-%d", i)
		path := fmt.Sprintf("/page/%d", i%5)
		ip := fmt.Sprintf("10.0.%d.%d", i/256, i%256)

		memWM.RecordPV(date, path)
		redisWM.RecordPV(date, path)
		memWM.RecordUV(date, userID)
		redisWM.RecordUV(date, userID)
		memWM.RecordIP(date, ip)
		redisWM.RecordIP(date, ip)
		memWM.RecordVV(date)
		redisWM.RecordVV(date)
		memWM.RecordRequest(date)
		redisWM.RecordRequest(date)
	}

	// Compare results.
	assert.Equal(t, memWM.GetPV(date, "/page/0"), redisWM.GetPV(date, "/page/0"))
	assert.Equal(t, memWM.GetVV(date), redisWM.GetVV(date))

	// UV and IP use HLL (probabilistic) — should be close but not exact.
	memUV := memWM.GetUV(date)
	redisUV := redisWM.GetUV(date)
	assert.True(t, memUV > 90 && memUV <= 100, "memory UV: %d", memUV)
	assert.True(t, redisUV > 90 && redisUV <= 100, "redis UV: %d", redisUV)

	memIP := memWM.GetIP(date)
	redisIP := redisWM.GetIP(date)
	assert.True(t, memIP > 90 && memIP <= 100, "memory IP: %d", memIP)
	assert.True(t, redisIP > 90 && redisIP <= 100, "redis IP: %d", redisIP)

	t.Logf("Dual backend consistency: UV(mem=%d, redis=%d), IP(mem=%d, redis=%d)",
		memUV, redisUV, memIP, redisIP)
}

// TestRedisCollectorInterface verifies the Redis collector implements stats.Collector.
func TestRedisCollectorInterface(t *testing.T) {
	var _ stats.Collector = redisstats.New(redis.NewClient(&redis.Options{}))
}

// TestMain flushes Redis before and after all tests.
func TestMain(m *testing.M) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Redis not available, skipping redis tests: %v\n", err)
		os.Exit(m.Run())
	}
	client.FlushDB(ctx)
	client.Close()

	code := m.Run()

	// Cleanup.
	client = redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	client.FlushDB(ctx)
	client.Close()

	os.Exit(code)
}

func init() {
	// Ensure redis is running.
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Redis not available: %v\n", err)
	}
	client.Close()
	// Give redis a moment to be ready.
	time.Sleep(100 * time.Millisecond)
}
