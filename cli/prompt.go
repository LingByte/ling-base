// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompt provides interactive input helpers for the CLI.
type Prompt struct {
	reader *bufio.Reader
}

// NewPrompt creates a new interactive prompt.
func NewPrompt() *Prompt {
	return &Prompt{reader: bufio.NewReader(os.Stdin)}
}

// Input reads a line of text with a prompt. If defaultVal is non-empty,
// it is used when the user presses Enter without typing anything.
func (p *Prompt) Input(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

// Select presents a numbered menu and returns the selected index (0-based).
func (p *Prompt) Select(label string, count int) int {
	for {
		fmt.Printf("  %s (1-%d): ", label, count)
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return 0
		}
		line = strings.TrimSpace(line)
		var idx int
		if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > count {
			fmt.Printf("  Invalid choice. Please enter a number between 1 and %d.\n", count)
			continue
		}
		return idx - 1
	}
}

// Confirm asks a yes/no question.
func (p *Prompt) Confirm(label string) bool {
	for {
		fmt.Printf("  %s [y/N]: ", label)
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return false
		}
		line = strings.ToLower(strings.TrimSpace(line))
		switch line {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("  Please enter y or n.")
		}
	}
}

// MultiSelect presents a list of options and returns the selected indices.
func (p *Prompt) MultiSelect(label string, options []string) []int {
	fmt.Printf("  %s (comma-separated numbers, e.g. 1,3,5, or 'all'):\n", label)
	for i, opt := range options {
		fmt.Printf("    [%d] %s\n", i+1, opt)
	}
	for {
		fmt.Printf("  Select: ")
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return nil
		}
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "all" {
			result := make([]int, len(options))
			for i := range options {
				result[i] = i
			}
			return result
		}
		if line == "" {
			return nil
		}
		var result []int
		parts := strings.Split(line, ",")
		valid := true
		for _, part := range parts {
			var idx int
			if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &idx); err != nil || idx < 1 || idx > len(options) {
				fmt.Printf("  Invalid: %s\n", part)
				valid = false
				break
			}
			result = append(result, idx - 1)
		}
		if valid && len(result) > 0 {
			return result
		}
	}
}
