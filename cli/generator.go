// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Generator creates module files on disk and registers the module in go.work.
type Generator struct {
	rootDir string
}

// NewGenerator creates a Generator rooted at the repository root.
// It auto-detects the repo root by searching for go.work.
func NewGenerator() *Generator {
	root, err := findRepoRoot()
	if err != nil {
		root = "."
	}
	return &Generator{rootDir: root}
}

// Generate creates all files for the given module spec.
func (g *Generator) Generate(spec *ModuleSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	// Find the template for this module type.
	var template *Template
	for i := range Templates {
		if Templates[i].Type == spec.Type {
			template = &Templates[i]
			break
		}
	}
	if template == nil {
		return fmt.Errorf("no template found for type: %s", spec.Type)
	}

	// Generate files.
	files := template.Generate(spec)

	fmt.Printf("\n=== Generating module: %s ===\n", spec.ModulePath())
	fmt.Println(spec.Summary())
	fmt.Println()

	for _, f := range files {
		fullPath := filepath.Join(g.rootDir, f.Path)

		// Check if file exists.
		if !f.Overwrite {
			if _, err := os.Stat(fullPath); err == nil {
				fmt.Printf("  [SKIP] %s (already exists)\n", f.Path)
				continue
			}
		}

		if spec.DryRun {
			fmt.Printf("  [DRY]  %s (%d bytes)\n", f.Path, len(f.Content))
			continue
		}

		// Create directory.
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}

		// Write file.
		if err := os.WriteFile(fullPath, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}

		fmt.Printf("  [OK]   %s (%d bytes)\n", f.Path, len(f.Content))
	}

	// Register in go.work.
	if !spec.DryRun {
		if err := g.registerInGoWork(spec); err != nil {
			fmt.Printf("  [WARN] could not register in go.work: %v\n", err)
		} else {
			fmt.Printf("  [OK]   go.work (added ./%s)\n", spec.ModulePath())
		}
	}

	fmt.Println()
	if spec.DryRun {
		fmt.Println("Dry run complete. No files were written.")
	} else {
		fmt.Printf("Module created at: %s\n", filepath.Join(g.rootDir, spec.ModulePath()))
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", spec.ModulePath())
		fmt.Println("  go mod tidy")
		fmt.Println("  go test ./...")
		fmt.Println()
		fmt.Println("  # Register in go.work (if not done automatically):")
		fmt.Printf("  # Add './%s' to go.work\n", spec.ModulePath())
	}

	return nil
}

// registerInGoWork adds the new module to the go.work file.
func (g *Generator) registerInGoWork(spec *ModuleSpec) error {
	workPath := filepath.Join(g.rootDir, "go.work")
	content, err := os.ReadFile(workPath)
	if err != nil {
		return fmt.Errorf("read go.work: %w", err)
	}

	entry := "./" + spec.ModulePath()

	// Check if already registered.
	if strings.Contains(string(content), entry) {
		return nil // Already registered.
	}

	// Find the "use (" block and insert the entry.
	lines := strings.Split(string(content), "\n")
	var result []string
	inserted := false

	inUseBlock := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "use (" {
			inUseBlock = true
			result = append(result, line)
			continue
		}
		if inUseBlock && strings.TrimSpace(line) == ")" {
			// Insert before closing paren, maintaining alphabetical order.
			// Collect all existing entries.
			var entries []string
			for _, l := range result {
				t := strings.TrimSpace(l)
				if strings.HasPrefix(t, "./") {
					entries = append(entries, t)
				}
			}
			entries = append(entries, entry)
			sort.Strings(entries)

			// Rebuild the use block.
			var newResult []string
			for _, l := range result {
				t := strings.TrimSpace(l)
				if strings.HasPrefix(t, "./") {
					continue // Skip old entries, we'll re-add sorted.
				}
				newResult = append(newResult, l)
			}
			// Find position of "use (" in newResult.
			useIdx := -1
			for i, l := range newResult {
				if strings.TrimSpace(l) == "use (" {
					useIdx = i
					break
				}
			}
			// Insert entries after "use (".
			finalResult := append([]string{}, newResult[:useIdx+1]...)
			finalResult = append(finalResult, entries...)
			finalResult = append(finalResult, newResult[useIdx+1:]...)
			finalResult = append(finalResult, line) // The closing ")"

			result = finalResult
			inUseBlock = false
			inserted = true
			continue
		}
		result = append(result, line)
	}

	if !inserted {
		// No "use (" block found, try to add one.
		result = append(result, "")
		result = append(result, "use (")
		result = append(result, entry)
		result = append(result, ")")
	}

	newContent := strings.Join(result, "\n")
	return os.WriteFile(workPath, []byte(newContent), 0644)
}

// findRepoRoot finds the repository root by looking for go.work.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.work not found")
		}
		dir = parent
	}
}

// sanitizeName ensures a module name is a valid Go identifier.
func sanitizeName(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(name, "_")
}
