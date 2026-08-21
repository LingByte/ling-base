package app

import (
	"io"
	"os"

	"github.com/LingByte/ling-base/agent/config"
)

// Options configures a ling-agent run. Zero values mean “use defaults / config.toml”.
type Options struct {
	// --- Provider overlay (wins over config.toml when non-empty) ---
	Provider      string   // anthropic | openai | relay
	Model         string   // model id or alias; also settable via CLI --model
	BaseURL       string   // required for openai/relay
	APIKey        string   // inline key (prefer APIKeyEnv)
	APIKeyEnv     string   // env var name holding the key
	Temperature   *float64 // omit from request when nil
	ContextWindow int      // 0 = package default
	Theme         string

	// --- Run mode ---
	Print           bool   // non-interactive: print result and exit
	Prompt          string // user prompt (implies Print when set via CLI)
	OutputFormat    string // text|json|stream-json (default text)
	InputFormat     string // text|stream-json (default text)
	PermissionMode  string // default|acceptEdits|bypassPermissions|plan|dontAsk
	DangerouslySkip bool   // force bypassPermissions
	Verbose         bool
	MaxTurns        int
	Resume          string // session id
	ContinueSession bool   // resume most recent session for CWD
	NewSession      bool
	ForkSession     bool
	FullResume      bool // replay full transcript instead of summary
	AllowedTools    []string
	DisallowedTools []string
	PartialMessages bool
	CreateConfig    string // "global" | "local" — write starter config and exit
	Loop            bool
	MaxIterations   int
	Deep            bool // wrap prompt in deep-thinking template

	// --- I/O / workspace ---
	CWD    string // default: os.Getwd()
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (o Options) cwd() string {
	if o.CWD != "" {
		return o.CWD
	}
	cwd, _ := os.Getwd()
	return cwd
}

func (o Options) stdin() io.Reader {
	if o.Stdin != nil {
		return o.Stdin
	}
	return os.Stdin
}

func (o Options) stdout() io.Writer {
	if o.Stdout != nil {
		return o.Stdout
	}
	return os.Stdout
}

func (o Options) stderr() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return os.Stderr
}

func (o Options) outputFormatOrDefault() string {
	if o.OutputFormat != "" {
		return o.OutputFormat
	}
	return "text"
}

func (o Options) inputFormatOrDefault() string {
	if o.InputFormat != "" {
		return o.InputFormat
	}
	return "text"
}

// applyOverlay merges programmatic Options onto a loaded config.Config.
func (o Options) applyOverlay(cfg config.Config) config.Config {
	if o.Provider != "" {
		cfg.Provider = o.Provider
	}
	if o.Model != "" {
		cfg.Model = o.Model
	}
	if o.BaseURL != "" {
		cfg.BaseURL = o.BaseURL
	}
	if o.APIKey != "" {
		cfg.APIKey = o.APIKey
	}
	if o.APIKeyEnv != "" {
		cfg.APIKeyEnv = o.APIKeyEnv
	}
	if o.Temperature != nil {
		cfg.Temperature = o.Temperature
	}
	if o.ContextWindow != 0 {
		cfg.ContextWindow = o.ContextWindow
	}
	if o.Theme != "" {
		cfg.Theme = o.Theme
	}
	return cfg
}
