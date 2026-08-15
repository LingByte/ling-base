package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSandbox_Type(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	assert.Equal(t, SandboxTypeLocal, s.Type())
}

func TestLocalSandbox_IsAvailable(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	assert.True(t, s.IsAvailable(nil))
}

func TestLocalSandbox_Cleanup(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	err := s.Cleanup(nil)
	assert.NoError(t, err)
}

func TestNewLocalSandbox_NilConfig(t *testing.T) {
	s := NewLocalSandbox(nil)
	require.NotNil(t, s)
	assert.Equal(t, SandboxTypeLocal, s.Type())
}

func TestLocalSandbox_Execute_NilConfig(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	_, err := s.Execute(nil, nil)
	assert.ErrorIs(t, err, ErrInvalidScript)
}

func TestLocalSandbox_GetInterpreter(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())

	tests := []struct {
		name string
		path string
		want string
	}{
		{"python", "/tmp/script.py", "python3"},
		{"python uppercase", "/tmp/SCRIPT.PY", "python3"},
		{"bash", "/tmp/run.sh", "bash"},
		{"bash extension", "/tmp/run.bash", "bash"},
		{"javascript", "/tmp/app.js", "node"},
		{"ruby", "/tmp/app.rb", "ruby"},
		{"perl", "/tmp/app.pl", "perl"},
		{"php", "/tmp/app.php", "php"},
		{"unknown ext", "/tmp/app.xyz", "sh"},
		{"no extension", "/tmp/Makefile", "sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.getInterpreter(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLocalSandbox_IsAllowedCommand_DefaultList(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())

	// Commands in the default list
	allowed := []string{"python", "python3", "node", "bash", "sh", "cat", "echo", "head", "tail", "grep", "sed", "awk", "sort", "uniq", "wc", "cut", "tr", "ls", "pwd", "date"}
	for _, cmd := range allowed {
		assert.True(t, s.isAllowedCommand(cmd), "expected %q to be allowed", cmd)
	}

	// Commands NOT in the default list
	denied := []string{"ruby", "perl", "php", "rm", "dd", "mkfs", "chmod", "chown", "curl", "wget"}
	for _, cmd := range denied {
		assert.False(t, s.isAllowedCommand(cmd), "expected %q to be denied", cmd)
	}
}

func TestLocalSandbox_IsAllowedCommand_CustomList(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowedCommands = []string{"python3", "ruby"}
	s := NewLocalSandbox(cfg)

	assert.True(t, s.isAllowedCommand("python3"))
	assert.True(t, s.isAllowedCommand("ruby"))
	// bash is in default list but not in custom list
	assert.False(t, s.isAllowedCommand("bash"))
	assert.False(t, s.isAllowedCommand("sh"))
}

func TestLocalSandbox_IsAllowedCommand_EmptyList(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowedCommands = []string{}
	s := NewLocalSandbox(cfg)

	// Empty custom list should fall back to defaults
	assert.True(t, s.isAllowedCommand("python3"))
	assert.True(t, s.isAllowedCommand("bash"))
	assert.False(t, s.isAllowedCommand("ruby"))
}

func TestLocalSandbox_BuildEnvironment_Basic(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	env := s.buildEnvironment(nil)

	// Should contain minimal environment
	assert.Contains(t, env, "PATH=/usr/local/bin:/usr/bin:/bin")
	assert.Contains(t, env, "HOME=/tmp")
	assert.Contains(t, env, "LANG=en_US.UTF-8")
	assert.Contains(t, env, "LC_ALL=en_US.UTF-8")
}

func TestLocalSandbox_BuildEnvironment_WithExtra(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	env := s.buildEnvironment(map[string]string{
		"FOO":        "bar",
		"MY_API_KEY": "secret",
		"LD_PRELOAD": "/evil.so", // dangerous, should be filtered
		"PATH":       "/evil",    // dangerous, should be filtered
	})

	// Safe vars should be present
	foundFOO := false
	foundAPIKey := false
	for _, e := range env {
		if e == "FOO=bar" {
			foundFOO = true
		}
		if e == "MY_API_KEY=secret" {
			foundAPIKey = true
		}
		// Dangerous vars should NOT be present
		assert.NotContains(t, e, "LD_PRELOAD")
		assert.NotContains(t, e, "PATH=/evil")
	}
	assert.True(t, foundFOO, "FOO should be in environment")
	assert.True(t, foundAPIKey, "MY_API_KEY should be in environment")
}

func TestLocalSandbox_ValidateScript_NotFound(t *testing.T) {
	s := NewLocalSandbox(DefaultConfig())
	err := s.validateScript("/nonexistent/path/script.sh")
	assert.ErrorIs(t, err, ErrScriptNotFound)
}

func TestLocalSandbox_ValidateScript_Directory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-validate-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	s := NewLocalSandbox(DefaultConfig())
	err = s.validateScript(tmpDir)
	assert.ErrorIs(t, err, ErrInvalidScript)
}

func TestLocalSandbox_ValidateScript_RelativePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-validate-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0755))

	// Change to the temp dir so the relative path resolves to an existing file
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmpDir))

	s := NewLocalSandbox(DefaultConfig())
	err = s.validateScript("test.sh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be absolute")
}

func TestLocalSandbox_ValidateScript_ValidAbsolute(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-validate-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0755))

	s := NewLocalSandbox(DefaultConfig())
	err = s.validateScript(scriptPath)
	assert.NoError(t, err)
}

func TestLocalSandbox_ValidateScript_AllowedPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-validate-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0755))

	// Allowed path includes the temp dir
	cfg := DefaultConfig()
	cfg.AllowedPaths = []string{tmpDir}
	s := NewLocalSandbox(cfg)
	err = s.validateScript(scriptPath)
	assert.NoError(t, err)
}

func TestLocalSandbox_ValidateScript_NotInAllowedPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-validate-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0755))

	// Allowed path does NOT include the temp dir
	cfg := DefaultConfig()
	cfg.AllowedPaths = []string{"/some/other/dir"}
	s := NewLocalSandbox(cfg)
	err = s.validateScript(scriptPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed paths")
}

func TestLocalSandbox_Execute_NotAllowedInterpreter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-exec-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a .rb script but don't allow ruby
	scriptPath := filepath.Join(tmpDir, "test.rb")
	require.NoError(t, os.WriteFile(scriptPath, []byte("puts 'hi'\n"), 0755))

	cfg := DefaultConfig()
	cfg.AllowedCommands = []string{"python3", "bash"} // no ruby
	s := NewLocalSandbox(cfg)

	_, err = s.Execute(nil, &ExecuteConfig{
		Script:  scriptPath,
		Timeout: 5e9,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interpreter not allowed")
}
