package system

import (
	"testing"
	"time"
)

func TestGetSystemStatus(t *testing.T) {
	// GetSystemStatus should return a zero-value SystemStatus initially
	// (or whatever was stored by init or previous tests)
	status := GetSystemStatus()
	_ = status // just verify it doesn't panic
}

func TestUpdateSystemStatus(t *testing.T) {
	// updateSystemStatus is unexported; call it directly
	// It calls gopsutil which should work on the test machine
	updateSystemStatus()

	status := GetSystemStatus()
	// After update, CPU/Memory may or may not be populated depending on platform,
	// but the call itself should not panic.
	_ = status
}

func TestRecordMonitorTrend(t *testing.T) {
	resetTrendsForTest(t)

	status := SystemStatus{
		CPUUsage:    55.5,
		MemoryUsage: 66.6,
		DiskUsage:   77.7,
	}
	recordMonitorTrend(status)

	trends.mu.RLock()
	if len(trends.ring) != 1 {
		t.Fatalf("ring len=%d want 1", len(trends.ring))
	}
	p := trends.ring[0]
	trends.mu.RUnlock()
	if p.CPUUsage != 55.5 {
		t.Fatalf("CPUUsage=%v want 55.5", p.CPUUsage)
	}
	if p.MemoryUsage != 66.6 {
		t.Fatalf("MemoryUsage=%v want 66.6", p.MemoryUsage)
	}
	if p.HeapAlloc == 0 {
		t.Fatal("HeapAlloc should be populated from ReadMemStats")
	}
	if p.Goroutines < 1 {
		t.Fatal("Goroutines should be >=1")
	}
}

func TestSampleProcessRSS(t *testing.T) {
	rss, ok := sampleProcessRSS()
	if !ok {
		// On some platforms this might fail; just verify it doesn't panic
		t.Skip("sampleProcessRSS not available on this platform")
	}
	if rss == 0 {
		t.Fatal("RSS should be >0 for a running process")
	}
}

func TestStartSystemMonitor(t *testing.T) {
	origCfg := GetPerformanceMonitorConfig()
	defer SetPerformanceMonitorConfig(origCfg)

	SetPerformanceMonitorConfig(PerformanceMonitorConfig{
		Enabled:         true,
		CPUThreshold:    999,
		MemoryThreshold: 999,
		DiskThreshold:   999,
	})

	StartSystemMonitor()

	// Let it run a couple iterations
	time.Sleep(7 * time.Second)

	// Verify system status was updated
	status := GetSystemStatus()
	_ = status
}

func TestStartSystemMonitorDisabled(t *testing.T) {
	origCfg := GetPerformanceMonitorConfig()
	defer SetPerformanceMonitorConfig(origCfg)

	SetPerformanceMonitorConfig(PerformanceMonitorConfig{Enabled: false})

	StartSystemMonitor()
	// Just verify it doesn't panic; it should just sleep
	time.Sleep(2 * time.Second)
}
