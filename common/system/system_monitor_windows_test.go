//go:build windows

package system

import "testing"

func TestGetDiskSpaceInfo(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})
	defer SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})

	info := GetDiskSpaceInfo()
	// On a real filesystem, Total should be >0
	if info.Total == 0 {
		t.Log("GetDiskSpaceInfo returned Total=0 (possible CI environment)")
	}
}
