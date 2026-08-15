package system

import "testing"

func TestDiskCacheConfigGetSet(t *testing.T) {
	SetDiskCacheConfig(DiskCacheConfig{
		Enabled:     true,
		ThresholdMB: 5,
		MaxSizeMB:   100,
		Path:        "/tmp/test-cache",
	})
	cfg := GetDiskCacheConfig()
	if !cfg.Enabled {
		t.Fatal("config should be enabled")
	}
	if cfg.ThresholdMB != 5 {
		t.Fatalf("ThresholdMB=%d want 5", cfg.ThresholdMB)
	}
	if GetDiskCacheThresholdBytes() != 5<<20 {
		t.Fatalf("threshold bytes=%d want %d", GetDiskCacheThresholdBytes(), 5<<20)
	}
	if GetDiskCacheMaxSizeBytes() != 100<<20 {
		t.Fatalf("max bytes=%d want %d", GetDiskCacheMaxSizeBytes(), 100<<20)
	}
	if GetDiskCachePath() != "/tmp/test-cache" {
		t.Fatalf("path=%q want /tmp/test-cache", GetDiskCachePath())
	}
	if !IsDiskCacheEnabled() {
		t.Fatal("IsDiskCacheEnabled should be true")
	}

	// ShouldUseDiskCache
	if !ShouldUseDiskCache(10 << 20) {
		t.Fatal("should use disk cache for 10MB when threshold is 5MB and available")
	}
	if ShouldUseDiskCache(1 << 20) {
		t.Fatal("should not use disk cache for 1MB when threshold is 5MB")
	}

	// reset
	SetDiskCacheConfig(DiskCacheConfig{
		Enabled:     false,
		ThresholdMB: 10,
		MaxSizeMB:   1024,
		Path:        "",
	})
}

func TestDiskCacheStatsIncrementDecrement(t *testing.T) {
	IncrementDiskFiles(100)
	IncrementDiskFiles(200)
	s := GetDiskCacheStats()
	if s.ActiveDiskFiles < 2 {
		t.Fatalf("ActiveDiskFiles=%d want >=2", s.ActiveDiskFiles)
	}

	DecrementDiskFiles(100)
	DecrementDiskFiles(200)

	IncrementMemoryBuffers(50)
	IncrementMemoryCacheHits()
	IncrementDiskCacheHits()

	ResetDiskCacheStats()
	s = GetDiskCacheStats()
	if s.DiskCacheHits != 0 || s.MemoryCacheHits != 0 {
		t.Fatal("hits should be reset")
	}

	ResetDiskCacheUsage()
}

func TestDecrementDiskFilesClamp(t *testing.T) {
	ResetDiskCacheUsage()
	// decrement below zero → clamped to 0
	DecrementDiskFiles(100)
	s := GetDiskCacheStats()
	if s.ActiveDiskFiles != 0 {
		t.Fatalf("clamped ActiveDiskFiles=%d want 0", s.ActiveDiskFiles)
	}
	if s.CurrentDiskUsageBytes != 0 {
		t.Fatalf("clamped CurrentDiskUsageBytes=%d want 0", s.CurrentDiskUsageBytes)
	}
}

func TestDecrementMemoryBuffers(t *testing.T) {
	ResetDiskCacheStats()
	IncrementMemoryBuffers(100)
	DecrementMemoryBuffers(50)
	s := GetDiskCacheStats()
	if s.ActiveMemoryBuffers != 1 {
		t.Fatalf("ActiveMemoryBuffers=%d want 1", s.ActiveMemoryBuffers)
	}
	// underflow protection
	DecrementMemoryBuffers(1000)
	s = GetDiskCacheStats()
	if s.ActiveMemoryBuffers != 0 {
		t.Fatalf("underflow ActiveMemoryBuffers=%d want 0", s.ActiveMemoryBuffers)
	}
	ResetDiskCacheStats()
}

func TestIsDiskCacheAvailable(t *testing.T) {
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, MaxSizeMB: 100, Path: ""})
	if IsDiskCacheAvailable(1) {
		t.Fatal("should not be available when disabled")
	}
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, MaxSizeMB: 1, Path: ""})
	if !IsDiskCacheAvailable(100) {
		t.Fatal("100 bytes should fit in 1MB")
	}
	if IsDiskCacheAvailable(2 << 20) {
		t.Fatal("2MB should not fit in 1MB")
	}
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestShouldUseDiskCacheDisabled(t *testing.T) {
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 1, MaxSizeMB: 100, Path: ""})
	if ShouldUseDiskCache(100 << 20) {
		t.Fatal("should not use disk cache when disabled")
	}
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestSyncDiskCacheStats(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	ResetDiskCacheUsage()

	_, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("sync test data"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}

	SyncDiskCacheStats()
	s := GetDiskCacheStats()
	if s.ActiveDiskFiles < 1 {
		t.Fatalf("after sync ActiveDiskFiles=%d want >=1", s.ActiveDiskFiles)
	}
	if s.CurrentDiskUsageBytes <= 0 {
		t.Fatalf("after sync CurrentDiskUsageBytes=%d want >0", s.CurrentDiskUsageBytes)
	}

	// sync on nonexistent dir → no crash, resets to 0
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: "/nonexistent-sync-xyz"})
	SyncDiskCacheStats()

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
	ResetDiskCacheUsage()
}
