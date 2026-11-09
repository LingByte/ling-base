package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStorageTemplateModulePaths(t *testing.T) {
	_, content, err := RenderTemplate("web-api/pkg/storage/storage.go.tmpl", &TemplateData{
		Module: "example.com/app",
		Year:   2026,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"github.com/LingByte/ling-base/common/stores",
		"github.com/LingByte/ling-base/common/stores/local",
	} {
		if !strings.Contains(content, `"`+path+`"`) {
			t.Errorf("storage template is missing import %q", path)
		}
	}

	full := rewriteTemplateImports(content, "example.com/app")
	for _, path := range []string{
		"example.com/app/pkg/common/stores",
		"example.com/app/pkg/common/stores/local",
	} {
		if !strings.Contains(full, `"`+path+`"`) {
			t.Errorf("full-mode storage template is missing import %q", path)
		}
	}
	if strings.Contains(full, "github.com/LingByte/ling-base") {
		t.Error("full-mode storage template still imports ling-base")
	}
}

func TestResolveRelocatedLingBaseImports(t *testing.T) {
	dir := t.TempDir()
	source := `package app
import (
	"github.com/LingByte/ling-base/common/stores"
	"github.com/LingByte/ling-base/common/stores/local"
	"github.com/LingByte/ling-base/providers/voice/vad"
	"github.com/LingByte/ling-base/providers/voice/voiceclone"
	"github.com/LingByte/ling-base/providers/voice/voiceprint"
)
`
	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"github.com/LingByte/ling-base/common/stores",
		"github.com/LingByte/ling-base/common/stores/local",
		"github.com/LingByte/ling-base/providers/voice/vad",
		"github.com/LingByte/ling-base/providers/voice/voiceclone",
		"github.com/LingByte/ling-base/providers/voice/voiceprint",
	}
	if got := resolveLingBaseImports(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("resolveLingBaseImports() = %v, want %v", got, want)
	}
}

func TestCopyModuleSourceRelocatedPackages(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	files := map[string]string{
		"common/stores/stores.go":                       "package stores\n",
		"common/stores/local/local.go":                  "package local\nimport _ \"github.com/LingByte/ling-base/common/stores\"\n",
		"providers/voice/recognizer/recognizer.go":      "package recognizer\n",
		"providers/voice/recognizer/local/local.go":     "package local\nimport _ \"github.com/LingByte/ling-base/providers/voice/recognizer\"\n",
		"common/stores/go.mod":                          "module github.com/LingByte/ling-base/common/stores\n",
		"providers/voice/recognizer/go.mod":             "module github.com/LingByte/ling-base/providers/voice/recognizer\n",
		"providers/voice/recognizer/recognizer_test.go": "package recognizer\n",
		"example/cmd/demo/main.go":                      "package main\n",
	}
	for name, content := range files {
		path := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copyModuleSource(source, target, "example.com/app"); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(target, filepath.FromSlash(name))
		if strings.HasPrefix(name, "example/") || strings.HasSuffix(name, "go.mod") || strings.HasSuffix(name, "_test.go") {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("excluded file %s: expected not to exist, got %v", name, err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		want := strings.ReplaceAll(content, "github.com/LingByte/ling-base", "example.com/app/pkg")
		if string(got) != want {
			t.Errorf("copied %s = %q, want %q", name, got, want)
		}
	}
}
