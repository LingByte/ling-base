// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command ling-agent is a coding agent CLI built on ling-base.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/LingByte/ling-base/agent"
	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/relay"
)

func main() {
	cfg := agent.ParseArgs(os.Args[1:])

	if cfg.Help {
		agent.PrintHelp()
		return
	}

	if cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "ling-agent: no API key found.\n")
		fmt.Fprintf(os.Stderr, "Set %s or pass --api-key.\n", envKeyForProvider(cfg.Provider))
		os.Exit(1)
	}

	// Create provider and relay client.
	provider, model := agent.CreateProvider(agent.ProviderConfig{
		Name:   cfg.Provider,
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
	client := relay.New(relay.WithProvider(provider))

	// Build system prompt.
	system := agent.BuildSystemPrompt(cfg.CWD)

	// Create agent with tools.
	a := agent.New(client, model,
		agent.WithSystem(system),
		agent.WithTools(
			tools.NewRead(cfg.CWD),
			tools.NewWrite(cfg.CWD),
			tools.NewEdit(cfg.CWD),
			tools.NewBash(cfg.CWD),
		),
		agent.WithMaxSteps(cfg.MaxSteps),
		agent.WithAutoApprove(cfg.AutoApprove),
	)

	// Resume session if requested.
	if cfg.ResumeID != "" {
		path := agent.SessionPath(cfg.ResumeID)
		if cfg.SessionPath != "" {
			path = cfg.SessionPath
		}
		sess, err := agent.LoadSession(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ling-agent: failed to load session: %v\n", err)
			os.Exit(1)
		}
		sess.ApplyToAgent(a)
	}

	// Determine prompt source.
	prompt := cfg.Prompt
	if prompt == "" {
		// Check if stdin is piped.
		stat, _ := os.Stdin.Stat()
		if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			// Piped stdin — read from it.
			if err := agent.RunFromStdin(context.Background(), a, cfg.JSON); err != nil {
				fmt.Fprintf(os.Stderr, "ling-agent: %v\n", err)
				os.Exit(1)
			}
			return
		}
		fmt.Fprintf(os.Stderr, "ling-agent: no prompt provided. Use --help for usage.\n")
		os.Exit(1)
	}

	// Set up signal handling for Ctrl+C.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nling-agent: interrupted\n")
		cancel()
	}()

	// Run the agent.
	var err error
	if cfg.JSON {
		err = agent.RunJSONMode(ctx, a, prompt)
	} else {
		err = agent.RunPrintMode(ctx, a, prompt)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ling-agent: %v\n", err)
		os.Exit(1)
	}

	// Save session.
	sess := agent.NewSession(model, system)
	sess.SyncFromAgent(a)
	sessPath := cfg.SessionPath
	if sessPath == "" {
		sessPath = agent.SessionPath(sess.ID)
	}
	if err := sess.Save(sessPath); err != nil {
		fmt.Fprintf(os.Stderr, "ling-agent: warning: failed to save session: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "[session] saved: %s\n", sess.ID)
	}
}

func envKeyForProvider(provider string) string {
	switch provider {
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}
