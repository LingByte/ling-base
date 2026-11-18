package sandbox

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

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

// SafePathUnderBase returns absPath when filePath is under baseDir.
func SafePathUnderBase(baseDir, filePath string) (string, error) {
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

// SanitizeForLog removes control characters and newlines to prevent log injection.
func SanitizeForLog(input string) string {
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

func isDangerousEnvVar(name string) bool {
	for _, pattern := range dangerousEnvVarPatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// FilterSafeEnvVars returns env entries with dangerous names removed.
func FilterSafeEnvVars(extra map[string]string) map[string]string {
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
