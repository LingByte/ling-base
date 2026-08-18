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
	sb.WriteString(fmt.Sprintf("  Type:        %s\n", s.Type))
	sb.WriteString(fmt.Sprintf("  Name:        %s\n", s.Name))
	if s.Parent != "" {
		sb.WriteString(fmt.Sprintf("  Parent:      %s\n", s.Parent))
	}
	sb.WriteString(fmt.Sprintf("  Path:        %s\n", s.ModulePath()))
	sb.WriteString(fmt.Sprintf("  Go module:   %s\n", s.FullModulePath()))
	if s.Description != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", s.Description))
	}
	if s.DryRun {
		sb.WriteString("  Mode:        dry-run (no files written)\n")
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
	{Type: TypeCore, Description: "Core interface module (e.g., cache, mq, search)", Generate: generateCore},
	{Type: TypeBackend, Description: "Backend implementation (e.g., cache/redis, mq/rabbitmq)", Generate: generateBackend},
	{Type: TypeUtil, Description: "Utility module (e.g., common/hash, common/convert)", Generate: generateUtil},
	{Type: TypeProvider, Description: "Provider adapter (e.g., notification/email, ocr/aliyun)", Generate: generateProvider},
}
