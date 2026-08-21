package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAndMatch(t *testing.T) {
	g := Parse(lines("# comment", "", ".terraform/", ".terragrunt-cache/", "node_modules/", "*.log", "/build", "!keep.log"))

	cases := []struct {
		rel   string
		isDir bool
		want  bool
	}{
		{".terraform", true, true},
		{".terraform/providers/x", false, false},
		{".terragrunt-cache", true, true},
		{"modules/.terragrunt-cache", true, true},
		{"node_modules", true, true},
		{"src/node_modules/pkg", true, true},
		{"debug.log", false, true},
		{"keep.log", false, false}, // re-included by negation
		{"build", true, true},      // anchored
		{"sub/build", true, false}, // anchored: only at root
		{"main.tf", false, false},
		{"src/app.go", false, false},
	}
	for _, c := range cases {
		if got := g.Match(c.rel, c.isDir); got != c.want {
			t.Errorf("Match(%q, dir=%v) = %v, want %v", c.rel, c.isDir, got, c.want)
		}
	}
}

func TestEmptyIgnoresNothing(t *testing.T) {
	g := Parse("")
	if g.Match("anything", false) || g.Match("dir", true) {
		t.Fatal("empty matcher should ignore nothing")
	}
}

func TestStackHonorsNestedGitignore(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opencode := filepath.Join(root, ".opencode")
	if err := os.MkdirAll(opencode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opencode, ".gitignore"), []byte("node_modules\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewStack(root)
	if s.Match("node_modules", true) {
		t.Fatal("root-level node_modules should not be ignored by an unloaded nested rule")
	}
	s.Push(opencode, ".opencode")
	if !s.Match(".opencode/node_modules", true) {
		t.Fatal("nested .opencode/.gitignore should ignore .opencode/node_modules")
	}
	if !s.Match(".opencode/node_modules/zod/src/v3/tests/pipeline.test.ts", false) {
		t.Fatal("files under nested-ignored node_modules should be ignored")
	}
	if s.Match(".opencode/config.json", false) {
		t.Fatal(".opencode/config.json should not be ignored")
	}
	if !s.Match("build", true) {
		t.Fatal("root build/ rule should still apply while nested frame is pushed")
	}
	s.Pop()
	if s.Match(".opencode/node_modules", true) {
		t.Fatal("after popping, the nested rule should no longer be in scope")
	}
}

func TestStackNestedNegationReincludes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".gitignore"), []byte("!keep.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStack(root)
	s.Push(sub, "sub")
	if s.Match("sub/keep.log", false) {
		t.Fatal("nested !keep.log should re-include a *.log excluded by root")
	}
	if !s.Match("sub/other.log", false) {
		t.Fatal("sub/other.log should still be excluded by root *.log")
	}
}

func lines(ls ...string) string {
	out := ""
	for i, l := range ls {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
