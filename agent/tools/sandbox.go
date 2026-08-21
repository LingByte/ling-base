// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package tools implements built-in tools for the agent: read, write,
// edit, and bash. Each tool confines filesystem access to a working
// directory (CWD) to prevent accidental modification of files outside
// the project.
package tools

import (
	"path/filepath"
	"strings"
)

// resolvePath resolves a potentially relative path against the CWD,
// returning an absolute path. It does NOT perform symlink resolution
// or jail checking — that's the caller's responsibility.
func resolvePath(cwd, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(cwd, p))
}

// isInside checks whether path is inside dir (after cleaning).
func isInside(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, "../")
}
