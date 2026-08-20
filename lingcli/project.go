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
	Name         string   // 项目名称 / 目录名
	Module       string   // Go module 路径 (如 github.com/me/myapp)
	Template     string   // 项目模板 ID
	Author       string   // 作者
	Port         int      // 服务端口 (web-api / grpc-service)
	Docker       bool     // 是否生成 Docker 部署文件
	Git          bool     // 是否初始化 git
	Modules      []string // 要集成的 ling-base 模块 ID 列表
	ConfigFormat string   // 配置文件格式: "yaml" 或 "env"（默认 yaml）
	CIPlatform   string   // CI 平台: "github" 或 "gitlab"（默认 github）
	Mode         string   // 生成模式: "lib"（引入库）或 "full"（复制源码到 pkg/）
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
	if s.ConfigFormat == "" {
		s.ConfigFormat = "yaml"
	}
	if s.CIPlatform == "" {
		s.CIPlatform = "github"
	}
	if s.Mode == "" {
		s.Mode = "lib"
	}
	// web-api 模板自动集成 response 模块（统一响应封装）
	if s.Template == "web-api" && !s.HasModule("response") {
		s.Modules = append(s.Modules, "response")
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
	if s.ConfigFormat != "" {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m配置格式:\x1b[0m    %s\n", s.ConfigFormat))
	}
	if s.CIPlatform != "" {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117mCI 平台:\x1b[0m      %s\n", s.CIPlatform))
	}
	if s.Mode != "" {
		modeDesc := s.Mode
		if s.Mode == "full" {
			modeDesc = "full（复制源码到 pkg/）"
		} else if s.Mode == "lib" {
			modeDesc = "lib（引入 ling-base 库）"
		}
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m生成模式:\x1b[0m    %s\n", modeDesc))
	}
	if len(s.Modules) > 0 {
		var moduleNames []string
		for _, id := range s.Modules {
			if m := findLingBaseModule(id); m != nil {
				moduleNames = append(moduleNames, m.Name)
			}
		}
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m集成模块:\x1b[0m    %s\n", strings.Join(moduleNames, ", ")))
	}
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
	ID          string                              // 模板标识
	Description string                              // 模板描述
	Structure   string                              // 目录结构摘要
	NeedsPort   bool                                // 是否需要端口配置
	Generate    func(spec *ProjectSpec) []FileEntry // 生成文件列表
}

// FileEntry 表示一个待生成的文件。
type FileEntry struct {
	Path    string // 相对于项目根目录的路径
	Content string // 文件内容
	NoFmt   bool   // 如果为 true，不运行 gofmt
}

// ProjectTemplates 是所有可用项目模板的注册表。
var ProjectTemplates = []ProjectTemplate{
	{
		ID:          "web-api",
		Description: "HTTP REST API 服务（路由、中间件、配置、优雅关闭）",
		Structure:   "cmd/ internal/ configs/ docker/ .github/ Makefile",
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

// ──────────────────────────────────────────────
// ling-base 模块注册表
// ──────────────────────────────────────────────

// LingBaseModule 描述一个可集成到生成项目中的 ling-base 模块。
type LingBaseModule struct {
	ID          string // 唯一标识
	Name        string // 显示名称
	Description string // 简短说明
	ImportPath  string // Go import 路径
	Core        bool   // 是否核心模块（基础版会询问）
}

// LingBaseModules 是可选的 ling-base 模块列表。
var LingBaseModules = []LingBaseModule{
	{ID: "apidocs", Name: "API 文档", Description: "Huma + OpenAPI 3.1 文档 UI（Scalar/Swagger/Redoc）", ImportPath: "github.com/LingByte/ling-base/apidocs", Core: true},
	{ID: "middleware", Name: "Gin 中间件", Description: "超时 + 熔断 + 恢复 + CORS 等中间件", ImportPath: "github.com/LingByte/ling-base/middleware", Core: true},
	{ID: "jwt", Name: "JWT 鉴权", Description: "Access/Refresh token + 黑名单 + 中间件", ImportPath: "github.com/LingByte/ling-base/common/jwtutil", Core: true},
	{ID: "limiter", Name: "限流", Description: "令牌桶 / 并发数 / 按 key 限流（内存/Redis）", ImportPath: "github.com/LingByte/ling-base/common/limiter", Core: true},
	{ID: "circuitbreaker", Name: "熔断器", Description: "滑动窗口熔断器（Closed/Open/Half-Open）", ImportPath: "github.com/LingByte/ling-base/common/circuitbreaker", Core: true},
	{ID: "response", Name: "统一响应", Description: "统一 JSON 响应封装 + 错误码 + Gin 集成", ImportPath: "github.com/LingByte/ling-base/common/response", Core: true},
	{ID: "cache", Name: "缓存", Description: "统一缓存接口（Redis/Memcache/FreeCache/Ristretto）", ImportPath: "github.com/LingByte/ling-base/common/cache"},
	{ID: "lock", Name: "分布式锁", Description: "Redis/MySQL/PostgreSQL/Etcd/Zookeeper 锁", ImportPath: "github.com/LingByte/ling-base/common/lock"},
	{ID: "retry", Name: "重试", Description: "指数退避/固定间隔重试 + 熔断组合", ImportPath: "github.com/LingByte/ling-base/common/retry"},
	{ID: "scheduler", Name: "分布式定时任务", Description: "分布式锁 + 任务分发（区别于 cron）", ImportPath: "github.com/LingByte/ling-base/common/scheduler"},
	{ID: "eventbus", Name: "事件总线", Description: "进程内事件发布/订阅 + 通配符匹配", ImportPath: "github.com/LingByte/ling-base/common/eventbus"},
	{ID: "i18n", Name: "国际化", Description: "多语言翻译（Gin 中间件 + MyMemory）", ImportPath: "github.com/LingByte/ling-base/common/i18n"},
	{ID: "notification", Name: "通知", Description: "Email/SMS/Push/Webhook/IM 多渠道通知", ImportPath: "github.com/LingByte/ling-base/common/notification"},
	{ID: "mq", Name: "消息队列", Description: "Kafka/RabbitMQ/Redis Stream/RocketMQ 统一接口", ImportPath: "github.com/LingByte/ling-base/mq"},
	{ID: "search", Name: "搜索引擎", Description: "Elasticsearch/Bleve 统一搜索接口", ImportPath: "github.com/LingByte/ling-base/common/search"},
	{ID: "stores", Name: "对象存储", Description: "S3/OSS/COS/MinIO/Kodo 等统一存储接口", ImportPath: "github.com/LingByte/ling-base/stores"},
	{ID: "stats", Name: "统计", Description: "PV/UV/VV/QPS/延迟分位等指标采集", ImportPath: "github.com/LingByte/ling-base/common/stats"},
	{ID: "opentelemetry", Name: "OpenTelemetry", Description: "链路追踪 + Metrics + Logs", ImportPath: "github.com/LingByte/ling-base/common/opentelemetry"},
	{ID: "metrics", Name: "Metrics 指标", Description: "Prometheus 兼容的应用指标采集", ImportPath: "github.com/LingByte/ling-base/common/metrics"},
	{ID: "tracing", Name: "链路追踪", Description: "OpenTracing/OpenTelemetry 兼容的分布式追踪", ImportPath: "github.com/LingByte/ling-base/common/tracing"},
	{ID: "bloom", Name: "布隆过滤器", Description: "内存/Redis/Scalable 布隆过滤器", ImportPath: "github.com/LingByte/ling-base/common/bloom"},
	{ID: "captcha", Name: "验证码", Description: "图形/短信/行为验证码", ImportPath: "github.com/LingByte/ling-base/common/captcha"},
	{ID: "validate", Name: "数据校验", Description: "结构体标签驱动校验（内置规则 + 自定义规则 + 嵌套）", ImportPath: "github.com/LingByte/ling-base/common/validate"},
}

// HasModule 检查 spec 是否选了某个模块。
func (s *ProjectSpec) HasModule(id string) bool {
	for _, m := range s.Modules {
		if m == id {
			return true
		}
	}
	return false
}

// findLingBaseModule 根据 ID 查找模块定义。
func findLingBaseModule(id string) *LingBaseModule {
	for i := range LingBaseModules {
		if LingBaseModules[i].ID == id {
			return &LingBaseModules[i]
		}
	}
	return nil
}
