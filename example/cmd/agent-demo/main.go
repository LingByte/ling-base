// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command agent-demo is an interactive coding agent built on ling-base/agent.
// It connects to a real LLM API (Qiniu llmapi.qiniu.io, OpenAI-compatible)
// and uses read/write/edit/bash tools to operate on files in the current
// working directory.
//
// Unlike a one-shot CLI, this is a persistent REPL: you type a prompt,
// the agent responds (possibly calling tools across multiple turns),
// then you can type another prompt and the agent continues with full
// conversation context. Type /exit or press Ctrl+D to quit.
//
// Usage:
//
//	cd example
//	go run ./cmd/agent-demo
//	go run ./cmd/agent-demo "read go.mod and tell me what this project uses"
//
// Environment:
//
//	LING_AGENT_API_KEY  — API key (default: built-in demo key)
//	LING_AGENT_BASE_URL — API base URL (default: https://llmapi.qiniu.io)
//	LING_AGENT_MODEL    — model name (default: gpt-5.4-mini)
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/LingByte/ling-base/agent"
	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/channel/openai"
)

const (
	defaultAPIKey  = "sk-c3qxB9P3y1hq9xuiqOduUg"
	defaultBaseURL = "https://llmapi.qiniu.io"
	defaultModel   = "gpt-5.4-mini"
)

func main() {
	apiKey := envOr("LING_AGENT_API_KEY", defaultAPIKey)
	baseURL := envOr("LING_AGENT_BASE_URL", defaultBaseURL)
	model := envOr("LING_AGENT_MODEL", defaultModel)

	jsonMode := false
	cwd, _ := os.Getwd()
	var initialPrompt []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--model":
			i++
			if i < len(args) {
				model = args[i]
			}
		case "--cwd":
			i++
			if i < len(args) {
				cwd = args[i]
			}
		case "--base-url":
			i++
			if i < len(args) {
				baseURL = args[i]
			}
		case "--api-key":
			i++
			if i < len(args) {
				apiKey = args[i]
			}
		case "--help", "-h":
			printHelp()
			return
		default:
			initialPrompt = append(initialPrompt, args[i])
		}
	}

	// ── Create provider + relay client ──────────────────────────
	provider := openai.NewProvider(apiKey, openai.WithBaseURL(baseURL))
	client := relay.New(relay.WithProvider(provider))

	// ── Build system prompt ─────────────────────────────────────
	system := agent.BuildSystemPrompt(cwd)

	// ── Create agent with tools ─────────────────────────────────
	a := agent.New(client, model,
		agent.WithSystem(system),
		agent.WithTools(
			tools.NewRead(cwd),
			tools.NewWrite(cwd),
			tools.NewEdit(cwd),
			tools.NewBash(cwd),
		),
		agent.WithMaxSteps(20),
	)

	// ── Signal handling ─────────────────────────────────────────
	// Ctrl+C cancels the current turn (not the whole session).
	// Ctrl+D or /exit quits.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ── Print header ────────────────────────────────────────────
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "╭─ ling-agent ──────────────────────────────────────────────╮\n")
		fmt.Fprintf(os.Stderr, "│ model:    %-48s│\n", model)
		fmt.Fprintf(os.Stderr, "│ base URL: %-48s│\n", baseURL)
		absCwd, _ := filepath.Abs(cwd)
		fmt.Fprintf(os.Stderr, "│ cwd:      %-48s│\n", truncateStr(absCwd, 48))
		fmt.Fprintf(os.Stderr, "╰──────────────────────────────────────────────────────────╯\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Interactive coding agent. Type your request, or:\n")
		fmt.Fprintf(os.Stderr, "  /exit  — quit    /clear — reset conversation\n")
		fmt.Fprintf(os.Stderr, "  /help  — commands\n\n")
	}

	// ── If initial prompt provided via CLI, run it first ────────
	if len(initialPrompt) > 0 {
		prompt := strings.Join(initialPrompt, " ")
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "┌─ you ─────────────────────────────────────────────────────\n")
			fmt.Fprintf(os.Stderr, "%s\n", prompt)
			fmt.Fprintf(os.Stderr, "└───────────────────────────────────────────────────────────\n\n")
			fmt.Fprintf(os.Stderr, "┌─ agent ───────────────────────────────────────────────────\n")
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-sigCh
			cancel()
		}()
		err := runTurn(ctx, a, prompt, jsonMode)
		cancel()
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "└───────────────────────────────────────────────────────────\n\n")
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "[error] %v\n", err)
		}
	}

	// ── Interactive REPL ────────────────────────────────────────
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB buffer for long pastes

	for {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "❯ ")
		}
		if !scanner.Scan() {
			break // EOF (Ctrl+D)
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// ── Slash commands ──────────────────────────────────────
		switch input {
		case "/exit", "/quit":
			fmt.Fprintf(os.Stderr, "bye.\n")
			return
		case "/clear":
			a.SetMessages(nil)
			fmt.Fprintf(os.Stderr, "[conversation cleared]\n\n")
			continue
		case "/help":
			fmt.Fprintf(os.Stderr, "Commands:\n")
			fmt.Fprintf(os.Stderr, "  /exit   quit the agent\n")
			fmt.Fprintf(os.Stderr, "  /clear  reset conversation history\n")
			fmt.Fprintf(os.Stderr, "  /help   show this message\n")
			fmt.Fprintf(os.Stderr, "  /model  show current model\n")
			fmt.Fprintf(os.Stderr, "  /msgs   show message count\n\n")
			continue
		case "/model":
			fmt.Fprintf(os.Stderr, "current model: %s\n\n", model)
			continue
		case "/msgs":
			msgs := a.Messages()
			fmt.Fprintf(os.Stderr, "message count: %d\n\n", len(msgs))
			continue
		}

		// ── Run agent turn ──────────────────────────────────────
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "┌─ agent ───────────────────────────────────────────────────\n")
		}

		ctx, cancel := context.WithCancel(context.Background())
		// Ctrl+C cancels current turn only.
		go func() {
			select {
			case <-sigCh:
				cancel()
				fmt.Fprintf(os.Stderr, "\n[interrupting current turn...]\n")
			case <-ctx.Done():
			}
		}()

		err := runTurn(ctx, a, input, jsonMode)
		cancel()

		if !jsonMode {
			fmt.Fprintf(os.Stderr, "└───────────────────────────────────────────────────────────\n\n")
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "[error] %v\n\n", err)
			// Don't exit — let the user retry in the REPL.
		}
	}

	// ── Save session on exit ────────────────────────────────────
	sess := agent.NewSession(model, system)
	sess.SyncFromAgent(a)
	sessPath := agent.SessionPath(sess.ID)
	if saveErr := sess.Save(sessPath); saveErr == nil {
		fmt.Fprintf(os.Stderr, "[session] saved: %s\n", sess.ID)
	}
}

// runTurn executes one user prompt through the agent and prints events.
func runTurn(ctx context.Context, a *agent.Agent, prompt string, jsonMode bool) error {
	if jsonMode {
		return agent.RunJSONMode(ctx, a, prompt)
	}
	return agent.RunPrintMode(ctx, a, prompt)
}

func printHelp() {
	fmt.Println(`agent-demo — an interactive coding agent built on ling-base

Usage:
  go run ./cmd/agent-demo [flags] [initial prompt]

  Without a prompt, starts an interactive REPL.
  With a prompt, runs it first then enters the REPL.

Flags:
  --json        Emit newline-delimited JSON events (non-interactive)
  --model       Model name (default: gpt-5.4-mini)
  --cwd         Working directory (default: current)
  --base-url    API base URL (default: https://llmapi.qiniu.io)
  --api-key     API key (default: built-in demo key)
  --help, -h    Show this help

Slash commands (in REPL):
  /exit   quit          /clear  reset conversation
  /model  show model    /msgs   show message count

Environment:
  LING_AGENT_API_KEY   Override API key
  LING_AGENT_BASE_URL  Override base URL
  LING_AGENT_MODEL     Override model

Examples:
  go run ./cmd/agent-demo
  go run ./cmd/agent-demo "read go.mod and list the modules"
  go run ./cmd/agent-demo "create hello.txt" "then read it back"`)
}

func envOr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
