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
type Generator struct{}

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
			// 回退到本地 ling-base 源码
			lingBaseRoot := findLingBaseRoot()
			if lingBaseRoot == "" {
				fmt.Printf("  \x1b[31m[错误] 无法定位 ling-base 源码目录\x1b[0m\n")
				fmt.Println()
				fmt.Println("  \x1b[38;5;245mfull 模式需要 ling-base 源码。请用以下任一方式指定:\x1b[0m")
				fmt.Println("    1. 设置环境变量: export LING_BASE_ROOT=/path/to/ling-base")
				fmt.Printf("    2. 使用 --ling-base-root 参数: --ling-base-root /path/to/ling-base\n")
				fmt.Println("    3. 在 ling-base 目录下运行 lingcli")
				fmt.Println("    4. 使用 --mode lib（默认，引入库而非复制源码）")
				fmt.Println()
				return fmt.Errorf("full 模式无法定位 ling-base 源码")
			}
			if err := generateFullMode(spec, lingBaseRoot, targetDir); err != nil {
				return fmt.Errorf("源码复制失败: %w", err)
			}
		}
	}

	// 运行 go mod init + tidy。
	fmt.Println()
	fmt.Println("\x1b[38;5;117m━━━ 初始化 Go module ━━━\x1b[0m")
	if err := g.runGoMod(targetDir, spec.Module); err != nil {
		fmt.Printf("  \x1b[33m[警告] go mod 初始化失败: %v\x1b[0m\n", err)
		fmt.Println("  \x1b[38;5;245m请手动运行: go mod init && go mod tidy\x1b[0m")
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
	fmt.Println("  \x1b[38;5;245mgo mod tidy\x1b[0m")
	fmt.Println("  \x1b[38;5;245mgo run ./cmd/...\x1b[0m")
	if spec.Docker {
		fmt.Printf("  \x1b[38;5;245mdocker build -t %s .\x1b[0m\n", spec.ProjectName())
	}
	fmt.Println()
	fmt.Printf("  \x1b[38;5;245m祝您编码愉快! 🚀\x1b[0m\n")

	return nil
}

// runGoMod 在目标目录运行 go mod init + tidy。
func (g *Generator) runGoMod(dir, module string) error {
	goBin := findGoBin()

	cmd := exec.Command(goBin, "mod", "init", module)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command(goBin, "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

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
