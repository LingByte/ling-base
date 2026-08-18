// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectSpec 持有新项目的完整规格。
type ProjectSpec struct {
	Name     string // 项目名称 / 目录名
	Module   string // Go module 路径 (如 github.com/me/myapp)
	Template string // 项目模板 ID
	Author   string // 作者
	Port     int    // 服务端口 (web-api / grpc-service)
	Docker   bool   // 是否生成 Docker 部署文件
	Git      bool   // 是否初始化 git
}

// FillDefaults 填充未设置的字段。
func (s *ProjectSpec) FillDefaults() {
	if s.Module == "" {
		s.Module = fmt.Sprintf("github.com/yourname/%s", s.Name)
	}
	if s.Port == 0 {
		switch s.Template {
		case "grpc-service":
			s.Port = 50051
		case "web-api":
			s.Port = 8080
		case "worker":
			s.Port = 8080
		}
	}
}

// Validate 校验规格。
func (s *ProjectSpec) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("项目名称不能为空")
	}
	if s.Name != "." && strings.ContainsAny(s.Name, " \t\n\r") {
		return fmt.Errorf("项目名称不能包含空格")
	}
	if s.Module == "" {
		return fmt.Errorf("module 路径不能为空")
	}
	if s.Template == "" {
		return fmt.Errorf("必须选择项目模板")
	}
	if findTemplate(s.Template) == nil {
		return fmt.Errorf("未知模板: %s", s.Template)
	}
	return nil
}

// TargetDir 返回项目目标目录。
func (s *ProjectSpec) TargetDir() string {
	if s.Name == "." {
		return "."
	}
	return s.Name
}

// ProjectName 返回规范化的项目名（从目录名或 module 路径提取）。
func (s *ProjectSpec) ProjectName() string {
	if s.Name != "." {
		return filepath.Base(s.Name)
	}
	parts := strings.Split(s.Module, "/")
	return parts[len(parts)-1]
}

// PackageName 返回合法的 Go 包名（将连字符等替换为下划线）。
func (s *ProjectSpec) PackageName() string {
	name := s.ProjectName()
	return sanitizePackageName(name)
}

// sanitizePackageName 将非合法 Go 标识符字符替换为下划线。
func sanitizePackageName(name string) string {
	var sb strings.Builder
	for i, r := range name {
		if r == '-' || r == '.' || r == ' ' {
			sb.WriteByte('_')
		} else if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	result := sb.String()
	if result == "" {
		return "pkg"
	}
	// 首字母不能是数字
	if result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}
	return result
}

// Summary 返回人类可读的项目规格摘要。
func (s *ProjectSpec) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m项目名称:\x1b[0m    %s\n", s.ProjectName()))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m项目模板:\x1b[0m    %s\n", s.Template))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117mModule:\x1b[0m      %s\n", s.Module))
	if s.Author != "" {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m作者:\x1b[0m        %s\n", s.Author))
	}
	tmpl := findTemplate(s.Template)
	if tmpl != nil && tmpl.NeedsPort {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m端口:\x1b[0m        %d\n", s.Port))
	}
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117mDocker:\x1b[0m      %s\n", boolYesNo(s.Docker)))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117mGit 初始化:\x1b[0m  %s\n", boolYesNo(s.Git)))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m目标目录:\x1b[0m    %s\n", s.TargetDir()))
	return sb.String()
}

func boolYesNo(b bool) string {
	if b {
		return "\x1b[32m是\x1b[0m"
	}
	return "\x1b[33m否\x1b[0m"
}

// ──────────────────────────────────────────────
// Template registry
// ──────────────────────────────────────────────

// ProjectTemplate 定义一个项目模板。
type ProjectTemplate struct {
	ID          string   // 模板标识
	Description string   // 模板描述
	Structure   string   // 目录结构摘要
	NeedsPort   bool     // 是否需要端口配置
	Generate    func(spec *ProjectSpec) []FileEntry // 生成文件列表
}

// FileEntry 表示一个待生成的文件。
type FileEntry struct {
	Path     string // 相对于项目根目录的路径
	Content  string // 文件内容
	NoFmt    bool   // 如果为 true，不运行 gofmt
}

// ProjectTemplates 是所有可用项目模板的注册表。
var ProjectTemplates = []ProjectTemplate{
	{
		ID:          "web-api",
		Description: "HTTP REST API 服务（路由、中间件、配置、优雅关闭）",
		Structure:   "cmd/ internal/ pkg/ configs/ Dockerfile docker-compose.yml",
		NeedsPort:   true,
		Generate:    generateWebAPI,
	},
	{
		ID:          "grpc-service",
		Description: "gRPC 服务（proto 定义、服务实现、拦截器）",
		Structure:   "cmd/ internal/ api/proto/ Dockerfile docker-compose.yml",
		NeedsPort:   true,
		Generate:    generateGRPCService,
	},
	{
		ID:          "cli-tool",
		Description: "命令行工具（子命令、flag 解析、配置文件）",
		Structure:   "cmd/ internal/ pkg/ README.md",
		NeedsPort:   false,
		Generate:    generateCLITool,
	},
	{
		ID:          "library",
		Description: "可复用 Go 库（pkg 包、完整测试、示例）",
		Structure:   "pkg/ examples/ README.md LICENSE",
		NeedsPort:   false,
		Generate:    generateLibrary,
	},
	{
		ID:          "worker",
		Description: "后台任务/定时任务 Worker（任务调度、队列消费）",
		Structure:   "cmd/ internal/ pkg/ Dockerfile docker-compose.yml",
		NeedsPort:   true,
		Generate:    generateWorker,
	},
}

// findTemplate 根据 ID 查找模板。
func findTemplate(id string) *ProjectTemplate {
	for i := range ProjectTemplates {
		if ProjectTemplates[i].ID == id {
			return &ProjectTemplates[i]
		}
	}
	return nil
}

// joinPath 连接路径片段。
func joinPath(parts ...string) string {
	return filepath.Join(parts...)
}
