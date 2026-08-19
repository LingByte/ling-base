// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the ling-cli project scaffolding tool.
//
// 类似 create-vue / create-react-app，一键生成完整的 Go 项目骨架，
// 包含 cmd/internal/pkg 目录结构、Docker 部署、Makefile、README 等。
//
// 用法:
//
//	# 交互模式（推荐）
//	lingcli create myproject
//
//	# 在当前目录初始化
//	lingcli create .
//
//	# 非交互模式
//	lingcli create myproject --template web-api --module github.com/me/myproject
//
//	# 列出可用模板
//	lingcli list
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const cliVersion = "v0.5.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "create":
		runCreate(os.Args[2:])
	case "list":
		runList()
	case "version", "-v", "--version":
		fmt.Println("lingcli", cliVersion)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`
  _      _  _   _  _____         ______   ___   _____  ______  _____  _      _
 | |    | || \ | ||  __ \       | ___ \  / _ \ /  ___||  ____|/  __ \| |    | |
 | |    | ||  \| || |  \/       | |_/ / / /_\ \\ --.  | |__   | /  \/| |    | |
 | |    | || . ` + "`" + ` || | __  _____ | ___ \ |  _  |  --. \|  __|  | |    | |    | |
 | |___ | || |\  || |_\ \       | |_/ / | | | |/\__/ /| |____ | \__/\| |___ | |
 |_____||_||_| \_| \____/       \____/  \_| |_/\____/ |______| \____/|_____||_|

  LingByte 项目脚手架 — 一键生成完整 Go 项目骨架`)

	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  lingcli create <项目名>              交互模式创建项目（推荐）")
	fmt.Println("  lingcli create .                      在当前目录初始化项目")
	fmt.Println("  lingcli create <项目名> [参数]        非交互模式创建项目")
	fmt.Println("  lingcli list                          列出可用项目模板")
	fmt.Println("  lingcli version                       打印版本号")
	fmt.Println("  lingcli help                          显示帮助")
	fmt.Println()
	fmt.Println("create 命令参数:")
	fmt.Println("  --template <模板>       项目模板: web-api, grpc-service, cli-tool, library, worker")
	fmt.Println("  --module <模块路径>     Go module 路径（如 github.com/me/myproject）")
	fmt.Println("  --author <作者>         作者名称")
	fmt.Println("  --port <端口>           服务端口（web-api / grpc-service）")
	fmt.Println("  --modules <模块列表>    要集成的 ling-base 模块（逗号分隔，如 apidocs,limiter,jwt）")
	fmt.Println("  --docker                生成 Docker 部署文件（默认 true）")
	fmt.Println("  --no-docker             不生成 Docker 部署文件")
	fmt.Println("  --git                   初始化 git 仓库（默认 true）")
	fmt.Println("  --no-git                不初始化 git 仓库")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  lingcli create myapp")
	fmt.Println("  lingcli create myapp --template web-api --module github.com/me/myapp")
	fmt.Println("  lingcli create . --template library --module github.com/me/mylib")
}

// runCreate 处理 create 子命令。
func runCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	tmpl := fs.String("template", "", "项目模板")
	module := fs.String("module", "", "Go module 路径")
	author := fs.String("author", "", "作者名称")
	port := fs.Int("port", 0, "服务端口")
	modules := fs.String("modules", "", "要集成的 ling-base 模块（逗号分隔）")
	docker := fs.Bool("docker", true, "生成 Docker 部署文件")
	git := fs.Bool("git", true, "初始化 git 仓库")

	// 分离位置参数和 flag 参数。
	// Go flag 包在遇到第一个非 flag 参数后停止解析，
	// 所以我们需要把位置参数移到末尾。
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			// 如果是 --flag=value 形式，不需要额外取值
			// 如果是 --flag value 形式，需要跳过下一个
			if !strings.Contains(args[i], "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// 检查是否是 bool flag
				switch args[i] {
				case "-docker", "--docker", "-no-docker", "--no-docker", "-git", "--git", "-no-git", "--no-git":
					// bool flag, 不取值
				default:
					flagArgs = append(flagArgs, args[i+1])
					i++
				}
			}
		} else {
			posArgs = append(posArgs, args[i])
		}
	}

	fs.Parse(flagArgs)

	// 位置参数：项目名或 "."
	target := "."
	if len(posArgs) > 0 {
		target = posArgs[0]
	}

	spec := &ProjectSpec{
		Name:     target,
		Module:   *module,
		Author:   *author,
		Port:     *port,
		Docker:   *docker,
		Git:      *git,
		Template: *tmpl,
	}

	// 解析 --modules
	if *modules != "" {
		for _, id := range strings.Split(*modules, ",") {
			id = strings.TrimSpace(id)
			if findLingBaseModule(id) != nil {
				spec.Modules = append(spec.Modules, id)
			}
		}
	}

	// 如果缺少必要信息，进入交互模式。
	if *tmpl == "" || *module == "" {
		runInteractive(spec)
		return
	}

	// 非交互模式：补全默认值。
	spec.FillDefaults()

	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	gen := NewGenerator()
	if err := gen.Generate(spec); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// runList 打印所有可用项目模板。
func runList() {
	fmt.Println()
	fmt.Println("可用项目模板:")
	fmt.Println()
	for _, t := range ProjectTemplates {
		fmt.Printf("  \x1b[38;5;39m%-15s\x1b[0m %s\n", t.ID, t.Description)
		fmt.Printf("                  目录结构: %s\n", t.Structure)
		fmt.Println()
	}
}

// runInteractive 启动交互式创建流程。
func runInteractive(spec *ProjectSpec) {
	p := NewPrompt()

	// 项目名称。
	if spec.Name == "." || spec.Name == "" {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 1/6: 项目名称 ━━━\x1b[0m")
		fmt.Println("  \x1b[38;5;245m提示: 小写英文，不含空格，如 myapp、my-service\x1b[0m")
		spec.Name = p.Input("请输入项目名称", "myapp")
	} else {
		fmt.Printf("\n\x1b[32m✓ 项目名称: %s\x1b[0m\n", spec.Name)
	}

	// 选择模板。
	if spec.Template == "" {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 2/6: 选择项目模板 ━━━\x1b[0m")
		for i, t := range ProjectTemplates {
			fmt.Printf("  \x1b[38;5;39m[%d]\x1b[0m \x1b[1m%-15s\x1b[0m %s\n", i+1, t.ID, t.Description)
		}
		idx := p.Select("请选择项目模板", len(ProjectTemplates))
		spec.Template = ProjectTemplates[idx].ID
		fmt.Printf("  \x1b[32m✓ 已选择: %s\x1b[0m\n", spec.Template)
	} else {
		fmt.Printf("\x1b[32m✓ 项目模板: %s\x1b[0m\n", spec.Template)
	}

	// Module 路径。
	if spec.Module == "" {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 3/6: Go Module 路径 ━━━\x1b[0m")
		fmt.Println("  \x1b[38;5;245m提示: 通常为 github.com/<用户>/<项目名>\x1b[0m")
		defaultMod := fmt.Sprintf("github.com/yourname/%s", spec.Name)
		spec.Module = p.Input("请输入 module 路径", defaultMod)
	} else {
		fmt.Printf("\x1b[32m✓ Module 路径: %s\x1b[0m\n", spec.Module)
	}

	// 作者。
	if spec.Author == "" {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 4/6: 作者信息 ━━━\x1b[0m")
		spec.Author = p.Input("请输入作者名称（可选，回车跳过）", "")
	}

	// 端口（web-api / grpc-service）。
	tmpl := findTemplate(spec.Template)
	if tmpl != nil && tmpl.NeedsPort {
		if spec.Port == 0 {
			fmt.Println("\n\x1b[38;5;117m━━━ 步骤 5/6: 服务端口 ━━━\x1b[0m")
			defaultPort := 8080
			if spec.Template == "grpc-service" {
				defaultPort = 50051
			}
			spec.Port = p.InputInt("请输入监听端口", defaultPort)
		}
	} else {
		fmt.Println("\n\x1b[38;5;117m━━━ 步骤 5/6: 跳过（此模板无需端口）━━━\x1b[0m")
	}

	// 选择 ling-base 模块。
	fmt.Println("\n\x1b[38;5;117m━━━ 步骤 6/6: 集成 ling-base 模块（可选）━━━\x1b[0m")
	fmt.Println("  \x1b[38;5;245m输入模块编号（逗号分隔），回车跳过。如: 1,2,3\x1b[0m")
	for i, m := range LingBaseModules {
		fmt.Printf("  \x1b[38;5;39m[%d]\x1b[0m %-16s %s\n", i+1, m.Name, m.Description)
	}
	fmt.Println()
	selected := p.Input("请选择要集成的模块", "")
	if selected != "" {
		spec.Modules = parseModuleSelection(selected)
	}
	if len(spec.Modules) > 0 {
		var names []string
		for _, id := range spec.Modules {
			if m := findLingBaseModule(id); m != nil {
				names = append(names, m.Name)
			}
		}
		fmt.Printf("  \x1b[32m✓ 已选择: %s\x1b[0m\n", strings.Join(names, ", "))
	} else {
		fmt.Println("  \x1b[38;5;245m未选择任何模块\x1b[0m")
	}

	// 确认。
	fmt.Println("\n\x1b[38;5;117m━━━ 确认信息 ━━━\x1b[0m")
	fmt.Print(spec.Summary())
	fmt.Println()

	if !p.Confirm("确认创建项目？") {
		fmt.Println("\n\x1b[33m已取消\x1b[0m")
		return
	}

	spec.FillDefaults()

	gen := NewGenerator()
	if err := gen.Generate(spec); err != nil {
		fmt.Fprintf(os.Stderr, "\n\x1b[31m错误: %v\x1b[0m\n", err)
		os.Exit(1)
	}
}

// parseModuleSelection 解析用户输入的模块编号列表（如 "1,3,5"）。
func parseModuleSelection(input string) []string {
	var ids []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil || idx < 1 || idx > len(LingBaseModules) {
			continue
		}
		ids = append(ids, LingBaseModules[idx-1].ID)
	}
	return ids
}
