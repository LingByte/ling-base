package system

import (
	"runtime"
	"testing"
)

func TestCollectRuntimeSnapshot(t *testing.T) {
	snap := CollectRuntimeSnapshot()
	if snap.GoVersion == "" {
		t.Fatal("GoVersion should not be empty")
	}
	if snap.Goroutine.NumGoroutine < 1 {
		t.Fatal("should have at least 1 goroutine")
	}
}

func TestMemStatsToSnapshot(t *testing.T) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	snap := memStatsToSnapshot(&m)
	if snap.HeapAlloc != m.HeapAlloc {
		t.Fatalf("HeapAlloc=%d want %d", snap.HeapAlloc, m.HeapAlloc)
	}
	if snap.Sys != m.Sys {
		t.Fatalf("Sys=%d want %d", snap.Sys, m.Sys)
	}
}

func TestBuildGCSnapshot(t *testing.T) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	snap := buildGCSnapshot(&m)
	if snap.NumGC != m.NumGC {
		t.Fatalf("NumGC=%d want %d", snap.NumGC, m.NumGC)
	}
}

func TestBuildGCSnapshotNoGC(t *testing.T) {
	// Simulate zero GC
	var m runtime.MemStats
	snap := buildGCSnapshot(&m)
	if snap.NumGC != 0 {
		t.Fatalf("NumGC=%d want 0", snap.NumGC)
	}
	if snap.GCPerMinute != 0 {
		t.Fatalf("GCPerMinute=%v want 0", snap.GCPerMinute)
	}
}
