package sandbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInterpreter(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"python", "script.py", "python3"},
		{"python uppercase", "SCRIPT.PY", "python3"},
		{"bash", "run.sh", "bash"},
		{"bash extension", "run.bash", "bash"},
		{"javascript", "app.js", "node"},
		{"ruby", "app.rb", "ruby"},
		{"perl", "app.pl", "perl"},
		{"unknown ext", "app.xyz", "sh"},
		{"no extension", "Makefile", "sh"},
		{"php not supported in docker", "app.php", "sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInterpreter(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDockerSandbox_Type(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	assert.Equal(t, SandboxTypeDocker, s.Type())
}

func TestNewDockerSandbox_NilConfig(t *testing.T) {
	s := NewDockerSandbox(nil)
	require.NotNil(t, s)
	assert.Equal(t, SandboxTypeDocker, s.Type())
	assert.Equal(t, DefaultDockerImage, s.config.DockerImage)
}

func TestNewDockerSandbox_EmptyImage(t *testing.T) {
	cfg := &Config{Type: SandboxTypeDocker, DockerImage: ""}
	s := NewDockerSandbox(cfg)
	assert.Equal(t, DefaultDockerImage, s.config.DockerImage)
}

func TestDockerSandbox_Cleanup(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	err := s.Cleanup(context.Background())
	assert.NoError(t, err)
}

func TestDockerSandbox_Execute_NilConfig(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	_, err := s.Execute(context.Background(), nil)
	assert.ErrorIs(t, err, ErrInvalidScript)
}

func TestDockerSandbox_BuildDockerArgs_Basic(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script: "/workspace/test.py",
		Args:   []string{"arg1", "arg2"},
	})

	// Check basic structure
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, "--user")
	assert.Contains(t, args, "1000:1000")
	assert.Contains(t, args, "--cap-drop")
	assert.Contains(t, args, "ALL")
	assert.Contains(t, args, "--network")
	assert.Contains(t, args, "none")
	assert.Contains(t, args, "--pids-limit")
	assert.Contains(t, args, "100")
	assert.Contains(t, args, "--security-opt")
	assert.Contains(t, args, "no-new-privileges")

	// Check image is included
	assert.Contains(t, args, DefaultDockerImage)

	// Check interpreter and script name
	assert.Contains(t, args, "python3")
	assert.Contains(t, args, "test.py")

	// Check args are appended
	assert.Contains(t, args, "arg1")
	assert.Contains(t, args, "arg2")
}

func TestDockerSandbox_BuildDockerArgs_ReadOnlyRootfs(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script:         "/workspace/test.py",
		ReadOnlyRootfs: true,
	})
	assert.Contains(t, args, "--read-only")
	assert.Contains(t, args, "--tmpfs")
}

func TestDockerSandbox_BuildDockerArgs_AllowNetwork(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script:       "/workspace/test.py",
		AllowNetwork: true,
	})
	// When network is allowed, --network none should NOT be present
	for _, a := range args {
		assert.NotEqual(t, "none", a)
	}
}

func TestDockerSandbox_BuildDockerArgs_MemoryLimit(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script:      "/workspace/test.py",
		MemoryLimit: 512 * 1024 * 1024,
	})
	assert.Contains(t, args, "--memory")
	assert.Contains(t, args, "--memory-swap")
}

func TestDockerSandbox_BuildDockerArgs_CPULimit(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script:   "/workspace/test.py",
		CPULimit: 2.0,
	})
	assert.Contains(t, args, "--cpus")
}

func TestDockerSandbox_BuildDockerArgs_EnvVars(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	args := s.buildDockerArgs(&ExecuteConfig{
		Script: "/workspace/test.py",
		Env: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	})
	// Check -e flags are present
	foundE := false
	for i, a := range args {
		if a == "-e" {
			foundE = true
			// Next element should be KEY=VALUE
			if i+1 < len(args) {
				envStr := args[i+1]
				assert.Contains(t, []string{"FOO=bar", "BAZ=qux"}, envStr)
			}
		}
	}
	assert.True(t, foundE, "expected -e flags in docker args")
}

func TestDockerSandbox_BuildDockerArgs_DifferentScripts(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())

	tests := []struct {
		name        string
		script      string
		interpreter string
	}{
		{"python", "/workspace/app.py", "python3"},
		{"bash", "/workspace/app.sh", "bash"},
		{"node", "/workspace/app.js", "node"},
		{"ruby", "/workspace/app.rb", "ruby"},
		{"perl", "/workspace/app.pl", "perl"},
		{"unknown", "/workspace/app.xyz", "sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := s.buildDockerArgs(&ExecuteConfig{
				Script: tt.script,
			})
			assert.Contains(t, args, tt.interpreter)
		})
	}
}

func TestDockerSandbox_ImageExists(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	// This will actually call docker; if docker is not available it returns false.
	// We just verify it doesn't panic.
	ctx := context.Background()
	_ = s.ImageExists(ctx)
}

func TestDockerSandbox_EnsureImage(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	ctx := context.Background()
	// EnsureImage calls ImageExists then potentially docker pull.
	// If docker is not available, it will return an error. We just verify
	// it doesn't panic and returns either nil or an error.
	_ = s.EnsureImage(ctx)
}

func TestDockerSandbox_IsAvailable(t *testing.T) {
	s := NewDockerSandbox(DefaultConfig())
	ctx := context.Background()
	// Just verify it doesn't panic; result depends on docker availability.
	_ = s.IsAvailable(ctx)
}
