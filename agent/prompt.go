// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultSystemPrompt returns a system prompt for a coding agent
// working in the given directory. It includes environment info
// (OS, arch, working directory) and instructions for using tools.
func DefaultSystemPrompt(cwd string) string {
	var b strings.Builder

	b.WriteString("You are a coding agent running in the user's terminal. ")
	b.WriteString("You help with software engineering tasks: reading code, writing code, ")
	b.WriteString("running commands, and debugging.\n\n")

	b.WriteString("## Environment\n\n")
	b.WriteString(fmt.Sprintf("- OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	b.WriteString(fmt.Sprintf("- Working directory: %s\n", cwd))
	b.WriteString(fmt.Sprintf("- Go version: %s\n", runtime.Version()))

	// List files in CWD (top level only).
	if entries, err := os.ReadDir(cwd); err == nil {
		var files []string
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				name += "/"
			}
			files = append(files, name)
			if len(files) >= 20 {
				break
			}
		}
		if len(files) > 0 {
			b.WriteString(fmt.Sprintf("- Files: %s\n", strings.Join(files, ", ")))
		}
	}

	b.WriteString("\n## Tools\n\n")
	b.WriteString("You have these tools available:\n")
	b.WriteString("- read: Read a file (supports offset/limit for large files).\n")
	b.WriteString("- write: Write content to a file (creates or overwrites).\n")
	b.WriteString("- edit: Replace an exact string in a file (old_string must be unique).\n")
	b.WriteString("- bash: Execute a shell command.\n\n")

	b.WriteString("## Principles\n\n")
	b.WriteString("1. **Read before writing.** Always read a file before editing it to understand the context.\n")
	b.WriteString("2. **Verify after changes.** After writing or editing, run relevant checks (build, test, lint).\n")
	b.WriteString("3. **Don't guess.** Use tools to confirm file contents, command output, and types.\n")
	b.WriteString("4. **Use relative paths.** All file paths should be relative to the working directory.\n")
	b.WriteString("5. **Be precise with edits.** The old_string in edit must match exactly, including whitespace.\n")
	b.WriteString("6. **Explain what you're doing.** Before calling a tool, briefly state your intent.\n")
	b.WriteString("7. **Keep it simple.** Prefer minimal, targeted changes over large rewrites.\n")

	return b.String()
}

// loadAGENTSMD reads an AGENTS.md file from the given directory if it
// exists, returning its content as additional system context.
func loadAGENTSMD(cwd string) string {
	for _, name := range []string{"AGENTS.md", "CLAUDE.md", ".agent"} {
		p := filepath.Join(cwd, name)
		data, err := os.ReadFile(p)
		if err == nil {
			return fmt.Sprintf("\n## Project instructions (%s)\n\n%s", name, string(data))
		}
	}
	return ""
}

// BuildSystemPrompt constructs the full system prompt, combining the
// default coding agent prompt with any AGENTS.md project instructions.
func BuildSystemPrompt(cwd string) string {
	base := DefaultSystemPrompt(cwd)
	if extra := loadAGENTSMD(cwd); extra != "" {
		return base + extra
	}
	return base
}
