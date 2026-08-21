// Package skill loads reusable prompt/command skills from Markdown files with
// YAML frontmatter. Skills are discovered from multiple directories with
// priority order (first match wins per name):
//
//	./.ling-agent/skills/<name>/SKILL.md   — project (native)
//	~/.ling-agent/skills/<name>/SKILL.md   — global (native)
//	./.ling-agent/skills/<name>.md         — project (flat, legacy)
//	~/.ling-agent/skills/<name>.md         — global (flat, legacy)
//	./.claude/skills/<name>/SKILL.md       — project (claude-compat)
//	~/.claude/skills/<name>/SKILL.md       — global (claude-compat)
//	./.agents/skills/<name>/SKILL.md       — project (agents-compat)
//	~/.agents/skills/<name>/SKILL.md       — global (agents-compat)
//
// The compat paths are deliberate: a SKILL.md written for any of the
// related ecosystems works in LingAgent unchanged.
//
// A skill file looks like:
//
//	---
//	name: review
//	description: Structured review of the current diff
//	type: prompt        # prompt | command
//	tools: [Bash, Read] # optional allowlist (growth point; unused in v1)
//	---
//	Review the staged changes carefully. $ARGUMENTS
//
// The body supports $ARGUMENTS substitution when the skill is invoked.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill types.
const (
	TypePrompt  = "prompt"  // body is injected as instructions for the model
	TypeCommand = "command" // body is a command template (v1: same handling as prompt)
)

// Skill is one loaded skill definition.
type Skill struct {
	Name        string   // invocation name (also the /<name> slash command)
	Description string   // one-line, model-facing
	Type        string   // TypePrompt (default) | TypeCommand
	Tools       []string // optional tool allowlist (growth point; unused in v1)
	Body        string   // template body; supports $ARGUMENTS
	Path        string   // source file, for diagnostics
	Source      string   // where the skill came from ("project", "global (claude)", etc.)
}

// frontmatter is the YAML header schema.
type frontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`
	Tools       []string `yaml:"tools"`
}

// Render substitutes $ARGUMENTS in the body with args (the text after the skill
// name). When the body contains no $ARGUMENTS placeholder and args is non-empty,
// the args are appended on a new line so they are never silently dropped.
func (s Skill) Render(args string) string {
	if strings.Contains(s.Body, "$ARGUMENTS") {
		return strings.ReplaceAll(s.Body, "$ARGUMENTS", args)
	}
	if strings.TrimSpace(args) == "" {
		return s.Body
	}
	return strings.TrimRight(s.Body, "\n") + "\n\n" + args
}

// SystemPromptAddendum returns the text to append to the system prompt
// when at least one skill is loaded. Empty string if none. The format is
// compact: name, source, and one-line description.
func SystemPromptAddendum(skills []Skill) string {
	var sb strings.Builder
	for _, s := range skills {
		if sb.Len() == 0 {
			sb.WriteString("Available skills (use /<name> to invoke):\n")
		}
		desc := strings.TrimSpace(s.Description)
		if desc == "" {
			desc = "(no description)"
		}
		source := s.Source
		if source == "" {
			source = "unknown"
		}
		fmt.Fprintf(&sb, "- %s [%s]: %s\n", s.Name, source, desc)
	}
	return sb.String()
}

// FindByName returns the skill with the given name, or nil.
func FindByName(skills []Skill, name string) *Skill {
	for i := range skills {
		if skills[i].Name == name {
			return &skills[i]
		}
	}
	return nil
}

// Load discovers skills from all supported directories. Project skills win
// over global; native (.ling-agent) wins over compat (.claude, .agents).
// Malformed files are skipped, reporting the reason to warn (warn may be nil).
// The result is sorted by name.
func Load(cwd string, warn func(string)) []Skill {
	byName := map[string]Skill{}
	home, _ := os.UserHomeDir()

	for _, loc := range searchDirs(cwd, home) {
		for _, sk := range loadDir(loc.dir, loc.label, warn) {
			if _, dup := byName[sk.Name]; !dup {
				byName[sk.Name] = sk
			}
		}
	}

	out := make([]Skill, 0, len(byName))
	for _, sk := range byName {
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

type skillLocation struct {
	dir   string
	label string
}

func searchDirs(cwd, home string) []skillLocation {
	var out []skillLocation
	add := func(dir, label string) {
		if dir == "" {
			return
		}
		out = append(out, skillLocation{dir: dir, label: label})
	}
	// Native: project before global
	if cwd != "" {
		add(filepath.Join(cwd, ".ling-agent", "skills"), "project")
	}
	if home != "" {
		add(filepath.Join(home, ".ling-agent", "skills"), "global")
	}
	// Claude-compat: project before global
	if cwd != "" {
		add(filepath.Join(cwd, ".claude", "skills"), "project (claude)")
	}
	if home != "" {
		add(filepath.Join(home, ".claude", "skills"), "global (claude)")
	}
	// Agents-compat: project before global
	if cwd != "" {
		add(filepath.Join(cwd, ".agents", "skills"), "project (agents)")
	}
	if home != "" {
		add(filepath.Join(home, ".agents", "skills"), "global (agents)")
	}
	return out
}

// loadDir parses skills from a directory. Supports both directory-style
// (<name>/SKILL.md) and flat-style (<name>.md). A missing dir yields nothing.
func loadDir(dir, label string, warn func(string)) []Skill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // missing/unreadable dir is not an error
	}
	var out []Skill
	for _, e := range entries {
		if e.IsDir() {
			// Directory-style skill: <name>/SKILL.md
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if data, err := os.ReadFile(skillPath); err == nil {
				sk, err := parse(data, skillPath)
				if err != nil {
					warnf(warn, "skill %s: %v", skillPath, err)
					continue
				}
				if sk.Name == "" {
					sk.Name = e.Name()
				}
				sk.Source = label
				out = append(out, sk)
				continue
			}
			continue
		}
		// Flat-style: <name>.md
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			warnf(warn, "skill %s: %v", path, err)
			continue
		}
		sk, err := parse(data, path)
		if err != nil {
			warnf(warn, "skill %s: %v", path, err)
			continue
		}
		sk.Source = label
		out = append(out, sk)
	}
	return out
}

// parse splits frontmatter from body and validates required fields. The name
// defaults to the file's base name (sans .md) when the frontmatter omits it.
func parse(data []byte, path string) (Skill, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return Skill{}, err
	}
	var meta frontmatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return Skill{}, fmt.Errorf("invalid frontmatter: %w", err)
	}

	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	typ := strings.TrimSpace(meta.Type)
	switch typ {
	case "":
		typ = TypePrompt
	case TypePrompt, TypeCommand:
		// ok
	default:
		return Skill{}, fmt.Errorf("invalid type %q (want %q or %q)", typ, TypePrompt, TypeCommand)
	}

	return Skill{
		Name:        name,
		Description: strings.TrimSpace(meta.Description),
		Type:        typ,
		Tools:       meta.Tools,
		Body:        strings.TrimSpace(string(body)),
		Path:        path,
	}, nil
}

// splitFrontmatter separates a leading `---\n … \n---` YAML block from the body.
// A file with no frontmatter returns empty frontmatter and the whole content as
// body (a bare-Markdown skill is valid; its name comes from the filename).
func splitFrontmatter(data []byte) (front, body []byte, err error) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return nil, data, nil
	}
	// Drop the opening fence, then find the closing one at a line start.
	rest := s[strings.IndexByte(s, '\n')+1:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
	}
	front = []byte(rest[:idx])
	after := rest[idx+len("\n---"):]
	// Skip to the end of the closing fence line.
	if nl := strings.IndexByte(after, '\n'); nl >= 0 {
		after = after[nl+1:]
	} else {
		after = ""
	}
	return front, []byte(after), nil
}

func warnf(warn func(string), format string, args ...any) {
	if warn != nil {
		warn(fmt.Sprintf(format, args...))
	}
}
