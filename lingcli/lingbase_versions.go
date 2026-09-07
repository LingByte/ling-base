// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// lingBaseSubmoduleVersions maps LingByte submodule paths to their latest
// published version. This is needed because `go mod tidy` cannot auto-discover
// submodules when the root module path is a prefix — it resolves to the root
// module instead of the submodule.
//
// To update: run `git tag -l --sort=-v:refname` in the ling-base repo and
// extract the latest version for each submodule.
var lingBaseSubmoduleVersions = map[string]string{
	"github.com/LingByte/ling-base/apidocs":                     "v0.4.0",
	"github.com/LingByte/ling-base/bootstrap":                   "v0.1.6",
	"github.com/LingByte/ling-base/common":                      "v0.3.1",
	"github.com/LingByte/ling-base/common/alerting":             "v0.1.0",
	"github.com/LingByte/ling-base/common/authcontext":          "v0.1.0",
	"github.com/LingByte/ling-base/common/authcontext/gin":      "v0.1.0",
	"github.com/LingByte/ling-base/common/batch":                "v0.1.0",
	"github.com/LingByte/ling-base/common/cache":                "v0.1.1",
	"github.com/LingByte/ling-base/common/chunk":                "v0.1.1",
	"github.com/LingByte/ling-base/common/codegen":              "v0.1.0",
	"github.com/LingByte/ling-base/common/configcenter":         "v0.1.0",
	"github.com/LingByte/ling-base/common/circuitbreaker":       "v0.1.0",
	"github.com/LingByte/ling-base/common/constants":            "v0.1.1",
	"github.com/LingByte/ling-base/common/embedder":             "v0.1.0",
	"github.com/LingByte/ling-base/common/extension":            "v0.1.0",
	"github.com/LingByte/ling-base/common/grpc":                 "v0.1.0",
	"github.com/LingByte/ling-base/common/i18n":                 "v0.1.0",
	"github.com/LingByte/ling-base/common/i18n/gin":             "v0.1.0",
	"github.com/LingByte/ling-base/common/idempotency":          "v0.1.0",
	"github.com/LingByte/ling-base/common/jwtutil":              "v0.2.2",
	"github.com/LingByte/ling-base/common/jwtutil/gin":          "v0.2.0",
	"github.com/LingByte/ling-base/common/limiter/tokenbucket":  "v0.1.2",
	"github.com/LingByte/ling-base/common/lock":                 "v0.1.0",
	"github.com/LingByte/ling-base/common/logger":               "v0.1.2",
	"github.com/LingByte/ling-base/common/middleware":           "v0.1.4",
	"github.com/LingByte/ling-base/common/password":             "v0.1.0",
	"github.com/LingByte/ling-base/common/probe":                "v0.1.0",
	"github.com/LingByte/ling-base/common/rbac":                 "v0.1.0",
	"github.com/LingByte/ling-base/common/rbac/gin":             "v0.1.0",
	"github.com/LingByte/ling-base/common/reconciler":           "v0.1.0",
	"github.com/LingByte/ling-base/common/registry":             "v0.1.0",
	"github.com/LingByte/ling-base/common/registry/consul":      "v0.1.0",
	"github.com/LingByte/ling-base/common/response":             "v0.1.1",
	"github.com/LingByte/ling-base/common/response/gin":         "v0.1.1",
	"github.com/LingByte/ling-base/common/retrieve":             "v0.1.0",
	"github.com/LingByte/ling-base/common/retry":                "v0.1.1",
	"github.com/LingByte/ling-base/common/shutdown":             "v0.1.0",
	"github.com/LingByte/ling-base/common/snapshot":             "v0.1.0",
	"github.com/LingByte/ling-base/common/ssh":                  "v0.1.0",
	"github.com/LingByte/ling-base/common/tree":                 "v0.1.0",
	"github.com/LingByte/ling-base/common/tree/gormstore":       "v0.1.0",
	"github.com/LingByte/ling-base/common/tree/memory":          "v0.1.1",
	"github.com/LingByte/ling-base/common/tree/mysql":           "v0.1.0",
	"github.com/LingByte/ling-base/common/tree/postgres":        "v0.1.0",
	"github.com/LingByte/ling-base/common/tree/sqlite":          "v0.1.0",
	"github.com/LingByte/ling-base/common/uaparse":              "v0.1.0",
	"github.com/LingByte/ling-base/common/validate":             "v0.2.1",
	"github.com/LingByte/ling-base/stores":                      "v0.1.5",
	"github.com/LingByte/ling-base/stores/local":                "v0.1.4",
	"github.com/LingByte/ling-base/voice/vad":                   "v0.1.0",
	"github.com/LingByte/ling-base/voice/voiceclone":            "v0.1.0",
	"github.com/LingByte/ling-base/voice/voiceprint":            "v0.1.0",
}

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
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		// Find all LingByte imports
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "\"github.com/LingByte/ling-base") {
				continue
			}
			imp := strings.Trim(line, "\"")
			// Find the longest matching submodule prefix
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
func goGetLingBaseModules(dir string, imports []string) {
	goBin := findGoBin()
	// 1. Add require directives for direct imports
	for _, imp := range imports {
		ver, ok := lingBaseSubmoduleVersions[imp]
		if !ok {
			continue
		}
		target := fmt.Sprintf("%s@%s", imp, ver)
		cmd := exec.Command(goBin, "mod", "edit", "-require="+target)
		cmd.Dir = dir
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()
	}
	// 2. Add replace directives for broken transitive dependencies.
	// These force the correct version even when a published submodule
	// references an unpublished version of another submodule.
	for modPath, ver := range lingBaseTransitiveReplaces {
		target := fmt.Sprintf("%s@%s", modPath, ver)
		cmd := exec.Command(goBin, "mod", "edit", "-replace="+modPath+"="+target)
		cmd.Dir = dir
		cmd.Stdout = nil
		cmd.Stderr = nil
		_ = cmd.Run()
	}
}
