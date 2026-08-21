// Command ling-agent is the ling-base coding agent. It dispatches to headless
// (-p) or interactive (TUI) mode, powered by ling-base/relay for LLM access.
package main

import (
	"os"

	"github.com/LingByte/ling-base/agent/cli"
)

func main() {
	os.Exit(cli.Execute())
}
