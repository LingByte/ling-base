// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the ling-base CLI 脚手架工具。
//
// 通过交互式命令行或参数模式，自动生成符合 ling-base 规范的模块基础代码。
//
// 用法:
//
//	# 交互模式（推荐）
//	cli
//
//	# 非交互模式
//	cli new --type core --name mycache
//	cli new --type backend --name redis --parent cache
//	cli new --type util --name strutil --parent common
//	cli new --type provider --name aliyun --parent notification/sms
//
//	# 列出可用模板
//	cli list
//
//	# 查看帮助
//	cli help
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/LingByte/ling-base/bootstrap"
)

const cliVersion = "v0.1.0"

func main() {
	if len(os.Args) < 2 {
		runInteractive()
		return
	}

	switch os.Args[1] {
	case "new":
		runNew(os.Args[2:])
	case "list":
		runList()
	case "version", "-v", "--version":
		fmt.Println("ling-base/cli", cliVersion)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	printBanner()
	fmt.Println("用法:")
	fmt.Println("  cli                      交互模式（推荐）")
	fmt.Println("  cli new [参数]           生成新模块")
	fmt.Println("  cli list                 列出可用模块模板")
	fmt.Println("  cli version              打印版本号")
	fmt.Println("  cli help                 显示帮助")
	fmt.Println()
	fmt.Println("new 命令参数:")
	fmt.Println("  --type <类型>            模块类型: core, backend, util, provider（必填）")
	fmt.Println("  --name <名称>            模块名称（必填）")
	fmt.Println("  --parent <父模块>        父模块路径（backend/util/provider 必填）")
	fmt.Println("  --description <描述>     模块描述")
	fmt.Println("  --dry-run                预览模式，不实际写文件")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  cli new --type core --name mycache --description \"我的缓存抽象\"")
	fmt.Println("  cli new --type backend --name redis --parent cache")
	fmt.Println("  cli new --type util --name strutil --parent common")
	fmt.Println("  cli new --type provider --name aliyun --parent notification/sms")
}

// printBanner 通过 bootstrap 的 banner 功能打印 ASCII 艺术字。
func printBanner() {
	_ = bootstrap.PrintBannerColored(os.Stdout, "LING-BASE CLI")
	fmt.Println()
	fmt.Println("  \x1b[38;5;39m模块脚手架工具 — 自动生成 ling-base 模块基础代码\x1b[0m")
	fmt.Println()
}

// runNew 处理非交互式 "new" 子命令。
func runNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	moduleType := fs.String("type", "", "模块类型: core, backend, util, provider")
	name := fs.String("name", "", "模块名称")
	parent := fs.String("parent", "", "父模块路径（backend/util/provider 必填）")
	desc := fs.String("description", "", "模块描述")
	dryRun := fs.Bool("dry-run", false, "预览模式，不实际写文件")
	fs.Parse(args)

	if *moduleType == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "错误: --type 和 --name 是必填参数")
		fs.Usage()
		os.Exit(1)
	}

	spec := ModuleSpec{
		Type:        ModuleType(*moduleType),
		Name:        *name,
		Parent:      *parent,
		Description: *desc,
		DryRun:      *dryRun,
	}

	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	gen := NewGenerator()
	if err := gen.Generate(&spec); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// runList 打印所有可用模块模板。
func runList() {
	printBanner()
	fmt.Println("可用模块模板:")
	fmt.Println()
	for _, t := range Templates {
		fmt.Printf("  \x1b[38;5;39m%-10s\x1b[0m  %s\n", t.Type, t.Description)
		fmt.Printf("             生成文件: %s\n", t.FileList())
		fmt.Println()
	}
}

// runInteractive 启动交互式菜单。
func runInteractive() {
	printBanner()

	p := NewPrompt()

	// 步骤 1: 选择模块类型。
	fmt.Println("\x1b[38;5;117m━━━ 步骤 1/4: 选择模块类型 ━━━\x1b[0m")
	for i, t := range Templates {
		fmt.Printf("  \x1b[38;5;39m[%d]\x1b[0m \x1b[1m%-10s\x1b[0m — %s\n", i+1, t.Type, t.Description)
	}
	idx := p.Select("请选择模块类型", len(Templates))
	selected := Templates[idx]

	fmt.Printf("\n  \x1b[32m✓ 已选择: %s\x1b[0m\n\n", selected.Type)

	// 步骤 2: 输入模块名称。
	fmt.Println("\x1b[38;5;117m━━━ 步骤 2/4: 模块名称 ━━━\x1b[0m")
	fmt.Println("  \x1b[38;5;245m提示: 小写英文，不含空格和斜杠，如 mycache、redis、strutil\x1b[0m")
	name := p.Input("请输入模块名称", "")
	if name == "" {
		fmt.Println("\n\x1b[31m✗ 模块名称不能为空，已取消\x1b[0m")
		return
	}

	// 步骤 3: 输入父模块（非 core 类型）。
	parent := ""
	if selected.Type != TypeCore {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 3/4: 父模块路径 ━━━\x1b[0m")
		fmt.Println("  \x1b[38;5;245m示例: cache, mq, notification, common, notification/sms\x1b[0m")
		parent = p.Input("请输入父模块路径", "")
		if parent == "" {
			fmt.Println("\n\x1b[31m✗ 父模块路径不能为空，已取消\x1b[0m")
			return
		}
	} else {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 3/4: 跳过（core 类型无需父模块）━━━\x1b[0m")
	}

	// 步骤 4: 模块描述。
	fmt.Println("\n\x1b[38;5;117m━━━ 步骤 4/4: 模块描述 ━━━\x1b[0m")
	desc := p.Input("请输入模块描述（可选，回车跳过）", "")

	// 确认。
	spec := ModuleSpec{
		Type:        selected.Type,
		Name:        name,
		Parent:      parent,
		Description: desc,
	}

	fmt.Println("\n\x1b[38;5;117m━━━ 确认信息 ━━━\x1b[0m")
	fmt.Print(spec.Summary())
	fmt.Println()

	if !p.Confirm("确认生成此模块？") {
		fmt.Println("\n\x1b[33m已取消\x1b[0m")
		return
	}

	gen := NewGenerator()
	if err := gen.Generate(&spec); err != nil {
		fmt.Fprintf(os.Stderr, "\n\x1b[31m错误: %v\x1b[0m\n", err)
		os.Exit(1)
	}
}
