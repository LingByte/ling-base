// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Generator 负责将项目文件写入磁盘。
type Generator struct{}

// NewGenerator 创建生成器。
func NewGenerator() *Generator {
	return &Generator{}
}

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
			// 允许在非空目录创建，但给出警告。
			fmt.Printf("\n  \x1b[33m[警告] 当前目录非空，文件可能会覆盖\x1b[0m\n")
		}
	}

	fmt.Printf("\n\x1b[38;5;117m━━━ 正在创建项目: %s ━━━\x1b[0m\n", spec.ProjectName())
	fmt.Print(spec.Summary())
	fmt.Println()

	// 写入所有文件。
	for _, f := range files {
		fullPath := filepath.Join(targetDir, f.Path)

		// 检查是否已存在。
		if _, err := os.Stat(fullPath); err == nil {
			fmt.Printf("  \x1b[33m[跳过]\x1b[0m %s \x1b[38;5;245m（文件已存在）\x1b[0m\n", f.Path)
			continue
		}

		// 创建目录。
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s: %w", dir, err)
		}

		// 写入文件。
		if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("写入 %s: %w", f.Path, err)
		}

		fmt.Printf("  \x1b[32m[完成]\x1b[0m %s\n", f.Path)
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
	fmt.Printf("  项目路径: \x1b[38;5;39m%s\x1b[0m\n", func() string {
		if abs, err := filepath.Abs(targetDir); err == nil {
			return abs
		}
		return targetDir
	}())
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
	// go mod init
	cmd := exec.Command("go", "mod", "init", module)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// go mod tidy (best effort, may fail if deps aren't available)
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // 不阻塞，tidy 失败不影响

	return nil
}

// runGitInit 在目标目录初始化 git 并创建首次提交。
func (g *Generator) runGitInit(dir string) error {
	commands := [][]string{
		{"git", "init"},
		{"git", "add", "-A"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// git commit
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf("Initial commit via lingcli at %s", time.Now().Format("2006-01-02")))
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // 可能没有 git config，不阻塞

	return nil
}

// ──────────────────────────────────────────────
// 公共文件生成辅助函数
// ──────────────────────────────────────────────

// generateGitignore 生成 .gitignore。
func generateGitignore() string {
	return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/bin/
/dist/

# Test
*.test
*.out
coverage.txt
coverage.html

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
*.pem
*.key

# Build
/tmp/
vendor/
`
}

// generateMakefile 生成通用 Makefile。
func generateMakefile(projectName string) string {
	return fmt.Sprintf(`# %s Makefile

APP := %s
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%%Y-%%m-%%dT%%H:%%M:%%SZ")
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

.PHONY: all build run test vet fmt fmt-check clean docker-build docker-up docker-down help

all: build

## build: 编译项目
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/...

## run: 本地运行
run:
	go run ./cmd/...

## test: 运行测试
test:
	go test -v -race -count=1 ./...

## vet: 静态检查
vet:
	go vet ./...

## fmt: 格式化代码
fmt:
	go fmt ./...

## fmt-check: 检查格式化
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "需要运行 make fmt" && exit 1)

## clean: 清理构建产物
clean:
	rm -rf bin/ dist/ coverage.txt

## docker-build: 构建 Docker 镜像
docker-build:
	docker build -t $(APP):$(VERSION) .

## docker-up: 启动 Docker Compose
docker-up:
	docker compose up -d

## docker-down: 停止 Docker Compose
docker-down:
	docker compose down

## help: 显示帮助
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
`, projectName, projectName)
}

// generateDockerfile 生成 Dockerfile。
func generateDockerfile(projectName, entrypoint string) string {
	return fmt.Sprintf(`# ---- 构建阶段 ----
FROM golang:1.23-alpine AS builder

WORKDIR /build

# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./%s

# ---- 运行阶段 ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app /app/app
COPY --from=builder /build/configs /app/configs

ENV TZ=Asia/Shanghai

EXPOSE 8080

ENTRYPOINT ["/app/app"]
`, entrypoint)
}

// generateDockerCompose 生成 docker-compose.yml。
func generateDockerCompose(projectName string, port int) string {
	return fmt.Sprintf(`version: "3.8"

services:
  app:
    build: .
    container_name: %s
    ports:
      - "%d:%d"
    environment:
      - APP_ENV=production
    restart: unless-stopped
    depends_on: []
`, projectName, port, port)
}

// generateREADME 生成 README.md。
func generateREADME(spec *ProjectSpec, extraSections string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", spec.ProjectName()))

	if spec.Template == "web-api" {
		sb.WriteString("HTTP REST API 服务。\n\n")
	} else if spec.Template == "grpc-service" {
		sb.WriteString("gRPC 服务。\n\n")
	} else if spec.Template == "cli-tool" {
		sb.WriteString("命令行工具。\n\n")
	} else if spec.Template == "library" {
		sb.WriteString("Go 库。\n\n")
	} else if spec.Template == "worker" {
		sb.WriteString("后台任务 Worker。\n\n")
	}

	// Badges.
	sb.WriteString("![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)\n")
	if spec.Docker {
		sb.WriteString("![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)\n")
	}
	sb.WriteString("\n")

	// Features.
	sb.WriteString("## 功能特性\n\n")
	for _, f := range getFeatures(spec.Template) {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}
	sb.WriteString("\n")

	// Quick start.
	sb.WriteString("## 快速开始\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 克隆项目\n")
	sb.WriteString(fmt.Sprintf("git clone <repo-url>\n"))
	sb.WriteString(fmt.Sprintf("cd %s\n\n", spec.ProjectName()))
	sb.WriteString("# 安装依赖\n")
	sb.WriteString("go mod tidy\n\n")
	sb.WriteString("# 运行\n")
	if spec.Template == "library" {
		sb.WriteString("go test ./...\n")
	} else {
		sb.WriteString("make run\n")
	}
	sb.WriteString("```\n\n")

	// Project structure.
	sb.WriteString("## 目录结构\n\n")
	sb.WriteString("```\n")
	sb.WriteString(strings.ReplaceAll(getStructureTree(spec.Template), "${projectName}", spec.ProjectName()))
	sb.WriteString("```\n\n")

	// Extra sections.
	if extraSections != "" {
		sb.WriteString(extraSections)
		sb.WriteString("\n")
	}

	// Docker.
	if spec.Docker {
		sb.WriteString("## Docker 部署\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString("# 构建镜像\n")
		sb.WriteString(fmt.Sprintf("docker build -t %s .\n\n", spec.ProjectName()))
		sb.WriteString("# 运行容器\n")
		if spec.Template == "library" || spec.Template == "cli-tool" {
			sb.WriteString(fmt.Sprintf("docker run --rm %s\n", spec.ProjectName()))
		} else {
			sb.WriteString(fmt.Sprintf("docker run -p %d:%d --rm %s\n", spec.Port, spec.Port, spec.ProjectName()))
		}
		sb.WriteString("\n")
		sb.WriteString("# 或使用 Docker Compose\n")
		sb.WriteString("docker compose up -d\n")
		sb.WriteString("```\n\n")
	}

	// Make commands.
	if spec.Template != "library" {
		sb.WriteString("## Make 命令\n\n")
		sb.WriteString("| 命令 | 说明 |\n|------|------|\n")
		sb.WriteString("| `make build` | 编译项目 |\n")
		sb.WriteString("| `make run` | 本地运行 |\n")
		sb.WriteString("| `make test` | 运行测试 |\n")
		sb.WriteString("| `make vet` | 静态检查 |\n")
		sb.WriteString("| `make fmt` | 格式化代码 |\n")
		if spec.Docker {
			sb.WriteString("| `make docker-build` | 构建 Docker 镜像 |\n")
			sb.WriteString("| `make docker-up` | 启动 Docker Compose |\n")
		}
		sb.WriteString("\n")
	}

	// License.
	sb.WriteString("## License\n\n")
	if spec.Author != "" {
		sb.WriteString(fmt.Sprintf("Copyright (c) %d %s. MIT License.\n", time.Now().Year(), spec.Author))
	} else {
		sb.WriteString("MIT License.\n")
	}

	return sb.String()
}

func getFeatures(template string) []string {
	switch template {
	case "web-api":
		return []string{
			"HTTP REST API 路由 (net/http)",
			"中间件支持 (日志、恢复、CORS)",
			"配置文件管理 (YAML)",
			"优雅关闭 (Graceful Shutdown)",
			"健康检查接口 (/health)",
			"结构化日志 (slog)",
			"Docker 容器化部署",
		}
	case "grpc-service":
		return []string{
			"gRPC 服务框架",
			"Protocol Buffers 定义",
			"拦截器支持 (日志、恢复)",
			"健康检查",
			"配置文件管理 (YAML)",
			"Docker 容器化部署",
		}
	case "cli-tool":
		return []string{
			"子命令模式",
			"Flag 解析",
			"配置文件支持 (YAML)",
			"彩色输出",
			"版本信息",
		}
	case "library":
		return []string{
			"完整的包文档",
			"单元测试",
			"使用示例",
			"GoDoc 兼容",
		}
	case "worker":
		return []string{
			"定时任务调度",
			"优雅关闭",
			"配置文件管理 (YAML)",
			"结构化日志 (slog)",
			"Docker 容器化部署",
		}
	default:
		return []string{"基础项目结构"}
	}
}

func getStructureTree(template string) string {
	switch template {
	case "web-api":
		return `${projectName}/
├── cmd/
│   └── server/
│       └── main.go          # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go        # 配置定义与加载
│   ├── handler/
│   │   └── handler.go       # HTTP 处理器
│   ├── middleware/
│   │   └── middleware.go     # 中间件
│   ├── model/
│   │   └── model.go         # 数据模型
│   └── service/
│       └── service.go       # 业务逻辑
├── pkg/
│   └── response/
│       └── response.go      # 统一响应封装
├── configs/
│   └── config.yaml          # 配置文件
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	case "grpc-service":
		return `${projectName}/
├── cmd/
│   └── server/
│       └── main.go          # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go        # 配置定义
│   ├── service/
│   │   └── service.go       # gRPC 服务实现
│   └── interceptor/
│       └── interceptor.go   # 拦截器
├── api/
│   └── proto/
│       └── service.proto    # Protobuf 定义
├── configs/
│   └── config.yaml
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	case "cli-tool":
		return `${projectName}/
├── cmd/
│   └── ${projectName}/
│       └── main.go          # 程序入口
├── internal/
│   ├── command/
│   │   └── command.go       # 子命令定义
│   └── config/
│       └── config.go        # 配置管理
├── pkg/
│   └── version/
│       └── version.go       # 版本信息
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	case "library":
		return `${projectName}/
├── pkg/
│   └── ${projectName}/
│       ├── ${projectName}.go    # 核心实现
│       └── ${projectName}_test.go
├── examples/
│   └── basic/
│       └── main.go          # 使用示例
├── .gitignore
├── go.mod
├── LICENSE
└── README.md
`
	case "worker":
		return `${projectName}/
├── cmd/
│   └── worker/
│       └── main.go          # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go        # 配置定义
│   ├── task/
│   │   └── task.go          # 任务定义
│   └── scheduler/
│       └── scheduler.go     # 调度器
├── pkg/
│   └── logger/
│       └── logger.go        # 日志工具
├── configs/
│   └── config.yaml
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	default:
		return "${projectName}/\n"
	}
}
