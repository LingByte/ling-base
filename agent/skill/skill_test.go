package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFrontmatterAndBody(t *testing.T) {
	sk, err := parse([]byte("---\nname: review\ndescription: Review the diff\ntype: prompt\ntools: [Bash, Read]\n---\nReview this: $ARGUMENTS\n"), "review.md")
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "review" || sk.Description != "Review the diff" || sk.Type != TypePrompt {
		t.Errorf("meta = %+v", sk)
	}
	if len(sk.Tools) != 2 || sk.Tools[0] != "Bash" {
		t.Errorf("tools = %v", sk.Tools)
	}
	if sk.Body != "Review this: $ARGUMENTS" {
		t.Errorf("body = %q", sk.Body)
	}
}

func TestParseDefaults(t *testing.T) {
	// No frontmatter: name from filename, type defaults to prompt.
	sk, err := parse([]byte("just a body"), "/x/quickfix.md")
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "quickfix" || sk.Type != TypePrompt || sk.Body != "just a body" {
		t.Errorf("got %+v", sk)
	}
}

func TestParseRejectsBadType(t *testing.T) {
	if _, err := parse([]byte("---\nname: x\ntype: bogus\n---\nbody"), "x.md"); err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestParseUnterminatedFrontmatter(t *testing.T) {
	if _, err := parse([]byte("---\nname: x\nno closing fence"), "x.md"); err == nil {
		t.Error("expected error for unterminated frontmatter")
	}
}

func TestRenderArguments(t *testing.T) {
	sk := Skill{Body: "Do $ARGUMENTS now"}
	if got := sk.Render("the thing"); got != "Do the thing now" {
		t.Errorf("render = %q", got)
	}
	// No placeholder + args → appended.
	sk2 := Skill{Body: "Standing instructions."}
	if got := sk2.Render("extra"); got != "Standing instructions.\n\nextra" {
		t.Errorf("append = %q", got)
	}
	// No placeholder + no args → unchanged.
	if got := sk2.Render(""); got != "Standing instructions." {
		t.Errorf("unchanged = %q", got)
	}
}

func TestLoadProjectOverlaysHome(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	write(t, filepath.Join(home, ".ling-agent", "skills", "review.md"), "---\nname: review\ndescription: home version\n---\nhome body")
	write(t, filepath.Join(home, ".ling-agent", "skills", "deploy.md"), "---\nname: deploy\ndescription: deploy\n---\ndeploy")
	// Project overrides review and adds plan.
	write(t, filepath.Join(cwd, ".ling-agent", "skills", "review.md"), "---\nname: review\ndescription: project version\n---\nproject body")
	write(t, filepath.Join(cwd, ".ling-agent", "skills", "plan.md"), "---\nname: plan\ndescription: plan\n---\nplan")
	// Malformed file is skipped (not fatal).
	write(t, filepath.Join(cwd, ".ling-agent", "skills", "bad.md"), "---\ntype: nonsense\n---\nx")

	var warnings []string
	got := Load(cwd, func(s string) { warnings = append(warnings, s) })

	byName := map[string]Skill{}
	for _, sk := range got {
		byName[sk.Name] = sk
	}
	if byName["review"].Description != "project version" || byName["review"].Body != "project body" {
		t.Errorf("project should win: %+v", byName["review"])
	}
	if _, ok := byName["deploy"]; !ok {
		t.Error("home-only skill deploy missing")
	}
	if _, ok := byName["plan"]; !ok {
		t.Error("project-only skill plan missing")
	}
	if _, ok := byName["bad"]; ok {
		t.Error("malformed skill should have been skipped")
	}
	if len(warnings) == 0 {
		t.Error("expected a warning for the malformed skill")
	}
}

func TestLoadCompatPaths(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Native: directory-style SKILL.md
	write(t, filepath.Join(cwd, ".ling-agent", "skills", "native", "SKILL.md"),
		"---\nname: native\ndescription: native skill\n---\nnative body")
	// Claude-compat: directory-style
	write(t, filepath.Join(cwd, ".claude", "skills", "claude-skill", "SKILL.md"),
		"---\nname: claude-skill\ndescription: from claude\n---\nclaude body")
	// Agents-compat: directory-style
	write(t, filepath.Join(home, ".agents", "skills", "agent-skill", "SKILL.md"),
		"---\nname: agent-skill\ndescription: from agents\n---\nagents body")

	got := Load(cwd, nil)
	byName := map[string]Skill{}
	for _, sk := range got {
		byName[sk.Name] = sk
	}

	if _, ok := byName["native"]; !ok {
		t.Error("native directory-style skill missing")
	}
	if _, ok := byName["claude-skill"]; !ok {
		t.Error("claude-compat skill missing")
	}
	if _, ok := byName["agent-skill"]; !ok {
		t.Error("agents-compat skill missing")
	}
	if byName["native"].Source != "project" {
		t.Errorf("native source = %q, want project", byName["native"].Source)
	}
}

func TestSystemPromptAddendum(t *testing.T) {
	skills := []Skill{
		{Name: "review", Description: "Review the diff", Source: "project"},
		{Name: "deploy", Description: "Deploy the app", Source: "global"},
	}
	got := SystemPromptAddendum(skills)
	if !strings.Contains(got, "review") || !strings.Contains(got, "deploy") {
		t.Errorf("addendum should list both skills: %s", got)
	}
	if !strings.Contains(got, "Review the diff") {
		t.Error("addendum should include descriptions")
	}
}

func TestFindByName(t *testing.T) {
	skills := []Skill{{Name: "a"}, {Name: "b"}}
	if s := FindByName(skills, "b"); s == nil || s.Name != "b" {
		t.Error("FindByName should find b")
	}
	if s := FindByName(skills, "c"); s != nil {
		t.Error("FindByName should return nil for missing")
	}
}
