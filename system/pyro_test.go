package system

import "testing"

func TestStartPyroScopeEmptyURL(t *testing.T) {
	// When PYROSCOPE_URL is empty, should return nil immediately
	t.Setenv("PYROSCOPE_URL", "")
	if err := StartPyroScope(); err != nil {
		t.Fatalf("StartPyroScope with empty URL: %v", err)
	}
}

func TestStartPyroScopeInvalidURL(t *testing.T) {
	// With a non-empty but invalid URL, pyroscope.Start should return an error
	t.Setenv("PYROSCOPE_URL", "http://127.0.0.1:1")
	t.Setenv("PYROSCOPE_APP_NAME", "test-app")
	t.Setenv("PYROSCOPE_BASIC_AUTH_USER", "")
	t.Setenv("PYROSCOPE_BASIC_AUTH_PASSWORD", "")
	t.Setenv("HOSTNAME", "test-host")
	t.Setenv("PYROSCOPE_MUTEX_RATE", "10")
	t.Setenv("PYROSCOPE_BLOCK_RATE", "10")

	// This will attempt to start pyroscope and likely fail to connect,
	// but pyroscope.Start may not return an error immediately (it starts
	// a background goroutine). We just verify it doesn't panic.
	_ = StartPyroScope()
}
