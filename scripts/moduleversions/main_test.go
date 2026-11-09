package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateCurrentModuleTags(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".", "lingcli", "common/stores", "common/stores/local", "providers/voice/recognizer"} {
		module := moduleRoot
		if dir != "." {
			module += "/" + dir
		}
		writeFixture(t, root, filepath.Join(dir, "go.mod"), "module "+module+"\n")
	}
	writeFixture(t, root, "lingcli/embed_source/_source/ignored/go.mod", "module invalid\n")
	content, err := generate(root, []string{
		"v0.18.0", "lingcli/v0.32.5", "stores/v0.1.5", "stores/local/v0.1.4",
		"voice/recognizer/v0.1.2", "common/stores/v0.1.9", "common/stores/v0.1.10",
		"common/stores/v0.2.0-rc.1", "common/stores/local/v9.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	versions := make(map[string]string)
	for _, match := range regexp.MustCompile(`"([^"]+)":\s+"([^"]+)"`).FindAllStringSubmatch(string(content), -1) {
		versions[match[1]] = match[2]
	}
	for module, want := range map[string]string{
		moduleRoot:                                 "v0.18.0",
		moduleRoot + "/common/stores":              "v0.1.10",
		moduleRoot + "/common/stores/local":        "v9.0.0",
		moduleRoot + "/providers/voice/recognizer": "v0.1.0",
	} {
		if got := versions[module]; got != want {
			t.Errorf("%s = %q, want %q", module, got, want)
		}
	}
	if len(versions) != 4 {
		t.Errorf("unexpected modules in generated catalog: %v", versions)
	}
}

func TestRequireLibraryTagsBeforeCLIRelease(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "common/stores/go.mod", "module "+moduleRoot+"/common/stores\n")
	if _, err := generateWithPolicy(root, []string{"stores/v0.1.5"}, true); err == nil {
		t.Fatal("legacy tags must not satisfy a relocated module's release check")
	}
	if _, err := generateWithPolicy(root, []string{"common/stores/v0.1.0"}, true); err != nil {
		t.Fatal(err)
	}
}

func TestRejectMismatchedModulePath(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "common/stores/go.mod", "module "+moduleRoot+"/stores\n")
	if _, err := generate(root, nil); err == nil {
		t.Fatal("expected mismatched module directory to fail")
	}
}

func fixtureCommand(t *testing.T, root, program string, args ...string) string {
	t.Helper()
	cmd := exec.Command(program, args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", program, args, err, output)
	}
	return string(output)
}

func releaseFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for release-script tests")
	}
	script, err := os.ReadFile(filepath.Join("..", "release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "scripts/release.sh", string(script))
	for _, dir := range []string{".", "common/stores", "common/stores/local", "providers/voice/recognizer", "lingcli"} {
		module := moduleRoot
		if dir != "." {
			module += "/" + dir
		}
		writeFixture(t, root, filepath.Join(dir, "go.mod"), "module "+module+"\n")
	}
	fixtureCommand(t, root, "git", "init", "-q")
	fixtureCommand(t, root, "git", "config", "user.name", "Release Tests")
	fixtureCommand(t, root, "git", "config", "user.email", "tests@example.invalid")
	fixtureCommand(t, root, "git", "config", "core.hooksPath", "/dev/null")
	fixtureCommand(t, root, "git", "add", ".")
	fixtureCommand(t, root, "git", "-c", "commit.gpgsign=false", "commit", "-qm", "Fixture")
	return root
}

func TestReleaseDryRunUsesNewPathAndExactStableTag(t *testing.T) {
	root := releaseFixture(t)
	for _, tag := range []string{"stores/v99.0.0", "common/stores/v0.1.9", "common/stores/v0.2.0-rc.1", "common/stores/local/v99.0.0"} {
		fixtureCommand(t, root, "git", "tag", tag)
	}
	before := fixtureCommand(t, root, "git", "tag", "--list")
	output := fixtureCommand(t, root, "bash", "scripts/release.sh", "--pkg", "./common/stores/", "--dry-run")
	if !strings.Contains(output, "common/stores/v0.1.10") {
		t.Fatalf("wrong release tag: %s", output)
	}
	output = fixtureCommand(t, root, "bash", "scripts/release.sh", "--pkg", "providers/voice/recognizer", "--dry-run")
	if !strings.Contains(output, "providers/voice/recognizer/v0.1.0") {
		t.Fatalf("wrong first-release tag: %s", output)
	}
	if after := fixtureCommand(t, root, "git", "tag", "--list"); after != before {
		t.Fatal("dry-run changed tags")
	}
}

func TestReleaseRefusesDirtyTreeAndOldPath(t *testing.T) {
	root := releaseFixture(t)
	writeFixture(t, root, "pending.txt", "uncommitted\n")
	for _, dir := range []string{"common/stores", "stores"} {
		cmd := exec.Command("bash", "scripts/release.sh", "--pkg", dir)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err == nil {
			t.Fatalf("expected rejection for %s: %s", dir, output)
		}
	}
	if tags := fixtureCommand(t, root, "git", "tag", "--list"); tags != "" {
		t.Fatalf("unexpected tag created: %s", tags)
	}
}

func TestReleaseAllExcludesCLIAndEmbeddedModules(t *testing.T) {
	root := releaseFixture(t)
	writeFixture(t, root, "lingcli/embed_source/_source/common/stores/go.mod", "module invalid\n")
	output := fixtureCommand(t, root, "bash", "scripts/release.sh", "--all", "--dry-run")
	if strings.Contains(output, "embed_source/") || strings.Contains(output, "[dry-run] lingcli/") {
		t.Fatalf("release-all included build artifacts or CLI: %s", output)
	}
	for _, tag := range []string{"common/stores/v0.1.0", "providers/voice/recognizer/v0.1.0"} {
		if !strings.Contains(output, tag) {
			t.Errorf("missing planned tag %s: %s", tag, output)
		}
	}
}
