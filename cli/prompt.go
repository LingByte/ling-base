// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompt 提供交互式输入工具。
type Prompt struct {
	reader *bufio.Reader
}

// NewPrompt 创建交互式输入器。
func NewPrompt() *Prompt {
	return &Prompt{reader: bufio.NewReader(os.Stdin)}
}

// Input 读取一行文本。如果用户直接回车，返回 defaultVal。
func (p *Prompt) Input(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  \x1b[38;5;117m%s\x1b[0m [\x1b[38;5;245m%s\x1b[0m]: ", label, defaultVal)
	} else {
		fmt.Printf("  \x1b[38;5;117m%s\x1b[0m: ", label)
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

// Select 呈现编号菜单，返回选中的索引（0-based）。
func (p *Prompt) Select(label string, count int) int {
	for {
		fmt.Printf("  \x1b[38;5;117m%s\x1b[0m (\x1b[38;5;245m1-%d\x1b[0m): ", label, count)
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return 0
		}
		line = strings.TrimSpace(line)
		var idx int
		if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > count {
			fmt.Printf("  \x1b[31m无效选择，请输入 1 到 %d 之间的数字\x1b[0m\n", count)
			continue
		}
		return idx - 1
	}
}

// Confirm 询问是/否问题。
func (p *Prompt) Confirm(label string) bool {
	for {
		fmt.Printf("  \x1b[38;5;117m%s\x1b[0m [\x1b[38;5;245m回车=否\x1b[0m] (y/n): ", label)
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
			fmt.Println("  \x1b[31m请输入 y 或 n\x1b[0m")
		}
	}
}

// MultiSelect 呈现多选列表，返回选中的索引。
func (p *Prompt) MultiSelect(label string, options []string) []int {
	fmt.Printf("  \x1b[38;5;117m%s\x1b[0m\n", label)
	fmt.Println("  \x1b[38;5;245m（逗号分隔数字，如 1,3,5；输入 all 全选）\x1b[0m")
	for i, opt := range options {
		fmt.Printf("    \x1b[38;5;39m[%d]\x1b[0m %s\n", i+1, opt)
	}
	for {
		fmt.Printf("  请选择: ")
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
				fmt.Printf("  \x1b[31m无效: %s\x1b[0m\n", part)
				valid = false
				break
			}
			result = append(result, idx-1)
		}
		if valid && len(result) > 0 {
			return result
		}
	}
}
