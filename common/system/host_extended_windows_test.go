//go:build windows

package system

import "testing"

func TestProcessMaxFDs(t *testing.T) {
	// On Windows, processMaxFDs always returns 0 (no RLIMIT_NOFILE).
	fds := processMaxFDs()
	if fds != 0 {
		t.Fatalf("processMaxFDs on Windows should return 0, got %d", fds)
	}
}
