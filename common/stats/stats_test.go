// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stats_test

import (
	"testing"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	c := memory.New()
	ctr := c.Counter("pv:home")
	assert.Equal(t, int64(0), ctr.Get())

	ctr.Incr()
	assert.Equal(t, int64(1), ctr.Get())

	ctr.IncrBy(10)
	assert.Equal(t, int64(11), ctr.Get())

	// Same key returns same counter.
	ctr2 := c.Counter("pv:home")
	assert.Equal(t, int64(11), ctr2.Get())

	ctr.Reset()
	assert.Equal(t, int64(0), ctr.Get())
}

func TestGauge(t *testing.T) {
	c := memory.New()
	g := c.Gauge("active_connections")

	g.Set(100)
	assert.Equal(t, int64(100), g.Get())

	g.Incr()
	assert.Equal(t, int64(101), g.Get())

	g.Decr()
	assert.Equal(t, int64(100), g.Get())
}

func TestSet(t *testing.T) {
	c := memory.New()
	s := c.Set("daily_users:2026-08-18")

	assert.True(t, s.Add("user1"))
	assert.False(t, s.Add("user1")) // duplicate
	assert.True(t, s.Add("user2"))

	assert.True(t, s.Has("user1"))
	assert.False(t, s.Has("user3"))
	assert.Equal(t, 2, s.Count())

	// Intersect
	s2 := c.Set("daily_users:2026-08-19")
	s2.Add("user1")
	s2.Add("user3")
	assert.Equal(t, 1, s.Intersect(s2)) // only user1 is in both
}

func TestHLL(t *testing.T) {
	c := memory.New()
	h := c.HLL("uv:2026-08-18")

	for i := 0; i < 10000; i++ {
		h.Add(string(rune(i)))
	}
	est := h.Estimate()
	// HLL has ~0.81% error; allow 5% tolerance for small samples.
	assert.True(t, est > 9500 && est < 10500, "expected ~10000, got %d", est)
}

func TestTimer(t *testing.T) {
	c := memory.New()
	timer := c.Timer("response_time:2026-08-18")

	samples := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, s := range samples {
		timer.Record(s)
	}

	assert.Equal(t, int64(10), timer.Count())
	assert.InDelta(t, 55.0, timer.Mean(), 0.1)
	// P50 with linear interpolation on [10..100] = 55.0
	assert.InDelta(t, 55.0, timer.Percentile(50), 0.1)
	// P95 with linear interpolation = 95.5
	assert.InDelta(t, 95.5, timer.Percentile(95), 0.5)
}

func TestWebsiteMetrics(t *testing.T) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"

	// PV
	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/about")
	assert.Equal(t, int64(2), wm.GetPV(date, "/home"))
	assert.Equal(t, int64(1), wm.GetPV(date, "/about"))

	// UV
	wm.RecordUV(date, "user1")
	wm.RecordUV(date, "user2")
	wm.RecordUV(date, "user1") // duplicate
	assert.Equal(t, uint64(2), wm.GetUV(date))

	// IP
	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "5.6.7.8")
	assert.Equal(t, uint64(2), wm.GetIP(date))

	// VV
	wm.RecordVV(date)
	wm.RecordVV(date)
	assert.Equal(t, int64(2), wm.GetVV(date))

	// Bounce rate
	wm.RecordBounce(date)
	assert.InDelta(t, 0.5, wm.GetBounceRate(date), 0.01)

	// CTR
	wm.RecordImpression(date, "banner")
	wm.RecordImpression(date, "banner")
	wm.RecordClick(date, "banner")
	assert.InDelta(t, 0.5, wm.GetCTR(date, "banner"), 0.01)

	// CVR
	wm.RecordConversion(date, "signup")
	assert.InDelta(t, 0.5, wm.GetCVR(date, "signup"), 0.01)

	// DAU / MAU
	wm.RecordDAU(date, "user1")
	wm.RecordDAU(date, "user2")
	assert.Equal(t, uint64(2), wm.GetDAU(date))

	wm.RecordMAU("2026-08", "user1")
	wm.RecordMAU("2026-08", "user2")
	assert.Equal(t, uint64(2), wm.GetMAU("2026-08"))

	// Retention
	wm.RecordDailyUserSet("2026-08-18", "user1")
	wm.RecordDailyUserSet("2026-08-18", "user2")
	wm.RecordDailyUserSet("2026-08-19", "user1")
	wm.RecordDailyUserSet("2026-08-19", "user3")
	assert.InDelta(t, 0.5, wm.GetRetention("2026-08-18", "2026-08-19"), 0.01)

	// Response time
	wm.RecordResponseTimeMs(date, 50)
	wm.RecordResponseTimeMs(date, 100)
	wm.RecordResponseTimeMs(date, 200)
	assert.Greater(t, wm.GetResponseTimeP95(date), 0.0)

	// QPS / Error rate
	wm.RecordRequest(date)
	wm.RecordRequest(date)
	wm.RecordError(date)
	assert.InDelta(t, 0.5, wm.GetErrorRate(date), 0.01)
	assert.Greater(t, wm.GetQPS(date), 0.0)
}

func TestCollectorInterface(t *testing.T) {
	// Verify memory.Collector implements stats.Collector.
	var _ stats.Collector = memory.New()
}
