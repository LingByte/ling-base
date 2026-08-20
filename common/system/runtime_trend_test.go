package system

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// resetTrendsForTest resets the global trend store to a clean state with a temp dir.
func resetTrendsForTest(t *testing.T) {
	t.Helper()
	trends.mu.Lock()
	defer trends.mu.Unlock()
	trends.ring = nil
	trends.loaded = false
	trends.lastDay = ""
	trends.lastSampleAt = time.Time{}
	if trends.file != nil {
		trends.file.Close()
		trends.file = nil
		trends.writer = nil
	}
	trends.dir = t.TempDir()
	trends.cap = defaultTrendRingCap
	trends.retainD = defaultTrendRetainD
	trends.intervalSec = defaultTrendInterval
}

func TestParseTrendWindow(t *testing.T) {
	cases := map[string]time.Duration{
		"":      time.Hour,
		"1h":    time.Hour,
		"15m":   15 * time.Minute,
		"24h":   24 * time.Hour,
		"bogus": time.Hour,
	}
	for in, want := range cases {
		if got := parseTrendWindow(in); got != want {
			t.Fatalf("parseTrendWindow(%q)=%v want %v", in, got, want)
		}
	}
}

func TestParseTrendWindowAllCases(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"", time.Hour},
		{"1h", time.Hour},
		{"60m", time.Hour},
		{"15m", 15 * time.Minute},
		{"30m", 30 * time.Minute},
		{"6h", 6 * time.Hour},
		{"12h", 12 * time.Hour},
		{"24h", 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"bogus", time.Hour},
		{"2h", 2 * time.Hour}, // valid ParseDuration
		{"999h", time.Hour},   // exceeds 7*24h, fallback
		{"-1h", time.Hour},    // negative, fallback
		{"0s", time.Hour},     // zero, fallback
	}
	for _, c := range cases {
		got := parseTrendWindow(c.in)
		if got != c.want {
			t.Fatalf("parseTrendWindow(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestSummarizeTrendRising(t *testing.T) {
	var points []RuntimeTrendPoint
	base := time.Now().UnixMilli()
	for i := 0; i < 20; i++ {
		points = append(points, RuntimeTrendPoint{
			Ts:         base + int64(i)*5000,
			HeapAlloc:  uint64(100<<20) + uint64(i)*(10<<20), // +10MiB each step
			Goroutines: 100 + i*5,
		})
	}
	sum := summarizeTrend(points, int64(time.Hour/time.Millisecond))
	if !sum.HeapAllocRising {
		t.Fatalf("expected heapAllocRising, delta=%d", sum.HeapAllocDelta)
	}
	if !sum.GoroutineRising {
		t.Fatalf("expected goroutineRising, delta=%d", sum.GoroutineDelta)
	}
}

func TestSummarizeTrendFlat(t *testing.T) {
	var points []RuntimeTrendPoint
	base := time.Now().UnixMilli()
	for i := 0; i < 20; i++ {
		// sawtooth: up then down — should not flag as rising
		heap := uint64(100 << 20)
		if i%2 == 0 {
			heap += 1 << 20
		}
		points = append(points, RuntimeTrendPoint{
			Ts:         base + int64(i)*5000,
			HeapAlloc:  heap,
			Goroutines: 100,
		})
	}
	sum := summarizeTrend(points, int64(time.Hour/time.Millisecond))
	if sum.HeapAllocRising {
		t.Fatalf("sawtooth should not be rising")
	}
	if sum.GoroutineRising {
		t.Fatalf("flat goroutines should not be rising")
	}
}

func TestSummarizeTrendEdgeCases(t *testing.T) {
	// empty
	sum := summarizeTrend(nil, 3600000)
	if sum.SampleCount != 0 {
		t.Fatalf("empty SampleCount=%d want 0", sum.SampleCount)
	}

	// single point
	sum = summarizeTrend([]RuntimeTrendPoint{{Ts: 100, HeapAlloc: 50}}, 3600000)
	if sum.SampleCount != 1 {
		t.Fatalf("single SampleCount=%d want 1", sum.SampleCount)
	}
	if sum.FirstTs != 100 || sum.LastTs != 100 {
		t.Fatalf("single FirstTs=%d LastTs=%d want 100,100", sum.FirstTs, sum.LastTs)
	}
}

// Test summarizeTrend with goroutine rising via percentage threshold
func TestSummarizeTrendGoroutineRisingPercent(t *testing.T) {
	var points []RuntimeTrendPoint
	base := time.Now().UnixMilli()
	// Start with 100 goroutines, end with 115 (15% growth, >10% threshold)
	for i := 0; i < 20; i++ {
		points = append(points, RuntimeTrendPoint{
			Ts:         base + int64(i)*5000,
			Goroutines: 100 + i,
		})
	}
	sum := summarizeTrend(points, int64(3600000))
	if !sum.GoroutineRising {
		t.Fatalf("expected goroutineRising, delta=%d", sum.GoroutineDelta)
	}
}

// Test summarizeTrend with heap rising via percentage threshold
func TestSummarizeTrendHeapRisingPercent(t *testing.T) {
	var points []RuntimeTrendPoint
	base := time.Now().UnixMilli()
	// Start with 100MB, end with 120MB (20% growth, >10% threshold)
	for i := 0; i < 20; i++ {
		points = append(points, RuntimeTrendPoint{
			Ts:        base + int64(i)*5000,
			HeapAlloc: uint64(100<<20) + uint64(i)*(1<<20),
		})
	}
	sum := summarizeTrend(points, int64(3600000))
	if !sum.HeapAllocRising {
		t.Fatalf("expected heapAllocRising, delta=%d", sum.HeapAllocDelta)
	}
}

func TestDownsampleTrend(t *testing.T) {
	in := make([]RuntimeTrendPoint, 100)
	for i := range in {
		in[i].Ts = int64(i)
		in[i].HeapAlloc = uint64(i)
	}
	out := downsampleTrend(in, 10)
	if len(out) != 10 {
		t.Fatalf("len=%d want 10", len(out))
	}
	if out[0].Ts != 0 || out[len(out)-1].Ts != 99 {
		t.Fatalf("endpoints lost: first=%d last=%d", out[0].Ts, out[len(out)-1].Ts)
	}
}

func TestDownsampleTrendEdgeCases(t *testing.T) {
	// empty input
	out := downsampleTrend(nil, 10)
	if len(out) != 0 {
		t.Fatalf("downsample(nil) len=%d want 0", len(out))
	}

	// maxPts=0 → return copy
	in := []RuntimeTrendPoint{{Ts: 1}, {Ts: 2}}
	out = downsampleTrend(in, 0)
	if len(out) != 2 {
		t.Fatalf("downsample(maxPts=0) len=%d want 2", len(out))
	}

	// len <= maxPts → return copy
	in = []RuntimeTrendPoint{{Ts: 1}, {Ts: 2}, {Ts: 3}}
	out = downsampleTrend(in, 5)
	if len(out) != 3 {
		t.Fatalf("downsample(len<max) len=%d want 3", len(out))
	}

	// exactly maxPts
	out = downsampleTrend(in, 3)
	if len(out) != 3 {
		t.Fatalf("downsample(len==max) len=%d want 3", len(out))
	}
}

// Test downsampleTrend with maxPts=1 (edge case)
func TestDownsampleTrendMaxPts1(t *testing.T) {
	in := []RuntimeTrendPoint{{Ts: 1}, {Ts: 2}, {Ts: 3}}
	out := downsampleTrend(in, 1)
	// With maxPts=1 and len > maxPts, the function keeps first and last
	// but inner=1-2=-1, so the loop doesn't execute, and we get first+last
	if len(out) == 0 {
		t.Fatal("downsample should not return empty")
	}
}

// Test downsampleTrend with maxPts=2
func TestDownsampleTrendMaxPts2(t *testing.T) {
	in := []RuntimeTrendPoint{{Ts: 1}, {Ts: 2}, {Ts: 3}, {Ts: 4}, {Ts: 5}}
	out := downsampleTrend(in, 2)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	if out[0].Ts != 1 || out[1].Ts != 5 {
		t.Fatalf("endpoints: first=%d last=%d want 1,5", out[0].Ts, out[1].Ts)
	}
}

func TestTrendSampleInterval(t *testing.T) {
	resetTrendsForTest(t)

	d := TrendSampleInterval()
	if d != 30*time.Second {
		t.Fatalf("TrendSampleInterval=%v want 30s", d)
	}

	trends.mu.Lock()
	trends.intervalSec = 60
	trends.mu.Unlock()
	d = TrendSampleInterval()
	if d != 60*time.Second {
		t.Fatalf("TrendSampleInterval=%v want 60s", d)
	}
}

func TestShouldSampleTrend(t *testing.T) {
	resetTrendsForTest(t)

	// First call should always return true (lastSampleAt is zero)
	if !ShouldSampleTrend() {
		t.Fatal("first ShouldSampleTrend should be true")
	}

	// Set lastSampleAt to now → should return false (interval not elapsed)
	trends.mu.Lock()
	trends.lastSampleAt = time.Now()
	trends.intervalSec = 30
	trends.mu.Unlock()
	if ShouldSampleTrend() {
		t.Fatal("ShouldSampleTrend should be false right after sample")
	}

	// Set lastSampleAt to 31 seconds ago → should return true
	trends.mu.Lock()
	trends.lastSampleAt = time.Now().Add(-31 * time.Second)
	trends.mu.Unlock()
	if !ShouldSampleTrend() {
		t.Fatal("ShouldSampleTrend should be true after interval elapsed")
	}
}

func TestRecordRuntimeTrend(t *testing.T) {
	resetTrendsForTest(t)

	now := time.Now().UnixMilli()
	p := RuntimeTrendPoint{
		Ts:         now,
		HeapAlloc:  1000,
		Goroutines: 10,
	}
	RecordRuntimeTrend(p)

	trends.mu.RLock()
	if len(trends.ring) != 1 {
		t.Fatalf("ring len=%d want 1", len(trends.ring))
	}
	if trends.ring[0].HeapAlloc != 1000 {
		t.Fatalf("HeapAlloc=%d want 1000", trends.ring[0].HeapAlloc)
	}
	trends.mu.RUnlock()

	// Record with Ts=0 → should auto-fill
	RecordRuntimeTrend(RuntimeTrendPoint{HeapAlloc: 2000})
	trends.mu.RLock()
	if len(trends.ring) != 2 {
		t.Fatalf("ring len=%d want 2", len(trends.ring))
	}
	if trends.ring[1].Ts <= 0 {
		t.Fatal("auto-filled Ts should be positive")
	}
	trends.mu.RUnlock()
}

func TestRecordRuntimeTrendRingOverflow(t *testing.T) {
	resetTrendsForTest(t)
	trends.mu.Lock()
	trends.cap = 5
	trends.mu.Unlock()

	for i := 0; i < 20; i++ {
		RecordRuntimeTrend(RuntimeTrendPoint{
			Ts:         time.Now().Add(-time.Duration(20-i) * time.Second).UnixMilli(),
			HeapAlloc:  uint64(i),
			Goroutines: i,
		})
	}

	trends.mu.RLock()
	if len(trends.ring) > trends.cap {
		t.Fatalf("ring len=%d exceeds cap=%d", len(trends.ring), trends.cap)
	}
	if len(trends.ring) == 0 {
		t.Fatal("ring should not be empty")
	}
	trends.mu.RUnlock()
}

func TestGetRuntimeTrend(t *testing.T) {
	resetTrendsForTest(t)

	now := time.Now().UnixMilli()
	// Insert points spanning 2 hours
	for i := 0; i < 10; i++ {
		RecordRuntimeTrend(RuntimeTrendPoint{
			Ts:         now - int64((10-i)*60*1000), // 10 min apart, oldest = 90 min ago
			HeapAlloc:  uint64(i * 1000),
			Goroutines: 10 + i,
		})
	}

	// 1h window should return points within the last hour
	snap := GetRuntimeTrend("1h")
	if snap.IntervalSeconds != defaultTrendInterval {
		t.Fatalf("IntervalSeconds=%d want %d", snap.IntervalSeconds, defaultTrendInterval)
	}
	if len(snap.Points) == 0 {
		t.Fatal("expected some points in 1h window")
	}

	// 24h window should return all points
	snap = GetRuntimeTrend("24h")
	if len(snap.Points) == 0 {
		t.Fatal("expected points in 24h window")
	}
	if snap.Summary.SampleCount == 0 {
		t.Fatal("SampleCount should not be 0")
	}
}

func TestGetRuntimeTrendEmpty(t *testing.T) {
	resetTrendsForTest(t)

	snap := GetRuntimeTrend("1h")
	if len(snap.Points) != 0 {
		t.Fatalf("expected 0 points, got %d", len(snap.Points))
	}
	if snap.Summary.SampleCount != 0 {
		t.Fatalf("expected SampleCount=0, got %d", snap.Summary.SampleCount)
	}
}

func TestGetRuntimeTrendSinglePoint(t *testing.T) {
	resetTrendsForTest(t)

	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         time.Now().UnixMilli(),
		HeapAlloc:  500,
		Goroutines: 5,
	})

	snap := GetRuntimeTrend("1h")
	if snap.Summary.SampleCount != 1 {
		t.Fatalf("SampleCount=%d want 1", snap.Summary.SampleCount)
	}
	if snap.Summary.FirstTs != snap.Summary.LastTs {
		t.Fatal("FirstTs should equal LastTs for single point")
	}
}

// Test GetRuntimeTrend with custom interval
func TestGetRuntimeTrendCustomInterval(t *testing.T) {
	resetTrendsForTest(t)

	trends.mu.Lock()
	trends.intervalSec = 0 // zero → should fallback to default
	trends.mu.Unlock()

	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         time.Now().UnixMilli(),
		HeapAlloc:  1,
		Goroutines: 1,
	})

	snap := GetRuntimeTrend("1h")
	if snap.IntervalSeconds != defaultTrendInterval {
		t.Fatalf("IntervalSeconds=%d want %d (default fallback)", snap.IntervalSeconds, defaultTrendInterval)
	}
}

func TestTrendPersistAndLoad(t *testing.T) {
	resetTrendsForTest(t)

	// Write a point → should persist to disk
	now := time.Now()
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         now.UnixMilli(),
		HeapAlloc:  12345,
		Goroutines: 42,
	})

	// Verify JSONL file exists
	day := now.Format(trendFileDateLayout)
	path := filepath.Join(trends.dir, day+".jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("JSONL file not created: %v", err)
	}

	// Reset the in-memory ring and reload
	trends.mu.Lock()
	trends.ring = nil
	trends.loaded = false
	trends.mu.Unlock()

	// Record another point → triggers loadLocked, should restore previous point
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         now.Add(time.Second).UnixMilli(),
		HeapAlloc:  67890,
		Goroutines: 99,
	})

	trends.mu.RLock()
	if len(trends.ring) < 2 {
		t.Fatalf("after reload ring len=%d want >=2", len(trends.ring))
	}
	// First loaded point should have HeapAlloc=12345
	found := false
	for _, p := range trends.ring {
		if p.HeapAlloc == 12345 {
			found = true
			break
		}
	}
	trends.mu.RUnlock()
	if !found {
		t.Fatal("persisted point not loaded from disk")
	}
}

func TestTrendPrune(t *testing.T) {
	resetTrendsForTest(t)

	// Create an old JSONL file (8 days ago → beyond 7-day retention)
	oldDay := time.Now().Add(-8 * 24 * time.Hour).Format(trendFileDateLayout)
	oldPath := filepath.Join(trends.dir, oldDay+".jsonl")
	if err := os.WriteFile(oldPath, []byte(`{"ts":1}`+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create a recent file
	recentDay := time.Now().Format(trendFileDateLayout)
	recentPath := filepath.Join(trends.dir, recentDay+".jsonl")
	if err := os.WriteFile(recentPath, []byte(`{"ts":2}`+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Trigger prune by recording a point
	RecordRuntimeTrend(RuntimeTrendPoint{Ts: time.Now().UnixMilli(), HeapAlloc: 1})

	// Old file should be pruned
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("old JSONL file should have been pruned")
	}
	// Recent file should still exist
	if _, err := os.Stat(recentPath); err != nil {
		t.Fatalf("recent JSONL file should still exist: %v", err)
	}
}

func TestTrendPersistMkdirFail(t *testing.T) {
	resetTrendsForTest(t)

	// Set dir to an unwritable path
	trends.mu.Lock()
	trends.dir = "/nonexistent-trend-xyz/sub"
	trends.loaded = true // skip loadLocked
	trends.mu.Unlock()

	// This should not panic, just log a warning
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         time.Now().UnixMilli(),
		HeapAlloc:  1,
		Goroutines: 1,
	})
}

// Test trend persistLocked with file open error
func TestTrendPersistFileOpenError(t *testing.T) {
	resetTrendsForTest(t)

	// Create the dir as a file to cause OpenFile to fail
	trends.mu.Lock()
	dir := trends.dir
	trends.mu.Unlock()
	os.RemoveAll(dir)
	if err := os.WriteFile(dir, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)

	// This should fail to create dir and log warning, not panic
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         1,
		HeapAlloc:  1,
		Goroutines: 1,
	})
}

// Test trend persistLocked with day change
func TestTrendPersistDayChange(t *testing.T) {
	resetTrendsForTest(t)

	// Record a point for "today"
	now := time.Now()
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         now.UnixMilli(),
		HeapAlloc:  100,
		Goroutines: 10,
	})

	// Record a point for "yesterday" to trigger day change
	yesterday := now.Add(-24 * time.Hour)
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         yesterday.UnixMilli(),
		HeapAlloc:  200,
		Goroutines: 20,
	})

	// Both should be in the ring
	trends.mu.RLock()
	if len(trends.ring) < 2 {
		t.Fatalf("ring len=%d want >=2", len(trends.ring))
	}
	trends.mu.RUnlock()
}

// Test loadLocked with mkdir failure
func TestTrendLoadMkdirFail(t *testing.T) {
	// Set dir to a path under a file
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	trends.mu.Lock()
	trends.ring = nil
	trends.loaded = false
	trends.dir = filepath.Join(blocker, "sub")
	trends.mu.Unlock()

	// Recording will trigger loadLocked which will fail to mkdir
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         time.Now().UnixMilli(),
		HeapAlloc:  1,
		Goroutines: 1,
	})

	// Should not panic, ring may be empty or have the point
	trends.mu.RLock()
	_ = trends.ring
	trends.mu.RUnlock()
}

// Test pruneLocked with nonexistent dir
func TestTrendPruneNonexistentDir(t *testing.T) {
	resetTrendsForTest(t)

	trends.mu.Lock()
	trends.dir = "/nonexistent-prune-test-xyz"
	trends.mu.Unlock()

	// Directly call pruneLocked via a record
	RecordRuntimeTrend(RuntimeTrendPoint{
		Ts:         time.Now().UnixMilli(),
		HeapAlloc:  1,
		Goroutines: 1,
	})
}

func TestTrendConcurrentAccess(t *testing.T) {
	resetTrendsForTest(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			RecordRuntimeTrend(RuntimeTrendPoint{
				Ts:         time.Now().UnixMilli(),
				HeapAlloc:  uint64(n),
				Goroutines: n,
			})
			_ = GetRuntimeTrend("1h")
			_ = ShouldSampleTrend()
			_ = TrendSampleInterval()
		}(i)
	}
	wg.Wait()
}
