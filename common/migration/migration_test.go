// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.sql
var testMigrationFS embed.FS

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantVer  uint64
		wantDesc string
		wantDir  string
		wantOK   bool
	}{
		{"up", "0001_create_users.up.sql", 1, "create_users", "up", true},
		{"down", "0001_create_users.down.sql", 1, "create_users", "down", true},
		{"no suffix", "0001_create_users.sql", 0, "", "", false},
		{"no underscore", "0001.up.sql", 0, "", "", false},
		{"non-numeric", "abc_create.up.sql", 0, "", "", false},
		{"not sql", "0001_create.up.txt", 0, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, desc, dir, ok := parseMigrationFilename(tt.filename)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantVer, ver)
				assert.Equal(t, tt.wantDesc, desc)
				assert.Equal(t, tt.wantDir, dir)
			}
		})
	}
}

func TestEmbedSource(t *testing.T) {
	src := NewEmbedSource(testMigrationFS, "testdata")
	migrations, err := src.Migrations()
	require.NoError(t, err)
	require.Len(t, migrations, 2)

	assert.Equal(t, uint64(1), migrations[0].Version)
	assert.Equal(t, "create_users", migrations[0].Description)
	assert.Contains(t, migrations[0].UpSQL, "CREATE TABLE")
	assert.Contains(t, migrations[0].DownSQL, "DROP TABLE")

	assert.Equal(t, uint64(2), migrations[1].Version)
	assert.Equal(t, "add_email_index", migrations[1].Description)
	assert.True(t, migrations[1].HasUp())
	assert.True(t, migrations[1].HasDown())
}

func TestEmbedSource_EmptyDir(t *testing.T) {
	// Use a non-existent directory.
	src := NewEmbedSource(testMigrationFS, "nonexistent")
	_, err := src.Migrations()
	assert.Error(t, err)
}

func TestFileSource(t *testing.T) {
	// Create a temporary directory with migration files.
	tmpDir := t.TempDir()

	writeFile(t, tmpDir, "0001_create_users.up.sql", "CREATE TABLE users (id INTEGER PRIMARY KEY);")
	writeFile(t, tmpDir, "0001_create_users.down.sql", "DROP TABLE users;")
	writeFile(t, tmpDir, "0002_add_index.up.sql", "CREATE INDEX idx ON users(id);")

	src := NewFileSource(tmpDir)
	migrations, err := src.Migrations()
	require.NoError(t, err)
	require.Len(t, migrations, 2)

	assert.Equal(t, uint64(1), migrations[0].Version)
	assert.Equal(t, "create_users", migrations[0].Description)
	assert.True(t, migrations[0].HasUp())
	assert.True(t, migrations[0].HasDown())

	assert.Equal(t, uint64(2), migrations[1].Version)
	assert.Equal(t, "add_index", migrations[1].Description)
	assert.True(t, migrations[1].HasUp())
	assert.False(t, migrations[1].HasDown()) // no down file
}

func TestFileSource_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	src := NewFileSource(tmpDir)
	_, err := src.Migrations()
	assert.ErrorIs(t, err, ErrNoMigrations)
}

func TestFileSource_NonexistentDir(t *testing.T) {
	src := NewFileSource("/nonexistent/path/12345")
	_, err := src.Migrations()
	assert.Error(t, err)
}

func TestStaticSource(t *testing.T) {
	src := NewStaticSource(
		Migration{Version: 2, Description: "second", UpSQL: "SELECT 2;"},
		Migration{Version: 1, Description: "first", UpSQL: "SELECT 1;"},
	)
	migrations, err := src.Migrations()
	require.NoError(t, err)
	require.Len(t, migrations, 2)

	// Should be sorted by version.
	assert.Equal(t, uint64(1), migrations[0].Version)
	assert.Equal(t, uint64(2), migrations[1].Version)
}

func TestMigration_HasUp_HasDown(t *testing.T) {
	m := Migration{Version: 1, UpSQL: "SELECT 1;", DownSQL: ""}
	assert.True(t, m.HasUp())
	assert.False(t, m.HasDown())

	m = Migration{Version: 1, UpSQL: "  ", DownSQL: "SELECT 0;"}
	assert.False(t, m.HasUp())
	assert.True(t, m.HasDown())
}

func TestMigration_String(t *testing.T) {
	m := Migration{Version: 1, Description: "create_users"}
	assert.Equal(t, "1_create_users", m.String())
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, "file.sql", joinPath(".", "file.sql"))
	assert.Equal(t, "file.sql", joinPath("", "file.sql"))
	assert.Equal(t, "dir/file.sql", joinPath("dir", "file.sql"))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}
