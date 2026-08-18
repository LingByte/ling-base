// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"fmt"
	"runtime"
	"testing"
)

// TestMemoryFootprint measures actual memory usage of each primitive.
func TestMemoryFootprint(t *testing.T) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	c := New()

	// Simulate: 1000 paths × 30 days of PV counters
	for d := 0; d < 30; d++ {
		for p := 0; p < 1000; p++ {
			c.Counter(fmt.Sprintf("pv:2026-%02d-%02d:/page/%d", d/28+1, d%28+1, p)).Incr()
		}
	}

	// Simulate: 100K UV via HLL (30 days)
	for d := 0; d < 30; d++ {
		h := c.HLL(fmt.Sprintf("uv:day-%d", d))
		for i := 0; i < 100000; i++ {
			h.Add(fmt.Sprintf("user-%d", i))
		}
	}

	// Simulate: 100K exact Set for retention (30 days)
	for d := 0; d < 30; d++ {
		s := c.Set(fmt.Sprintf("daily_users:day-%d", d))
		for i := 0; i < 100000; i++ {
			s.Add(fmt.Sprintf("user-%d", i))
		}
	}

	// Simulate: 1M Timer samples (response times)
	timer := c.Timer("response_time:day-0")
	for i := 0; i < 1000000; i++ {
		timer.Record(int64(i))
	}

	runtime.GC()
	runtime.ReadMemStats(&after)

	allocMB := float64(after.Alloc-before.Alloc) / 1024 / 1024
	totalAllocMB := float64(after.TotalAlloc-before.TotalAlloc) / 1024 / 1024

	t.Logf("══════════════════════════════════════════════════════")
	t.Logf("  内存占用分析（30天数据）")
	t.Logf("──────────────────────────────────────────────────────")
	t.Logf("  Counter:  30K keys (30天 × 1000路径)      ~%.1f MB", float64(30*1000*120)/1024/1024)
	t.Logf("  HLL:      30 keys × 100K users each       ~%.1f MB (12KB/key 固定)", float64(30*12*1024)/1024/1024)
	t.Logf("  Set:      30 keys × 100K users each       ~%.1f MB (精确去重, 每用户~80B)", float64(30*100000*80)/1024/1024)
	t.Logf("  Timer:    1M samples (无上限)             ~%.1f MB (8B/sample, 无限增长!)", float64(1000000*8)/1024/1024)
	t.Logf("──────────────────────────────────────────────────────")
	t.Logf("  实际 Alloc:        %.1f MB", allocMB)
	t.Logf("  实际 TotalAlloc:   %.1f MB", totalAllocMB)
	t.Logf("  NumGC:             %d", after.NumGC-before.NumGC)
	t.Logf("══════════════════════════════════════════════════════")

	// Show the danger zones
	setMB := float64(30*100000*80) / 1024 / 1024
	timerMB := float64(1000000*8) / 1024 / 1024
	t.Logf("\n⚠ 风险点:")
	t.Logf("  Set (精确去重):     %.1f MB — 1M用户/天 × 30天 = %.1f MB", setMB, float64(30*1000000*80)/1024/1024)
	t.Logf("  Timer (全量样本):   %.1f MB — 10M请求/天 × 30天 = %.1f MB", timerMB, float64(300000000*8)/1024/1024)
}
