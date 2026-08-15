package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiskCacheDirAndEnsure(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	dir := GetDiskCacheDir()
	if dir != filepath.Join(tmp, "ling-base-body-cache") {
		t.Fatalf("GetDiskCacheDir=%q want %q", dir, filepath.Join(tmp, "ling-base-body-cache"))
	}

	if err := EnsureDiskCacheDir(); err != nil {
		t.Fatalf("EnsureDiskCacheDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("not a directory")
	}

	// EnsureDiskCacheDir is idempotent
	if err := EnsureDiskCacheDir(); err != nil {
		t.Fatalf("EnsureDiskCacheDir second call: %v", err)
	}

	// empty path → temp dir fallback
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: ""})
	dir = GetDiskCacheDir()
	if dir == "" {
		t.Fatal("GetDiskCacheDir should fallback to temp dir")
	}
	// dir should be inside os.TempDir()
	if !strings.HasPrefix(dir, os.TempDir()) {
		t.Fatalf("expected dir under os.TempDir()=%q, got %q", os.TempDir(), dir)
	}

	// reset
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestCreateDiskCacheFile(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	path, file, err := CreateDiskCacheFile(DiskCacheTypeBody)
	if err != nil {
		t.Fatalf("CreateDiskCacheFile: %v", err)
	}
	if path == "" || file == nil {
		t.Fatal("path or file is nil")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("created file not found: %v", err)
	}
	file.Close()
	os.Remove(path)

	// invalid path → error
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: "/nonexistent-root-xyz/\x00"})
	_, _, err = CreateDiskCacheFile(DiskCacheTypeFile)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

// Test CreateDiskCacheFile with a path that causes OpenFile to fail
func TestCreateDiskCacheFileOpenError(t *testing.T) {
	// Set path to a location under a file (not a directory)
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: filepath.Join(blocker, "subdir")})
	_, _, err := CreateDiskCacheFile(DiskCacheTypeBody)
	if err == nil {
		t.Fatal("expected error when parent is a file")
	}
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestWriteAndReadDiskCacheFile(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	data := []byte("hello disk cache")
	path, err := WriteDiskCacheFile(DiskCacheTypeBody, data)
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}
	defer os.Remove(path)

	got, err := ReadDiskCacheFile(path)
	if err != nil {
		t.Fatalf("ReadDiskCacheFile: %v", err)
	}
	if string(got) != "hello disk cache" {
		t.Fatalf("got %q want %q", got, "hello disk cache")
	}

	// string variant
	path2, err := WriteDiskCacheFileString(DiskCacheTypeFile, "string data")
	if err != nil {
		t.Fatalf("WriteDiskCacheFileString: %v", err)
	}
	defer os.Remove(path2)

	s, err := ReadDiskCacheFileString(path2)
	if err != nil {
		t.Fatalf("ReadDiskCacheFileString: %v", err)
	}
	if s != "string data" {
		t.Fatalf("got %q want %q", s, "string data")
	}

	// read nonexistent
	if _, err := ReadDiskCacheFile("/nonexistent-file-xyz"); err == nil {
		t.Fatal("expected error reading nonexistent file")
	}
	if _, err := ReadDiskCacheFileString("/nonexistent-file-xyz"); err == nil {
		t.Fatal("expected error reading nonexistent file string")
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

// Test WriteDiskCacheFile with empty data and normal data
func TestWriteDiskCacheFileVariants(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	// Normal write
	path, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("test data"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}
	os.Remove(path)

	// Test with empty data
	path, err = WriteDiskCacheFile(DiskCacheTypeBody, []byte{})
	if err != nil {
		t.Fatalf("WriteDiskCacheFile empty: %v", err)
	}
	os.Remove(path)

	// Test with larger data
	path, err = WriteDiskCacheFile(DiskCacheTypeBody, make([]byte, 1024))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile 1KB: %v", err)
	}
	os.Remove(path)

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestRemoveDiskCacheFile(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	path, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("temp"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}

	if err := RemoveDiskCacheFile(path); err != nil {
		t.Fatalf("RemoveDiskCacheFile: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, stat err=%v", err)
	}

	// remove again → error
	if err := RemoveDiskCacheFile(path); err == nil {
		t.Fatal("expected error removing nonexistent file")
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestCleanupOldDiskCacheFiles(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	// create a fresh file
	_, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("fresh"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}

	// create an old file by backdating mod time
	oldPath, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("old"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	// cleanup files older than 1 hour
	if err := CleanupOldDiskCacheFiles(time.Hour); err != nil {
		t.Fatalf("CleanupOldDiskCacheFiles: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("old file should have been cleaned up")
	}

	// nonexistent dir → no error
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: "/nonexistent-cleanup-xyz"})
	if err := CleanupOldDiskCacheFiles(time.Hour); err != nil {
		t.Fatalf("CleanupOldDiskCacheFiles on nonexistent dir: %v", err)
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

// Test CleanupOldDiskCacheFiles with subdirectory (should skip dirs)
func TestCleanupOldDiskCacheFilesWithSubdir(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	cacheDir := GetDiskCacheDir()
	// Create a subdirectory
	if err := os.MkdirAll(filepath.Join(cacheDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Cleanup should not remove the subdirectory
	if err := CleanupOldDiskCacheFiles(time.Hour); err != nil {
		t.Fatalf("CleanupOldDiskCacheFiles: %v", err)
	}

	// Subdir should still exist
	if _, err := os.Stat(filepath.Join(cacheDir, "subdir")); err != nil {
		t.Fatalf("subdir should not be removed: %v", err)
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

func TestGetDiskCacheInfo(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	_, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("info test"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}

	count, size, err := GetDiskCacheInfo()
	if err != nil {
		t.Fatalf("GetDiskCacheInfo: %v", err)
	}
	if count < 1 {
		t.Fatalf("fileCount=%d want >=1", count)
	}
	if size <= 0 {
		t.Fatalf("totalSize=%d want >0", size)
	}

	// nonexistent dir → 0, 0, nil
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: "/nonexistent-info-xyz"})
	count, size, err = GetDiskCacheInfo()
	if err != nil {
		t.Fatalf("GetDiskCacheInfo on nonexistent dir: %v", err)
	}
	if count != 0 || size != 0 {
		t.Fatalf("expected 0,0 got %d,%d", count, size)
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}

// Test GetDiskCacheInfo with a subdirectory (should skip dirs)
func TestGetDiskCacheInfoWithSubdir(t *testing.T) {
	tmp := t.TempDir()
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, Path: tmp})

	// Create a subdirectory in the cache dir
	cacheDir := GetDiskCacheDir()
	if err := os.MkdirAll(filepath.Join(cacheDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file
	_, err := WriteDiskCacheFile(DiskCacheTypeBody, []byte("test"))
	if err != nil {
		t.Fatalf("WriteDiskCacheFile: %v", err)
	}

	count, size, err := GetDiskCacheInfo()
	if err != nil {
		t.Fatalf("GetDiskCacheInfo: %v", err)
	}
	if count != 1 {
		t.Fatalf("fileCount=%d want 1 (subdir should be skipped)", count)
	}
	if size <= 0 {
		t.Fatalf("totalSize=%d want >0", size)
	}

	SetDiskCacheConfig(DiskCacheConfig{Enabled: false, ThresholdMB: 10, MaxSizeMB: 1024, Path: ""})
}
