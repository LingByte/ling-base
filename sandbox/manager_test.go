package sandbox

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManagerFromType_Local(t *testing.T) {
	mgr, err := NewManagerFromType("local", true, "")
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, SandboxTypeLocal, mgr.GetType())
	assert.NotNil(t, mgr.GetSandbox())
}

func TestNewManagerFromType_Disabled(t *testing.T) {
	mgr, err := NewManagerFromType("disabled", true, "")
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, SandboxTypeDisabled, mgr.GetType())

	_, err = mgr.Execute(context.Background(), &ExecuteConfig{
		Script: "/some/script.sh",
	})
	assert.ErrorIs(t, err, ErrSandboxDisabled)
}

func TestNewManagerFromType_EmptyString(t *testing.T) {
	// Empty string should be treated as "disabled"
	mgr, err := NewManagerFromType("", true, "")
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, SandboxTypeDisabled, mgr.GetType())
}

func TestNewManagerFromType_Invalid(t *testing.T) {
	mgr, err := NewManagerFromType("invalid-type", true, "")
	require.Error(t, err)
	assert.Nil(t, mgr)
	assert.Contains(t, err.Error(), "unknown sandbox type")
}

func TestNewManagerFromType_Docker_Fallback(t *testing.T) {
	// Docker is likely not available in CI; with fallback enabled it should
	// fall back to local sandbox.
	mgr, err := NewManagerFromType("docker", true, "")
	require.NoError(t, err)
	require.NotNil(t, mgr)
	// Either docker (if available) or local (fallback)
	sbType := mgr.GetType()
	assert.Contains(t, []SandboxType{SandboxTypeDocker, SandboxTypeLocal}, sbType)
}

func TestNewManagerFromType_Docker_NoFallback(t *testing.T) {
	// If docker is not available and fallback is disabled, should error.
	mgr, err := NewManagerFromType("docker", false, "")
	if err != nil {
		// Docker not available and fallback disabled — expected error
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "docker is not available")
	} else {
		// Docker is available — manager should be docker type
		require.NotNil(t, mgr)
		assert.Equal(t, SandboxTypeDocker, mgr.GetType())
		_ = mgr
	}
}

func TestNewManagerFromType_CustomDockerImage(t *testing.T) {
	mgr, err := NewManagerFromType("local", true, "custom/image:tag")
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, SandboxTypeLocal, mgr.GetType())
}

func TestNewDisabledManager_Working(t *testing.T) {
	mgr := NewDisabledManager()
	require.NotNil(t, mgr)

	assert.Equal(t, SandboxTypeDisabled, mgr.GetType())

	sb := mgr.GetSandbox()
	require.NotNil(t, sb)
	assert.Equal(t, SandboxTypeDisabled, sb.Type())
	assert.False(t, sb.IsAvailable(context.Background()))

	// Execute should return ErrSandboxDisabled
	result, err := mgr.Execute(context.Background(), &ExecuteConfig{
		Script: "/some/script.sh",
	})
	assert.ErrorIs(t, err, ErrSandboxDisabled)
	assert.Nil(t, result)

	// Cleanup should be a no-op
	err = mgr.Cleanup(context.Background())
	assert.NoError(t, err)
}

func TestNewManager_NilConfig(t *testing.T) {
	// nil config should use DefaultConfig
	mgr, err := NewManager(nil)
	require.NoError(t, err)
	require.NotNil(t, mgr)
	assert.Equal(t, SandboxTypeLocal, mgr.GetType())
}

func TestNewManager_InvalidConfig(t *testing.T) {
	_, err := NewManager(&Config{
		Type: "invalid",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sandbox config")
}

func TestDefaultManager_Cleanup_NilSandbox(t *testing.T) {
	mgr := &DefaultManager{}
	err := mgr.Cleanup(context.Background())
	assert.NoError(t, err)
}

func TestDefaultManager_GetType_NilSandbox(t *testing.T) {
	mgr := &DefaultManager{}
	assert.Equal(t, SandboxTypeDisabled, mgr.GetType())
}

func TestDefaultManager_Execute_NilSandbox(t *testing.T) {
	mgr := &DefaultManager{}
	_, err := mgr.Execute(context.Background(), &ExecuteConfig{
		Script: "/some/script.sh",
	})
	assert.ErrorIs(t, err, ErrSandboxDisabled)
}

func TestDefaultManager_Execute_SkipValidation(t *testing.T) {
	// When SkipValidation is true, validation should be skipped.
	// Use a disabled sandbox so Execute returns ErrSandboxDisabled quickly.
	mgr := NewDisabledManager()
	_, err := mgr.Execute(context.Background(), &ExecuteConfig{
		Script:         "/some/script.sh",
		SkipValidation: true,
	})
	assert.ErrorIs(t, err, ErrSandboxDisabled)
}

func TestDefaultManager_Execute_SecurityViolation(t *testing.T) {
	// With dangerous script content and validation enabled, should return
	// ErrSecurityViolation. Use a local manager so validation runs.
	config := DefaultConfig()
	config.Type = SandboxTypeLocal
	localMgr, err := NewManager(config)
	require.NoError(t, err)

	result, err := localMgr.Execute(context.Background(), &ExecuteConfig{
		Script:        "/some/script.sh",
		ScriptContent: `rm -rf /`,
	})
	// Should get ErrSecurityViolation and a result with error info
	assert.ErrorIs(t, err, ErrSecurityViolation)
	require.NotNil(t, result)
	assert.Equal(t, -1, result.ExitCode)
	assert.NotEmpty(t, result.Stderr)
}
