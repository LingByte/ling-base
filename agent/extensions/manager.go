// Package extensions implements the host side of LingAgent's subprocess
// extension protocol. The Manager discovers extensions in well-known
// directories, spawns each one, completes the hello handshake, and
// routes slash commands to the right extension.
//
// Each extension is its own process, communicating with LingAgent over
// its stdin/stdout in newline-delimited JSON. Stderr is redirected to a
// per-extension log file. Crashing one extension does not affect the
// others or the host.
//
// Extension layout:
//
//	~/.ling-agent/extensions/<name>/
//	  extension.json   — manifest (name, version, executable)
//	  <binary>         — the extension executable
//	  *.log            — stderr log
package extensions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/agent/extproto"
)

// Manifest is the extension.json file shipped alongside an extension's
// executable.
type Manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Executable  string `json:"executable"`
	Description string `json:"description,omitempty"`
}

// Command is a slash command registered by an extension.
type Command struct {
	Name        string
	Description string
	Extension   string // owning extension name
}

// Tool is a tool registered by an extension.
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Deferred    bool
	Extension   string
}

// Extension is one loaded extension subprocess.
type Extension struct {
	Name    string
	Manifest Manifest
	Path    string

	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	commands map[string]Command
	tools    map[string]Tool
	ready    bool
}

// Manager discovers, launches, and manages extension subprocesses.
type Manager struct {
	mu         sync.Mutex
	extensions map[string]*Extension
	rootDirs   []string
	hostInfo   extproto.HelloAckFromHost

	pendingCommands map[string]chan extproto.CommandResponseFromExt
	pendingTools    map[string]chan extproto.ToolResultFromExt
}

// New creates a Manager that discovers extensions from the given root
// directories (typically ~/.ling-agent/extensions and
// <cwd>/.ling-agent/extensions).
func New(rootDirs []string, hostInfo extproto.HelloAckFromHost) *Manager {
	return &Manager{
		extensions:      make(map[string]*Extension),
		rootDirs:        rootDirs,
		hostInfo:        hostInfo,
		pendingCommands: make(map[string]chan extproto.CommandResponseFromExt),
		pendingTools:    make(map[string]chan extproto.ToolResultFromExt),
	}
}

// Discover finds and launches all extensions. Returns the list of
// launched extensions and any errors encountered. A failed extension
// does not prevent the others from launching.
func (m *Manager) Discover(ctx context.Context) ([]*Extension, []error) {
	var errs []error
	seen := map[string]bool{}

	for _, root := range m.rootDirs {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue // missing dir is fine
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if seen[name] {
				continue
			}
			seen[name] = true

			extDir := filepath.Join(root, name)
			manifestPath := filepath.Join(extDir, "extension.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue // no manifest, skip
			}
			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				errs = append(errs, fmt.Errorf("extension %s: bad manifest: %w", name, err))
				continue
			}
			if manifest.Name == "" {
				manifest.Name = name
			}

			ext, err := m.launch(ctx, manifest, extDir)
			if err != nil {
				errs = append(errs, fmt.Errorf("extension %s: launch: %w", name, err))
				continue
			}

			m.mu.Lock()
			m.extensions[ext.Name] = ext
			m.mu.Unlock()
		}
	}

	m.mu.Lock()
	list := make([]*Extension, 0, len(m.extensions))
	for _, e := range m.extensions {
		list = append(list, e)
	}
	m.mu.Unlock()
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, errs
}

// launch starts an extension subprocess and performs the hello handshake.
func (m *Manager) launch(ctx context.Context, manifest Manifest, extDir string) (*Extension, error) {
	exePath := manifest.Executable
	if !filepath.IsAbs(exePath) {
		exePath = filepath.Join(extDir, exePath)
	}

	cmd := exec.CommandContext(ctx, exePath)
	cmd.Dir = extDir

	// Redirect stderr to a log file.
	logPath := filepath.Join(extDir, "stderr.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		cmd.Stderr = logFile
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	ext := &Extension{
		Name:      manifest.Name,
		Manifest:  manifest,
		Path:      extDir,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		commands:  make(map[string]Command),
		tools:     make(map[string]Tool),
	}

	// Send hello_ack.
	hostInfo := m.hostInfo
	hostInfo.Type = "hello_ack"
	hostInfo.ExtensionDir = extDir
	if err := ext.send(hostInfo); err != nil {
		return nil, fmt.Errorf("hello_ack: %w", err)
	}

	// Read frames until ready. This is synchronous during startup;
	// after ready, a goroutine takes over reading.
	readyCh := make(chan error, 1)
	go func() {
		readyCh <- ext.readUntilReady()
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			return nil, err
		}
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for ready")
	}

	// Start the background reader for async frames.
	go ext.readLoop(m)

	return ext, nil
}

// readUntilReady reads frames until it sees ready.
func (e *Extension) readUntilReady() error {
	sc := bufio.NewScanner(e.stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var frame struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(sc.Bytes(), &frame); err != nil {
			continue
		}
		switch frame.Type {
		case "hello":
			var hello extproto.HelloFromExt
			if err := json.Unmarshal(sc.Bytes(), &hello); err == nil {
				e.Name = hello.Name
			}
		case "register_command":
			var reg extproto.RegisterCommandFromExt
			if err := json.Unmarshal(sc.Bytes(), &reg); err == nil {
				e.mu.Lock()
				e.commands[reg.Name] = Command{
					Name:        reg.Name,
					Description: reg.Description,
					Extension:   e.Name,
				}
				e.mu.Unlock()
			}
		case "register_tool":
			var reg extproto.RegisterToolFromExt
			if err := json.Unmarshal(sc.Bytes(), &reg); err == nil {
				e.mu.Lock()
				e.tools[reg.Name] = Tool{
					Name:        reg.Name,
					Description: reg.Description,
					Schema:      reg.Schema,
					Deferred:    reg.Deferred,
					Extension:   e.Name,
				}
				e.mu.Unlock()
			}
		case "ready":
			e.mu.Lock()
			e.ready = true
			e.mu.Unlock()
			return nil
		}
	}
	return fmt.Errorf("stdout closed before ready")
}

// readLoop reads async frames after the extension is ready.
func (e *Extension) readLoop(m *Manager) {
	sc := bufio.NewScanner(e.stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var frame struct {
			Type string `json:"type"`
			ID   string `json:"id,omitempty"`
		}
		if err := json.Unmarshal(sc.Bytes(), &frame); err != nil {
			continue
		}
		switch frame.Type {
		case "command_response":
			var resp extproto.CommandResponseFromExt
			if err := json.Unmarshal(sc.Bytes(), &resp); err == nil {
				m.mu.Lock()
				if ch, ok := m.pendingCommands[resp.ID]; ok {
					ch <- resp
					delete(m.pendingCommands, resp.ID)
				}
				m.mu.Unlock()
			}
		case "tool_result":
			var result extproto.ToolResultFromExt
			if err := json.Unmarshal(sc.Bytes(), &result); err == nil {
				m.mu.Lock()
				if ch, ok := m.pendingTools[result.ID]; ok {
					ch <- result
					delete(m.pendingTools, result.ID)
				}
				m.mu.Unlock()
			}
		case "notify":
			var notify extproto.NotifyFromExt
			if err := json.Unmarshal(sc.Bytes(), &notify); err == nil {
				// Could route to TUI; for now, log.
				fmt.Fprintf(os.Stderr, "[ext:%s] %s: %s\n", e.Name, notify.Level, notify.Message)
			}
		}
	}
}

// send writes a frame to the extension's stdin.
func (e *Extension) send(v any) error {
	data, err := extproto.Encode(v)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err = e.stdin.Write(data)
	return err
}

// InvokeCommand sends a slash command to the extension and waits for
// the response.
func (m *Manager) InvokeCommand(ctx context.Context, cmdName, args string) (extproto.CommandResponseFromExt, error) {
	m.mu.Lock()
	ext := m.findExtensionByCommand(cmdName)
	if ext == nil {
		m.mu.Unlock()
		return extproto.CommandResponseFromExt{}, fmt.Errorf("no extension handles command %s", cmdName)
	}
	id := fmt.Sprintf("cmd-%d", time.Now().UnixNano())
	ch := make(chan extproto.CommandResponseFromExt, 1)
	m.pendingCommands[id] = ch
	m.mu.Unlock()

	err := ext.send(extproto.CommandInvokedFromHost{
		Type: "command_invoked",
		ID:   id,
		Name: cmdName,
		Args: args,
	})
	if err != nil {
		return extproto.CommandResponseFromExt{}, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return extproto.CommandResponseFromExt{}, ctx.Err()
	case <-time.After(30 * time.Second):
		return extproto.CommandResponseFromExt{}, fmt.Errorf("timeout waiting for command response")
	}
}

// CallTool sends a tool call to the extension and waits for the result.
func (m *Manager) CallTool(ctx context.Context, toolName string, args json.RawMessage) (extproto.ToolResultFromExt, error) {
	m.mu.Lock()
	ext := m.findExtensionByTool(toolName)
	if ext == nil {
		m.mu.Unlock()
		return extproto.ToolResultFromExt{}, fmt.Errorf("no extension handles tool %s", toolName)
	}
	id := fmt.Sprintf("tool-%d", time.Now().UnixNano())
	ch := make(chan extproto.ToolResultFromExt, 1)
	m.pendingTools[id] = ch
	m.mu.Unlock()

	err := ext.send(extproto.ToolCallFromHost{
		Type: "tool_call",
		ID:   id,
		Name: toolName,
		Args: args,
	})
	if err != nil {
		return extproto.ToolResultFromExt{}, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return extproto.ToolResultFromExt{}, ctx.Err()
	case <-time.After(60 * time.Second):
		return extproto.ToolResultFromExt{}, fmt.Errorf("timeout waiting for tool result")
	}
}

// Commands returns all slash commands registered by all extensions.
func (m *Manager) Commands() []Command {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Command
	for _, ext := range m.extensions {
		ext.mu.Lock()
		for _, cmd := range ext.commands {
			out = append(out, cmd)
		}
		ext.mu.Unlock()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Tools returns all tools registered by all extensions.
func (m *Manager) Tools() []Tool {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Tool
	for _, ext := range m.extensions {
		ext.mu.Lock()
		for _, tool := range ext.tools {
			out = append(out, tool)
		}
		ext.mu.Unlock()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Close shuts down all extensions gracefully.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ext := range m.extensions {
		ext.send(extproto.ShutdownFromHost{Type: "shutdown"})
		if ext.cmd != nil && ext.cmd.Process != nil {
			ext.cmd.Process.Kill()
		}
	}
}

// findExtensionByCommand finds the extension that registered a command.
// Caller must hold m.mu.
func (m *Manager) findExtensionByCommand(name string) *Extension {
	for _, ext := range m.extensions {
		ext.mu.Lock()
		if _, ok := ext.commands[name]; ok {
			ext.mu.Unlock()
			return ext
		}
		ext.mu.Unlock()
	}
	return nil
}

// findExtensionByTool finds the extension that registered a tool.
// Caller must hold m.mu.
func (m *Manager) findExtensionByTool(name string) *Extension {
	for _, ext := range m.extensions {
		ext.mu.Lock()
		if _, ok := ext.tools[name]; ok {
			ext.mu.Unlock()
			return ext
		}
		ext.mu.Unlock()
	}
	return nil
}

// Extensions returns all loaded extensions, sorted by name.
func (m *Manager) Extensions() []*Extension {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := make([]*Extension, 0, len(m.extensions))
	for _, e := range m.extensions {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// HasCommand reports whether any extension handles the given command.
func (m *Manager) HasCommand(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.findExtensionByCommand(name) != nil
}

// HasTool reports whether any extension handles the given tool.
func (m *Manager) HasTool(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.findExtensionByTool(name) != nil
}

// FormatExtensions returns a human-readable summary of loaded extensions.
func (m *Manager) FormatExtensions() string {
	exts := m.Extensions()
	if len(exts) == 0 {
		return "No extensions loaded."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Extensions (%d):\n", len(exts)))
	for _, e := range exts {
		b.WriteString(fmt.Sprintf("  %s v%s", e.Name, e.Manifest.Version))
		if e.Manifest.Description != "" {
			b.WriteString(" — " + e.Manifest.Description)
		}
		b.WriteString("\n")
		e.mu.Lock()
		if len(e.commands) > 0 {
			cmds := make([]string, 0, len(e.commands))
			for c := range e.commands {
				cmds = append(cmds, "/"+c)
			}
			sort.Strings(cmds)
			b.WriteString("    commands: " + strings.Join(cmds, ", ") + "\n")
		}
		if len(e.tools) > 0 {
			tools := make([]string, 0, len(e.tools))
			for t := range e.tools {
				tools = append(tools, t)
			}
			sort.Strings(tools)
			b.WriteString("    tools: " + strings.Join(tools, ", ") + "\n")
		}
		e.mu.Unlock()
	}
	return strings.TrimSpace(b.String())
}
