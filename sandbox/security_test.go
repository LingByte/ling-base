package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafePathUnderBase_Valid(t *testing.T) {
	_, err := safePathUnderBase("/tmp", "/tmp/subdir/file.txt")
	if err != nil {
		t.Errorf("expected no error for path under base, got: %v", err)
	}
}

func TestSafePathUnderBase_Traversal(t *testing.T) {
	_, err := safePathUnderBase("/tmp/safe", "/tmp/../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestSafePathUnderBase_EmptyBase(t *testing.T) {
	_, err := safePathUnderBase("", "/tmp/file")
	if err == nil {
		t.Error("expected error for empty base dir")
	}
}

func TestSafePathUnderBase_EmptyPath(t *testing.T) {
	_, err := safePathUnderBase("/tmp", "")
	if err == nil {
		t.Error("expected error for empty file path")
	}
}

func TestSafePathUnderBase_ExactBase(t *testing.T) {
	_, err := safePathUnderBase("/tmp", "/tmp")
	if err != nil {
		t.Errorf("expected no error for path equal to base, got: %v", err)
	}
}

func TestSanitizeForLog_RemovesNewlines(t *testing.T) {
	input := "hello\nworld\r\nfoo\tbar"
	result := sanitizeForLog(input)
	// Each control char is replaced with a space; \r\n becomes two spaces.
	if result != "hello world  foo bar" {
		t.Errorf("expected sanitized output, got %q", result)
	}
}

func TestSanitizeForLog_RemovesControlChars(t *testing.T) {
	input := "hello\x00\x01\x02world"
	result := sanitizeForLog(input)
	if result != "helloworld" {
		t.Errorf("expected control chars removed, got %q", result)
	}
}

func TestSanitizeForLog_Empty(t *testing.T) {
	if sanitizeForLog("") != "" {
		t.Error("expected empty string for empty input")
	}
}

func TestIsDangerousEnvVar(t *testing.T) {
	dangerous := []string{"LD_PRELOAD", "LD_LIBRARY_PATH", "PATH", "PYTHONPATH", "NODE_OPTIONS", "BASH_ENV", "ENV", "SHELL", "DYLD_INSERT_LIBRARIES"}
	for _, name := range dangerous {
		if !isDangerousEnvVar(name) {
			t.Errorf("expected %q to be dangerous", name)
		}
	}
}

func TestIsDangerousEnvVar_Safe(t *testing.T) {
	safe := []string{"HOME", "USER", "LANG", "LC_ALL", "MY_API_KEY", "DEBUG"}
	for _, name := range safe {
		if isDangerousEnvVar(name) {
			t.Errorf("expected %q to be safe", name)
		}
	}
}

func TestFilterSafeEnvVars(t *testing.T) {
	input := map[string]string{
		"HOME":         "/tmp",
		"LD_PRELOAD":   "/evil.so",
		"MY_API_KEY":   "secret",
		"PATH":         "/evil",
		"PYTHONPATH":   "/evil",
		"DEBUG":        "true",
		"NODE_OPTIONS": "--inspect",
	}
	result := filterSafeEnvVars(input)
	if _, ok := result["LD_PRELOAD"]; ok {
		t.Error("LD_PRELOAD should be filtered out")
	}
	if _, ok := result["PATH"]; ok {
		t.Error("PATH should be filtered out")
	}
	if _, ok := result["PYTHONPATH"]; ok {
		t.Error("PYTHONPATH should be filtered out")
	}
	if _, ok := result["NODE_OPTIONS"]; ok {
		t.Error("NODE_OPTIONS should be filtered out")
	}
	if _, ok := result["HOME"]; !ok {
		t.Error("HOME should be kept")
	}
	if _, ok := result["MY_API_KEY"]; !ok {
		t.Error("MY_API_KEY should be kept")
	}
	if _, ok := result["DEBUG"]; !ok {
		t.Error("DEBUG should be kept")
	}
}

func TestFilterSafeEnvVars_Empty(t *testing.T) {
	result := filterSafeEnvVars(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestSafePathUnderBase_RelativePath(t *testing.T) {
	// Relative path is resolved against CWD, not base dir.
	// Use CWD as the base to ensure the path is under it.
	cwd, err := os.Getwd()
	require.NoError(t, err)
	result, err := safePathUnderBase(cwd, "subdir/file.txt")
	require.NoError(t, err)
	// The result should be an absolute path under cwd
	assert.True(t, filepath.IsAbs(result))
	assert.Contains(t, result, cwd)
}

func TestSafePathUnderBase_Symlink(t *testing.T) {
	// Create a symlink and verify safePathUnderBase resolves it
	tmpDir, err := os.MkdirTemp("", "sandbox-symlink-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	targetDir := filepath.Join(tmpDir, "target")
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	linkPath := filepath.Join(tmpDir, "link")
	require.NoError(t, os.Symlink(targetDir, linkPath))

	// Path through symlink should be resolved
	_, err = safePathUnderBase(tmpDir, linkPath)
	// symlink resolves to target which is under tmpDir, so no error
	assert.NoError(t, err)
}

func TestSafePathUnderBase_AbsolutePathOutsideBase(t *testing.T) {
	_, err := safePathUnderBase("/tmp/safe", "/etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal denied")
}

func TestSafePathUnderBase_PathWithTraversal(t *testing.T) {
	// /tmp/safe/../etc/passwd resolves to /tmp/etc/passwd which is not under /tmp/safe
	_, err := safePathUnderBase("/tmp/safe", "/tmp/safe/../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal denied")
}

func TestSafePathUnderBase_DeepNestedPath(t *testing.T) {
	_, err := safePathUnderBase("/tmp", "/tmp/a/b/c/d/e/f/file.txt")
	assert.NoError(t, err)
}

func TestSafePathUnderBase_DotDotInMiddle(t *testing.T) {
	// /tmp/a/../b should resolve to /tmp/b which is under /tmp
	_, err := safePathUnderBase("/tmp", "/tmp/a/../b/file.txt")
	assert.NoError(t, err)
}

func TestSanitizeForLog_SpecialChars(t *testing.T) {
	input := "hello\x00world\x01foo\x02bar"
	result := sanitizeForLog(input)
	// Control chars below 32 should be removed
	assert.Equal(t, "helloworldfoobar", result)
}

func TestSanitizeForLog_OnlyControlChars(t *testing.T) {
	input := "\x00\x01\x02\x03"
	result := sanitizeForLog(input)
	assert.Equal(t, "", result)
}

func TestSanitizeForLog_MixedContent(t *testing.T) {
	input := "normal text\nwith\nnewlines\tand\ttabs"
	result := sanitizeForLog(input)
	// Newlines and tabs should be replaced with spaces
	assert.NotContains(t, result, "\n")
	assert.NotContains(t, result, "\t")
	assert.Contains(t, result, "normal text")
}

func TestSanitizeForLog_UnicodePreserved(t *testing.T) {
	input := "Hello 世界 🌍"
	result := sanitizeForLog(input)
	assert.Equal(t, input, result)
}

func TestIsDangerousEnvVar_CaseInsensitive(t *testing.T) {
	// All patterns are case-insensitive
	dangerous := []string{"ld_preload", "path", "Path", "PATH", "pythonpath", "PYTHONPATH", "node_options", "bash_env", "env", "shell", "SHELL"}
	for _, name := range dangerous {
		assert.True(t, isDangerousEnvVar(name), "expected %q to be dangerous (case-insensitive)", name)
	}
}

func TestIsDangerousEnvVar_DyldPrefix(t *testing.T) {
	// Any DYLD_ prefixed var should be dangerous
	dyldVars := []string{"DYLD_INSERT_LIBRARIES", "DYLD_LIBRARY_PATH", "DYLD_FALLBACK_LIBRARY_PATH", "dyld_foo"}
	for _, name := range dyldVars {
		assert.True(t, isDangerousEnvVar(name), "expected %q to be dangerous", name)
	}
}

func TestIsDangerousEnvVar_Empty(t *testing.T) {
	assert.False(t, isDangerousEnvVar(""))
}

func TestFilterSafeEnvVars_AllDangerous(t *testing.T) {
	input := map[string]string{
		"LD_PRELOAD":      "/evil.so",
		"PATH":            "/evil",
		"PYTHONPATH":      "/evil",
		"NODE_OPTIONS":    "--inspect",
		"BASH_ENV":        "/evil",
		"ENV":             "/evil",
		"SHELL":           "/evil",
		"LD_LIBRARY_PATH": "/evil",
	}
	result := filterSafeEnvVars(input)
	assert.Empty(t, result)
}

func TestFilterSafeEnvVars_AllSafe(t *testing.T) {
	input := map[string]string{
		"HOME":    "/tmp",
		"USER":    "test",
		"LANG":    "en_US.UTF-8",
		"MY_VAR":  "value",
		"DEBUG":   "true",
		"API_KEY": "secret",
	}
	result := filterSafeEnvVars(input)
	assert.Len(t, result, len(input))
	for k, v := range input {
		assert.Equal(t, v, result[k])
	}
}

func TestFilterSafeEnvVars_Mixed(t *testing.T) {
	input := map[string]string{
		"HOME":         "/tmp",
		"LD_PRELOAD":   "/evil.so",
		"MY_API_KEY":   "secret",
		"PATH":         "/evil",
		"DEBUG":        "true",
		"DYLD_FOO":     "bar",
		"NODE_OPTIONS": "--inspect",
	}
	result := filterSafeEnvVars(input)
	// Only HOME, MY_API_KEY, DEBUG should remain
	assert.Len(t, result, 3)
	assert.Equal(t, "/tmp", result["HOME"])
	assert.Equal(t, "secret", result["MY_API_KEY"])
	assert.Equal(t, "true", result["DEBUG"])
}
