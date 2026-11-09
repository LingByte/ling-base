package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// An underscore directory keeps embedded .go files out of `go list ./...`.
const embeddedSourceRoot = "embed_source/_source"
const embeddedRequiresPath = "embed_source/dependencies.json"

// prepareEmbeddedSource is a repository maintenance command, used by Make/CI.
// Use the same exclusions and directory layout as disk-based full mode.
func prepareEmbeddedSource(root string) error {
	for _, path := range []string{"go.work", "lingcli/go.mod"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			return fmt.Errorf("prepare-embed must run at the ling-base repository root: %w", err)
		}
	}
	tmp, err := os.MkdirTemp(filepath.Join(root, "lingcli"), ".embed-source-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := copyModuleSource(root, filepath.Join(tmp, "_source"), ""); err != nil {
		return err
	}
	dependencies := collectSubmoduleRequires(root)
	requires, err := json.MarshalIndent(dependencies, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "dependencies.json"), append(requires, '\n'), 0644); err != nil {
		return err
	}

	target := filepath.Join(root, "lingcli", "embed_source")
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != ".gitkeep" {
			if err := os.RemoveAll(filepath.Join(target, entry.Name())); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"_source", "dependencies.json"} {
		if err := os.Rename(filepath.Join(tmp, name), filepath.Join(target, name)); err != nil {
			return err
		}
	}
	fmt.Printf("Embedded library source and %d dependency versions prepared.\n", len(dependencies))
	return nil
}
