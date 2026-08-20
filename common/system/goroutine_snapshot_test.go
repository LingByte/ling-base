package system

import "testing"

func TestBuildGoroutineSnapshot(t *testing.T) {
	// buildGoroutineSnapshot is called by CollectRuntimeSnapshot
	snap := CollectRuntimeSnapshot()
	if snap.Goroutine.NumGoroutine < 1 {
		t.Fatal("NumGoroutine should be >=1")
	}
	if snap.Goroutine.NumThread < 1 {
		t.Fatal("NumThread should be >=1")
	}
}

func TestBuildGoroutineSnapshotDirect(t *testing.T) {
	snap := buildGoroutineSnapshot(42)
	if snap.NumGoroutine != 42 {
		t.Fatalf("NumGoroutine=%d want 42", snap.NumGoroutine)
	}
	if snap.NumThread < 1 {
		t.Fatal("NumThread should be >=1")
	}
}
