// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

//go:embed all:templates
var templateFS embed.FS

// TemplateData 传递给 .tmpl 模板的渲染数据。
type TemplateData struct {
	Module       string
	ProjectName  string
	PackageName  string
	Author       string
	Year         int
	Port         int
	Description  string
	Features     []string
	Structure    string
	Modules      []string // 选中的 ling-base 模块 ID
	ConfigFormat string   // "yaml" 或 "env"
	CIPlatform   string   // "github", "gitlab" 或 "jenkins"

	// 条件渲染标志（根据 Modules 计算）
	HasAPIDocs        bool
	HasLimiter        bool
	HasCircuitBreaker bool
	HasMiddleware     bool
	HasJWT            bool
	HasResponse       bool
	HasI18n           bool
	IsEnvConfig       bool
	IsYAMLConfig      bool

	// 生成模式: "lib"（引入 ling-base 库）或 "full"（复制源码到 pkg/）
	Mode string
}

// RenderTemplate 渲染单个 .tmpl 文件，返回输出路径和内容。
// srcPath 是相对于 templates/ 的路径（含 .tmpl 后缀）。
// 返回去掉 .tmpl 后缀的目标路径和渲染后的内容。
func RenderTemplate(srcPath string, data *TemplateData) (string, string, error) {
	// 读取模板文件
	content, err := templateFS.ReadFile(filepath.Join("templates", srcPath))
	if err != nil {
		return "", "", fmt.Errorf("读取模板 %s: %w", srcPath, err)
	}

	// 渲染
	tmpl, err := template.New(srcPath).Parse(string(content))
	if err != nil {
		return "", "", fmt.Errorf("解析模板 %s: %w", srcPath, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("渲染模板 %s: %w", srcPath, err)
	}

	// 去掉 .tmpl 后缀得到目标路径
	outPath := strings.TrimSuffix(srcPath, ".tmpl")
	return outPath, buf.String(), nil
}

// ListTemplateFiles 列出某个模板目录下所有 .tmpl 文件（递归）。
// templateID 如 "web-api"、"cli-tool"。
func ListTemplateFiles(templateID string) ([]string, error) {
	root := "templates/" + templateID
	var files []string

	err := fs.WalkDir(templateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// 转为相对于 templates/ 的路径
		rel, _ := filepath.Rel("templates", path)
		if strings.HasSuffix(rel, ".tmpl") {
			files = append(files, rel)
		}
		return nil
	})

	return files, err
}

// ListStaticFiles 列出某个模板目录下所有非 .tmpl 文件（递归）。
// 这些文件不经过模板渲染，直接原样复制到目标项目。
// 排除 .DS_Store 等系统文件。
func ListStaticFiles(templateID string) ([]string, error) {
	root := "templates/" + templateID
	var files []string

	err := fs.WalkDir(templateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		// 排除 .tmpl 文件和系统文件
		if strings.HasSuffix(name, ".tmpl") || name == ".DS_Store" || name == ".gitkeep" {
			return nil
		}
		rel, _ := filepath.Rel("templates", path)
		files = append(files, rel)
		return nil
	})

	return files, err
}

// ──────────────────────────────────────────────
// 模板生成函数 — 每个模板类型对应一个
// ──────────────────────────────────────────────

func generateWebAPI(spec *ProjectSpec) []FileEntry {
	return renderTemplateFiles("web-api", spec)
}

func generateGRPCService(spec *ProjectSpec) []FileEntry {
	return renderTemplateFiles("grpc-service", spec)
}

func generateCLITool(spec *ProjectSpec) []FileEntry {
	return renderTemplateFiles("cli-tool", spec)
}

func generateLibrary(spec *ProjectSpec) []FileEntry {
	files := renderTemplateFiles("library", spec)
	// library 还需要 LICENSE
	return files
}

func generateWorker(spec *ProjectSpec) []FileEntry {
	return renderTemplateFiles("worker", spec)
}

// renderTemplateFiles 渲染指定模板目录下的所有 .tmpl 文件。
// 同时自动包含 shared/ 下的公共文件（.gitignore, Makefile, Dockerfile, README.md）。
func renderTemplateFiles(templateID string, spec *ProjectSpec) []FileEntry {
	data := &TemplateData{
		Module:            spec.Module,
		ProjectName:       spec.ProjectName(),
		PackageName:       spec.PackageName(),
		Author:            spec.Author,
		Year:              time.Now().Year(),
		Port:              spec.Port,
		Description:       getTemplateDescription(templateID),
		Features:          getFeatures(templateID),
		Structure:         getStructureTree(templateID, spec.ProjectName()),
		Modules:           spec.Modules,
		ConfigFormat:      spec.ConfigFormat,
		CIPlatform:        spec.CIPlatform,
		HasAPIDocs:        spec.HasModule("apidocs"),
		HasLimiter:        spec.HasModule("limiter"),
		HasCircuitBreaker: spec.HasModule("circuitbreaker"),
		HasMiddleware:     spec.HasModule("middleware"),
		HasJWT:            spec.HasModule("jwt"),
		HasResponse:       spec.HasModule("response"),
		HasI18n:           spec.HasModule("i18n"),
		IsEnvConfig:       spec.ConfigFormat == "env",
		IsYAMLConfig:      spec.ConfigFormat != "env",
		Mode:              spec.Mode,
	}

	var files []FileEntry

	// 1. 渲染模板专属文件
	tmplFiles, err := ListTemplateFiles(templateID)
	if err != nil {
		fmt.Fprintf(stderr(), "警告: 无法列出模板文件 %s: %v\n", templateID, err)
		return files
	}

	for _, tmplFile := range tmplFiles {
		outPath, content, err := RenderTemplate(tmplFile, data)
		if err != nil {
			fmt.Fprintf(stderr(), "警告: 渲染 %s 失败: %v\n", tmplFile, err)
			continue
		}
		// 如果渲染后内容为空或只有注释（条件模板未满足），跳过此文件
		if strings.TrimSpace(content) == "" {
			continue
		}
		// Go 文件如果没有 package 声明，说明条件不满足，跳过
		if strings.HasSuffix(outPath, ".go") && !strings.Contains(content, "package ") {
			continue
		}
		// 去掉模板 ID 前缀（如 "web-api/cmd/server/main.go" → "cmd/server/main.go"）
		outPath = strings.TrimPrefix(outPath, templateID+"/")
		files = append(files, FileEntry{Path: outPath, Content: content})
	}

	// 2. 渲染 shared 公共文件（如果模板目录中没有同名文件）
	sharedFiles, _ := ListTemplateFiles("shared")
	for _, sf := range sharedFiles {
		outPath, content, err := RenderTemplate(sf, data)
		if err != nil {
			continue
		}
		// 去掉 "shared/" 前缀
		outPath = strings.TrimPrefix(outPath, "shared/")
		// 检查模板是否已有同名文件
		if !hasFile(files, outPath) {
			files = append(files, FileEntry{Path: outPath, Content: content})
		}
	}

	// 3. 复制静态文件（非 .tmpl，不经过模板渲染，直接原样复制）
	staticFiles, _ := ListStaticFiles(templateID)
	for _, sf := range staticFiles {
		content, err := templateFS.ReadFile(filepath.Join("templates", sf))
		if err != nil {
			continue
		}
		// 去掉模板 ID 前缀
		outPath := strings.TrimPrefix(sf, templateID+"/")
		if !hasFile(files, outPath) {
			files = append(files, FileEntry{Path: outPath, Content: string(content)})
		}
	}

	return files
}

// hasFile 检查文件列表中是否已存在指定路径。
func hasFile(files []FileEntry, path string) bool {
	for _, f := range files {
		if f.Path == path {
			return true
		}
	}
	return false
}

// getTemplateDescription 返回模板描述。
func getTemplateDescription(id string) string {
	switch id {
	case "web-api":
		return "HTTP REST API 服务，基于 ling-base bootstrap 框架。"
	case "grpc-service":
		return "gRPC 服务。"
	case "cli-tool":
		return "命令行工具。"
	case "library":
		return "Go 库。"
	case "worker":
		return "后台任务 Worker。"
	default:
		return ""
	}
}

// getFeatures 返回模板的功能特性列表。
func getFeatures(id string) []string {
	switch id {
	case "web-api":
		return []string{
			"HTTP REST API 路由 (Gin)",
			"中间件链 (RequestID / 日志 / 恢复 / CORS)",
			"配置文件管理 (YAML / .env，多环境，支持环境变量覆盖)",
			"数据库支持 (MySQL / PostgreSQL / SQLite，GORM)",
			"统一响应封装 (ling-base/common/response)",
			"优雅关闭 (Graceful Shutdown)",
			"健康检查接口 (/health)",
			"Bootstrap 启动框架 (Banner / 生命周期 / 事件)",
			"结构化日志 (zap + lumberjack 轮转)",
			"API 文档集成 (apidocs / Huma / OpenAPI 3.1)",
			"限流 + 熔断配置 (ling-base limiter / circuitbreaker)",
			"JWT 鉴权配置 (ling-base jwtutil)",
			"Docker 容器化部署 (docker/ 目录，多阶段构建)",
			"Docker Compose 编排 (App + MySQL + Redis)",
			"CI/CD 配置 (GitHub Actions 或 GitLab CI)",
			"代码质量工具 (.golangci.yml + .dockerignore)",
		}
	case "grpc-service":
		return []string{
			"gRPC 服务框架",
			"Protocol Buffers 定义",
			"拦截器支持 (日志、恢复)",
			"Bootstrap 启动框架",
			"配置文件管理 (YAML)",
			"Docker 容器化部署",
		}
	case "cli-tool":
		return []string{
			"子命令模式",
			"Flag 解析",
			"配置文件支持 (YAML)",
			"版本信息",
		}
	case "library":
		return []string{
			"完整的包文档",
			"单元测试",
			"使用示例",
			"函数式选项模式",
			"GoDoc 兼容",
		}
	case "worker":
		return []string{
			"定时任务调度",
			"Bootstrap 启动框架 (Banner / 生命周期)",
			"优雅关闭",
			"配置文件管理 (YAML)",
			"结构化日志 (slog)",
			"Docker 容器化部署",
		}
	default:
		return []string{"基础项目结构"}
	}
}

// getStructureTree 返回目录结构树。
func getStructureTree(templateID, projectName string) string {
	switch templateID {
	case "web-api":
		return projectName + `/├── cmd/
│   └── server/
│       └── main.go              # 程序入口 + 应用启动逻辑
├── internal/
│   ├── configs/
│   │   ├── config.go            # 配置定义与加载
│   │   └── config_test.go
│   ├── handlers/
│   │   ├── urls.go              # 路由注册（humax.Group / Gin）
│   │   ├── handler.go           # HTTP 处理器实现
│   │   ├── auth.go              # JWT 鉴权 handler（可选）
│   │   └── handler_test.go
│   ├── middlewares/
│   │   └── middleware.go        # 项目特定中间件（限流配置等）
│   └── models/
│       └── user.go              # 数据模型
│   └── types/
│       └── types.go             # 通用 DTO
├── configs/
│   ├── .env.example             # 环境变量参考
│   ├── config.yaml              # 开发环境配置
│   └── config.prod.yaml         # 生产环境配置
├── docker/
│   ├── Dockerfile               # 多阶段构建 + HEALTHCHECK
│   └── docker-compose.yml       # App + MySQL + Redis
├── docs/
│   └── README.md                # 项目文档
├── migrations/
│   ├── 001_init.up.sql          # 初始迁移
│   └── 001_init.down.sql        # 回滚迁移
├── scripts/
│   └── migrate.sh               # 数据库迁移脚本
├── skills/                      # AI 开发技能（TDD/代码审查/调试等）
├── i18n/
│   └── translations/            # 翻译文件
├── .github/workflows/ci.yml     # GitHub Actions CI（或 .gitlab-ci.yml / Jenkinsfile）
├── .air.toml                    # Air 热重载配置
├── .dockerignore
├── .editorconfig                # 编辑器一致性配置
├── .golangci.yml
├── CHANGELOG.md                 # 版本变更记录
├── CONTRIBUTING.md              # 贡献指南
├── SECURITY.md                  # 安全策略
├── LICENSE                      # MIT 许可证
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	case "grpc-service":
		return projectName + `/├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── app/
│   │   └── app.go               # 应用启动 (bootstrap)
│   ├── config/
│   │   └── config.go            # 配置定义
│   ├── service/
│   │   └── service.go           # gRPC 服务实现
│   └── interceptor/
│       └── interceptor.go       # 拦截器
├── api/
│   └── proto/
│       └── service.proto        # Protobuf 定义
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
		return projectName + `/├── cmd/
│   └── main.go                  # 程序入口
├── internal/
│   ├── command/
│   │   └── command.go           # 子命令定义
│   └── config/
│       └── config.go            # 配置管理
├── pkg/
│   └── version/
│       └── version.go           # 版本信息
├── Makefile
├── .gitignore
├── go.mod
└── README.md
`
	case "library":
		return projectName + `/├── pkg/
│   ├── library.go               # 核心实现
│   └── library_test.go          # 单元测试
├── examples/
│   └── basic/
│       └── main.go              # 使用示例
├── LICENSE
├── .gitignore
├── go.mod
└── README.md
`
	case "worker":
		return projectName + `/├── cmd/
│   └── worker/
│       └── main.go              # 程序入口
├── internal/
│   ├── app/
│   │   └── app.go               # 应用启动 (bootstrap)
│   ├── config/
│   │   └── config.go            # 配置定义
│   ├── task/
│   │   └── task.go              # 任务定义
│   └── scheduler/
│       └── scheduler.go         # 调度器
├── pkg/
│   └── logger/
│       └── logger.go            # 日志工具
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
		return projectName + "/\n"
	}
}
