// Package cli is the cobra command-line surface for ling-agent.
// It parses flags into [app.Options] and delegates to the [app] runtime.
//
// Prefer importing [app] directly when embedding ling-agent in your own
// binary; use this package when you want the stock flag set.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/LingByte/ling-base/agent/app"
	"github.com/LingByte/ling-base/version"
)

// NewRootCommand builds the top-level `ling-agent` command.
// Optional defaults are applied before flag parsing (flags still win).
func NewRootCommand(defaults ...app.Options) *cobra.Command {
	var opts app.Options
	if len(defaults) > 0 {
		opts = defaults[0]
	}

	cmd := &cobra.Command{
		Use:           "ling-agent [prompt]",
		Short:         "LingAgent — a coding agent powered by ling-base/relay",
		Version:       fmt.Sprintf("%s (LingAgent)", version.Version),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Prompt == "" && len(args) > 0 {
				opts.Prompt = args[0]
				opts.Print = true
			}
			opts.Stdin = cmd.InOrStdin()
			opts.Stdout = cmd.OutOrStdout()
			opts.Stderr = cmd.ErrOrStderr()
			return app.Run(cmd.Context(), opts)
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	f := cmd.Flags()
	f.BoolVarP(&opts.Print, "print", "p", opts.Print, "Non-interactive mode: print result to stdout and exit")
	f.StringVar(&opts.Model, "model", opts.Model, "Model alias (haiku|sonnet|opus) or full model ID")
	outFmt := opts.OutputFormat
	if outFmt == "" {
		outFmt = "text"
	}
	inFmt := opts.InputFormat
	if inFmt == "" {
		inFmt = "text"
	}
	f.StringVar(&opts.OutputFormat, "output-format", outFmt, "Output format: text|json|stream-json")
	f.StringVar(&opts.InputFormat, "input-format", inFmt, "Input format: text|stream-json (stream-json drives a persistent agent over stdin)")
	f.StringVar(&opts.PermissionMode, "permission-mode", opts.PermissionMode, "Permission mode: default|acceptEdits|bypassPermissions|plan|dontAsk (default: config [permissions] mode, else \"default\")")
	f.BoolVar(&opts.DangerouslySkip, "dangerously-skip-permissions", opts.DangerouslySkip, "Skip all permission checks (sets bypassPermissions)")
	f.BoolVar(&opts.Verbose, "verbose", opts.Verbose, "Verbose output (required for stream-json)")
	f.IntVar(&opts.MaxTurns, "max-turns", opts.MaxTurns, "Limit the number of agentic loop turns (0 = unlimited)")
	f.StringVarP(&opts.Resume, "resume", "r", opts.Resume, "Resume a session by ID")
	f.BoolVar(&opts.ContinueSession, "continue", opts.ContinueSession, "Resume the most recent session in this directory (default when available)")
	f.BoolVar(&opts.NewSession, "new-session", opts.NewSession, "Start a fresh session instead of auto-resuming the most recent session in this directory")
	f.BoolVar(&opts.ForkSession, "fork-session", opts.ForkSession, "When resuming, start a new session ID (preserves the original)")
	f.BoolVar(&opts.FullResume, "full", opts.FullResume, "When resuming, replay the entire transcript instead of the compacted summary")
	f.StringSliceVar(&opts.AllowedTools, "allowedTools", opts.AllowedTools, "Auto-allow tool rules, e.g. 'Edit' or 'Bash(git status:*)' (repeatable, comma-separated)")
	f.StringSliceVar(&opts.DisallowedTools, "disallowedTools", opts.DisallowedTools, "Deny tool rules (same format as --allowedTools)")
	f.BoolVar(&opts.PartialMessages, "include-partial-messages", opts.PartialMessages, "Include partial message chunks as they arrive (only with --print and --output-format=stream-json)")
	f.StringVar(&opts.CreateConfig, "create-config", opts.CreateConfig, "Create a starter TOML config and exit: global (~/.ling-agent/config.toml) or local (./.ling-agent/config.toml)")
	f.BoolVar(&opts.Loop, "loop", opts.Loop, "Autonomous loop: iterate against the goal spec (PRD.md or .ling-agent/GOAL.md) until complete or --max-iterations. Requires --dangerously-skip-permissions.")
	f.IntVar(&opts.MaxIterations, "max-iterations", opts.MaxIterations, "Max iterations for --loop (0 = default 10, hard cap 50)")
	f.BoolVar(&opts.Deep, "deep", opts.Deep, "Wrap the prompt in a deep-thinking template (bidirectional steel-man argumentation)")

	return cmd
}

// Execute runs the stock CLI. Optional defaults seed [app.Options] before flags
// (provider/baseURL/apiKeyEnv are typical). Returns a process exit code.
func Execute(defaults ...app.Options) int {
	if err := NewRootCommand(defaults...).ExecuteContext(context.Background()); err != nil {
		if !errors.Is(err, app.ErrRendered) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return 1
	}
	return 0
}
