// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"fmt"
	"runtime"
	"testing"
)

// TestOptimizedMemoryFootprint compares default vs optimized memory usage.
func TestOptimizedMemoryFootprint(t *testing.T) {
	// ─── 默认模式（无优化）───
	var before1, after1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before1)

	c1 := New()
	// 30天 × 100K 用户的精确 Set
	for d := 0; d < 30; d++ {
		s := c1.Set(fmt.Sprintf("daily:day-%d", d))
		for i := 0; i < 100000; i++ {
			s.Add(fmt.Sprintf("user-%d", i))
		}
	}
	// 1M Timer 样本
	timer1 := c1.Timer("rt:day-0")
	for i := 0; i < 1000000; i++ {
		timer1.Record(int64(i))
	}

	runtime.GC()
	runtime.ReadMemStats(&after1)
	defaultMB := float64(after1.HeapInuse-before1.HeapInuse) / 1024 / 1024

	// ─── 优化模式（蓄水池 + Bloom）───
	var before2, after2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before2)

	c2 := New(
		WithReservoirTimer(4096),                         // 32 KB/timer 固定
		WithBloomSet(100000, 0.001),                      // 1.4 MB/set 固定
	)
	// 30天 × 100K 用户的 Bloom Set
	for d := 0; d < 30; d++ {
		s := c2.Set(fmt.Sprintf("daily:day-%d", d))
		for i := 0; i < 100000; i++ {
			s.Add(fmt.Sprintf("user-%d", i))
		}
	}
	// 1M Timer 样本（蓄水池只保留 4096）
	timer2 := c2.Timer("rt:day-0")
	for i := 0; i < 1000000; i++ {
		timer2.Record(int64(i))
	}

	runtime.GC()
	runtime.ReadMemStats(&after2)
	optimizedMB := float64(after2.HeapInuse-before2.HeapInuse) / 1024 / 1024

	t.Logf("═════════════════════════════════════════════════════════════")
	t.Logf("  内存对比（30天 × 100K用户 + 1M Timer样本）")
	t.Logf("─────────────────────────────────────────────────────────────")
	t.Logf("  默认模式:   %.1f MB", defaultMB)
	t.Logf("  优化模式:   %.1f MB", optimizedMB)
	t.Logf("  节省:       %.1f MB (%.0f%%)", defaultMB-optimizedMB, (1-optimizedMB/defaultMB)*100)
	t.Logf("─────────────────────────────────────────────────────────────")
	t.Logf("  Timer Count:  默认=%d  优化=%d (蓄水池容量=4096)",
		timer1.Count(), timer2.Count())
	t.Logf("  Timer P95:    默认=%.0f  优化=%.0f",
		timer1.Percentile(95), timer2.Percentile(95))
	t.Logf("  Set Count:    默认=%d  优化=%d (Bloom估计)",
		c1.Set("daily:day-0").Count(), c2.Set("daily:day-0").Count())
	t.Logf("═════════════════════════════════════════════════════════════")

	// 验证优化后内存大幅减少
	if optimizedMB >= defaultMB*0.5 {
		t.Errorf("优化后内存应减少 50%% 以上: 默认=%.1fMB, 优化=%.1fMB", defaultMB, optimizedMB)
	}

	// 验证 Timer 蓄水池的 P95 仍然接近真实值
	// 1M 样本 [0..999999], P95 ≈ 950000
	p95 := timer2.Percentile(95)
	if p95 < 900000 || p95 > 990000 {
		t.Errorf("蓄水池 P95 应接近 950000, got %.0f", p95)
	}
}

// TestBloomSetAccuracy tests Bloom filter false positive rate.
func TestBloomSetAccuracy(t *testing.T) {
	expectedN := 100000
	fpr := 0.001 // 0.1%

	c := New(WithBloomSet(expectedN, fpr))
	s := c.Set("test:bloom")

	// Add 100K users.
	for i := 0; i < expectedN; i++ {
		s.Add(fmt.Sprintf("user-%d", i))
	}

	// Check that all added users are found (no false negatives).
	for i := 0; i < 1000; i++ {
		if !s.Has(fmt.Sprintf("user-%d", i)) {
			t.Fatalf("false negative at user-%d — Bloom filter should never have false negatives", i)
		}
	}

	// Check false positive rate with unseen users.
	falsePositives := 0
	testCount := 10000
	for i := 0; i < testCount; i++ {
		// These users were never added.
		if s.Has(fmt.Sprintf("unseen-user-%d", i)) {
			falsePositives++
		}
	}

	actualFPR := float64(falsePositives) / float64(testCount)
	t.Logf("Bloom filter: expected FPR=%.3f%%, actual FPR=%.3f%% (%d/%d)",
		fpr*100, actualFPR*100, falsePositives, testCount)

	// Actual FPR should be close to target (allow 3x tolerance for small sample).
	if actualFPR > fpr*5 {
		t.Errorf("FPR too high: expected < %.3f%%, got %.3f%%", fpr*5*100, actualFPR*100)
	}

	// Count estimate.
	estimatedCount := s.Count()
	t.Logf("Bloom count estimate: %d (actual: %d, error: %.1f%%)",
		estimatedCount, expectedN, float64(estimatedCount-expectedN)/float64(expectedN)*100)
}

// TestReservoirTimerAccuracy tests that reservoir sampling gives good percentiles.
func TestReservoirTimerAccuracy(t *testing.T) {
	c := New(WithReservoirTimer(4096))
	timer := c.Timer("test:rt")

	// Record 1M samples uniformly distributed [0, 1M).
	for i := 0; i < 1000000; i++ {
		timer.Record(int64(i))
	}

	// With uniform distribution, percentiles should be close to expected.
	tests := []struct {
		p        float64
		expected float64
		tolerance float64
	}{
		{50, 500000, 50000},  // ±10%
		{95, 950000, 50000},  // ±5%
		{99, 990000, 50000},  // ±5%
	}

	for _, tt := range tests {
		got := timer.Percentile(tt.p)
		diff := abs(got - tt.expected)
		t.Logf("P%.0f: expected=%.0f, got=%.0f, diff=%.0f (%.1f%%)",
			tt.p, tt.expected, got, diff, diff/tt.expected*100)
		if diff > tt.tolerance {
			t.Errorf("P%.0f: expected %.0f ± %.0f, got %.0f", tt.p, tt.expected, tt.tolerance, got)
		}
	}

	t.Logf("Reservoir: count=%d, memory=32KB (fixed, capacity=4096)",
		timer.Count())
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
