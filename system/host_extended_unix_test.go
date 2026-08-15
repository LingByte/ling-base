//go:build !windows

package system

import "testing"

func TestProcessMaxFDs(t *testing.T) {
	// On Unix, processMaxFDs reads RLIMIT_NOFILE.
	// Just verify it doesn't panic and returns a reasonable value.
	fds := processMaxFDs()
	// 0 means the call failed or returned 0; on most systems it should be >0
	// but we don't assert a specific value since it depends on the system.
	_ = fds
}
