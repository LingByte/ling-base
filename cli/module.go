// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"strings"
)

// ModuleType represents the type of module to generate.
type ModuleType string

const (
	TypeCore     ModuleType = "core"     // Core interface module (e.g., cache, mq)
	TypeBackend  ModuleType = "backend"  // Backend implementation (e.g., cache/redis)
	TypeUtil     ModuleType = "util"     // Utility module (e.g., common/hash)
	TypeProvider ModuleType = "provider" // Provider adapter (e.g., notification/email)
)

// ModuleSpec holds the specification for a new module.
type ModuleSpec struct {
	Type        ModuleType // core, backend, util, provider
	Name        string     // module name (e.g., "redis", "mycache")
	Parent      string     // parent module path (e.g., "cache", "common")
	Description string     // human-readable description
	DryRun      bool       // if true, preview without writing files
}

// Validate checks the spec for correctness.
func (s *ModuleSpec) Validate() error {
	if s.Type != TypeCore && s.Type != TypeBackend && s.Type != TypeUtil && s.Type != TypeProvider {
		return fmt.Errorf("invalid module type: %q (must be core, backend, util, or provider)", s.Type)
	}
	if s.Name == "" {
		return fmt.Errorf("module name is required")
	}
	if strings.ContainsAny(s.Name, " /\\") {
		return fmt.Errorf("module name must not contain spaces or slashes")
	}
	if s.Type != TypeCore && s.Parent == "" {
		return fmt.Errorf("parent module is required for %s type", s.Type)
	}
	return nil
}

// ModulePath returns the full module path relative to the repo root.
// e.g., "cache/redis", "common/hash", "mycache".
func (s *ModuleSpec) ModulePath() string {
	if s.Type == TypeCore {
		return s.Name
	}
	return s.Parent + "/" + s.Name
}

// FullModulePath returns the complete Go module path.
// e.g., "github.com/LingByte/ling-base/cache/redis".
func (s *ModuleSpec) FullModulePath() string {
	return "github.com/LingByte/ling-base/" + s.ModulePath()
}

// ParentModulePath returns the parent module's Go path (for backend/util/provider).
func (s *ModuleSpec) ParentModulePath() string {
	if s.Type == TypeCore {
		return ""
	}
	return "github.com/LingByte/ling-base/" + s.Parent
}

// PackageName returns the Go package name (last component of module path).
func (s *ModuleSpec) PackageName() string {
	return s.Name
}

// Summary returns a human-readable summary of the spec.
func (s *ModuleSpec) Summary() string {
	var sb strings.Builder
	typeName := map[ModuleType]string{
		TypeCore:     "核心接口 (core)",
		TypeBackend:  "后端实现 (backend)",
		TypeUtil:     "工具模块 (util)",
		TypeProvider: "Provider 适配器 (provider)",
	}[s.Type]

	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m模块类型:\x1b[0m    %s\n", typeName))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m模块名称:\x1b[0m    %s\n", s.Name))
	if s.Parent != "" {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m父模块:\x1b[0m      %s\n", s.Parent))
	}
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m目录路径:\x1b[0m    %s\n", s.ModulePath()))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;117mGo module:\x1b[0m   %s\n", s.FullModulePath()))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("  \x1b[38;5;117m描述:\x1b[0m        %s\n", s.Description))
	}
	if s.DryRun {
		sb.WriteString("  \x1b[33m模式:\x1b[0m        预览 (dry-run，不写文件)\n")
	}
	return sb.String()
}

// FileToGenerate represents a single file to be generated.
type FileToGenerate struct {
	Path     string // relative to repo root
	Content  string // file content
	Overwrite bool   // whether to overwrite if file exists
}

// Template defines a module template.
type Template struct {
	Type        ModuleType
	Description string
	Generate    func(spec *ModuleSpec) []FileToGenerate
}

// FileList returns a comma-separated list of files this template generates.
func (t *Template) FileList() string {
	spec := &ModuleSpec{
		Type:   t.Type,
		Name:   "example",
		Parent: "parent",
	}
	files := t.Generate(spec)
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return strings.Join(paths, ", ")
}

// Templates is the registry of all available module templates.
var Templates = []Template{
	{Type: TypeCore, Description: "核心接口模块（如 cache, mq, search）", Generate: generateCore},
	{Type: TypeBackend, Description: "后端实现模块（如 cache/redis, mq/rabbitmq）", Generate: generateBackend},
	{Type: TypeUtil, Description: "工具函数模块（如 common/hash, common/convert）", Generate: generateUtil},
	{Type: TypeProvider, Description: "Provider 适配器（如 notification/email, ocr/aliyun）", Generate: generateProvider},
}
