package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/LingByte/ling-base/agent/agent"
	"github.com/LingByte/ling-base/agent/browser"
	"github.com/LingByte/ling-base/agent/config"
	"github.com/LingByte/ling-base/agent/doctor"
	"github.com/LingByte/ling-base/agent/lsp"
	"github.com/LingByte/ling-base/agent/mcp"
	"github.com/LingByte/ling-base/agent/memory"
	"github.com/LingByte/ling-base/agent/permission"
	"github.com/LingByte/ling-base/agent/prompt"
	"github.com/LingByte/ling-base/agent/sandbox"
	"github.com/LingByte/ling-base/agent/session"
	"github.com/LingByte/ling-base/agent/skill"
	"github.com/LingByte/ling-base/agent/streamjson"
	"github.com/LingByte/ling-base/agent/subagent"
	"github.com/LingByte/ling-base/agent/swarm"
	"github.com/LingByte/ling-base/agent/tools"
	"github.com/LingByte/ling-base/agent/tui"
	"github.com/LingByte/ling-base/relay"
	"github.com/LingByte/ling-base/relay/channel/claude"
	"github.com/LingByte/ling-base/relay/channel/openai"
	"github.com/LingByte/ling-base/version"
)

func compactAndPersist(ctx context.Context, history []agent.Message, compact tui.CompactFunc, onSummary func(string)) ([]agent.Message, string, error) {
	newHistory, summary, err := compact(ctx, history)
	if err == nil && onSummary != nil {
		onSummary(summary)
	}
	return newHistory, summary, err
}

// withAgentTool returns a registry that is the base tools plus the Agent tool,
// wired to a sub-agent spawner that draws from the base tools.
func withAgentTool(base *tools.Registry, provider agent.Provider, model agent.Model, perm permission.Context, approver agent.Approver, maxTurns int, deferred map[string]bool) (*tools.Registry, error) {
	spawner := agent.NewSpawnerWithDeferred(provider, base, model, perm, approver, maxTurns, deferred)

	infos := make([]tools.AgentTypeInfo, 0)
	for _, t := range subagent.Builtin() {
		infos = append(infos, tools.AgentTypeInfo{Name: t.Name, Description: t.Description})
	}
	agentTool, err := tools.NewAgent(spawner, infos)
	if err != nil {
		return nil, err
	}
	return tools.NewRegistry(append(base.All(), agentTool)...), nil
}

// skillToolInfos adapts loaded skills into the tools package's SkillInfo,
// binding each skill's Render so the tools package stays decoupled from skill.
func skillToolInfos(skills []skill.Skill) []tools.SkillInfo {
	infos := make([]tools.SkillInfo, 0, len(skills))
	for _, sk := range skills {
		sk := sk
		infos = append(infos, tools.SkillInfo{
			Name:        sk.Name,
			Description: sk.Description,
			Render:      sk.Render,
		})
	}
	return infos
}

// builtinSlashCommands are the TUI commands a skill cannot override (the slash
// switch handles them before the /<skill> default branch).
var builtinSlashCommands = map[string]bool{
	"help": true, "?": true, "quit": true, "exit": true, "clear": true,
	"model": true, "mode": true, "goal": true,
	"memory": true, "mcp": true, "stats": true,
	"allow": true, "deny": true, "status": true,
	"config": true, "agents": true, "context": true,
	"compact": true, "add-dir": true,
	"plan": true, "doctor": true, "diff": true, "commit": true, "export": true,
	"last": true,
}

// withExtraDirs appends an "additional working directories" note to the system
// prompt when /add-dir has registered any (v1: informational context only).
func withExtraDirs(sys string, dirs []string) string {
	if len(dirs) == 0 {
		return sys
	}
	return sys + "\n\nAdditional working directories the user has made available:\n- " +
		strings.Join(dirs, "\n- ")
}

// buildDoctorInput gathers the facts /doctor reports, without prompting.
func buildDoctorInput(cfg config.Config, model agent.Model, cwd string, mcpServers int) doctor.Input {
	servers, hints := lsp.Survey(cwd)
	lspServers := make([]doctor.LSPServer, 0, len(servers))
	for _, s := range servers {
		lspServers = append(lspServers, doctor.LSPServer{Name: s.Bin, Language: s.Language, Version: s.Version})
	}
	ctxLimit, ctxSource := contextWindow(string(model), cfg.ContextWindow)
	in := doctor.Input{
		Provider:        providerName(cfg),
		Model:           string(model),
		SandboxMode:     sandboxMode(cfg.Sandbox),
		ConfigFound:     configFileExists(cwd),
		MCPServers:      mcpServers,
		LSPServers:      lspServers,
		MissingLSPHints: hints,
		AuthKind:        "none",
		ContextWindow:   ctxLimit,
		ContextSource:   ctxSource,
	}
	if cfg.Provider == config.ProviderOpenAI {
		if cfg.ResolveAPIKey() != "" {
			in.AuthOK, in.AuthKind = true, "api-key"
		}
	} else if key, err := resolveCredential(); err == nil {
		in.AuthOK = true
		if strings.HasPrefix(key, "sk-ant-oauth") {
			in.AuthKind = "oauth"
		} else {
			in.AuthKind = "api-key"
		}
	}
	return in
}

// configFileExists reports whether a home or project .ling-agent/config.toml exists.
func configFileExists(cwd string) bool {
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".ling-agent", "config.toml")); err == nil {
			return true
		}
	}
	_, err := os.Stat(config.ProjectPath(cwd))
	return err == nil
}

// providerName returns the resolved provider for display ("anthropic" default).
func providerName(cfg config.Config) string {
	if cfg.Provider != "" {
		return cfg.Provider
	}
	return config.ProviderAnthropic
}

// sandboxMode returns the resolved sandbox mode for display ("local" default).
func sandboxMode(sb config.Sandbox) string {
	if sb.Mode != "" {
		return sb.Mode
	}
	return config.SandboxLocal
}

// themeOrWarn passes a configured theme through, warning (and falling back to
// the default) when it names an unknown theme so a typo isn't silent.
func themeOrWarn(theme string, warn func(string)) string {
	if theme == "" || tui.IsTheme(theme) {
		return theme
	}
	warn(fmt.Sprintf("unknown theme %q in config; using default (valid: %s)", theme, tui.ThemeNames()))
	return ""
}

// tuiAgents adapts the built-in sub-agent types into the TUI's AgentInfo for
// the /agents command.
func tuiAgents() []tui.AgentInfo {
	bs := subagent.Builtin()
	out := make([]tui.AgentInfo, 0, len(bs))
	for _, t := range bs {
		out = append(out, tui.AgentInfo{Name: t.Name, Description: t.Description})
	}
	return out
}

// tuiSkills adapts loaded skills into TUI /<name> commands, warning when a skill
// name shadows a built-in command (the built-in wins, so the skill is
// unreachable as a slash command but remains callable via the Skill tool).
func tuiSkills(skills []skill.Skill, warn func(string)) []tui.SkillCommand {
	out := make([]tui.SkillCommand, 0, len(skills))
	for _, sk := range skills {
		sk := sk
		if builtinSlashCommands[sk.Name] {
			warn(fmt.Sprintf("skill %q shadows built-in /%s; reachable only via the Skill tool", sk.Name, sk.Name))
		}
		out = append(out, tui.SkillCommand{Name: sk.Name, Description: sk.Description, Render: sk.Render})
	}
	return out
}

// buildProvider selects and constructs the model provider from config. It
// returns the provider and the provider's default model. Anthropic is the
// default; "openai" uses an OpenAI-compatible Chat Completions endpoint;
// "relay" uses ling-base/relay (40+ provider adaptors).
func buildProvider(cfg config.Config) (agent.Provider, string, error) {
	switch cfg.Provider {
	case config.ProviderRelay, config.ProviderOpenAI:
		if cfg.BaseURL == "" {
			return nil, "", fmt.Errorf("provider %q requires baseURL in config", cfg.Provider)
		}
		key := cfg.ResolveAPIKey()
		if key == "" {
			return nil, "", fmt.Errorf("provider %q needs apiKey or apiKeyEnv in config; if using apiKeyEnv, export that variable before running", cfg.Provider)
		}
		model := cfg.Model
		if model == "" {
			model = "gpt-5.4-mini"
		}
		// Use the OpenAI channel for both "relay" and "openai" providers,
		// since relay's OpenAI channel handles all OpenAI-compatible endpoints.
		provider := openai.NewProvider(key, openai.WithBaseURL(strings.TrimSuffix(cfg.BaseURL, "/")))
		client := relay.New(relay.WithProvider(provider))
		return agent.NewRelayProvider(client), model, nil
	default:
		// Anthropic native: use the Claude channel directly.
		key := cfg.ResolveAPIKey()
		if key == "" {
			key = os.Getenv("ANTHROPIC_API_KEY")
		}
		if key == "" {
			return nil, "", fmt.Errorf("anthropic provider needs apiKey or ANTHROPIC_API_KEY env var")
		}
		model := cfg.Model
		if model == "" {
			model = "claude-sonnet-4-5-20250929"
		}
		var opts []claude.Option
		if cfg.BaseURL != "" {
			opts = append(opts, claude.WithBaseURL(cfg.BaseURL))
		}
		provider := claude.NewProvider(key, opts...)
		client := relay.New(relay.WithProvider(provider))
		return agent.NewRelayProvider(client), model, nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func buildBrowserOptions(bc config.Browser) browser.Options {
	opts := browser.Options{
		Mode:           browser.ModeLaunch,
		Headless:       true,
		ChromePath:     strings.TrimSpace(os.Getenv("LING_AGENT_CHROME_PATH")),
		UserDataDir:    firstNonEmpty(strings.TrimSpace(os.Getenv("LING_AGENT_CHROME_USER_DATA_DIR")), browser.DefaultUserDataDir()),
		RemoteURL:      strings.TrimSpace(os.Getenv("LING_AGENT_CHROME_REMOTE_URL")),
		HeadedFallback: true,
	}
	if bc.Headless != nil {
		opts.Headless = *bc.Headless
	} else if v := strings.TrimSpace(strings.ToLower(os.Getenv("LING_AGENT_BROWSER_HEADLESS"))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			opts.Headless = true
		case "0", "false", "no", "off":
			opts.Headless = false
		}
	}
	if bc.ChromePath != "" {
		opts.ChromePath = bc.ChromePath
	}
	if bc.RemoteURL != "" {
		opts.RemoteURL = bc.RemoteURL
	}
	if bc.UserDataDir != "" {
		opts.UserDataDir = bc.UserDataDir
	}
	if bc.HeadedFallback != nil {
		opts.HeadedFallback = *bc.HeadedFallback
	} else if v := strings.TrimSpace(strings.ToLower(os.Getenv("LING_AGENT_BROWSER_HEADED_FALLBACK"))); v != "" {
		switch v {
		case "1", "true", "yes", "on":
			opts.HeadedFallback = true
		case "0", "false", "no", "off":
			opts.HeadedFallback = false
		}
	}
	if bc.SearchEngine != "" {
		opts.SearchEngine = bc.SearchEngine
	}
	if opts.RemoteURL != "" {
		opts.Mode = browser.ModeAttach
	}
	return opts
}

// buildShellRunner selects the Bash execution backend from config. Both "os"
// (host confinement via sandbox-exec/bwrap) and "container" modes degrade
// gracefully to the local runner when the required tool is absent or the
// config is incomplete (warn explains why).
func buildShellRunner(cwd string, sb config.Sandbox, warn func(string)) sandbox.ShellRunner {
	switch sb.Mode {
	case config.SandboxOS:
		return buildOSShellRunner(cwd, sb, warn)
	case config.SandboxContainer:
		return buildContainerShellRunner(sb, warn)
	default:
		return sandbox.NewLocalShellRunner()
	}
}

// buildOSShellRunner picks the OS-native confinement tool for the current
// platform: sandbox-exec on macOS, bubblewrap on Linux. Falls back to local
// (unconfined) execution with a warning when the tool isn't available.
func buildOSShellRunner(cwd string, sb config.Sandbox, warn func(string)) sandbox.ShellRunner {
	switch goruntime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			warn("sandbox mode \"os\": sandbox-exec not found; falling back to local execution")
			return sandbox.NewLocalShellRunner()
		}
	case "linux":
		if _, err := exec.LookPath("bwrap"); err != nil {
			warn("sandbox mode \"os\": bwrap (bubblewrap) not found; falling back to local execution")
			return sandbox.NewLocalShellRunner()
		}
	default:
		warn("sandbox mode \"os\" is unsupported on " + goruntime.GOOS + "; falling back to local execution")
		return sandbox.NewLocalShellRunner()
	}
	runner, err := sandbox.NewOSShellRunner(cwd, sb.WriteRoots, sb.Network)
	if err != nil {
		warn(fmt.Sprintf("sandbox mode \"os\": %v; falling back to local execution", err))
		return sandbox.NewLocalShellRunner()
	}
	return runner
}

func buildContainerShellRunner(sb config.Sandbox, warn func(string)) sandbox.ShellRunner {
	runtime := sb.Runtime
	if runtime == "" {
		runtime = "docker"
	}
	switch {
	case sb.Image == "":
		warn("sandbox mode \"container\" has no image set; falling back to local execution")
	case !sandbox.RuntimeAvailable(runtime):
		warn(runtime + " is not installed; falling back to local execution")
	default:
		return sandbox.NewContainerShellRunner(
			sandbox.NewContainer(runtime, sb.Image, sb.MountCWDOr(true), sb.ReadOnly, sb.Network),
		)
	}
	return sandbox.NewLocalShellRunner()
}

// mcpController adapts the mcp.Manager to the TUI's MCPController so /mcp can
// inspect and reconnect/disconnect servers without the TUI owning the manager.
type mcpController struct {
	mgr *mcp.Manager
	ctx context.Context
}

func (c mcpController) Servers() []tui.MCPServerInfo {
	// Count live tools per server (mcp__<server>__<tool>).
	counts := map[string]int{}
	for _, t := range c.mgr.Tools(c.ctx) {
		n := t.Name()
		const pfx = "mcp__"
		if !strings.HasPrefix(n, pfx) {
			continue
		}
		if i := strings.Index(n[len(pfx):], "__"); i >= 0 {
			counts[n[len(pfx):len(pfx)+i]]++
		}
	}
	out := make([]tui.MCPServerInfo, 0)
	for _, s := range c.mgr.Servers() {
		out = append(out, tui.MCPServerInfo{Name: s.Name, Connected: s.Connected(), Tools: counts[s.Name]})
	}
	return out
}

func (c mcpController) Reconnect(name string) error  { return c.mgr.Reconnect(name) }
func (c mcpController) Disconnect(name string) error { return c.mgr.Disconnect(name) }

// gitBranch returns the current git branch for dir, or "" if not a repo.
func gitBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// gitCommit returns the short HEAD commit hash, or "" outside a repo.
func gitCommit(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

const starterConfig = `# LingAgent config
# Global: ~/.ling-agent/config.toml
# Local:  ./.ling-agent/config.toml (overrides global settings)

provider = "openai"
model = "openai/gpt-5.5"
baseURL = "https://api.example.com/v1"

# apiKeyEnv is the NAME of the environment variable that holds your API key:
# pick any name, then export a variable of that name, e.g. for the value below:
#   export MY_API_KEY="sk-..."
# Or set apiKey = "sk-..." inline — but the env form keeps secrets out of files.
apiKeyEnv = "MY_API_KEY"

# Temperature for the model. Omitted by default (server picks its own default).
# Some providers reject this field; remove the line to omit it from the request.
# temperature = 1.0

# Context window in tokens — drives autocompaction so long sessions don't
# overflow the model upstream ("max_tokens must be at least 1, got -N" or
# "context length exceeded"). Defaults to 200000 (Anthropic-sized); set this
# to the actual window your provider/model exposes. Common values: 8192, 16384,
# 32768, 128000.
# contextWindow = 8192

# TUI theme (Markdown + chrome). /theme switches it for a session.
# theme = "nord" # dracula | gruvbox | tokyo-night | nord | catppuccin

# Optional examples:
# [sandbox]
# mode = "local" # local | os | container
#
# [browser]
# searchEngine = "ddg" # ddg | google
# headless = true
#
# [permissions]
# mode = "default" # default(ask) | acceptEdits | plan | dontAsk | bypassPermissions
# allow = ["Bash(go test:*)"]
# deny = ["Bash(rm:*)"]
`

func CreateConfig(scope, cwd string) (string, error) {
	var path string
	switch scope {
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}
		path = filepath.Join(home, ".ling-agent", "config.toml")
	case "local":
		path = config.ProjectPath(cwd)
	default:
		return "", fmt.Errorf("--create-config must be global or local")
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("config already exists: %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// resolveResumeID selects the prior session to seed from, if any.
// resolveResumeID selects the prior session to seed from, if any. An explicit
// --resume/--continue is honored in any mode; the implicit "pick up the most
// recent project session" is an interactive convenience only, so headless (-p)
// and embedding (stream-json) runs stay stateless unless asked.
func resolveResumeID(cwd string, opts Options, interactive bool) (string, error) {
	if opts.NewSession && (opts.Resume != "" || opts.ContinueSession) {
		return "", fmt.Errorf("--new-session cannot be combined with --resume or --continue")
	}
	if opts.Resume != "" {
		return opts.Resume, nil
	}
	if opts.ContinueSession {
		if id, ok := session.MostRecent(cwd); ok {
			return id, nil
		}
		return "", fmt.Errorf("--continue: no previous session found in this directory")
	}
	if interactive && !opts.NewSession {
		if id, ok := session.MostRecent(cwd); ok {
			return id, nil
		}
	}
	return "", nil
}

// run dispatches to headless or interactive mode. Phase 0 only implements a
// headless stub that emits a well-formed result so the output renderers and
// differential harness can be exercised end-to-end before the agent loop lands.
func Run(ctx context.Context, opts Options) error {
	format, err := ParseOutputFormat(opts.outputFormatOrDefault())
	if err != nil {
		return err
	}

	cwd := opts.cwd()
	if opts.CreateConfig != "" {
		path, err := CreateConfig(opts.CreateConfig, cwd)
		if err != nil {
			return err
		}
		fmt.Fprintf(opts.stdout(), "Created %s\n\nEdit it with your provider/baseURL/model/apiKeyEnv, then export the named API key env var and run ling-agent.\n", path)
		return nil
	}

	// Mode: autonomous --loop | stream-json input (embedding) | headless -p |
	// interactive TUI. --loop is non-interactive and renders as text.
	interactive := !opts.Print && !opts.Loop && opts.inputFormatOrDefault() != "stream-json"
	if opts.Loop {
		if opts.inputFormatOrDefault() == "stream-json" {
			return fmt.Errorf("--loop cannot be combined with --input-format stream-json")
		}
		if format != FormatText {
			return fmt.Errorf("--loop only supports --output-format text")
		}
	}
	if opts.Print && format == FormatStreamJSON && !opts.Verbose {
		return fmt.Errorf("--output-format stream-json requires --verbose")
	}
	if opts.PartialMessages && (!opts.Print || format != FormatStreamJSON) {
		return fmt.Errorf("--include-partial-messages only works with --print and --output-format=stream-json")
	}

	start := time.Now()
	r := NewRenderer(format, opts.stdout())

	// Resolve the session: explicit resume by id, auto-resume the most recent
	// project session by default, or start new when requested/no prior session.
	// --fork-session writes to a fresh id while preserving the original.
	var initialMessages []agent.Message
	resumeID, err := resolveResumeID(cwd, opts, interactive)
	if err != nil {
		return err
	}
	sessionID := uuid.NewString()
	if resumeID != "" {
		// Token-saving resume: if a persisted compaction summary exists and the
		// user didn't ask for a --full replay, seed from the summary instead of
		// the entire transcript. Falls back to full replay when no summary exists.
		if summary, ok := session.ReadSummary(cwd, resumeID); ok && !opts.FullResume {
			initialMessages = []agent.Message{
				agent.NewUserMessage(
					"Summary of the earlier conversation in this session:\n\n" + summary),
			}
			fmt.Fprintln(opts.stderr(), "resuming from compacted summary (--full for the entire transcript)")
		} else {
			entries, rerr := session.Read(session.ExistingPath(cwd, resumeID))
			if rerr != nil {
				return fmt.Errorf("resume %s: %w", resumeID, rerr)
			}
			initialMessages, rerr = agent.MessagesFromEntries(entries)
			if rerr != nil {
				return fmt.Errorf("resume %s: %w", resumeID, rerr)
			}
		}
		if !opts.ForkSession {
			sessionID = resumeID // continue appending to the same transcript
		}
	}

	// Select the model provider (.ling-agent/config.toml: anthropic | openai).
	cfg := opts.applyOverlay(config.Load(cwd))
	provider, providerModel, err := buildProvider(cfg)
	if err != nil {
		return err
	}

	// Resolve the permission mode: --permission-mode flag wins, else the config
	// default ([permissions] mode), else "default". --dangerously-skip wins over all.
	modeStr := opts.PermissionMode
	if modeStr == "" {
		modeStr = cfg.Permissions.Mode
	}
	if modeStr == "" {
		modeStr = string(permission.ModeDefault)
	}
	mode := permission.Mode(modeStr)
	if !mode.Valid() {
		return fmt.Errorf("invalid permission mode %q (default|acceptEdits|bypassPermissions|plan|dontAsk)", modeStr)
	}
	if opts.DangerouslySkip {
		mode = permission.ModeBypassPermissions
	}
	// --model overrides the config/provider default.
	modelStr := opts.Model
	if modelStr == "" {
		modelStr = providerModel
	}
	model := agent.Model(resolveModel(modelStr))

	// Build allow/deny rules from config (.ling-agent) + CLI flags.
	allowRules, err := permission.ParseRules(append(append([]string{}, cfg.Permissions.Allow...), opts.AllowedTools...))
	if err != nil {
		return fmt.Errorf("--allowedTools/permissions.allow: %w", err)
	}
	denyRules, err := permission.ParseRules(append(append([]string{}, cfg.Permissions.Deny...), opts.DisallowedTools...))
	if err != nil {
		return fmt.Errorf("--disallowedTools/permissions.deny: %w", err)
	}
	// Headless path: the mode is fixed for the lifetime of this command.
	permCtx := permission.Context{Mode: permission.StaticMode(mode), Allow: allowRules, Deny: denyRules}
	// Refresh MEMORY.md's links to the .ling-agent/memory/*.md detail notes before
	// building the prompt, so recall surfaces them (best-effort; idempotent).
	_ = memory.New(filepath.Join(cwd, ".ling-agent")).SyncLinks()
	// Assemble the full system prompt (base instructions + env context +
	// CLAUDE.md) once for this run.
	sysPrompt := prompt.System(cwd, string(model))

	// Build the tool registry. Sub-agents draw from the base tools (incl. any
	// MCP tools); the top-level registry adds the Agent tool.
	executor := buildShellRunner(cwd, cfg.Sandbox, func(m string) { fmt.Fprintln(opts.stderr(), "warning:", m) })
	// Lazy browser engine for the web tools, tied to the run context and closed
	// at session end so any launched Chrome is reliably terminated (it launches
	// nothing until a web tool actually runs).
	browserEngine := browser.NewEngine(ctx, buildBrowserOptions(cfg.Browser))
	defer browserEngine.Close()
	// Background shells (Bash run_in_background) are session-scoped and tied to
	// the run context; KillAll terminates any still running at session end.
	shellStore := tools.NewShellStore(ctx)
	defer shellStore.KillAll()
	// Lazy language-server pool for code-intel tools (Diagnostics/Definition/
	// References). Servers are detected on PATH + toolchain dirs, spawned on
	// first use, and shut down at session end. Not downloaded.
	lspPool := lsp.NewPool(ctx, cwd, cfg.LSP.Disabled, nil)
	defer lspPool.Close()
	// Multi-agent swarm supervisor for parallel background tasks.
	homeDir, _ := os.UserHomeDir()
	swarmRoot := filepath.Join(homeDir, ".ling-agent", "swarm")
	swarmSupervisor := swarm.New(swarm.Config{
		Root:     swarmRoot,
		RepoRoot: cwd,
	})
	base, err := tools.DefaultRegistry(executor,
		tools.WithBrowserEngine(browserEngine),
		tools.WithShellStore(shellStore),
		tools.WithLSP(lspPool),
		tools.WithSwarm(swarmSupervisor),
	)
	if err != nil {
		return err
	}

	// Connect configured MCP servers (.mcp.json) and fold in their tools and
	// resource tools. Best effort: a server failure does not abort the run.
	mcpCfg, _ := mcp.LoadConfig(cwd)
	mcpMgr, mcpErrs := mcp.Connect(ctx, mcpCfg)
	defer mcpMgr.Close()
	for _, e := range mcpErrs {
		fmt.Fprintln(opts.stderr(), "warning:", e)
	}
	baseTools := base.All()
	baseTools = append(baseTools, mcpMgr.Tools(ctx)...)
	if rts, rerr := mcpMgr.ResourceTools(); rerr == nil && len(mcpMgr.Servers()) > 0 {
		baseTools = append(baseTools, rts...)
	}
	// Persistent memory: one store shared by the Memory tool (agent + sub-agents)
	// and the /memory command.
	memStore := memory.New(filepath.Join(cwd, ".ling-agent"))
	if memTool, merr := tools.NewMemoryForProject(memStore, cwd); merr == nil {
		baseTools = append(baseTools, memTool)
	}

	// User-defined skills (~/.ling-agent/skills overlaid by .ling-agent/skills) become a
	// single Skill tool the model can invoke; the TUI also dispatches /<skill>.
	skills := skill.Load(cwd, func(m string) { fmt.Fprintln(opts.stderr(), "warning:", m) })
	skillInfos := skillToolInfos(skills)
	if skillTool, serr := tools.NewSkill(skillInfos); serr == nil && skillTool != nil {
		baseTools = append(baseTools, skillTool)
	}

	// Deferred tool loading: MCP tools can be numerous, so withhold them from
	// the initial request behind a ToolSearch tool that loads them on demand.
	deferredTools := map[string]bool{}
	for _, t := range baseTools {
		if strings.HasPrefix(t.Name(), "mcp__") {
			deferredTools[t.Name()] = true
		}
	}
	if len(deferredTools) > 0 {
		catalog := make([]tools.ToolInfo, 0, len(baseTools))
		for _, t := range baseTools {
			desc, _ := t.Description(ctx)
			catalog = append(catalog, tools.ToolInfo{Name: t.Name(), Description: desc})
		}
		if ts, terr := tools.NewToolSearch(catalog); terr == nil {
			baseTools = append(baseTools, ts)
		}
	}
	base = tools.NewRegistry(baseTools...)

	// Headless has no interactive approver, so permission "ask" denies.
	approver := agent.DenyAll
	registry, err := withAgentTool(base, provider, model, permCtx, approver, opts.MaxTurns, deferredTools)
	if err != nil {
		return err
	}

	// Open the transcript for this session (best effort: a transcript failure
	// should not abort the run).
	var recorder agent.Recorder
	if tr, terr := session.NewTranscript(session.Meta{
		SessionID:      sessionID,
		CWD:            cwd,
		Version:        version.GetVersion(),
		GitBranch:      gitBranch(cwd),
		PermissionMode: string(mode),
	}); terr == nil {
		defer func() { _ = tr.Close() }()
		recorder = tr
	}

	loop := agent.New(provider, registry)

	// Persist compaction summaries for token-saving resume (a LingAgent divergence).
	onSummary := func(summary string) {
		_ = session.WriteSummary(cwd, sessionID, summary, gitCommit(cwd))
	}

	// Interactive TUI: the default when not headless and not stream-json input.
	// It drives the same loop, prompting the user to resolve permission asks.
	if interactive {
		// Shared settings so slash commands can read/change them between turns.
		ctxLimit, ctxSource := contextWindow(string(model), cfg.ContextWindow)
		sess := &tui.Session{
			// Effective model (flag or config default), never "" — otherwise a
			// non-Anthropic provider would wrongly resolve to the Anthropic default.
			SessionID:           sessionID,
			Model:               modelStr,
			ResolvedModel:       string(model),
			Theme:               themeOrWarn(cfg.Theme, func(m string) { fmt.Fprintln(opts.stderr(), "warning:", m) }), // user default (~/.ling-agent) overlaid by project; /theme overrides per session
			PermissionMode:      string(mode),
			Memory:              memStore,
			MCP:                 mcpController{mgr: mcpMgr, ctx: ctx},
			Skills:              tuiSkills(skills, func(m string) { fmt.Fprintln(opts.stderr(), "warning:", m) }),
			Provider:            providerName(cfg),
			SandboxMode:         sandboxMode(cfg.Sandbox),
			CWD:                 cwd,
			GitBranch:           gitBranch(cwd),
			Agents:              tuiAgents(),
			ContextWindow:       ctxLimit,
			ContextWindowSource: ctxSource,
			Compact: func(ctx context.Context, history []agent.Message) ([]agent.Message, string, error) {
				return compactAndPersist(ctx, history, func(ctx context.Context, history []agent.Message) ([]agent.Message, string, error) {
					return loop.Compact(ctx, history, agent.Model(resolveModel(modelStr)))
				}, onSummary)
			},

			Doctor: func() string {
				return doctor.Format(doctor.Run(buildDoctorInput(cfg, model, cwd, len(mcpCfg.MCPServers))))
			},
		}
		runFn := func(ctx context.Context, prompt string, history []agent.Message, ap agent.Approver, asker tools.Asker, planner tools.Planner, emit agent.Emitter) (agent.Result, error) {
			// Permission mode reads live from the session every check, so a
			// /mode bypass (or ExitPlanMode flipping out of plan) takes effect
			// on the very next tool dispatch inside the agent loop — not just
			// at the next TUI turn boundary. The Context itself is rebuilt
			// here so the rule lists stay snapshot-stable for the duration of
			// this turn, but the mode probe is live.
			turnPerm := permission.Context{
				Mode:  func() permission.Mode { return permission.Mode(sess.PermissionMode) },
				Allow: allowRules,
				Deny:  denyRules,
			}
			return loop.Run(ctx, agent.Options{
				Prompt:          prompt,
				Model:           agent.Model(resolveModel(sess.Model)), // resolved fresh each turn
				System:          withExtraDirs(sysPrompt, sess.ExtraDirs),
				MaxTurns:        opts.MaxTurns,
				ContextWindow:   cfg.ContextWindow,
				Permission:      turnPerm,
				DeferredTools:   deferredTools,
				Approver:        ap,
				Asker:           asker,
				Planner:         planner,
				InitialMessages: history,
				Recorder:        recorder,
				WebTools:        true,
				OnSummary:       onSummary,
			}, emit)
		}
		return tui.Run(ctx, tui.RunFunc(runFn), initialMessages, sess)
	}

	// Stream-json input: drive a persistent agent over stdin/stdout (the
	// embedding channel). Each user message is a turn; permission asks are
	// surfaced as control_request and answered by the peer.
	if opts.inputFormatOrDefault() == "stream-json" {
		driver := streamjson.NewDriver(opts.stdout())
		runFn := func(ctx context.Context, prompt string, history []agent.Message, ap agent.Approver, emit agent.Emitter) (agent.Result, error) {
			return loop.Run(ctx, agent.Options{
				Prompt:          prompt,
				Model:           model,
				System:          sysPrompt,
				MaxTurns:        opts.MaxTurns,
				ContextWindow:   cfg.ContextWindow,
				Permission:      permCtx,
				Approver:        ap,
				DeferredTools:   deferredTools,
				InitialMessages: history,
				Recorder:        recorder,
				WebTools:        true,
				OnSummary:       onSummary,
			}, emit)
		}
		return driver.Run(ctx, opts.stdin(), runFn)
	}

	// Autonomous goal loop: iterate against the spec until complete or capped.
	if opts.Loop {
		return runGoalLoop(ctx, opts.stderr(), loopRun{
			loop:       loop,
			cwd:        cwd,
			mode:       mode,
			model:      model,
			system:     sysPrompt,
			maxTurns:   opts.MaxTurns,
			iterations: opts.MaxIterations,
			permCtx:    permCtx,
			approver:   approver,
			deferred:   deferredTools,
			recorder:   recorder,
			onSummary:  onSummary,
			render:     r,
		})
	}

	// Single-shot headless run. For stream-json output, emit JS-compatible
	// message envelopes (assistant/user) via an envelope recorder alongside the
	// transcript; the simplified delta events are not used in that mode.
	runRecorder := agent.Recorder(recorder)
	emit := func(ev agent.Event) { _ = r.Event(ev) }
	var partial func(agent.StreamChunk)
	if format == FormatStreamJSON {
		// Serialize envelope + partial writes to the same stream.
		var writeMu sync.Mutex
		out := opts.stdout()
		runRecorder = multiRecorder{recorder, newEnvelopeRecorder(out, sessionID)}
		emit = func(agent.Event) {}
		if opts.PartialMessages {
			partial = newPartialEmitter(out, sessionID, &writeMu).emit
		}
	}
	// --deep wraps the user's prompt in the deep-thinking template.
	submittedPrompt := opts.Prompt
	if opts.Deep && submittedPrompt != "" {
		submittedPrompt = prompt.DeepThinkingPrompt(submittedPrompt)
	}
	res, err := loop.Run(ctx, agent.Options{
		Prompt:          submittedPrompt,
		Model:           model,
		System:          sysPrompt,
		MaxTurns:        opts.MaxTurns,
		ContextWindow:   cfg.ContextWindow,
		Permission:      permCtx,
		Approver:        approver,
		DeferredTools:   deferredTools,
		InitialMessages: initialMessages,
		Recorder:        runRecorder,
		WebTools:        true,
		OnSummary:       onSummary,
		PartialMessages: partial,
	}, emit)

	out := ResultMessage{
		Type:          "result",
		Subtype:       "success",
		IsError:       err != nil,
		DurationMS:    time.Since(start).Milliseconds(),
		DurationAPIMS: time.Since(start).Milliseconds(),
		NumTurns:      res.NumTurns,
		Result:        res.Text,
		StopReason:    res.StopReason,
		SessionID:     sessionID,
		TotalCostUSD:  0, // Phase 3: derive from usage + pricing.
		Usage: map[string]any{
			"input_tokens":                res.InputTokens,
			"output_tokens":               res.OutputTokens,
			"cache_read_input_tokens":     res.CacheReadInputTokens,
			"cache_creation_input_tokens": res.CacheCreationInputTokens,
		},
		UUID: uuid.NewString(),
	}
	if err != nil {
		out.Subtype = "error_during_execution"
		out.Result = "Error: " + friendlyError(err)
	}
	if rerr := r.Result(out); rerr != nil {
		return rerr
	}
	if err != nil {
		// The error is already rendered into the result payload; signal a
		// non-zero exit without printing it again to stderr.
		return ErrRendered
	}
	return nil
}

// ErrRendered marks a run error already emitted in the result payload,
// so Execute exits non-zero without re-printing it.
var ErrRendered = fmt.Errorf("run failed")

// Execute runs with opts and returns a process exit code.
func Execute(opts Options) int {
	if err := Run(context.Background(), opts); err != nil {
		if !errors.Is(err, ErrRendered) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return 1
	}
	return 0
}
