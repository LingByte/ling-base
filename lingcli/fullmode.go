// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ModuleSource 描述一个 ling-base 模块的源码位置和依赖。
type ModuleSource struct {
	// ID 是模块标识（与 LingBaseModules 中的 ID 对应）
	ID string
	// Dirs 是源码目录列表（相对于 ling-base 根目录）
	// 例如: ["middleware"] 或 ["common/jwtutil", "common/jwtutil/gin"]
	Dirs []string
	// Deps 是依赖的其他模块 ID（传递依赖）
	Deps []string
}

// moduleSources 定义所有模块的源码映射。
// full 模式下，这些目录的 .go 文件会被复制到 pkg/ 下。
var moduleSources = map[string]ModuleSource{
	"response": {
		ID:   "response",
		Dirs: []string{"common/response", "common/response/gin"},
	},
	"jwt": {
		ID:   "jwt",
		Dirs: []string{"common/jwtutil", "common/jwtutil/gin"},
		Deps: []string{"crypto"},
	},
	"limiter": {
		ID:   "limiter",
		Dirs: []string{"common/limiter", "common/limiter/count", "common/limiter/tokenbucket", "common/limiter/keycount"},
	},
	"circuitbreaker": {
		ID:   "circuitbreaker",
		Dirs: []string{"common/circuitbreaker"},
	},
	"middleware": {
		ID:   "middleware",
		Dirs: []string{"middleware"},
		Deps: []string{"logger", "constants", "response", "circuitbreaker"},
	},
	"apidocs": {
		ID:   "apidocs",
		Dirs: []string{"apidocs", "apidocs/humax"},
	},
	"i18n": {
		ID:   "i18n",
		Dirs: []string{"i18n", "i18n/gin"},
	},
	// 核心依赖（不直接选择，但被其他模块依赖）
	"logger": {
		ID:   "logger",
		Dirs: []string{"logger", "logger/gin"},
		Deps: []string{"constants"},
	},
	"constants": {
		ID:   "constants",
		Dirs: []string{"constants"},
	},
	"bootstrap": {
		ID:   "bootstrap",
		Dirs: []string{"bootstrap"},
		Deps: []string{"logger", "constants", "eventbus", "version"},
	},
	"crypto": {
		ID:   "crypto",
		Dirs: []string{"common/crypto"},
	},
	"eventbus": {
		ID:   "eventbus",
		Dirs: []string{"eventbus"},
	},
	"version": {
		ID:   "version",
		Dirs: []string{"version"},
	},
}

// coreModules 是 web-api 项目始终需要的核心模块（无论用户选了什么）。
var coreModules = []string{"logger", "constants", "bootstrap", "response", "middleware"}

// collectModuleDirs 收集选中模块 + 核心模块 + 传递依赖的所有源码目录。
func collectModuleDirs(selectedModules []string) []string {
	// 用 map 去重
	visited := map[string]bool{}
	var dirs []string

	var visit func(moduleID string)
	visit = func(moduleID string) {
		if visited[moduleID] {
			return
		}
		visited[moduleID] = true

		src, ok := moduleSources[moduleID]
		if !ok {
			fmt.Fprintf(os.Stderr, "  \x1b[33m[警告] 未知模块源码映射: %s\x1b[0m\n", moduleID)
			return
		}

		dirs = append(dirs, src.Dirs...)
		for _, dep := range src.Deps {
			visit(dep)
		}
	}

	// 先收集核心模块
	for _, id := range coreModules {
		visit(id)
	}
	// 再收集用户选的模块
	for _, id := range selectedModules {
		visit(id)
	}

	return dirs
}

// copyModuleSource 将 ling-base 源码目录复制到目标项目的 pkg/ 下，
// 并重写 import 路径。
//
// lingBaseRoot 是 ling-base 仓库的根目录路径。
// targetPkgDir 是目标项目的 pkg/ 目录路径。
// modulePath 是目标项目的 Go module 路径（如 github.com/me/myapp）。
func copyModuleSource(lingBaseRoot, targetPkgDir, modulePath string, dirs []string) error {
	lingBaseImport := "github.com/LingByte/ling-base"

	for _, dir := range dirs {
		srcDir := filepath.Join(lingBaseRoot, dir)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "  \x1b[33m[警告] 源码目录不存在: %s\x1b[0m\n", srcDir)
			continue
		}

		// 目标目录: pkg/<dir>
		dstDir := filepath.Join(targetPkgDir, dir)

		// 确保目录存在
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", dstDir, err)
		}

		// 复制 .go 文件（排除 _test.go）
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("读取目录失败 %s: %w", srcDir, err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			// 跳过 go.mod / go.sum（ling-base 是多模块仓库，子目录有自己的 go.mod）
			// full 模式下所有代码在同一个 module 下，不需要子 go.mod
			if name == "go.mod" || name == "go.sum" {
				continue
			}
			// 跳过 README 等文档文件
			if strings.HasSuffix(name, ".md") {
				continue
			}

			srcFile := filepath.Join(srcDir, name)
			dstFile := filepath.Join(dstDir, name)

			// 子目录：只复制不含 go.mod 的资源目录（如 assets/）
			// 含 go.mod 的子目录是独立 Go 模块，由 Dirs 列表显式处理
			if entry.IsDir() {
				if fileExists(filepath.Join(srcFile, "go.mod")) {
					continue // 跳过 Go 子模块（由 Dirs 列表显式处理）
				}
				if err := copyDirRecursive(srcFile, dstFile); err != nil {
					return fmt.Errorf("复制子目录失败 %s: %w", srcFile, err)
				}
				continue
			}

			// .go 文件需要重写 import 路径
			if strings.HasSuffix(name, ".go") {
				if err := copyAndRewriteFile(srcFile, dstFile, lingBaseImport, modulePath); err != nil {
					return fmt.Errorf("复制文件失败 %s: %w", srcFile, err)
				}
				continue
			}

			// 非 Go 文件（embed 资源如 .css, .svg, .png, .font 等）直接复制
			content, err := os.ReadFile(srcFile)
			if err != nil {
				continue // 跳过无法读取的文件
			}
			if err := os.WriteFile(dstFile, content, 0644); err != nil {
				return fmt.Errorf("复制资源文件失败 %s: %w", srcFile, err)
			}
		}
	}

	return nil
}

// fileExists 检查文件是否存在。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyDirRecursive 递归复制目录（用于 embed 资源目录如 assets/）。
// 跳过 go.mod / go.sum / .md 文件。
func copyDirRecursive(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "go.mod" || name == "go.sum" || strings.HasSuffix(name, ".md") {
			continue
		}
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(dstDir, name)
		if entry.IsDir() {
			if fileExists(filepath.Join(src, "go.mod")) {
				continue
			}
			if err := copyDirRecursive(src, dst); err != nil {
				return err
			}
			continue
		}
		content, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

// copyAndRewriteFile 复制一个 Go 文件并将 ling-base import 路径重写为目标项目路径。
func copyAndRewriteFile(srcPath, dstPath, oldImport, newModulePath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// 解析 Go 文件获取 import 列表
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, content, parser.ParseComments)
	if err != nil {
		// 如果解析失败，直接复制原文
		return os.WriteFile(dstPath, content, 0644)
	}

	// 收集需要替换的 import 路径
	rewritten := string(content)

	// 遍历所有 import，替换 ling-base 路径
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(importPath, oldImport) {
			// 替换: github.com/LingByte/ling-base/xxx → github.com/user/project/pkg/xxx
			newPath := strings.Replace(importPath, oldImport, newModulePath+"/pkg", 1)
			rewritten = strings.ReplaceAll(rewritten, `"`+importPath+`"`, `"`+newPath+`"`)
		}
	}

	return os.WriteFile(dstPath, []byte(rewritten), 0644)
}

// generateFullMode 在 full 模式下复制源码到 pkg/ 并重写模板中的 import。
// 返回需要额外写入的文件列表。
func generateFullMode(spec *ProjectSpec, lingBaseRoot, targetDir string) error {
	pkgDir := filepath.Join(targetDir, "pkg")

	// 收集所有需要的源码目录
	dirs := collectModuleDirs(spec.Modules)

	fmt.Printf("  \x1b[38;5;245m复制 %d 个源码目录到 pkg/...\x1b[0m\n", len(dirs))

	// 复制源码
	if err := copyModuleSource(lingBaseRoot, pkgDir, spec.Module, dirs); err != nil {
		return err
	}

	// 统计复制的文件数
	count := 0
	filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			count++
		}
		return nil
	})
	fmt.Printf("  \x1b[32m✓ 已复制 %d 个 Go 文件到 pkg/\x1b[0m\n", count)

	return nil
}

// rewriteTemplateImports 在 full 模式下重写模板渲染后的 import 路径。
// 将 github.com/LingByte/ling-base/xxx → github.com/user/project/pkg/xxx
func rewriteTemplateImports(content, modulePath string) string {
	oldImport := "github.com/LingByte/ling-base"
	newImport := modulePath + "/pkg"
	return strings.ReplaceAll(content, oldImport, newImport)
}

// isFullMode 返回 spec 是否为 full 模式。
func isFullMode(spec *ProjectSpec) bool {
	return spec.Mode == "full"
}

// ast 包的 unused import 检查（避免编译器报 unused import）
var _ = ast.Print

// findLingBaseRoot 定位 ling-base 仓库根目录。
// 优先用 LING_BASE_ROOT 环境变量，其次尝试从可执行文件位置向上查找。
func findLingBaseRoot() string {
	if root := os.Getenv("LING_BASE_ROOT"); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.work")); err == nil {
			return root
		}
	}

	// 从当前工作目录向上查找 go.work
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			// 检查是否是 ling-base（有 lingcli 目录）
			if _, err := os.Stat(filepath.Join(dir, "lingcli")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
