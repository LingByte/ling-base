package main

import (
	"fmt"
	"os"

	"github.com/LingByte/ling-base/agentkit/flowcraft/forge/internal/commands"
)

func main() {
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "forge:", err)
		os.Exit(1)
	}
	if err := commands.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "forge:", err)
		os.Exit(1)
	}
}
