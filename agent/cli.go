// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/channel/claude"
	"github.com/LingByte/ling-base/relay/channel/openai"
)

// ProviderConfig holds the configuration for creating a relay provider.
type ProviderConfig struct {
	Name   string // openai, claude, deepseek
	APIKey string
	Model  string
}

// DefaultModel returns the default model for a provider.
func DefaultModel(provider string) string {
	switch provider {
	case "claude":
		return "claude-sonnet-4-20250514"
	case "deepseek":
		return "deepseek-chat"
	case "openai":
		return "gpt-4o"
	default:
		return "gpt-4o"
	}
}

// CreateProvider creates a relay Provider from the config.
func CreateProvider(cfg ProviderConfig) (relay.Provider, string) {
	model := cfg.Model
	if model == "" {
		model = DefaultModel(cfg.Name)
	}

	switch cfg.Name {
	case "claude":
		return claude.NewProvider(cfg.APIKey), model
	case "deepseek":
		// DeepSeek is OpenAI-compatible; use openai adaptor with deepseek base URL.
		return openai.NewProvider(cfg.APIKey, openai.WithBaseURL("https://api.deepseek.com")), model
	case "openai":
		return openai.NewProvider(cfg.APIKey), model
	default:
		// Default to OpenAI for unknown providers.
		return openai.NewProvider(cfg.APIKey), model
	}
}

// ResolveAPIKey finds the API key from flag, env, or returns empty.
func ResolveAPIKey(provider string) string {
	envMap := map[string]string{
		"openai":   "OPENAI_API_KEY",
		"claude":   "ANTHROPIC_API_KEY",
		"deepseek": "DEEPSEEK_API_KEY",
	}
	if envVar, ok := envMap[provider]; ok {
		if key := os.Getenv(envVar); key != "" {
			return key
		}
	}
	return ""
}

// PrintHelp prints usage to stdout.
func PrintHelp() {
	fmt.Println(`ling-agent — a coding agent harness built on ling-base.

Usage:
  ling-agent [flags] "your prompt"
  echo "prompt" | ling-agent [flags]

Modes:
  (default)     Print assistant text to stdout, stream tool calls to stderr.
  --json        Emit newline-delimited JSON events to stdout.

Flags:
  --provider    LLM provider: openai|claude|deepseek (default: openai)
  --model       Model name (default: provider-specific)
  --api-key     API key (also: OPENAI_API_KEY, ANTHROPIC_API_KEY, DEEPSEEK_API_KEY)
  --cwd         Working directory (default: current directory)
  --max-steps   Maximum agent loop steps (default: 0 = unlimited)
  --auto-approve  Skip confirmation for all tool calls
  --resume      Resume a session by ID
  --session     Save session to this path (default: auto in ~/.ling-agent/)
  --json        JSON event stream mode
  --help, -h    Show this help

Examples:
  ling-agent "read main.go and explain it"
  ling-agent --provider claude --model claude-sonnet-4-20250514 "fix the bug in utils.go"
  echo "refactor" | ling-agent --json
  ling-agent --resume abc-123 "continue"`)
}

// ParseArgs parses command-line arguments into a config.
type CLIConfig struct {
	Provider    string
	Model       string
	APIKey      string
	CWD         string
	MaxSteps    int
	AutoApprove bool
	ResumeID    string
	SessionPath string
	JSON        bool
	Help        bool
	Prompt      string
}

// ParseArgs parses os.Args[1:] into a CLIConfig.
func ParseArgs(args []string) CLIConfig {
	cfg := CLIConfig{
		Provider: "openai",
		CWD:      "",
	}

	var promptParts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			cfg.Help = true
		case "--json":
			cfg.JSON = true
		case "--auto-approve":
			cfg.AutoApprove = true
		case "--provider":
			i++
			if i < len(args) {
				cfg.Provider = args[i]
			}
		case "--model":
			i++
			if i < len(args) {
				cfg.Model = args[i]
			}
		case "--api-key":
			i++
			if i < len(args) {
				cfg.APIKey = args[i]
			}
		case "--cwd":
			i++
			if i < len(args) {
				cfg.CWD = args[i]
			}
		case "--max-steps":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &cfg.MaxSteps)
			}
		case "--resume":
			i++
			if i < len(args) {
				cfg.ResumeID = args[i]
			}
		case "--session":
			i++
			if i < len(args) {
				cfg.SessionPath = args[i]
			}
		default:
			if !strings.HasPrefix(arg, "--") {
				promptParts = append(promptParts, arg)
			}
		}
	}

	cfg.Prompt = strings.Join(promptParts, " ")

	// Resolve CWD.
	if cfg.CWD == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		cfg.CWD = cwd
	}

	// Resolve API key.
	if cfg.APIKey == "" {
		cfg.APIKey = ResolveAPIKey(cfg.Provider)
	}

	return cfg
}
