package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
)

func writeReleaseFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAliasedRelocatedImports(t *testing.T) {
	dir := t.TempDir()
	writeReleaseFixture(t, dir, "app.go", `package app
import (
 storage "github.com/LingByte/ling-base/common/stores/local"
 _ "github.com/LingByte/ling-base/providers/voice/realtime/openai"
 asr "github.com/LingByte/ling-base/providers/voice/recognizer"
)
`)
	want := []string{
		"github.com/LingByte/ling-base/common/stores/local",
		"github.com/LingByte/ling-base/providers/voice/realtime/openai",
		"github.com/LingByte/ling-base/providers/voice/recognizer",
	}
	if got := resolveLingBaseImports(dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("imports = %v, want %v", got, want)
	}
}

func TestRelocatedFirstReleaseRequirements(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	dir := t.TempDir()
	writeReleaseFixture(t, dir, "go.mod", "module example.com/app\n\ngo 1.26.2\n")
	modules := []string{
		"github.com/LingByte/ling-base/common/stores",
		"github.com/LingByte/ling-base/common/stores/local",
		"github.com/LingByte/ling-base/providers/voice/recognizer",
		"github.com/LingByte/ling-base/providers/voice/realtime/openai",
	}
	if err := goGetLingBaseModules(dir, modules); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, module := range modules {
		if !strings.Contains(string(content), module+" v0.1.0") {
			t.Errorf("incorrect first-release requirement for %s:\n%s", module, content)
		}
	}
	if err := goGetLingBaseModules(dir, []string{"unknown/module"}); err == nil {
		t.Fatal("unknown module versions must not be silently skipped")
	}
}

func TestPrepareAndExtractRelocatedEmbeddedSource(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"LICENSE":                                   "MIT fixture license\n",
		"go.work":                                   "go 1.26.2\n",
		"lingcli/go.mod":                            "module github.com/LingByte/ling-base/lingcli\n",
		"lingcli/embed_source/.gitkeep":             "",
		"common/stores/go.mod":                      "module github.com/LingByte/ling-base/common/stores\nrequire golang.org/x/text v0.1.0\n",
		"common/stores/stores.go":                   "package stores\n",
		"common/stores/local/local.go":              "package local\nimport _ \"github.com/LingByte/ling-base/common/stores\"\n",
		"common/stores/local/local_test.go":         "package local\n",
		"providers/voice/recognizer/go.mod":         "module github.com/LingByte/ling-base/providers/voice/recognizer\nrequire (\nmodernc.org/sqlite v1.2.3\ngolang.org/x/text v0.2.0\n)\nreplace modernc.org/sqlite => modernc.org/sqlite v9.0.0\n",
		"providers/voice/recognizer/recognizer.go":  "package recognizer\n",
		"providers/voice/recognizer/local/local.go": "package local\nimport _ \"github.com/LingByte/ling-base/providers/voice/recognizer\"\n",
		"apidocs/assets/style.css":                  "body {}\n",
		"example/demo.go":                           "package main\n",
		"scripts/tool.go":                           "package main\n",
	} {
		writeReleaseFixture(t, root, name, content)
	}
	if err := prepareEmbeddedSource(root); err != nil {
		t.Fatal(err)
	}
	source := os.DirFS(filepath.Join(root, "lingcli"))
	requires, err := embeddedSubmoduleRequires(source)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"golang.org/x/text": "v0.2.0", "modernc.org/sqlite": "v1.2.3"}
	if !reflect.DeepEqual(requires, want) {
		t.Fatalf("embedded requires = %v, want %v", requires, want)
	}
	for _, excluded := range []string{"example", "scripts", "common/stores/go.mod", "common/stores/local/local_test.go"} {
		if _, err := os.Stat(filepath.Join(root, "lingcli", embeddedSourceRoot, excluded)); !os.IsNotExist(err) {
			t.Errorf("excluded source %s exists: %v", excluded, err)
		}
	}
	original, err := os.ReadFile(filepath.Join(root, "lingcli", embeddedSourceRoot, "common/stores/local/local.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(original), `"github.com/LingByte/ling-base/common/stores"`) {
		t.Fatal("preparation must preserve imports until extraction")
	}
	target := t.TempDir()
	if err := extractEmbeddedSourceFrom(source, &ProjectSpec{Module: "example.com/app"}, target); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"pkg/LICENSE":                                   "MIT fixture license",
		"pkg/common/stores/local/local.go":              "example.com/app/pkg/common/stores",
		"pkg/providers/voice/recognizer/local/local.go": "example.com/app/pkg/providers/voice/recognizer",
		"pkg/apidocs/assets/style.css":                  "body {}",
	} {
		content, err := os.ReadFile(filepath.Join(target, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), want) {
			t.Errorf("%s does not contain %q: %s", name, want, content)
		}
	}
}

func TestCloneSourceRefMatchesCLIRelease(t *testing.T) {
	if got := lingBaseSourceRef("v0.33.0"); got != "lingcli/v0.33.0" {
		t.Fatalf("source ref = %q", got)
	}
	if got := lingBaseSourceRef("dev"); got != "main" {
		t.Fatalf("development source ref = %q", got)
	}
}

func TestCLIVersionFromReleaseAndGoInstall(t *testing.T) {
	for _, tc := range []struct {
		injected string
		info     *debug.BuildInfo
		want     string
	}{
		{"v0.33.0", nil, "v0.33.0"},
		{"dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.33.0"}}, "v0.33.0"},
		{"dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev"},
		{"", nil, "dev"},
	} {
		if got := cliVersionFromBuildInfo(tc.injected, tc.info); got != tc.want {
			t.Errorf("CLI version = %q, want %q", got, tc.want)
		}
	}
}
