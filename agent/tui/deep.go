package tui

import "github.com/LingByte/ling-base/agent/prompt"

// deepThinkingPrompt wraps a user question in the deep-thinking template.
// Delegates to prompt.DeepThinkingPrompt so the template is shared between
// the TUI /deep command and the CLI --deep flag.
func deepThinkingPrompt(question string) string {
	return prompt.DeepThinkingPrompt(question)
}
