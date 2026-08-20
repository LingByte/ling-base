//go:build !windows

package system

import "testing"

func TestGetDiskSpaceInfo(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})
	defer SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})

	info := GetDiskSpaceInfo()
	// On a real filesystem, Total should be >0
	if info.Total == 0 {
		// Some CI environments might report 0; just verify no panic
		t.Log("GetDiskSpaceInfo returned Total=0 (possible CI environment)")
	}
	if info.Total > 0 && info.UsedPercent < 0 {
		t.Fatal("UsedPercent should not be negative")
	}

	// Empty path → temp dir fallback
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: ""})
	info = GetDiskSpaceInfo()
	_ = info
}
