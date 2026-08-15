package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetDiskPathProvider(t *testing.T) {
	orig := diskPathProvider
	defer func() { diskPathProvider = orig }()

	called := false
	SetDiskPathProvider(func() []string {
		called = true
		return []string{"/tmp"}
	})

	paths := diskPathProvider()
	if !called {
		t.Fatal("custom provider was not set")
	}
	if len(paths) != 1 || paths[0] != "/tmp" {
		t.Fatalf("got %v want [/tmp]", paths)
	}

	// setting nil should not replace
	SetDiskPathProvider(nil)
	paths = diskPathProvider()
	if len(paths) != 1 || paths[0] != "/tmp" {
		t.Fatalf("nil setter should not replace: got %v", paths)
	}
}

func TestCollectHostSnapshot(t *testing.T) {
	snap := CollectHostSnapshot()
	if snap.CPU.NumCPU == 0 {
		t.Fatal("NumCPU should not be 0")
	}
}

func TestCollectDiskPaths(t *testing.T) {
	orig := diskPathProvider
	defer func() { diskPathProvider = orig }()

	// custom provider returning a real temp dir
	tmp := t.TempDir()
	diskPathProvider = func() []string {
		return []string{tmp}
	}

	paths := collectDiskPaths()
	if len(paths) == 0 {
		t.Fatal("expected at least one disk path")
	}
	if paths[0].Path != tmp {
		t.Fatalf("Path=%q want %q", paths[0].Path, tmp)
	}
}

func TestCollectDiskPathsDuplicatesAndEmpty(t *testing.T) {
	orig := diskPathProvider
	defer func() { diskPathProvider = orig }()

	tmp := t.TempDir()

	// duplicate paths should be deduplicated
	diskPathProvider = func() []string {
		return []string{tmp, tmp, "", ".", "/nonexistent-xyz-123"}
	}
	paths := collectDiskPaths()
	// only tmp should produce a result; empty/dot/nonexistent are filtered
	if len(paths) != 1 {
		t.Fatalf("expected 1 unique path, got %d: %+v", len(paths), paths)
	}
}

func TestDiskMonitorPath(t *testing.T) {
	// empty
	if p := diskMonitorPath(""); p != "" {
		t.Fatalf("diskMonitorPath('')=%q want ''", p)
	}
	// dot
	if p := diskMonitorPath("."); p != "" {
		t.Fatalf("diskMonitorPath('.')=%q want ''", p)
	}
	// whitespace
	if p := diskMonitorPath("   "); p != "" {
		t.Fatalf("diskMonitorPath('   ')=%q want ''", p)
	}
	// directory path → returned as-is
	tmp := t.TempDir()
	if p := diskMonitorPath(tmp); p != tmp {
		t.Fatalf("diskMonitorPath(dir)=%q want %q", p, tmp)
	}
	// file path → parent directory returned
	tmpFile := filepath.Join(tmp, "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p := diskMonitorPath(tmpFile); p != tmp {
		t.Fatalf("diskMonitorPath(file)=%q want %q", p, tmp)
	}
}

func TestProcessSelfThreads(t *testing.T) {
	// processSelfThreads may fail on some platforms, just verify no panic
	threads, err := processSelfThreads()
	if err != nil {
		// On some platforms this might fail; just verify it doesn't panic
		t.Logf("processSelfThreads returned error: %v (acceptable on some platforms)", err)
		return
	}
	if threads < 0 {
		t.Fatalf("threads=%d should be >=0", threads)
	}
}

func TestCollectHostSnapshotCustomProvider(t *testing.T) {
	orig := diskPathProvider
	defer func() { diskPathProvider = orig }()

	tmp := t.TempDir()
	// Create a file in tmp to test the file→dir path
	tmpFile := filepath.Join(tmp, "test.log")
	os.WriteFile(tmpFile, []byte("x"), 0o644)

	diskPathProvider = func() []string {
		return []string{tmp, tmpFile, "/nonexistent-xyz", "", "."}
	}

	snap := CollectHostSnapshot()
	// Should have at least one disk entry (for tmp)
	if len(snap.Disks) == 0 {
		t.Fatal("expected at least one disk entry")
	}
}

func TestDefaultDiskPathProvider(t *testing.T) {
	// just verify it doesn't panic and returns something
	paths := defaultDiskPathProvider()
	_ = paths
}
