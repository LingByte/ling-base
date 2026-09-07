// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// embed_source 目录由 `make prepare-cli-embed` 在构建前同步。
// 开发时只有 .gitkeep，发布构建时包含完整 ling-base 源码。
//
//go:embed all:embed_source
var embeddedSource embed.FS

// hasEmbeddedSource 检查 embed_source 是否包含实际源码（不只是 .gitkeep）。
func hasEmbeddedSource() bool {
	entries, err := embeddedSource.ReadDir("embed_source")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			return true
		}
	}
	return false
}

// extractEmbeddedSource extracts the entire embedded ling-base source tree
// into the target project's pkg/ directory, rewriting all import paths.
// This mirrors the full-mode copyModuleSource but reads from the embedded
// filesystem instead of disk.
func extractEmbeddedSource(spec *ProjectSpec, targetDir string) error {
	pkgDir := filepath.Join(targetDir, "pkg")
	lingBaseImport := "github.com/LingByte/ling-base"
	newImport := spec.Module + "/pkg"

	fmt.Printf("  \x1b[38;5;245m从嵌入源码提取...\x1b[0m\n")

	count := 0
	err := fs.WalkDir(embeddedSource, "embed_source", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Compute relative path from embed_source root
		relPath, err := filepath.Rel("embed_source", path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}

		// Get top-level directory name
		topDir := relPath
		if idx := strings.Index(relPath, "/"); idx >= 0 {
			topDir = relPath[:idx]
		}

		// Skip excluded top-level directories
		if fullExcludeDirs[topDir] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip root go.mod / go.sum / go.work
		if relPath == "go.mod" || relPath == "go.sum" || relPath == "go.work" {
			return nil
		}

		if d.IsDir() {
			dstDir := filepath.Join(pkgDir, relPath)
			return os.MkdirAll(dstDir, 0755)
		}

		name := d.Name()

		// Skip test files, go.mod, go.sum, .md, dot files
		if strings.HasSuffix(name, "_test.go") ||
			name == "go.mod" || name == "go.sum" ||
			strings.HasSuffix(name, ".md") ||
			strings.HasPrefix(name, ".") {
			return nil
		}

		dstFile := filepath.Join(pkgDir, relPath)

		// Go files: rewrite import paths
		if strings.HasSuffix(name, ".go") {
			data, err := embeddedSource.ReadFile(path)
			if err != nil {
				return nil
			}
			rewritten := string(data)
			// Apply import fixups first, then rewrite to local paths
			for old, fixed := range importFixups {
				rewritten = strings.ReplaceAll(rewritten, `"`+old+`"`, `"`+fixed+`"`)
			}
			rewritten = strings.ReplaceAll(rewritten, lingBaseImport, newImport)
			if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(dstFile, []byte(rewritten), 0644); err != nil {
				return err
			}
			count++
			return nil
		}

		// Non-Go files: copy as-is
		data, err := embeddedSource.ReadFile(path)
		if err != nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dstFile), 0755); err != nil {
			return err
		}
		return os.WriteFile(dstFile, data, 0644)
	})
	if err != nil {
		return err
	}

	fmt.Printf("  \x1b[32m✓ 已提取 %d 个 Go 文件到 pkg/\x1b[0m\n", count)
	return nil
}

// 确保 embed.FS 被 fs 包引用（避免 unused import）
var _ = fs.WalkDir
