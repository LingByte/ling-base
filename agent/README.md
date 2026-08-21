# agent

A coding agent harness built on top of [`ling-base/relay`](../relay). It provides
an agent loop that drives an LLM to use tools (read, write, edit, bash) for
software engineering tasks — similar to Claude Code, Codex CLI, or zot, but as
a lightweight library with no TUI framework dependency.

## Features

- **Agent loop** — sends messages to an LLM, executes tool calls, feeds results
  back, and loops until the task is done.
- **Built-in tools** — `read`, `write`, `edit`, `bash` with CWD confinement.
- **Provider-neutral** — works with any provider supported by `relay/` (OpenAI,
  Claude, DeepSeek, and 35+ more channels).
- **Session persistence** — save/load conversations as JSON files.
- **Two output modes** — `print` (human-readable) and `json` (NDJSON event
  stream for programmatic use).
- **Event system** — every step of the loop emits typed events for UIs, logs,
  or extensions to consume.
- **Zero TUI dependency** — pure stdlib + relay, suitable for embedding in
  other tools or running in CI.

## Module structure

```
agent/
├── tool.go              # Tool interface + Registry + ToolCall
├── events.go            # Event types (8 variants)
├── agent.go             # Agent struct + agent loop
├── prompt.go            # System prompt builder (+ AGENTS.md loading)
├── session.go           # Session save/load/resume
├── cli.go               # CLI config + provider factory
├── mode.go              # print mode + json mode
├── agent_test.go        # Core tests
├── tools/
│   ├── sandbox.go       # Path resolution + CWD confinement
│   ├── file.go          # read / write / edit tools
│   ├── bash.go          # bash tool (cross-platform + timeout)
│   └── tools_test.go    # Tool tests
└── cmd/ling-agent/
    └── main.go          # CLI entry point
```

## Quick start

### As a library

```go
import (
    "github.com/LingByte/ling-base/agent"
    "github.com/LingByte/ling-base/agent/tools"
    "github.com/LingByte/ling-base/relay"
    "github.com/LingByte/ling-base/relay/channel/openai"
)

func main() {
    provider := openai.NewProvider("sk-xxx")
    client := relay.New(relay.WithProvider(provider))

    a := agent.New(client, "gpt-4o",
        agent.WithSystem(agent.BuildSystemPrompt(".")),
        agent.WithTools(
            tools.NewRead("."),
            tools.NewWrite("."),
            tools.NewEdit("."),
            tools.NewBash("."),
        ),
        agent.WithMaxSteps(20),
    )

    err := a.Prompt(ctx, "read main.go and explain it", func(ev agent.Event) {
        switch e := ev.(type) {
        case agent.EvAssistantText:
            fmt.Print(e.Text)
        case agent.EvToolCallStart:
            fmt.Fprintf(os.Stderr, "[tool] %s\n", e.Name)
        }
    })
}
```

### As a CLI

```bash
# Build
go build -o ling-agent ./cmd/ling-agent

# One-shot
export OPENAI_API_KEY=sk-xxx
./ling-agent "read main.go and explain it"

# Interactive REPL
./ling-agent

# JSON event stream
echo "list files" | ./ling-agent --json

# Resume a saved session
./ling-agent --resume <session-id> "continue"
```

### Demo (with built-in API key)

```bash
cd example
go run ./cmd/agent-demo
```

## Tools

| Tool | Description | Safety |
|------|-------------|--------|
| `read` | Read file contents (supports offset/limit). Directories return listing. | Confined to CWD |
| `write` | Create or overwrite a file. Creates parent dirs. | Confined to CWD |
| `edit` | Exact string replacement. `old_string` must be unique. | Confined to CWD |
| `bash` | Execute shell command with timeout. Cross-platform (bash/cmd). | Timeout + CWD |

### Custom tools

Implement the `Tool` interface and register it:

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "Does something custom" }
func (t *MyTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object",...}`) }
func (t *MyTool) Execute(ctx context.Context, args json.RawMessage, progress func(string)) (agent.ToolResult, error) {
    // ...
    return agent.ToolResult{Content: "result"}, nil
}

a := agent.New(client, model, agent.WithTools(&MyTool{}))
```

## Events

The agent loop emits typed events via the `sink` callback:

| Event | When |
|-------|------|
| `EvUserMessage` | User message added to transcript |
| `EvTurnStart` | Beginning of each loop step |
| `EvTurnEnd` | End of each step (with stop reason) |
| `EvAssistantText` | LLM produced text |
| `EvToolCallStart` | Tool is about to execute |
| `EvToolCallEnd` | Tool finished (with result/error) |
| `EvUsage` | Token usage after each LLM call |
| `EvDone` | Agent loop completed |
| `EvError` | Error occurred |

## Session management

```go
// Save
sess := agent.NewSession("gpt-4o", system)
sess.SyncFromAgent(a)
sess.Save(agent.SessionPath(sess.ID))

// Load and resume
sess, _ := agent.LoadSession(agent.SessionPath("abc-123"))
sess.ApplyToAgent(a)
a.Continue(ctx, sink)
```

Session files are stored at:
- **macOS**: `~/Library/Application Support/ling-agent/sessions/`
- **Linux**: `~/.local/state/ling-agent/sessions/`
- **Windows**: `%LOCALAPPDATA%\ling-agent\sessions\`

## Configuration options

| Option | Description |
|--------|-------------|
| `WithSystem(s)` | Set the system prompt |
| `WithTools(t...)` | Register tools |
| `WithMaxSteps(n)` | Max loop iterations (0 = unlimited) |
| `WithAutoApprove(b)` | Skip tool confirmation |
| `BeforeToolExecute(fn)` | Hook to allow/deny/modify tool calls |
| `OnEvent(fn)` | Mirror all events to an observer |

## CLI flags

```
--provider      openai|claude|deepseek (default: openai)
--model         Model name
--api-key       API key (or env: OPENAI_API_KEY, ANTHROPIC_API_KEY, DEEPSEEK_API_KEY)
--cwd           Working directory (default: current)
--max-steps     Max loop steps (default: 0 = unlimited)
--auto-approve  Skip confirmation
--resume        Resume session by ID
--session       Custom session file path
--json          JSON event stream mode
```

## Comparison with zot

| Feature | zot | ling-agent |
|---------|-----|------------|
| Agent loop | ✅ | ✅ |
| LLM providers | 30+ | 3 (extensible via relay's 40 channels) |
| read/write/edit/bash | ✅ | ✅ |
| Session persistence | ✅ | ✅ |
| AGENTS.md loading | ✅ | ✅ |
| Interactive TUI | ✅ | ❌ |
| Print mode | ✅ | ✅ |
| JSON mode | ✅ | ✅ |
| Streaming output | ✅ | ❌ (non-streaming for tool_calls support) |
| OAuth login | ✅ | ❌ (API key only) |
| Extensions (JSON-RPC) | ✅ | ❌ |
| Telegram bot | ✅ | ❌ |

## License

MIT
