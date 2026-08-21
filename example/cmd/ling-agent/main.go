// Command ling-agent is a thin example that configures the agent/app runtime
// and starts either the interactive TUI or a headless (-p) run.
package main

import (
	"os"

	"github.com/LingByte/ling-base/agent/app"
	"github.com/LingByte/ling-base/agent/cli"
)

func main() {
	os.Exit(cli.Execute(app.Options{
		Provider:  "openai",
		Model:     "gpt-5.4",
		BaseURL:   "https://rightapi.ai/codex",
		APIKeyEnv: "RIGHTAPI_API_KEY",
	}))
}
