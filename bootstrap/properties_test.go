// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProperties_SetGet(t *testing.T) {
	p := NewProperties()
	p.Set("key", "value")
	assert.Equal(t, "value", p.Get("key"))
}

func TestProperties_GetDefault(t *testing.T) {
	p := NewProperties()
	assert.Equal(t, "default", p.GetDefault("missing", "default"))
	p.Set("key", "value")
	assert.Equal(t, "value", p.GetDefault("key", "default"))
}

func TestProperties_GetInt(t *testing.T) {
	p := NewProperties()
	p.Set("port", "8080")
	assert.Equal(t, 8080, p.GetInt("port", 3000))
	assert.Equal(t, 3000, p.GetInt("missing", 3000))
	p.Set("bad", "abc")
	assert.Equal(t, 3000, p.GetInt("bad", 3000))
}

func TestProperties_GetInt64(t *testing.T) {
	p := NewProperties()
	p.Set("big", "9999999999")
	assert.Equal(t, int64(9999999999), p.GetInt64("big", 0))
}

func TestProperties_GetBool(t *testing.T) {
	p := NewProperties()
	tests := map[string]bool{
		"true": true, "1": true, "yes": true, "on": true,
		"false": false, "0": false, "no": false, "off": false,
	}
	for val, expected := range tests {
		p.Set("flag", val)
		assert.Equal(t, expected, p.GetBool("flag", false), "value=%s", val)
	}
	assert.Equal(t, true, p.GetBool("missing", true))
}

func TestProperties_GetDuration(t *testing.T) {
	p := NewProperties()
	p.Set("timeout", "30s")
	assert.Equal(t, 30*time.Second, p.GetDuration("timeout", 0))
	assert.Equal(t, 5*time.Minute, p.GetDuration("missing", 5*time.Minute))
	p.Set("bad", "xyz")
	assert.Equal(t, time.Second, p.GetDuration("bad", time.Second))
}

func TestProperties_GetFloat64(t *testing.T) {
	p := NewProperties()
	p.Set("ratio", "0.85")
	assert.Equal(t, 0.85, p.GetFloat64("ratio", 0))
	assert.Equal(t, 1.5, p.GetFloat64("missing", 1.5))
}

func TestProperties_GetStringSlice(t *testing.T) {
	p := NewProperties()
	p.Set("hosts", "a.com, b.com, c.com")
	slice := p.GetStringSlice("hosts", ",")
	assert.Equal(t, []string{"a.com", "b.com", "c.com"}, slice)

	assert.Nil(t, p.GetStringSlice("missing", ","))
}

func TestProperties_Has(t *testing.T) {
	p := NewProperties()
	p.Set("key", "value")
	assert.True(t, p.Has("key"))
	assert.False(t, p.Has("missing"))
}

func TestProperties_Keys(t *testing.T) {
	p := NewProperties()
	p.Set("k1", "v1")
	p.Set("k2", "v2")
	keys := p.Keys()
	assert.Len(t, keys, 2)
}

func TestProperties_Count(t *testing.T) {
	p := NewProperties()
	p.Set("k1", "v1")
	p.Set("k2", "v2")
	assert.Equal(t, 2, p.Count())
}

func TestProperties_LoadFromMap(t *testing.T) {
	p := NewProperties()
	p.LoadFromMap(map[string]string{
		"db.host": "localhost",
		"db.port": "5432",
	})
	assert.Equal(t, "localhost", p.Get("db.host"))
	assert.Equal(t, "5432", p.Get("db.port"))
}

func TestProperties_LoadFromEnv(t *testing.T) {
	os.Setenv("APP_DB_HOST", "db.example.com")
	os.Setenv("APP_DB_PORT", "3306")
	os.Setenv("OTHER_VAR", "ignored")
	defer os.Unsetenv("APP_DB_HOST")
	defer os.Unsetenv("APP_DB_PORT")
	defer os.Unsetenv("OTHER_VAR")

	p := NewProperties()
	p.LoadFromEnv("APP")
	assert.Equal(t, "db.example.com", p.Get("db.host"))
	assert.Equal(t, "3306", p.Get("db.port"))
	assert.False(t, p.Has("other.var"))
}

func TestProperties_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "app.properties")
	content := `# comment
app.name=test
app.port=8080
empty.line.should.be.skipped

app.debug=true
`
	os.WriteFile(filename, []byte(content), 0644)

	p := NewProperties()
	err := p.LoadFromFile(filename)
	assert.NoError(t, err)
	assert.Equal(t, "test", p.Get("app.name"))
	assert.Equal(t, "8080", p.Get("app.port"))
	assert.Equal(t, "true", p.Get("app.debug"))
}

func TestProperties_LoadFromFileNotFound(t *testing.T) {
	p := NewProperties()
	err := p.LoadFromFile("/nonexistent/file.properties")
	assert.Error(t, err)
}

func TestProperties_Sources(t *testing.T) {
	p := NewProperties()
	p.LoadFromMap(map[string]string{"k": "v"})
	p.LoadFromEnv("APP")
	sources := p.Sources()
	assert.Len(t, sources, 2)
}
