// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// lingBaseTransitiveReplaces maps LingByte submodules that have broken
// transitive dependencies (referencing unpublished versions). The generator
// adds replace directives for these to force the correct published version.
// This overrides broken version references in published submodules.
var lingBaseTransitiveReplaces = map[string]string{
	// common/crypto@v0.2.1 references common/hash@v0.1.0 (unpublished)
	"github.com/LingByte/ling-base/common/hash": "v0.2.0",
	// common/go.mod references common/constants@v0.1.0 (unpublished, only v0.1.1 exists)
	"github.com/LingByte/ling-base/common/constants": "v0.1.1",
}

// resolveLingBaseImports scans generated Go source files for LingByte imports
// and returns the set of unique submodule paths that need to be fetched.
func resolveLingBaseImports(dir string) []string {
	importSet := make(map[string]bool)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip vendor and .git
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		for _, spec := range file.Imports {
			imp := strings.Trim(spec.Path.Value, "\"")
			// Find the actual owning module, not an ancestor such as common.
			bestMatch := ""
			for modPath := range lingBaseSubmoduleVersions {
				if imp == modPath || strings.HasPrefix(imp, modPath+"/") {
					if len(modPath) > len(bestMatch) {
						bestMatch = modPath
					}
				}
			}
			if bestMatch != "" {
				importSet[bestMatch] = true
			}
		}
		return nil
	})

	result := make([]string, 0, len(importSet))
	for imp := range importSet {
		result = append(result, imp)
	}
	sort.Strings(result)
	return result
}

// goGetLingBaseModules explicitly adds each LingBase submodule as a require
// directive with its published version using `go mod edit -require`.
// This is necessary because `go get` and `go mod tidy` cannot auto-discover
// submodules when the root module path is a prefix — they resolve to the
// root module instead of the submodule.
// It also adds replace directives for known broken transitive dependencies.
func goGetLingBaseModules(dir string, imports []string) error {
	args := []string{"mod", "edit"}
	for _, imp := range imports {
		ver, ok := lingBaseSubmoduleVersions[imp]
		if !ok {
			return fmt.Errorf("未找到模块 %s 的发布版本，请运行 make sync-cli-versions", imp)
		}
		args = append(args, "-require="+imp+"@"+ver)
	}
	modules := make([]string, 0, len(lingBaseTransitiveReplaces))
	for module := range lingBaseTransitiveReplaces {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	for _, module := range modules {
		args = append(args, "-replace="+module+"="+module+"@"+lingBaseTransitiveReplaces[module])
	}
	cmd := exec.Command(findGoBin(), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("写入 ling-base 模块版本失败: %w\n%s", err, output)
	}
	return nil
}
