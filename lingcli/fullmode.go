// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// fullExcludeDirs are top-level directories in ling-base that are NOT copied
// into the generated project's pkg/ directory in full mode.
var fullExcludeDirs = map[string]bool{
	"lingcli":  true, // the generator itself
	"docs":     true, // documentation site
	"example":  true, // examples
	"pentest":  true, // penetration testing tools
	".git":     true, // git metadata
	"agentkit": true, // AI agent kit (heavy, separate concern)
}

// fullExcludeSubdirs are specific subdirectory paths to exclude (relative to
// ling-base root). These are typically test-only or build artifacts.
var fullExcludeSubdirs = map[string]bool{}

// copyModuleSource walks the entire ling-base source tree and copies all
// Go source files (excluding _test.go, go.mod, go.sum, .md) into the
// target project's pkg/ directory, rewriting all ling-base import paths
// to local paths.
//
// This ensures that full mode includes ALL ling-base packages — common/*,
// stores/* (all backends), relay/* (all channels), bootstrap, apidocs,
// version, providers/*, voice/*, etc. — so the generated project is fully
// self-contained with zero external LingByte dependencies.
func copyModuleSource(lingBaseRoot, targetPkgDir, modulePath string) error {
	lingBaseImport := "github.com/LingByte/ling-base"

	// Walk the entire ling-base tree
	return filepath.Walk(lingBaseRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}

		// Compute relative path from ling-base root
		relPath, err := filepath.Rel(lingBaseRoot, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}

		// Get top-level directory name
		topDir := relPath
		if idx := strings.Index(relPath, "/"); idx >= 0 {
			topDir = relPath[:idx]
		}

		// Skip excluded top-level directories
		if fullExcludeDirs[topDir] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded subdirectories
		if fullExcludeSubdirs[relPath] && info.IsDir() {
			return filepath.SkipDir
		}

		// Skip the root go.mod / go.sum / go.work (we're merging everything
		// into one module)
		if relPath == "go.mod" || relPath == "go.sum" || relPath == "go.work" {
			return nil
		}

		if info.IsDir() {
			// Create the target directory
			dstDir := filepath.Join(targetPkgDir, relPath)
			return os.MkdirAll(dstDir, 0755)
		}

		// Skip test files
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Skip go.mod / go.sum in subdirectories (each submodule has its own;
		// in full mode everything is one module)
		if info.Name() == "go.mod" || info.Name() == "go.sum" {
			return nil
		}

		// Skip documentation files
		if strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Skip non-essential files (gitignore, etc.)
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		srcFile := path
		dstFile := filepath.Join(targetPkgDir, relPath)

		// Go files: rewrite import paths
		if strings.HasSuffix(info.Name(), ".go") {
			return copyAndRewriteFile(srcFile, dstFile, lingBaseImport, modulePath)
		}

		// Non-Go files (embed resources like .css, .svg, .png, .font, .sql, etc.)
		// Copy as-is, but skip large binary files that aren't needed
		content, err := os.ReadFile(srcFile)
		if err != nil {
			return nil // skip unreadable files
		}
		return os.WriteFile(dstFile, content, 0644)
	})
}

// importFixups maps incorrect import paths found in ling-base source to
// their correct module paths. Some ling-base packages use shorthand import
// paths that only work in the workspace context (via go.work re-exports)
// but break when everything is merged into a single module in full mode.
var importFixups = map[string]string{
	// common/mq subpackages import "github.com/LingByte/ling-base/mq"
	// but the actual module path is common/mq
	"github.com/LingByte/ling-base/mq": "github.com/LingByte/ling-base/common/mq",
}

// copyAndRewriteFile copies a Go file and rewrites ling-base import paths
// to the target project's local pkg/ paths.
func copyAndRewriteFile(srcPath, dstPath, oldImport, newModulePath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Parse the Go file to get import list
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, copy as-is
		return os.WriteFile(dstPath, content, 0644)
	}

	// Rewrite all ling-base imports
	rewritten := string(content)
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(importPath, oldImport) {
			continue
		}
		// Apply fixups for known incorrect import paths
		if fixed, ok := importFixups[importPath]; ok {
			importPath = fixed
		}
		// Replace: github.com/LingByte/ling-base/xxx → github.com/user/project/pkg/xxx
		newPath := strings.Replace(importPath, oldImport, newModulePath+"/pkg", 1)
		// Replace both the fixed and original forms in the source
		rewritten = strings.ReplaceAll(rewritten, `"`+strings.Trim(imp.Path.Value, `"`)+`"`, `"`+newPath+`"`)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(dstPath, []byte(rewritten), 0644)
}

// generateFullMode copies the entire ling-base source tree into pkg/ and
// rewrites all imports. The generated project has zero external LingByte
// dependencies.
func generateFullMode(spec *ProjectSpec, lingBaseRoot, targetDir string) error {
	pkgDir := filepath.Join(targetDir, "pkg")

	fmt.Printf("  \x1b[38;5;245m复制 ling-base 源码到 pkg/...\x1b[0m\n")

	if err := copyModuleSource(lingBaseRoot, pkgDir, spec.Module); err != nil {
		return err
	}

	// Count copied Go files
	count := 0
	filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			count++
		}
		return nil
	})
	fmt.Printf("  \x1b[32m✓ 已复制 %d 个 Go 文件到 pkg/\x1b[0m\n", count)

	return nil
}

// rewriteTemplateImports rewrites import paths in rendered template files.
// In full mode, github.com/LingByte/ling-base/xxx → github.com/user/project/pkg/xxx
func rewriteTemplateImports(content, modulePath string) string {
	oldImport := "github.com/LingByte/ling-base"
	newImport := modulePath + "/pkg"
	return strings.ReplaceAll(content, oldImport, newImport)
}

// isFullMode returns true if the spec is in full mode.
func isFullMode(spec *ProjectSpec) bool {
	return spec.Mode == "full"
}

// collectSubmoduleRequires scans all go.mod files in the ling-base tree
// (excluding the root and excluded dirs) and collects non-LingByte require
// directives. This is needed in full mode because merging all submodules
// into one go.mod loses the per-submodule version constraints for external
// dependencies (e.g. alibabacloud SDK versions).
//
// Returns a map of module path → version string.
func collectSubmoduleRequires(lingBaseRoot string) map[string]string {
	requires := map[string]string{}

	filepath.Walk(lingBaseRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() != "go.mod" {
			return nil
		}

		relPath, _ := filepath.Rel(lingBaseRoot, path)
		relPath = filepath.ToSlash(relPath)
		if relPath == "go.mod" {
			return nil // skip root go.mod
		}

		// Check excluded dirs
		topDir := relPath
		if idx := strings.Index(relPath, "/"); idx >= 0 {
			topDir = relPath[:idx]
		}
		if fullExcludeDirs[topDir] {
			return nil
		}

		// Parse go.mod and collect require directives
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			// Match require lines: "github.com/foo/bar v1.2.3"
			// or indented in require blocks: "	github.com/foo/bar v1.2.3"
			if !strings.HasPrefix(line, "github.com/") &&
				!strings.HasPrefix(line, "gopkg.in/") &&
				!strings.HasPrefix(line, "go.") &&
				!strings.HasPrefix(line, "cloud.google.com/") &&
				!strings.HasPrefix(line, "gorm.io/") {
				continue
			}
			// Skip LingByte imports
			if strings.HasPrefix(line, "github.com/LingByte/") {
				continue
			}
			// Extract module path and version
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			modPath := parts[0]
			modVer := parts[1]
			// Strip comments
			if idx := strings.Index(modVer, "//"); idx >= 0 {
				modVer = strings.TrimSpace(modVer[:idx])
			}
			// Only keep if version looks valid
			if modVer == "" {
				continue
			}
			// Keep pseudo-versions (v0.0.0-20260330155402-...) — they are valid
			// and necessary for modules that don't have tagged releases.
			// When multiple submodules require different versions of the same
			// dependency, keep the highest version (MVS semantics).
			if existing, exists := requires[modPath]; !exists || compareVersions(modVer, existing) > 0 {
				requires[modPath] = modVer
			}
		}
		return nil
	})

	return requires
}

// compareVersions compares two Go module version strings.
// Returns >0 if a > b, 0 if equal, <0 if a < b.
// Handles semver tags (v1.2.3) and pseudo-versions (v0.0.0-20260330155402-...).
func compareVersions(a, b string) int {
	// Simple comparison: for pseudo-versions, compare the timestamp portion
	// For semver, compare major.minor.patch numerically
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	// Pseudo-version: v0.0.0-YYYYMMDDHHMMSS-...
	if strings.HasPrefix(a, "0.0.0-") && strings.HasPrefix(b, "0.0.0-") {
		aTS := a[6:20] // YYYYMMDDHHMMSS
		bTS := b[6:20]
		if len(aTS) >= 14 && len(bTS) >= 14 {
			if aTS > bTS {
				return 1
			} else if aTS < bTS {
				return -1
			}
			return 0
		}
	}

	// Semver: split by . and compare numerically
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			// Strip suffixes like -rc1, +incompatible
			aPart := aParts[i]
			if idx := strings.IndexAny(aPart, "-+"); idx >= 0 {
				aPart = aPart[:idx]
			}
			fmt.Sscanf(aPart, "%d", &aNum)
		}
		if i < len(bParts) {
			bPart := bParts[i]
			if idx := strings.IndexAny(bPart, "-+"); idx >= 0 {
				bPart = bPart[:idx]
			}
			fmt.Sscanf(bPart, "%d", &bNum)
		}
		if aNum > bNum {
			return 1
		} else if aNum < bNum {
			return -1
		}
	}
	return 0
}

// injectSubmoduleRequires adds require directives from submodule go.mod
// files into the generated project's go.mod. This ensures external
// dependency version constraints are preserved in full mode.
// Uses `go get` to download dependencies and update go.sum, then
// `go mod edit` to pin the version as a direct require (prevents
// `go mod tidy` from downgrading it).
func injectSubmoduleRequires(dir string, requires map[string]string) {
	goBin := findGoBin()
	for modPath, modVer := range requires {
		target := fmt.Sprintf("%s@%s", modPath, modVer)
		// go get downloads the module and updates go.sum
		getCmd := exec.Command(goBin, "get", target)
		getCmd.Dir = dir
		getCmd.Stdout = nil
		getCmd.Stderr = nil
		_ = getCmd.Run()
		// go mod edit pins the version as a direct require
		editCmd := exec.Command(goBin, "mod", "edit", "-require="+target)
		editCmd.Dir = dir
		editCmd.Stdout = nil
		editCmd.Stderr = nil
		_ = editCmd.Run()
	}
}

// ast package unused import check (avoid compiler error)
var _ = ast.Print

// findLingBaseRoot locates the ling-base repository root.
// Search order:
//  1. LING_BASE_ROOT environment variable
//  2. Walk up from current working directory
//  3. Walk up from executable location
//  4. git clone to a temp directory (when installed via go install)
func findLingBaseRoot() string {
	// 1. Environment variable
	if root := os.Getenv("LING_BASE_ROOT"); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.work")); err == nil {
			return root
		}
		if _, err := os.Stat(filepath.Join(root, "lingcli")); err == nil {
			return root
		}
	}

	isLingBase := func(dir string) bool {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "lingcli")); err == nil {
				return true
			}
		}
		return false
	}

	// 2. Walk up from current working directory
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 10; i++ {
			if isLingBase(dir) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 3. Walk up from executable location
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		for i := 0; i < 10; i++ {
			if isLingBase(dir) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 4. git clone to temp directory
	return cloneLingBase()
}

// cloneLingBase clones the ling-base repository to a temp directory.
// Returns empty string on failure.
func cloneLingBase() string {
	tmpDir, err := os.MkdirTemp("", "ling-base-src-")
	if err != nil {
		return ""
	}

	cmd := exec.Command("git", "clone", "--depth=1", "--branch=main",
		"https://github.com/LingByte/ling-base.git", tmpDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return ""
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "lingcli")); err != nil {
		os.RemoveAll(tmpDir)
		return ""
	}

	return tmpDir
}
