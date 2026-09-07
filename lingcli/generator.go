// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// stderr 返回标准错误输出。
func stderr() io.Writer { return os.Stderr }

// Generator 负责将项目文件写入磁盘。
type Generator struct {
	// lingBaseRoot is the path to the ling-base source tree (used in full mode
	// to collect submodule dependency versions). Empty in lib mode or when
	// using embedded source.
	lingBaseRoot string
}

// NewGenerator 创建生成器。
func NewGenerator() *Generator { return &Generator{} }

// Generate 根据规格生成完整项目。
func (g *Generator) Generate(spec *ProjectSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	tmpl := findTemplate(spec.Template)
	if tmpl == nil {
		return fmt.Errorf("未找到模板: %s", spec.Template)
	}

	files := tmpl.Generate(spec)
	targetDir := spec.TargetDir()

	// 检查目标目录是否非空。
	if targetDir != "." {
		if entries, err := os.ReadDir(targetDir); err == nil && len(entries) > 0 {
			return fmt.Errorf("目录 %s 已存在且非空，请选择一个空目录", targetDir)
		}
	} else {
		if entries, err := os.ReadDir("."); err == nil && len(entries) > 0 {
			fmt.Printf("\n  \x1b[33m[警告] 当前目录非空，文件可能会覆盖\x1b[0m\n")
		}
	}

	fmt.Printf("\n\x1b[38;5;117m━━━ 正在创建项目: %s ━━━\x1b[0m\n", spec.ProjectName())
	fmt.Print(spec.Summary())
	fmt.Println()

	// 写入所有文件。
	for _, f := range files {
		fullPath := filepath.Join(targetDir, f.Path)

		if _, err := os.Stat(fullPath); err == nil {
			fmt.Printf("  \x1b[33m[跳过]\x1b[0m %s \x1b[38;5;245m（文件已存在）\x1b[0m\n", f.Path)
			continue
		}

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s: %w", dir, err)
		}

		// full 模式下重写模板中的 ling-base import 路径
		content := f.Content
		if isFullMode(spec) && strings.HasSuffix(f.Path, ".go") {
			content = rewriteTemplateImports(content, spec.Module)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("写入 %s: %w", f.Path, err)
		}

		fmt.Printf("  \x1b[32m[完成]\x1b[0m %s\n", f.Path)
	}

	// full 模式：复制 ling-base 源码到 pkg/
	if isFullMode(spec) {
		fmt.Println()
		fmt.Println("\x1b[38;5;117m━━━ 复制 ling-base 源码到 pkg/ ━━━\x1b[0m")

		// 优先使用嵌入源码（发布二进制时可用）
		if hasEmbeddedSource() {
			if err := extractEmbeddedSource(spec, targetDir); err != nil {
				return fmt.Errorf("提取嵌入源码失败: %w", err)
			}
		} else {
			// 回退到本地 ling-base 源码,或自动 git clone
			lingBaseRoot := findLingBaseRoot()
			g.lingBaseRoot = lingBaseRoot
			if lingBaseRoot == "" {
				fmt.Printf("  \x1b[31m[错误] 无法获取 ling-base 源码\x1b[0m\n")
				fmt.Println()
				fmt.Println("  \x1b[38;5;245mfull 模式需要 ling-base 完整源码。请用以下任一方式:\x1b[0m")
				fmt.Println("    1. 设置环境变量: export LING_BASE_ROOT=/path/to/ling-base")
				fmt.Printf("    2. 使用 --ling-base-root 参数: --ling-base-root /path/to/ling-base\n")
				fmt.Println("    3. 在 ling-base 目录下运行 lingcli")
				fmt.Println("    4. 使用 --mode lib（默认，引入库而非复制源码）")
				fmt.Println()
				return fmt.Errorf("full 模式无法获取 ling-base 源码")
			}
			// 如果是临时 clone 的目录,结束后清理
			if strings.HasPrefix(lingBaseRoot, os.TempDir()) {
				defer os.RemoveAll(lingBaseRoot)
			}
			if err := generateFullMode(spec, lingBaseRoot, targetDir); err != nil {
				return fmt.Errorf("源码复制失败: %w", err)
			}
		}
	}

	// 运行 go mod init + tidy。
	fmt.Println()
	fmt.Println("\x1b[38;5;117m━━━ 初始化 Go module ━━━\x1b[0m")
	if err := g.runGoMod(targetDir, spec.Module, isFullMode(spec), g.lingBaseRoot); err != nil {
		fmt.Printf("  \x1b[33m[警告] %v\x1b[0m\n", err)
		fmt.Println("  \x1b[38;5;245m项目文件已生成，但依赖未完全解析。请按上述提示操作后运行 go run ./cmd/...\x1b[0m")
	} else {
		fmt.Printf("  \x1b[32m[完成]\x1b[0m go mod init %s\n", spec.Module)
	}

	// 初始化 git。
	if spec.Git {
		fmt.Println()
		fmt.Println("\x1b[38;5;117m━━━ 初始化 Git 仓库 ━━━\x1b[0m")
		if err := g.runGitInit(targetDir); err != nil {
			fmt.Printf("  \x1b[33m[警告] git init 失败: %v\x1b[0m\n", err)
		} else {
			fmt.Printf("  \x1b[32m[完成]\x1b[0m git init + initial commit\n")
		}
	}

	// 完成。
	fmt.Println()
	fmt.Printf("\x1b[32m✓ 项目创建成功!\x1b[0m\n")
	fmt.Println()
	absDir := targetDir
	if abs, err := filepath.Abs(targetDir); err == nil {
		absDir = abs
	}
	fmt.Printf("  项目路径: \x1b[38;5;39m%s\x1b[0m\n", absDir)
	fmt.Println()
	fmt.Println("\x1b[38;5;117m后续步骤:\x1b[0m")
	if targetDir != "." {
		fmt.Printf("  \x1b[38;5;245mcd\x1b[0m %s\n", targetDir)
	}
	fmt.Println("  \x1b[38;5;245mgo run ./cmd/...\x1b[0m")
	if spec.Docker {
		fmt.Printf("  \x1b[38;5;245mdocker build -t %s .\x1b[0m\n", spec.ProjectName())
	}
	fmt.Println()
	fmt.Printf("  \x1b[38;5;245m祝您编码愉快! 🚀\x1b[0m\n")

	return nil
}

// runGoMod 在目标目录运行 go mod init + tidy。
func (g *Generator) runGoMod(dir, module string, fullMode bool, lingBaseRoot string) error {
	goBin := findGoBin()

	cmd := exec.Command(goBin, "mod", "init", module)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if fullMode {
		// In full mode, all ling-base source is copied to pkg/ and imports
		// are rewritten to local paths — no external LingByte dependencies.
		// Inject version constraints from submodule go.mod files BEFORE
		// `go mod tidy` so that correct versions are resolved from the
		// start (e.g. alibabacloud SDK versions that only exist in specific
		// minor versions).
		if lingBaseRoot != "" {
			requires := collectSubmoduleRequires(lingBaseRoot)
			if len(requires) > 0 {
				injectSubmoduleRequires(dir, requires)
			}
		}
	} else {
		// In lib mode, explicitly go get each LingByte submodule with its
		// published version. This is necessary because `go mod tidy` cannot
		// auto-discover submodules when the root module path is a prefix of
		// the submodule path.
		lingBaseImports := resolveLingBaseImports(dir)
		if len(lingBaseImports) > 0 {
			goGetLingBaseModules(dir, lingBaseImports)
		}
	}

	// go mod tidy — capture output to detect failures
	tidyCmd := exec.Command(goBin, "mod", "tidy")
	tidyCmd.Dir = dir
	tidyOut, err := tidyCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  \x1b[33m[警告] go mod tidy 失败\x1b[0m\n")
		fmt.Println()
		fmt.Println("  \x1b[38;5;245m可能原因: 部分 LingByte 子模块尚未发布到公共 Go module proxy\x1b[0m")
		fmt.Println()
		fmt.Println("  \x1b[38;5;117m解决方案（任选其一）:\x1b[0m")
		fmt.Println("  \x1b[38;5;39m1.\x1b[0m 使用 full 模式重新生成（复制源码，无需外部依赖）:")
		fmt.Println("     \x1b[38;5;245mlingcli create <项目名> --mode full\x1b[0m")
		fmt.Println("  \x1b[38;5;39m2.\x1b[0m 手动添加 replace 指令指向本地 ling-base 源码:")
		fmt.Println("     \x1b[38;5;245mgo mod edit -replace github.com/LingByte/ling-base=<本地路径>\x1b[0m")
		fmt.Println("  \x1b[38;5;39m3.\x1b[0m 等待 LingByte 子模块发布后重新运行:")
		fmt.Println("     \x1b[38;5;245mgo mod tidy\x1b[0m")
		fmt.Println()
		// Print first 10 lines of error output for debugging
		lines := strings.Split(string(tidyOut), "\n")
		maxLines := 10
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		fmt.Println("  \x1b[38;5;245m错误详情（前几行）:\x1b[0m")
		for i := 0; i < maxLines; i++ {
			if lines[i] != "" {
				fmt.Printf("  \x1b[38;5;245m%s\x1b[0m\n", lines[i])
			}
		}
		fmt.Println()
		return fmt.Errorf("go mod tidy 失败（模块未发布）")
	}

	// In full mode, re-inject version constraints AFTER `go mod tidy`.
	// This enforces correct versions that `go mod tidy` may have downgraded
	// (e.g. gorm.io/plugin/dbresolver v1.5.3 → v1.6.2). We do NOT run tidy
	// again after this — the pinned versions are intentional overrides.
	if fullMode && lingBaseRoot != "" {
		requires := collectSubmoduleRequires(lingBaseRoot)
		if len(requires) > 0 {
			injectSubmoduleRequires(dir, requires)
		}
	}

	fmt.Printf("  \x1b[32m[完成]\x1b[0m go mod tidy\n")
	return nil
}

// findGoBin 查找 go 可执行文件路径。
// 优先用 PATH 中的 go，找不到时尝试常见路径。
func findGoBin() string {
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}
	for _, candidate := range []string{
		"/usr/local/bin/go",
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/lib/go/bin/go",
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "go" // fallback
}

// runGitInit 在目标目录初始化 git 并创建首次提交。
func (g *Generator) runGitInit(dir string) error {
	gitBin := "git"
	if path, err := exec.LookPath("git"); err == nil {
		gitBin = path
	}

	for _, args := range [][]string{
		{gitBin, "init"},
		{gitBin, "add", "-A"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	cmd := exec.Command(gitBin, "commit", "-m", fmt.Sprintf("Initial commit via lingcli at %s", time.Now().Format("2006-01-02")))
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	return nil
}
