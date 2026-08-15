package sandbox

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// dangerousEnvVarPatterns are environment variable names that are unsafe for
// subprocesses because they can alter library loading paths, shell behavior,
// or interpreter execution.
var dangerousEnvVarPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^LD_PRELOAD$`),
	regexp.MustCompile(`(?i)^LD_LIBRARY_PATH$`),
	regexp.MustCompile(`(?i)^DYLD_`),
	regexp.MustCompile(`(?i)^PATH$`),
	regexp.MustCompile(`(?i)^PYTHONPATH$`),
	regexp.MustCompile(`(?i)^NODE_OPTIONS$`),
	regexp.MustCompile(`(?i)^BASH_ENV$`),
	regexp.MustCompile(`(?i)^ENV$`),
	regexp.MustCompile(`(?i)^SHELL$`),
}

// safePathUnderBase returns absPath when filePath is under baseDir.
// It prevents path traversal attacks by ensuring the resolved absolute path
// is contained within the base directory.
func safePathUnderBase(baseDir, filePath string) (string, error) {
	if baseDir == "" || filePath == "" {
		return "", fmt.Errorf("baseDir and filePath cannot be empty")
	}
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", fmt.Errorf("invalid base dir: %w", err)
	}
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	sep := string(filepath.Separator)
	if absPath != absBase && !strings.HasPrefix(absPath, absBase+sep) {
		return "", fmt.Errorf("path traversal denied: path is outside base directory")
	}
	return absPath, nil
}

// sanitizeForLog removes control characters and newlines to prevent log injection.
func sanitizeForLog(input string) string {
	if input == "" {
		return ""
	}
	sanitized := strings.ReplaceAll(input, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")
	var builder strings.Builder
	for _, r := range sanitized {
		if r >= 32 || r == ' ' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// isDangerousEnvVar reports whether an environment variable name is unsafe.
func isDangerousEnvVar(name string) bool {
	for _, pattern := range dangerousEnvVarPatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// filterSafeEnvVars returns env entries with dangerous names removed.
func filterSafeEnvVars(extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return nil
	}
	out := make(map[string]string, len(extra))
	for key, value := range extra {
		if isDangerousEnvVar(key) {
			continue
		}
		out[key] = value
	}
	return out
}
