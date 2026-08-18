// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the ling-base CLI scaffolding tool.
//
// It provides an interactive command-line interface for generating new
// module boilerplate code following the ling-base conventions.
//
// Usage:
//
//	# Interactive mode (recommended)
//	go run ./cli
//
//	# Non-interactive mode
//	go run ./cli new --type core --name mycache
//	go run ./cli new --type backend --name redis --parent cache
//	go run ./cli new --type util --name strutil --parent common
//	go run ./cli new --type provider --name aliyun --parent notification/sms
//
//	# List available module templates
//	go run ./cli list
package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	banner = `
 __    _  __  ____  __     ___   ___   _      __   __  ____   __
( ( )  / )(  (  _ \(  )   / __) / __) / \    /  \ (  )(  _ \ /  \
 )  \  ) (   ) __/ / (_  ( (__ ( (__  / _ \  () () ))(  )   /( (_/
(_)\_)\_/  (__)   \____/ \___) \___)(_/ \_)\__/(__)( (__\_) \__/

  Module Scaffolding Tool — generate boilerplate code for ling-base modules
`
)

func main() {
	if len(os.Args) < 2 {
		runInteractive()
		return
	}

	switch os.Args[1] {
	case "new":
		runNew(os.Args[2:])
	case "list":
		runList()
	case "version":
		fmt.Println("ling-base/cli v0.1.0")
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(banner)
	fmt.Println("Usage:")
	fmt.Println("  cli                    Interactive mode (recommended)")
	fmt.Println("  cli new [flags]        Generate a new module")
	fmt.Println("  cli list               List available module templates")
	fmt.Println("  cli version            Print version")
	fmt.Println("  cli help               Show this help")
	fmt.Println()
	fmt.Println("Flags for 'new':")
	fmt.Println("  --type <type>        Module type: core, backend, util, provider (required)")
	fmt.Println("  --name <name>        Module name (required)")
	fmt.Println("  --parent <parent>    Parent module path (for backend/util/provider)")
	fmt.Println("  --description <desc>  Module description")
	fmt.Println("  --dry-run            Preview without writing files")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cli new --type core --name mycache --description \"My cache abstraction\"")
	fmt.Println("  cli new --type backend --name redis --parent cache")
	fmt.Println("  cli new --type util --name strutil --parent common")
	fmt.Println("  cli new --type provider --name aliyun --parent notification/sms")
}

// runNew handles the non-interactive "new" subcommand.
func runNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	moduleType := fs.String("type", "", "module type: core, backend, util, provider")
	name := fs.String("name", "", "module name")
	parent := fs.String("parent", "", "parent module path (for backend/util/provider)")
	desc := fs.String("description", "", "module description")
	dryRun := fs.Bool("dry-run", false, "preview without writing files")
	fs.Parse(args)

	if *moduleType == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "Error: --type and --name are required")
		fs.Usage()
		os.Exit(1)
	}

	spec := ModuleSpec{
		Type:        ModuleType(*moduleType),
		Name:        *name,
		Parent:      *parent,
		Description: *desc,
		DryRun:      *dryRun,
	}

	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	gen := NewGenerator()
	if err := gen.Generate(&spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runList prints all available module templates.
func runList() {
	fmt.Print(banner)
	fmt.Println("Available module templates:")
	fmt.Println()
	for _, t := range Templates {
		fmt.Printf("  %-10s  %s\n", t.Type, t.Description)
		fmt.Printf("             Files: %s\n", t.FileList())
		fmt.Println()
	}
}

// runInteractive starts the interactive menu.
func runInteractive() {
	fmt.Print(banner)

	p := NewPrompt()

	// Step 1: Select module type.
	fmt.Println("=== Step 1: Select module type ===")
	for i, t := range Templates {
		fmt.Printf("  [%d] %-10s — %s\n", i+1, t.Type, t.Description)
	}
	idx := p.Select("Choose a module type", len(Templates))
	selected := Templates[idx]

	fmt.Printf("\n  Selected: %s\n\n", selected.Type)

	// Step 2: Enter module name.
	fmt.Println("=== Step 2: Module name ===")
	name := p.Input("Module name (e.g. mycache, redis, strutil)", "")

	// Step 3: Enter parent (for non-core modules).
	parent := ""
	if selected.Type != "core" {
		fmt.Println("\n=== Step 3: Parent module ===")
		fmt.Println("  Examples: cache, mq, notification, common, notification/sms")
		parent = p.Input("Parent module path", "")
	}

	// Step 4: Description.
	fmt.Println("\n=== Step 4: Description ===")
	desc := p.Input("Module description (optional)", "")

	// Step 5: Confirm.
	spec := ModuleSpec{
		Type:        selected.Type,
		Name:        name,
		Parent:      parent,
		Description: desc,
	}

	fmt.Println("\n=== Confirmation ===")
	fmt.Println(spec.Summary())

	if !p.Confirm("Generate this module?") {
		fmt.Println("Aborted.")
		return
	}

	gen := NewGenerator()
	if err := gen.Generate(&spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
