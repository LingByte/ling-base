// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"time"
)

// ──────────────────────────────────────────────
// web-api 模板
// ──────────────────────────────────────────────

func generateWebAPI(spec *ProjectSpec) []FileEntry {
	mod := spec.Module
	year := time.Now().Year()
	author := spec.Author
	if author == "" {
		author = "Your Name"
	}

	var files []FileEntry

	// cmd/server/main.go
	files = append(files, FileEntry{
		Path: "cmd/server/main.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"%s/internal/config"
	"%s/internal/handler"
	"%s/internal/middleware"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("启动服务", "version", Version, "buildTime", BuildTime, "port", cfg.Server.Port)

	mux := http.NewServeMux()

	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.Recover(h)
	h = middleware.CORS(h)

	apiHandler := handler.New()
	mux.HandleFunc("/health", apiHandler.Health)
	mux.HandleFunc("/api/v1/hello", apiHandler.Hello)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%%d", cfg.Server.Port),
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务异常", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("服务已启动", "addr", server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("服务关闭失败", "error", err)
	}
	slog.Info("服务已停止")
}
`, year, author, mod, mod, mod),
	})

	// internal/config/config.go
	files = append(files, FileEntry{
		Path: "internal/config/config.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `+"`"+`yaml:"server"`+"`"+`
	Logging LoggingConfig `+"`"+`yaml:"logging"`+"`"+`
	App     AppConfig     `+"`"+`yaml:"app"`+"`"+`
}

type ServerConfig struct {
	Port         int           `+"`"+`yaml:"port"`+"`"+`
	ReadTimeout  time.Duration `+"`"+`yaml:"readTimeout"`+"`"+`
	WriteTimeout time.Duration `+"`"+`yaml:"writeTimeout"`+"`"+`
	IdleTimeout  time.Duration `+"`"+`yaml:"idleTimeout"`+"`"+`
}

type LoggingConfig struct {
	Level  string `+"`"+`yaml:"level"`+"`"+`
	Format string `+"`"+`yaml:"format"`+"`"+`
}

type AppConfig struct {
	Name        string `+"`"+`yaml:"name"`+"`"+`
	Environment string `+"`"+`yaml:"environment"`+"`"+`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         %d,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Logging: LoggingConfig{Level: "info", Format: "json"},
		App:     AppConfig{Name: "%s", Environment: "development"},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info("配置文件未找到，使用默认配置", "path", path)
		return Default(), nil
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %%w", err)
	}
	return cfg, nil
}
`, year, author, spec.Port, spec.ProjectName()),
	})

	// internal/handler/handler.go
	files = append(files, FileEntry{
		Path: "internal/handler/handler.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct {
	startTime time.Time
}

func New() *Handler {
	return &Handler{startTime: time.Now()}
}

type Response struct {
	Code    int         `+"`"+`json:"code"`+"`"+`
	Message string      `+"`"+`json:"message"`+"`"+`
	Data    interface{} `+"`"+`json:"data,omitempty"`+"`"+`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data: map[string]interface{}{
			"status":  "healthy",
			"uptime":  time.Since(h.startTime).String(),
			"version": "dev",
		},
	})
}

func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Code: -1, Message: "method not allowed"})
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"message": "Hello, " + name + "!",
			"time":    time.Now().Format(time.RFC3339),
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
`, year, author),
	})

	// internal/middleware/middleware.go
	files = append(files, FileEntry{
		Path: "internal/middleware/middleware.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		slog.Info("HTTP 请求",
			"method", r.Method, "path", r.URL.Path,
			"status", wrapped.status, "duration", time.Since(start).String(),
			"ip", r.RemoteAddr,
		)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "stack", string(debug.Stack()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
`, year, author),
	})

	// pkg/response/response.go
	files = append(files, FileEntry{
		Path: "pkg/response/response.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Code    int         `+"`"+`json:"code"`+"`"+`
	Message string      `+"`"+`json:"message"`+"`"+`
	Data    interface{} `+"`"+`json:"data,omitempty"`+"`"+`
	Error   string      `+"`"+`json:"error,omitempty"`+"`"+`
}

func Success(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{Code: 0, Message: "success", Data: data})
}

func Error(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIResponse{Code: -1, Message: message})
}

func Created(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusCreated, APIResponse{Code: 0, Message: "created", Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
`, year, author),
	})

	// configs/config.yaml
	files = append(files, FileEntry{
		Path: "configs/config.yaml",
		Content: fmt.Sprintf(`# %s 配置文件

server:
  port: %d
  readTimeout: 10s
  writeTimeout: 10s
  idleTimeout: 60s

logging:
  level: info
  format: json

app:
  name: %s
  environment: development
`, spec.ProjectName(), spec.Port, spec.ProjectName()),
	})

	// 公共文件
	files = append(files, FileEntry{Path: ".gitignore", Content: generateGitignore()})
	files = append(files, FileEntry{Path: "Makefile", Content: generateMakefile(spec.ProjectName())})
	files = append(files, FileEntry{Path: "Dockerfile", Content: generateDockerfile(spec.ProjectName(), "cmd/server")})
	files = append(files, FileEntry{Path: "docker-compose.yml", Content: generateDockerCompose(spec.ProjectName(), spec.Port)})
	files = append(files, FileEntry{Path: "README.md", Content: generateREADME(spec, "")})

	return files
}

// ──────────────────────────────────────────────
// grpc-service 模板
// ──────────────────────────────────────────────

func generateGRPCService(spec *ProjectSpec) []FileEntry {
	mod := spec.Module
	year := time.Now().Year()
	author := spec.Author
	if author == "" {
		author = "Your Name"
	}

	var files []FileEntry

	// cmd/server/main.go
	files = append(files, FileEntry{
		Path: "cmd/server/main.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"%s/internal/config"
	"%s/internal/interceptor"
	"%s/internal/service"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	slog.Info("启动 gRPC 服务", "version", Version, "port", cfg.Server.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%%d", cfg.Server.Port))
	if err != nil {
		slog.Error("监听失败", "error", err)
		os.Exit(1)
	}

	srv := service.NewServer()
	srv.UseInterceptor(interceptor.Logging)
	srv.UseInterceptor(interceptor.Recover)

	go func() {
		if err := srv.Serve(lis); err != nil {
			slog.Error("gRPC 服务异常", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("gRPC 服务已启动", "addr", lis.Addr().String())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("正在关闭 gRPC 服务...")
	srv.GracefulStop()
	slog.Info("gRPC 服务已停止")
}
`, year, author, mod, mod, mod),
	})

	// internal/config/config.go
	files = append(files, FileEntry{
		Path: "internal/config/config.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `+"`"+`yaml:"server"`+"`"+`
	Logging LoggingConfig `+"`"+`yaml:"logging"`+"`"+`
	App     AppConfig     `+"`"+`yaml:"app"`+"`"+`
}

type ServerConfig struct {
	Port           int           `+"`"+`yaml:"port"`+"`"+`
	MaxRecvMsgSize int           `+"`"+`yaml:"maxRecvMsgSize"`+"`"+`
	MaxSendMsgSize int           `+"`"+`yaml:"maxSendMsgSize"`+"`"+`
	Timeout        time.Duration `+"`"+`yaml:"timeout"`+"`"+`
}

type LoggingConfig struct {
	Level  string `+"`"+`yaml:"level"`+"`"+`
	Format string `+"`"+`yaml:"format"`+"`"+`
}

type AppConfig struct {
	Name        string `+"`"+`yaml:"name"`+"`"+`
	Environment string `+"`"+`yaml:"environment"`+"`"+`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           %d,
			MaxRecvMsgSize: 4 * 1024 * 1024,
			MaxSendMsgSize: 4 * 1024 * 1024,
			Timeout:        30 * time.Second,
		},
		Logging: LoggingConfig{Level: "info", Format: "json"},
		App:     AppConfig{Name: "%s", Environment: "development"},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info("配置文件未找到，使用默认配置", "path", path)
		return Default(), nil
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %%w", err)
	}
	return cfg, nil
}
`, year, author, spec.Port, spec.ProjectName()),
	})

	// internal/service/service.go
	files = append(files, FileEntry{
		Path: "internal/service/service.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package service

import (
	"context"
	"log/slog"
	"net"
	"sync"
)

// Server 简化的 gRPC 服务容器。
// 实际使用时请替换为 google.golang.org/grpc.Server。
type Server struct {
	mu           sync.Mutex
	listener     net.Listener
	interceptors []Interceptor
	closed       bool
}

type Interceptor func(ctx context.Context, req interface{}, handler Handler) (interface{}, error)
type Handler func(ctx context.Context, req interface{}) (interface{}, error)

func NewServer() *Server {
	return &Server{}
}

func (s *Server) UseInterceptor(i Interceptor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interceptors = append(s.interceptors, i)
}

func (s *Server) Serve(lis net.Listener) error {
	s.listener = lis
	slog.Info("gRPC 服务监听中", "addr", lis.Addr().String())
	for {
		conn, err := lis.Accept()
		if err != nil {
			if s.closed {
				return nil
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	// TODO: 接入 google.golang.org/grpc 处理连接
}

func (s *Server) GracefulStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.listener != nil {
		_ = s.listener.Close()
	}
}
`, year, author),
	})

	// internal/interceptor/interceptor.go
	files = append(files, FileEntry{
		Path: "internal/interceptor/interceptor.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"%s/internal/service"
)

func Logging(ctx context.Context, req interface{}, handler service.Handler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	slog.Info("gRPC 请求", "duration", time.Since(start).String(), "error", err)
	return resp, err
}

func Recover(ctx context.Context, req interface{}, handler service.Handler) (resp interface{}, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("panic recovered", "error", rec, "stack", string(debug.Stack()))
			err = fmt.Errorf("internal error")
		}
	}()
	return handler(ctx, req)
}
`, year, author, mod),
	})

	// api/proto/service.proto
	files = append(files, FileEntry{
		Path: "api/proto/service.proto",
		Content: fmt.Sprintf(`syntax = "proto3";

package %s.v1;

option go_package = "%s/api/proto/v1;%s_v1";

service Greeter {
  rpc SayHello(HelloRequest) returns (HelloResponse);
  rpc HealthCheck(HealthRequest) returns (HealthResponse);
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}

message HealthRequest {}

message HealthResponse {
  string status = 1;
  string version = 2;
}
`, spec.ProjectName(), mod, spec.ProjectName()),
	})

	// configs/config.yaml
	files = append(files, FileEntry{
		Path: "configs/config.yaml",
		Content: fmt.Sprintf(`# %s gRPC 配置

server:
  port: %d
  maxRecvMsgSize: 4194304
  maxSendMsgSize: 4194304
  timeout: 30s

logging:
  level: info
  format: json

app:
  name: %s
  environment: development
`, spec.ProjectName(), spec.Port, spec.ProjectName()),
	})

	// 公共文件
	files = append(files, FileEntry{Path: ".gitignore", Content: generateGitignore()})
	files = append(files, FileEntry{Path: "Makefile", Content: generateMakefile(spec.ProjectName())})
	files = append(files, FileEntry{Path: "Dockerfile", Content: generateDockerfile(spec.ProjectName(), "cmd/server")})
	files = append(files, FileEntry{Path: "docker-compose.yml", Content: generateDockerCompose(spec.ProjectName(), spec.Port)})
	files = append(files, FileEntry{Path: "README.md", Content: generateREADME(spec, "## Proto 生成\n\n```bash\nprotoc --go_out=. --go-grpc_out=. api/proto/service.proto\n```\n")})

	return files
}

// ──────────────────────────────────────────────
// cli-tool 模板
// ──────────────────────────────────────────────

func generateCLITool(spec *ProjectSpec) []FileEntry {
	mod := spec.Module
	year := time.Now().Year()
	author := spec.Author
	if author == "" {
		author = "Your Name"
	}
	pn := spec.PackageName()

	var files []FileEntry

	// cmd/<pn>/main.go
	files = append(files, FileEntry{
		Path: fmt.Sprintf("cmd/%s/main.go", pn),
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"os"

	"%s/internal/command"
	"%s/pkg/version"
)

func main() {
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Usage = printUsage
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Info())
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	cmd := command.New()
	if err := cmd.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[31m错误: %%v\x1b[0m\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`+"`"+`%%s — 命令行工具

用法:
  %%s [命令] [参数]

命令:
  hello       打印问候语
  help        显示帮助

全局参数:
  --version   显示版本信息

示例:
  %%s hello --name World
`+"`"+`, %q, %q, %q)
}
`, year, author, mod, mod, pn, pn, pn),
	})

	// internal/command/command.go
	files = append(files, FileEntry{
		Path: "internal/command/command.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package command

import (
	"flag"
	"fmt"
)

type Command struct {
	name  string
	usage string
	run   func(args []string) error
}

type Commander struct {
	commands map[string]*Command
}

func New() *Commander {
	c := &Commander{commands: make(map[string]*Command)}
	c.register("hello", "打印问候语", c.runHello)
	c.register("help", "显示帮助", c.runHelp)
	return c
}

func (c *Commander) register(name, usage string, run func(args []string) error) {
	c.commands[name] = &Command{name: name, usage: usage, run: run}
}

func (c *Commander) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定子命令")
	}
	name := args[0]
	cmd, ok := c.commands[name]
	if !ok {
		return fmt.Errorf("未知命令: %%s", name)
	}
	return cmd.run(args[1:])
}

func (c *Commander) runHello(args []string) error {
	fs := flag.NewFlagSet("hello", flag.ExitOnError)
	name := fs.String("name", "World", "名称")
	fs.Parse(args)
	fmt.Printf("Hello, %%s! 👋\n", *name)
	return nil
}

func (c *Commander) runHelp(args []string) error {
	fmt.Println("可用命令:")
	for name, cmd := range c.commands {
		fmt.Printf("  %%-10s %%s\n", name, cmd.usage)
	}
	return nil
}
`, year, author),
	})

	// pkg/version/version.go
	files = append(files, FileEntry{
		Path: "pkg/version/version.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package version

import "fmt"

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "none"
)

func Info() string {
	return fmt.Sprintf("%%s %%s (built: %%s, commit: %%s)", "%s", Version, BuildTime, GitCommit)
}
`, year, author, pn),
	})

	// 公共文件
	files = append(files, FileEntry{Path: ".gitignore", Content: generateGitignore()})
	files = append(files, FileEntry{Path: "Makefile", Content: generateMakefile(pn)})
	files = append(files, FileEntry{Path: "README.md", Content: generateREADME(spec, "")})

	return files
}

// ──────────────────────────────────────────────
// library 模板
// ──────────────────────────────────────────────

func generateLibrary(spec *ProjectSpec) []FileEntry {
	mod := spec.Module
	year := time.Now().Year()
	author := spec.Author
	if author == "" {
		author = "Your Name"
	}
	pn := spec.PackageName()

	var files []FileEntry

	// pkg/<pn>/<pn>.go
	files = append(files, FileEntry{
		Path: fmt.Sprintf("pkg/%s/%s.go", pn, pn),
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

// Package %s 提供示例功能。
// 请根据实际需求修改此文件。
package %s

// DoSomething 示例函数。
func DoSomething(input string) string {
	return input
}

// Options 配置选项。
type Options struct {
	Verbose bool
}

// Option 函数式选项。
type Option func(*Options)

// WithVerbose 设置详细模式。
func WithVerbose(v bool) Option {
	return func(o *Options) { o.Verbose = v }
}

// ApplyOptions 应用选项。
func ApplyOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}
`, year, author, pn, pn),
	})

	// pkg/<pn>/<pn>_test.go
	files = append(files, FileEntry{
		Path: fmt.Sprintf("pkg/%s/%s_test.go", pn, pn),
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package %s

import "testing"

func TestDoSomething(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"普通", "hello", "hello"},
		{"空", "", ""},
		{"中文", "你好", "你好"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DoSomething(tt.input); got != tt.want {
				t.Errorf("DoSomething(%%q) = %%q, want %%q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyOptions(t *testing.T) {
	opts := ApplyOptions(WithVerbose(true))
	if !opts.Verbose {
		t.Error("Verbose 应为 true")
	}
	opts = ApplyOptions()
	if opts.Verbose {
		t.Error("默认 Verbose 应为 false")
	}
}
`, year, author, pn),
	})

	// examples/basic/main.go
	files = append(files, FileEntry{
		Path: "examples/basic/main.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package main

import (
	"fmt"

	"%s/pkg/%s"
)

func main() {
	result := %s.DoSomething("Hello, World!")
	fmt.Println(result)
}
`, year, author, mod, pn, pn),
	})

	// LICENSE
	files = append(files, FileEntry{
		Path: "LICENSE",
		Content: fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`, year, author),
	})

	// 公共文件
	files = append(files, FileEntry{Path: ".gitignore", Content: generateGitignore()})
	files = append(files, FileEntry{Path: "README.md", Content: generateREADME(spec, "")})

	return files
}

// ──────────────────────────────────────────────
// worker 模板
// ──────────────────────────────────────────────

func generateWorker(spec *ProjectSpec) []FileEntry {
	mod := spec.Module
	year := time.Now().Year()
	author := spec.Author
	if author == "" {
		author = "Your Name"
	}

	var files []FileEntry

	// cmd/worker/main.go
	files = append(files, FileEntry{
		Path: "cmd/worker/main.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"%s/internal/config"
	"%s/internal/scheduler"
	"%s/internal/task"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	slog.Info("启动 Worker", "version", Version, "app", cfg.App.Name)

	sched := scheduler.New()
	sched.Register("示例任务", task.SampleTask, cfg.Tasks.SampleInterval)
	sched.Register("健康检查", task.HealthCheck, cfg.Tasks.HealthInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		slog.Error("启动调度器失败", "error", err)
		os.Exit(1)
	}

	slog.Info("Worker 已启动，等待中断信号...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("正在关闭 Worker...")
	cancel()
	sched.Stop()
	slog.Info("Worker 已停止")
}
`, year, author, mod, mod, mod),
	})

	// internal/config/config.go
	files = append(files, FileEntry{
		Path: "internal/config/config.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App   AppConfig   `+"`"+`yaml:"app"`+"`"+`
	Tasks TasksConfig `+"`"+`yaml:"tasks"`+"`"+`
}

type AppConfig struct {
	Name        string `+"`"+`yaml:"name"`+"`"+`
	Environment string `+"`"+`yaml:"environment"`+"`"+`
}

type TasksConfig struct {
	SampleInterval time.Duration `+"`"+`yaml:"sampleInterval"`+"`"+`
	HealthInterval time.Duration `+"`"+`yaml:"healthInterval"`+"`"+`
}

func Default() *Config {
	return &Config{
		App:   AppConfig{Name: "%s", Environment: "development"},
		Tasks: TasksConfig{SampleInterval: 30 * time.Second, HealthInterval: 60 * time.Second},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info("配置文件未找到，使用默认配置", "path", path)
		return Default(), nil
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %%w", err)
	}
	return cfg, nil
}
`, year, author, spec.ProjectName()),
	})

	// internal/task/task.go
	files = append(files, FileEntry{
		Path: "internal/task/task.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package task

import (
	"context"
	"log/slog"
	"time"
)

type TaskFunc func(ctx context.Context) error

func SampleTask(ctx context.Context) error {
	slog.Info("执行示例任务", "time", time.Now().Format(time.RFC3339))
	return nil
}

func HealthCheck(ctx context.Context) error {
	slog.Info("健康检查", "status", "ok", "time", time.Now().Format(time.RFC3339))
	return nil
}
`, year, author),
	})

	// internal/scheduler/scheduler.go
	files = append(files, FileEntry{
		Path: "internal/scheduler/scheduler.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"%s/internal/task"
)

type Scheduler struct {
	mu      sync.Mutex
	tasks   []*scheduledTask
	cancel  context.CancelFunc
	running bool
}

type scheduledTask struct {
	name     string
	fn       task.TaskFunc
	interval time.Duration
}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Register(name string, fn task.TaskFunc, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, &scheduledTask{name: name, fn: fn, interval: interval})
	slog.Info("注册任务", "name", name, "interval", interval.String())
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	for _, t := range s.tasks {
		t := t
		go s.runTask(ctx, t)
	}
	return nil
}

func (s *Scheduler) runTask(ctx context.Context, t *scheduledTask) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	slog.Info("任务已启动", "name", t.name)
	for {
		select {
		case <-ctx.Done():
			slog.Info("任务已停止", "name", t.name)
			return
		case <-ticker.C:
			if err := t.fn(ctx); err != nil {
				slog.Error("任务执行失败", "name", t.name, "error", err)
			}
		}
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}
`, year, author, mod),
	})

	// pkg/logger/logger.go
	files = append(files, FileEntry{
		Path: "pkg/logger/logger.go",
		Content: fmt.Sprintf(`// Copyright (c) %d %s. All rights reserved.

package logger

import (
	"log/slog"
	"os"
)

func Init(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
`, year, author),
	})

	// configs/config.yaml
	files = append(files, FileEntry{
		Path: "configs/config.yaml",
		Content: fmt.Sprintf(`# %s Worker 配置

app:
  name: %s
  environment: development

tasks:
  sampleInterval: 30s
  healthInterval: 60s
`, spec.ProjectName(), spec.ProjectName()),
	})

	// 公共文件
	files = append(files, FileEntry{Path: ".gitignore", Content: generateGitignore()})
	files = append(files, FileEntry{Path: "Makefile", Content: generateMakefile(spec.ProjectName())})
	files = append(files, FileEntry{Path: "Dockerfile", Content: generateDockerfile(spec.ProjectName(), "cmd/worker")})
	files = append(files, FileEntry{Path: "docker-compose.yml", Content: generateDockerCompose(spec.ProjectName(), spec.Port)})
	files = append(files, FileEntry{Path: "README.md", Content: generateREADME(spec, "")})

	return files
}
