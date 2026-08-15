package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDefaultsNotUnknown(t *testing.T) {
	// After init() runs, GoVersion should be resolved from runtime.
	if strings.TrimSpace(GoVersion) == "unknown" {
		t.Error("GoVersion should be resolved, not 'unknown'")
	}
}

func TestResolveGitCommitFormat(t *testing.T) {
	// GitCommit is either "unknown" (no git) or a hex string.
	if GitCommit != "unknown" {
		for _, c := range GitCommit {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("GitCommit contains non-hex character: %q in %q", c, GitCommit)
			}
		}
	}
}

// ----- isUnknown -----

func TestIsUnknown(t *testing.T) {
	assert.True(t, isUnknown("unknown"), `"unknown" should be unknown`)
	assert.True(t, isUnknown(""), `"" should be unknown`)
	assert.True(t, isUnknown("  "), `whitespace-only should be unknown`)
	// Note: isUnknown checks v == "unknown" (exact, untrimmed) for the
	// non-empty case, so " unknown " with surrounding spaces is NOT unknown.
	assert.False(t, isUnknown(" unknown "), `" unknown " with spaces is NOT unknown (exact match only)`)
	assert.False(t, isUnknown("dev"), `"dev" should NOT be unknown`)
	assert.False(t, isUnknown("1.0.0"), `"1.0.0" should NOT be unknown`)
	assert.False(t, isUnknown("abc123def456"), `a real commit should NOT be unknown`)
}

// ----- shortRevision -----

func TestShortRevision(t *testing.T) {
	// Long hash is truncated to 12 chars.
	long := "abcdef1234567890abcdef1234567890abcdef12"
	assert.Equal(t, "abcdef123456", shortRevision(long))

	// Exactly 12 chars is returned as-is.
	exact := "abcdef123456"
	assert.Equal(t, exact, shortRevision(exact))

	// Shorter than 12 is returned as-is.
	short := "abc123"
	assert.Equal(t, short, shortRevision(short))

	// Empty string returns empty.
	assert.Equal(t, "", shortRevision(""))

	// Whitespace is trimmed before truncation.
	assert.Equal(t, "abcdef123456", shortRevision("  abcdef1234567890  "))
}

// ----- gitLine -----

func TestGitLine_Version(t *testing.T) {
	// "git --version" should return a non-empty string when git is installed.
	// If git is not available, gitLine returns "" and we skip assertions.
	out := gitLine("--version")
	if out == "" {
		t.Skip("git not available; skipping gitLine version test")
	}
	assert.Contains(t, out, "git version")
}

func TestGitLine_InvalidCommand(t *testing.T) {
	// An invalid git subcommand should return an empty string (error handled).
	out := gitLine("not-a-real-subcommand-xyz")
	assert.Empty(t, out)
}

func TestGitLine_RevParse(t *testing.T) {
	// In a git repo, rev-parse --short HEAD returns a short hash.
	out := gitLine("rev-parse", "--short", "HEAD")
	if out == "" {
		t.Skip("not in a git repo or git unavailable; skipping rev-parse test")
	}
	// Short hash should be a non-empty hex string.
	for _, c := range out {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"rev-parse output contains non-hex char %q in %q", c, out)
	}
}

// ----- resolveVersionDefaults -----

func TestResolveVersionDefaults_FillsDefaults(t *testing.T) {
	// Save and restore global state so other tests are unaffected.
	savedGoVersion := GoVersion
	savedGitCommit := GitCommit
	savedBuildTime := BuildTime
	t.Cleanup(func() {
		GoVersion = savedGoVersion
		GitCommit = savedGitCommit
		BuildTime = savedBuildTime
	})

	// Set everything to "unknown" so resolveVersionDefaults fills defaults.
	GoVersion = "unknown"
	GitCommit = "unknown"
	BuildTime = "unknown"

	resolveVersionDefaults()

	// GoVersion should be replaced with the runtime version.
	assert.NotEqual(t, "unknown", GoVersion)
	assert.NotEmpty(t, GoVersion)

	// BuildTime should never remain "unknown" — it falls back to now() if
	// git is unavailable.
	assert.NotEqual(t, "unknown", BuildTime)
	assert.NotEmpty(t, BuildTime)
}

func TestResolveVersionDefaults_PreservesRealValues(t *testing.T) {
	savedGoVersion := GoVersion
	savedGitCommit := GitCommit
	savedBuildTime := BuildTime
	t.Cleanup(func() {
		GoVersion = savedGoVersion
		GitCommit = savedGitCommit
		BuildTime = savedBuildTime
	})

	// Set real (non-unknown) values; they should be preserved.
	GoVersion = "go1.99.0"
	GitCommit = "abcdef123456"
	BuildTime = "2026-01-01T00:00:00Z"

	resolveVersionDefaults()

	assert.Equal(t, "go1.99.0", GoVersion)
	assert.Equal(t, "abcdef123456", GitCommit)
	assert.Equal(t, "2026-01-01T00:00:00Z", BuildTime)
}

func TestResolveVersionDefaults_EmptyGoVersion(t *testing.T) {
	savedGoVersion := GoVersion
	t.Cleanup(func() { GoVersion = savedGoVersion })

	GoVersion = ""
	resolveVersionDefaults()
	assert.NotEqual(t, "", GoVersion)
	assert.NotEqual(t, "unknown", GoVersion)
}
