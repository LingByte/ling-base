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

// InputInt 读取一个整数。如果用户直接回车，返回 defaultVal。
func (p *Prompt) InputInt(label string, defaultVal int) int {
	if defaultVal != 0 {
		fmt.Printf("  \x1b[38;5;117m%s\x1b[0m [\x1b[38;5;245m%d\x1b[0m]: ", label, defaultVal)
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
	var val int
	if _, err := fmt.Sscanf(line, "%d", &val); err != nil {
		fmt.Printf("  \x1b[31m无效数字，使用默认值 %d\x1b[0m\n", defaultVal)
		return defaultVal
	}
	return val
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
