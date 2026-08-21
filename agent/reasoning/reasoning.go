// Package reasoning provides unified reasoning-effort level management
// across different LLM providers. It normalizes user-facing levels
// (minimum, low, medium, high, xhigh, max) and maps them to
// provider-specific enums.
//
// This is adapted from zot's reasoning system, trimmed to the subset
// LingAgent needs. The Model struct here is a lightweight stand-in for
// the full model metadata; callers populate it from their provider's
// model discovery.
package reasoning

import "strings"

// Level is a canonical reasoning effort level.
type Level string

const (
	Off     Level = ""
	Minimum Level = "minimum"
	Low     Level = "low"
	Medium  Level = "medium"
	High    Level = "high"
	XHigh   Level = "xhigh"
	Max     Level = "max"
)

var levelOrder = []string{"", "minimum", "low", "medium", "high", "xhigh", "max"}

// Model carries the model metadata needed to determine available
// reasoning levels and how to map them to provider-specific enums.
type Model struct {
	ID                    string
	Provider              string
	API                   string // "anthropic", "responses", "chat", etc.
	Reasoning             bool   // model supports reasoning/thinking
	AdaptiveThinking      bool   // Anthropic adaptive thinking model
	AdaptiveThinkingCompat bool  // adaptive thinking over OpenAI-compat wire
	ReasoningLevelMap     map[string]string // per-model overrides
}

// AvailableLevels returns the distinct reasoning levels supported by a
// model. Optional per-model overrides can remove, remap, or extend
// protocol defaults. The empty string represents off.
func AvailableLevels(model Model) []string {
	defaults := defaultLevels(model)
	if !model.Reasoning || len(model.ReasoningLevelMap) == 0 {
		return defaults
	}

	available := map[string]bool{"": true}
	for _, level := range levelOrder[1:] {
		if !containsLevel(defaults, level) {
			if _, overridden := model.ReasoningLevelMap[level]; !overridden {
				continue
			}
		}
		effective := level
		if mapped, overridden := model.ReasoningLevelMap[level]; overridden {
			effective = Normalize(mapped)
		}
		if rank(effective) > 0 {
			available[effective] = true
		}
	}

	levels := []string{""}
	for _, level := range levelOrder[1:] {
		if available[level] {
			levels = append(levels, level)
		}
	}
	return levels
}

// Clamp maps a configured level to the nearest level exposed for the
// active model. Ties prefer the higher level.
func Clamp(model Model, level string) string {
	normalized := Normalize(level)
	available := AvailableLevels(model)
	if mapped, overridden := model.ReasoningLevelMap[normalized]; overridden {
		target := Normalize(mapped)
		if target != "" && containsLevel(available, target) {
			return target
		}
	}
	for _, candidate := range available {
		if candidate == normalized {
			return candidate
		}
	}
	return nearestLevel(available, normalized)
}

// Normalize canonicalizes user-facing reasoning levels. Empty string
// means reasoning is disabled. "maximum" remains an alias for xhigh;
// "max" is the separate opt-in tier above it.
func Normalize(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "off", "none", "no", "false", "disabled":
		return ""
	case "min", "minimal", "minimum":
		return "minimum"
	case "low":
		return "low"
	case "med", "medium":
		return "medium"
	case "hi", "high":
		return "high"
	case "xhigh", "maximum":
		return "xhigh"
	case "max":
		return "max"
	default:
		return strings.ToLower(strings.TrimSpace(level))
	}
}

// Budget returns the approximate token budget for reasoning-capable
// providers that accept explicit budgets (e.g. Anthropic thinking
// budget).
func Budget(level string) int {
	switch Normalize(level) {
	case "minimum":
		return 1024
	case "low":
		return 2048
	case "medium":
		return 8192
	case "high":
		return 16384
	case "xhigh", "max":
		return 32768
	default:
		return 0
	}
}

// AnthropicAdaptiveEffort maps levels onto the effort enum used by
// adaptive-thinking models. These models reject explicit thinking
// budgets; reasoning depth is controlled by output_config.effort.
func AnthropicAdaptiveEffort(level string) string {
	switch Normalize(level) {
	case "minimum", "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		return "xhigh"
	case "max":
		return "max"
	default:
		return ""
	}
}

// OpenAIEffort maps levels onto the effort enum accepted by generic
// OpenAI-compatible chat-completions endpoints.
func OpenAIEffort(level string) string {
	switch Normalize(level) {
	case "minimum", "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "xhigh", "max":
		return "high"
	default:
		return ""
	}
}

// ---- internals ----

func defaultLevels(model Model) []string {
	if !model.Reasoning {
		return []string{""}
	}

	id := strings.ToLower(model.ID)
	if (model.Provider == "google" || model.Provider == "google-vertex") && strings.Contains(id, "gemini-3") {
		if strings.Contains(id, "-pro") {
			return []string{"", "low", "high"}
		}
		return []string{"", "minimum", "low", "medium", "high"}
	}
	if model.AdaptiveThinkingCompat {
		return []string{"", "high"}
	}
	if model.AdaptiveThinking {
		return []string{"", "low", "medium", "high", "xhigh", "max"}
	}
	if model.API == "responses" || model.Provider == "openai-codex" || model.Provider == "openai-responses" {
		levels := []string{"", "low", "medium", "high", "xhigh"}
		if strings.HasPrefix(id, "gpt-5.6-") {
			levels = append(levels, "max")
		}
		return levels
	}
	if model.Provider == "google" || model.Provider == "google-vertex" {
		if strings.Contains(id, "gemini-2.5") {
			return []string{"", "minimum", "low", "medium", "high", "xhigh"}
		}
		return []string{""}
	}
	if usesReasoningBudget(model) {
		return []string{"", "minimum", "low", "medium", "high", "xhigh"}
	}
	if model.Provider == "amazon-bedrock" {
		return []string{""}
	}
	return []string{"", "low", "medium", "high"}
}

func containsLevel(levels []string, target string) bool {
	for _, level := range levels {
		if level == target {
			return true
		}
	}
	return false
}

func nearestLevel(available []string, requested string) string {
	if requested == "" || len(available) == 1 {
		return ""
	}
	requestedRank := rank(requested)
	if requestedRank == 0 {
		return ""
	}
	best, bestDistance := available[1], len(levelOrder)
	for _, candidate := range available[1:] {
		distance := rank(candidate) - requestedRank
		if distance < 0 {
			distance = -distance
		}
		if distance <= bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}

func rank(level string) int {
	for r, candidate := range levelOrder {
		if candidate == level {
			return r
		}
	}
	return 0
}

func usesReasoningBudget(model Model) bool {
	switch model.Provider {
	case "anthropic", "fireworks", "kimi", "minimax", "minimax-cn", "vercel-ai-gateway":
		return true
	}
	return model.API == "anthropic"
}
