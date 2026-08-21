package app

import (
	"fmt"
	"os"
	"strings"
)

// modelAliases maps short names to full model IDs.
var modelAliases = map[string]string{
	"haiku":  "claude-haiku-4-5",
	"sonnet": "claude-sonnet-4-6",
	"opus":   "claude-opus-4-8",
}

// modelContextWindows is the per-model input-token limit.
var modelContextWindows = map[string]int{
	"claude-opus-4-8":   200_000,
	"claude-opus-4-7":   200_000,
	"claude-sonnet-4-6": 200_000,
	"claude-haiku-4-5":  200_000,
}

// ContextWindow source labels.
const (
	ContextSourceConfig  = "config override"
	ContextSourceModel   = "model default"
	ContextSourceUnknown = "unknown — using compaction fallback"
)

// resolveModel turns a CLI --model value into a model ID. Empty → default.
func resolveModel(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return "claude-sonnet-4-6"
	}
	if full, ok := modelAliases[strings.ToLower(m)]; ok {
		return full
	}
	return m
}

// contextWindow returns the effective input-token limit and a human-readable
// source label.
func contextWindow(model string, override int) (limit int, source string) {
	if override > 0 {
		return override, ContextSourceConfig
	}
	resolved := resolveModel(model)
	if n, ok := modelContextWindows[resolved]; ok {
		return n, ContextSourceModel
	}
	return 0, ContextSourceUnknown
}

// resolveCredential resolves the API key from env vars.
func resolveCredential() (string, error) {
	if k := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); k != "" {
		return k, nil
	}
	if k := strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")); k != "" {
		return k, nil
	}
	return "", fmt.Errorf(`LingAgent needs credentials before it can start.

Choose one:
  1. Anthropic API key:
     export ANTHROPIC_API_KEY="sk-ant-..."
     ling-agent

  2. Set apiKey in ~/.ling-agent/config.toml:
     [provider]
     apiKey = "sk-ant-..."
`)
}

// friendlyError returns a human-readable error message.
func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Common patterns
	if strings.Contains(msg, "429") {
		return "Rate limited or insufficient quota (429). " + msg
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") {
		return "Authentication failed. Check your API key. " + msg
	}
	return msg
}
