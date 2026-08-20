package system

import "testing"

func TestPerformanceMonitorConfig(t *testing.T) {
	cfg := GetPerformanceMonitorConfig()
	if !cfg.Enabled {
		t.Fatal("default config should be enabled")
	}

	SetPerformanceMonitorConfig(PerformanceMonitorConfig{
		Enabled:         false,
		CPUThreshold:    50,
		MemoryThreshold: 60,
		DiskThreshold:   70,
	})
	cfg = GetPerformanceMonitorConfig()
	if cfg.Enabled {
		t.Fatal("config should be disabled")
	}
	if cfg.CPUThreshold != 50 {
		t.Fatalf("CPUThreshold=%d want 50", cfg.CPUThreshold)
	}

	// restore default
	SetPerformanceMonitorConfig(PerformanceMonitorConfig{
		Enabled:         true,
		CPUThreshold:    90,
		MemoryThreshold: 90,
		DiskThreshold:   90,
	})
}
