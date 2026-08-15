// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	originalNonExistent := os.Getenv("NON_EXISTENT_ENV_KEY")
	originalTest := os.Getenv("TEST_ENV_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_ENV_KEY")
		} else {
			os.Setenv("NON_EXISTENT_ENV_KEY", originalNonExistent)
		}
		if originalTest == "" {
			os.Unsetenv("TEST_ENV_KEY")
		} else {
			os.Setenv("TEST_ENV_KEY", originalTest)
		}
	}()

	os.Unsetenv("NON_EXISTENT_ENV_KEY")

	value := GetEnv("NON_EXISTENT_ENV_KEY")
	assert.Equal(t, "", value)

	os.Setenv("TEST_ENV_KEY", "test_env_value")

	value = GetEnv("TEST_ENV_KEY")
	assert.Equal(t, "test_env_value", value)
}

func TestLookupEnv(t *testing.T) {
	os.Unsetenv("TEST_ENV_KEY_FROM_FILE")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")

	envContent := `
# This is a comment
TEST_ENV_KEY_FROM_FILE=test_value_from_file
ANOTHER_KEY=another_value
INVALID_LINE
`
	err := os.WriteFile(".env", []byte(envContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(".env")

	PurgeEnvCacheForTest()

	value, found := LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "test_value_from_file", value)

	os.Setenv("TEST_ENV_KEY_FROM_FILE", "test_value_from_env")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")

	value, found = LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "test_value_from_env", value, "环境变量应该优先于.env文件中的值")

	os.Setenv("TEST_ENV_KEY_FROM_FILE", "")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")
	value, found = LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "", value)

	os.Unsetenv("NON_EXISTENT_KEY")
	defer os.Unsetenv("NON_EXISTENT_KEY")

	value, found = LookupEnv("NON_EXISTENT_KEY")
	assert.False(t, found)
	assert.Equal(t, "", value)
}

func TestLookupEnv_SearchesUpwardsAndTrimsQuotes(t *testing.T) {
	os.Unsetenv("SIP_TRANSFER_PORT")
	defer os.Unsetenv("SIP_TRANSFER_PORT")

	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	envContent := `
SIP_TRANSFER_PORT="50400"
SOME_OTHER_KEY=abc
`
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	PurgeEnvCacheForTest()

	v, ok := LookupEnv("SIP_TRANSFER_PORT")
	assert.True(t, ok)
	assert.Equal(t, "50400", v)
}

func TestLookupEnv_SupportsExportAndInlineComment(t *testing.T) {
	os.Unsetenv("SIP_TRANSFER_PORT")
	defer os.Unsetenv("SIP_TRANSFER_PORT")

	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	envContent := `
export SIP_TRANSFER_PORT=50400 # comment here
`
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	PurgeEnvCacheForTest()

	v, ok := LookupEnv("SIP_TRANSFER_PORT")
	assert.True(t, ok)
	assert.Equal(t, "50400", v)
}

func TestLookupEnv_DoesNotTruncateHashInsideValue(t *testing.T) {
	os.Unsetenv("DSN")
	defer os.Unsetenv("DSN")

	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	envContent := "DSN=root:pass##word@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4\n"
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	PurgeEnvCacheForTest()

	v, ok := LookupEnv("DSN")
	assert.True(t, ok)
	assert.Equal(t, "root:pass##word@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4", v)
}

func TestGetBoolEnv(t *testing.T) {
	originalNonExistent := os.Getenv("NON_EXISTENT_BOOL_KEY")
	originalTrue := os.Getenv("BOOL_TEST_KEY")
	originalFalse := os.Getenv("BOOL_FALSE_TEST_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_BOOL_KEY")
		} else {
			os.Setenv("NON_EXISTENT_BOOL_KEY", originalNonExistent)
		}
		if originalTrue == "" {
			os.Unsetenv("BOOL_TEST_KEY")
		} else {
			os.Setenv("BOOL_TEST_KEY", originalTrue)
		}
		if originalFalse == "" {
			os.Unsetenv("BOOL_FALSE_TEST_KEY")
		} else {
			os.Setenv("BOOL_FALSE_TEST_KEY", originalFalse)
		}
	}()

	value := GetBoolEnv("NON_EXISTENT_BOOL_KEY")
	assert.False(t, value)

	os.Setenv("BOOL_TEST_KEY", "true")

	value = GetBoolEnv("BOOL_TEST_KEY")
	assert.True(t, value)

	os.Setenv("BOOL_FALSE_TEST_KEY", "false")

	value = GetBoolEnv("BOOL_FALSE_TEST_KEY")
	assert.False(t, value)
}

func TestGetIntEnv(t *testing.T) {
	originalNonExistent := os.Getenv("NON_EXISTENT_INT_KEY")
	originalValid := os.Getenv("INT_TEST_KEY")
	originalInvalid := os.Getenv("INVALID_INT_TEST_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_INT_KEY")
		} else {
			os.Setenv("NON_EXISTENT_INT_KEY", originalNonExistent)
		}
		if originalValid == "" {
			os.Unsetenv("INT_TEST_KEY")
		} else {
			os.Setenv("INT_TEST_KEY", originalValid)
		}
		if originalInvalid == "" {
			os.Unsetenv("INVALID_INT_TEST_KEY")
		} else {
			os.Setenv("INVALID_INT_TEST_KEY", originalInvalid)
		}
	}()

	value := GetIntEnv("NON_EXISTENT_INT_KEY")
	assert.Equal(t, int64(0), value)

	os.Setenv("INT_TEST_KEY", "12345")

	value = GetIntEnv("INT_TEST_KEY")
	assert.Equal(t, int64(12345), value)

	os.Setenv("INVALID_INT_TEST_KEY", "not_a_number")

	value = GetIntEnv("INVALID_INT_TEST_KEY")
	assert.Equal(t, int64(0), value)
}

func TestLoadEnvs(t *testing.T) {
	type TestConfig struct {
		StringValue string `env:"STRING_TEST_KEY"`
		IntValue    int    `env:"INT_TEST_KEY"`
		BoolValue   bool   `env:"BOOL_TEST_KEY"`
		Ignored     string `env:"-"`
		Unset       string `env:"UNSET_TEST_KEY"`
	}

	os.Unsetenv("STRING_TEST_KEY")
	os.Unsetenv("INT_TEST_KEY")
	os.Unsetenv("BOOL_TEST_KEY")
	defer func() {
		os.Unsetenv("STRING_TEST_KEY")
		os.Unsetenv("INT_TEST_KEY")
		os.Unsetenv("BOOL_TEST_KEY")
	}()

	os.Setenv("STRING_TEST_KEY", "test_string")
	os.Setenv("INT_TEST_KEY", "42")
	os.Setenv("BOOL_TEST_KEY", "true")
	defer func() {
		os.Unsetenv("STRING_TEST_KEY")
		os.Unsetenv("INT_TEST_KEY")
		os.Unsetenv("BOOL_TEST_KEY")
	}()

	config := &TestConfig{}
	LoadEnvs(config)

	assert.Equal(t, "test_string", config.StringValue)
	assert.Equal(t, 42, config.IntValue)
	assert.True(t, config.BoolValue)
	assert.Equal(t, "", config.Ignored)
	assert.Equal(t, "", config.Unset)
}

func TestLoadEnv(t *testing.T) {
	originalTestKey := os.Getenv("TEST_ENV_FILE_KEY")
	originalAnotherKey := os.Getenv("ANOTHER_ENV_KEY")
	defer func() {
		if originalTestKey == "" {
			os.Unsetenv("TEST_ENV_FILE_KEY")
		} else {
			os.Setenv("TEST_ENV_FILE_KEY", originalTestKey)
		}
		if originalAnotherKey == "" {
			os.Unsetenv("ANOTHER_ENV_KEY")
		} else {
			os.Setenv("ANOTHER_ENV_KEY", originalAnotherKey)
		}
	}()

	envContent := `
# Test environment file
TEST_ENV_FILE_KEY=test_value_from_env_file
ANOTHER_ENV_KEY=another_value
`
	envFile := ".env.test"
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(envFile)

	err = LoadEnv("test")
	assert.NoError(t, err)

	value := os.Getenv("TEST_ENV_FILE_KEY")
	assert.Equal(t, "test_value_from_env_file", value)

	value = os.Getenv("ANOTHER_ENV_KEY")
	assert.Equal(t, "another_value", value)

	err = LoadEnv("nonexistent")
	assert.Error(t, err)
}

func TestEnvFloatAndPositiveInt(t *testing.T) {
	t.Setenv("TEST_POS_INT", "42")
	n, ok := PositiveIntEnv("TEST_POS_INT")
	if !ok || n != 42 {
		t.Fatalf("PositiveIntEnv: ok=%v n=%d", ok, n)
	}
	t.Setenv("TEST_POS_INT", "0")
	if _, ok := PositiveIntEnv("TEST_POS_INT"); ok {
		t.Fatal("zero should not be positive")
	}
	t.Setenv("TEST_ENV_FLOAT", "0.75")
	f, ok := EnvFloat("TEST_ENV_FLOAT")
	if !ok || f != 0.75 {
		t.Fatalf("EnvFloat: ok=%v f=%v", ok, f)
	}
}

func TestEnvCacheExpiration(t *testing.T) {
	// Test that the env cache TTL mechanism works
	t.Setenv("TEST_CACHE_KEY", "cached_value")
	PurgeEnvCacheForTest()

	// First lookup populates cache
	v, ok := LookupEnv("TEST_CACHE_KEY")
	assert.True(t, ok)
	assert.Equal(t, "cached_value", v)

	// Cached lookup should still work
	v, ok = LookupEnv("TEST_CACHE_KEY")
	assert.True(t, ok)
	assert.Equal(t, "cached_value", v)

	// Unset env, cache should still serve the value
	os.Unsetenv("TEST_CACHE_KEY")
	v, ok = LookupEnv("TEST_CACHE_KEY")
	assert.True(t, ok)
	assert.Equal(t, "cached_value", v)

	// Wait for cache to expire
	time.Sleep(envCacheTTL + 50*time.Millisecond)

	// After expiry, should not find it
	v, ok = LookupEnv("TEST_CACHE_KEY")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestGetFloatEnv(t *testing.T) {
	t.Setenv("TEST_FLOAT_ENV", "3.14")
	if v := GetFloatEnv("TEST_FLOAT_ENV"); v != 3.14 {
		t.Fatalf("GetFloatEnv = %v, want 3.14", v)
	}
	if v := GetFloatEnv("NON_EXISTENT_FLOAT"); v != 0 {
		t.Fatalf("GetFloatEnv = %v, want 0", v)
	}
}

func TestGetFloatEnvWithDefault(t *testing.T) {
	t.Setenv("TEST_FLOAT_DEFAULT", "2.5")
	if v := GetFloatEnvWithDefault("TEST_FLOAT_DEFAULT", 9.9); v != 2.5 {
		t.Fatalf("GetFloatEnvWithDefault = %v, want 2.5", v)
	}
	if v := GetFloatEnvWithDefault("NON_EXISTENT_FLOAT_DEFAULT", 9.9); v != 9.9 {
		t.Fatalf("GetFloatEnvWithDefault = %v, want 9.9", v)
	}
}

func TestGetIntEnvWithDefault(t *testing.T) {
	t.Setenv("TEST_INT_DEFAULT", "42")
	if v := GetIntEnvWithDefault("TEST_INT_DEFAULT", 99); v != 42 {
		t.Fatalf("GetIntEnvWithDefault = %v, want 42", v)
	}
	if v := GetIntEnvWithDefault("NON_EXISTENT_INT_DEFAULT", 99); v != 99 {
		t.Fatalf("GetIntEnvWithDefault = %v, want 99", v)
	}
}

func TestGetStringOrDefault(t *testing.T) {
	t.Setenv("TEST_STR_DEFAULT", "hello")
	if v := GetStringOrDefault("TEST_STR_DEFAULT", "fallback"); v != "hello" {
		t.Fatalf("GetStringOrDefault = %q, want %q", v, "hello")
	}
	if v := GetStringOrDefault("NON_EXISTENT_STR", "fallback"); v != "fallback" {
		t.Fatalf("GetStringOrDefault = %q, want %q", v, "fallback")
	}
}

func TestGetBoolOrDefault(t *testing.T) {
	t.Setenv("TEST_BOOL_DEFAULT", "true")
	if v := GetBoolOrDefault("TEST_BOOL_DEFAULT", false); !v {
		t.Fatal("GetBoolOrDefault = false, want true")
	}
	if v := GetBoolOrDefault("NON_EXISTENT_BOOL", true); !v {
		t.Fatal("GetBoolOrDefault = false, want true (default)")
	}
}

func TestGetIntOrDefault(t *testing.T) {
	t.Setenv("TEST_INT_OR_DEFAULT", "77")
	if v := GetIntOrDefault("TEST_INT_OR_DEFAULT", 100); v != 77 {
		t.Fatalf("GetIntOrDefault = %d, want 77", v)
	}
	if v := GetIntOrDefault("NON_EXISTENT_INT_OR", 100); v != 100 {
		t.Fatalf("GetIntOrDefault = %d, want 100", v)
	}
}

func TestGetFloatOrDefault(t *testing.T) {
	t.Setenv("TEST_FLOAT_OR_DEFAULT", "1.5")
	if v := GetFloatOrDefault("TEST_FLOAT_OR_DEFAULT", 9.9); v != 1.5 {
		t.Fatalf("GetFloatOrDefault = %v, want 1.5", v)
	}
	if v := GetFloatOrDefault("NON_EXISTENT_FLOAT_OR", 9.9); v != 9.9 {
		t.Fatalf("GetFloatOrDefault = %v, want 9.9", v)
	}
	t.Setenv("TEST_FLOAT_OR_INVALID", "not_a_float")
	if v := GetFloatOrDefault("TEST_FLOAT_OR_INVALID", 9.9); v != 9.9 {
		t.Fatalf("GetFloatOrDefault invalid = %v, want 9.9", v)
	}
}

func TestParseDuration(t *testing.T) {
	if d := ParseDuration("", 5*time.Second); d != 5*time.Second {
		t.Fatalf("ParseDuration empty = %v, want 5s", d)
	}
	if d := ParseDuration("30m", 5*time.Second); d != 30*time.Minute {
		t.Fatalf("ParseDuration 30m = %v, want 30m", d)
	}
	if d := ParseDuration("invalid", 5*time.Second); d != 5*time.Second {
		t.Fatalf("ParseDuration invalid = %v, want 5s", d)
	}
}

func TestEnvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "1h30m")
	d, ok := EnvDuration("TEST_DURATION")
	if !ok || d != 90*time.Minute {
		t.Fatalf("EnvDuration = %v ok=%v, want 1h30m true", d, ok)
	}
	if _, ok := EnvDuration("NON_EXISTENT_DURATION"); ok {
		t.Fatal("EnvDuration should return false for unset key")
	}
	t.Setenv("TEST_DURATION_INVALID", "not_a_duration")
	if _, ok := EnvDuration("TEST_DURATION_INVALID"); ok {
		t.Fatal("EnvDuration should return false for invalid value")
	}
}

func TestEnvIntInvalid(t *testing.T) {
	t.Setenv("TEST_ENV_INT_INVALID", "abc")
	if _, ok := EnvInt("TEST_ENV_INT_INVALID"); ok {
		t.Fatal("EnvInt should return false for invalid value")
	}
}

func TestEnvFloatInvalid(t *testing.T) {
	t.Setenv("TEST_ENV_FLOAT_INVALID", "not_a_float")
	if _, ok := EnvFloat("TEST_ENV_FLOAT_INVALID"); ok {
		t.Fatal("EnvFloat should return false for invalid value")
	}
}
