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

// extractEmbeddedSource 将嵌入的源码解压到 targetPkgDir/pkg/ 下，
// 并重写 import 路径。
func extractEmbeddedSource(spec *ProjectSpec, targetDir string) error {
	pkgDir := filepath.Join(targetDir, "pkg")
	dirs := collectModuleDirs(spec.Modules)

	fmt.Printf("  \x1b[38;5;245m从嵌入源码提取 %d 个目录...\x1b[0m\n", len(dirs))

	lingBaseImport := "github.com/LingByte/ling-base"
	count := 0

	for _, dir := range dirs {
		embedPath := filepath.Join("embed_source", dir)
		dstDir := filepath.Join(pkgDir, dir)

		// 检查嵌入源码中是否存在该目录
		entries, err := embeddedSource.ReadDir(embedPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  \x1b[38;5;245m[跳过] %s: 模板未直接引用，不复制源码\x1b[0m\n", dir)
			continue
		}

		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", dstDir, err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, "_test.go") || name == "go.mod" || name == "go.sum" || strings.HasSuffix(name, ".md") {
				continue
			}

			srcPath := filepath.Join(embedPath, name)
			dstPath := filepath.Join(dstDir, name)

			// 子目录（资源目录如 assets/）
			if entry.IsDir() {
				// 检查是否含 go.mod（独立子模块，跳过）
				subGoMod := filepath.Join(srcPath, "go.mod")
				if _, err := embeddedSource.ReadFile(subGoMod); err == nil {
					continue
				}
				if err := extractEmbeddedDir(srcPath, dstPath); err != nil {
					return err
				}
				continue
			}

			// .go 文件重写 import
			if strings.HasSuffix(name, ".go") {
				data, err := embeddedSource.ReadFile(srcPath)
				if err != nil {
					continue
				}
				rewritten := rewriteImportsInContent(string(data), lingBaseImport, spec.Module+"/pkg")
				if err := os.WriteFile(dstPath, []byte(rewritten), 0644); err != nil {
					return err
				}
				count++
				continue
			}

			// 资源文件直接复制
			data, err := embeddedSource.ReadFile(srcPath)
			if err != nil {
				continue
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	fmt.Printf("  \x1b[32m✓ 已提取 %d 个 Go 文件到 pkg/\x1b[0m\n", count)
	return nil
}

// extractEmbeddedDir 递归提取嵌入的子目录。
func extractEmbeddedDir(srcPath, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	entries, err := embeddedSource.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "go.mod" || name == "go.sum" || strings.HasSuffix(name, ".md") {
			continue
		}
		src := filepath.Join(srcPath, name)
		dst := filepath.Join(dstDir, name)
		if entry.IsDir() {
			// 跳过含 go.mod 的子目录
			subGoMod := filepath.Join(src, "go.mod")
			if _, err := embeddedSource.ReadFile(subGoMod); err == nil {
				continue
			}
			if err := extractEmbeddedDir(src, dst); err != nil {
				return err
			}
			continue
		}
		data, err := embeddedSource.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// rewriteImportsInContent 将 content 中的 oldImport 替换为 newImport。
func rewriteImportsInContent(content, oldImport, newImport string) string {
	return strings.ReplaceAll(content, oldImport, newImport)
}

// 确保 embed.FS 被 fs 包引用（避免 unused import）
var _ = fs.WalkDir
