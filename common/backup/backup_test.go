// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSourceFile(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "source.bin")
	require.NoError(t, os.WriteFile(p, content, 0o644))
	return p
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestManager_BackupAndRestore_NoCompression(t *testing.T) {
	content := bytes.Repeat([]byte("hello backup! "), 1000)
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()

	mgr := New(
		&FileSource{Path: srcPath},
		&FileDestination{Dir: dstDir},
		WithSink(&FileSink{Path: srcPath}),
	)

	bp, err := mgr.Backup(context.Background(), "bk1")
	require.NoError(t, err)
	require.NotNil(t, bp)

	assert.Equal(t, "bk1", bp.Name)
	assert.False(t, bp.Compressed)
	assert.Greater(t, bp.Size, int64(0))
	assert.Equal(t, sha256Hex(content), bp.Checksum)
	assert.True(t, strings.HasSuffix(bp.Key, "bk1"), "key=%s", bp.Key)

	_, err = os.Stat(bp.Key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(srcPath, []byte("corrupted"), 0o644))
	require.NoError(t, mgr.Restore(context.Background(), bp, nil))

	restored, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}

func TestManager_BackupAndRestore_WithCompression(t *testing.T) {
	content := bytes.Repeat([]byte("compress me compress me "), 500)
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()

	mgr := New(
		&FileSource{Path: srcPath},
		&FileDestination{Dir: dstDir},
		WithCompression(),
		WithSink(&FileSink{Path: srcPath}),
	)

	bp, err := mgr.Backup(context.Background(), "bk-gz")
	require.NoError(t, err)
	assert.True(t, bp.Compressed)
	assert.True(t, strings.HasSuffix(bp.Key, ".gz"), "key=%s", bp.Key)

	f, err := os.Open(bp.Key)
	require.NoError(t, err)
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	_ = f.Close()
	assert.Equal(t, content, decompressed)
	assert.Less(t, bp.Size, int64(len(content)))

	require.NoError(t, os.WriteFile(srcPath, []byte("x"), 0o644))
	require.NoError(t, mgr.Restore(context.Background(), bp, nil))
	restored, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}

func TestManager_Verify(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()

	mgr := New(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "ck")
	require.NoError(t, err)
	require.NoError(t, mgr.Verify(context.Background(), bp))

	require.NoError(t, os.WriteFile(bp.Key, []byte("tampered"), 0o644))
	err = mgr.Verify(context.Background(), bp)
	assert.ErrorIs(t, err, ErrChecksumMismatch)
}

func TestManager_ListGetDelete(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := New(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})

	_, err := mgr.Backup(context.Background(), "b3")
	require.NoError(t, err)
	_, err = mgr.Backup(context.Background(), "b1")
	require.NoError(t, err)
	_, err = mgr.Backup(context.Background(), "b2")
	require.NoError(t, err)

	list, err := mgr.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "b1", list[0].Name)

	got, err := mgr.Get(context.Background(), "b2")
	require.NoError(t, err)
	assert.Equal(t, "b2", got.Name)

	bp, err := mgr.Get(context.Background(), "b1")
	require.NoError(t, err)
	require.NoError(t, mgr.Delete(context.Background(), "b1"))
	_, err = os.Stat(bp.Key)
	assert.True(t, os.IsNotExist(err))
	_, err = mgr.Get(context.Background(), "b1")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestManager_EmptyName(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := New(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	_, err := mgr.Backup(context.Background(), "")
	assert.ErrorIs(t, err, ErrEmptyName)
}

func TestManager_RestoreRequiresSink(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := New(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "r")
	require.NoError(t, err)
	err = mgr.Restore(context.Background(), bp, nil)
	assert.ErrorIs(t, err, ErrNoSink)
}

func TestFileCatalog_Persists(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("persist-me"))
	dstDir := t.TempDir()
	catPath := filepath.Join(dstDir, "catalog.json")

	mgr := New(
		&FileSource{Path: srcPath},
		&FileDestination{Dir: dstDir},
		WithCatalog(&FileCatalog{Path: catPath}),
	)
	bp, err := mgr.Backup(context.Background(), "p1")
	require.NoError(t, err)

	mgr2 := New(
		&FileSource{Path: srcPath},
		&FileDestination{Dir: dstDir},
		WithCatalog(&FileCatalog{Path: catPath}),
		WithSink(&FileSink{Path: srcPath}),
	)
	got, err := mgr2.Get(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, bp.Checksum, got.Checksum)

	require.NoError(t, os.WriteFile(srcPath, []byte("x"), 0o644))
	require.NoError(t, mgr2.Restore(context.Background(), got, nil))
	restored, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("persist-me"), restored)
}

func TestRetention_KeepLastN_Prune(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	tick := 0
	mgr := New(
		&FileSource{Path: srcPath},
		&FileDestination{Dir: dstDir},
		WithRetention(KeepLastN(2)),
		WithClock(func() time.Time {
			tick++
			return now.Add(time.Duration(tick) * time.Hour)
		}),
	)
	for _, name := range []string{"a", "b", "c"} {
		_, err := mgr.Backup(context.Background(), name)
		require.NoError(t, err)
	}
	removed, err := mgr.Prune(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, removed)

	list, err := mgr.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestRetention_KeepNewerThan(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	all := []*Backup{
		{Name: "old", CreatedAt: now.Add(-48 * time.Hour)},
		{Name: "new", CreatedAt: now.Add(-1 * time.Hour)},
	}
	v := KeepNewerThan(24*time.Hour).Victims(all, now)
	assert.Equal(t, []string{"old"}, v)
}

func TestCommandSource_Cat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses cat")
	}
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "in.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("from-cat"), 0o644))
	dstDir := filepath.Join(dir, "out")

	mgr := New(
		&CommandSource{Name: "cat", Args: []string{srcFile}},
		&FileDestination{Dir: dstDir},
	)
	bp, err := mgr.Backup(context.Background(), "cmd")
	require.NoError(t, err)
	assert.Equal(t, sha256Hex([]byte("from-cat")), bp.Checksum)

	sinkFile := filepath.Join(dir, "restored.txt")
	require.NoError(t, mgr.Restore(context.Background(), bp, &FileSink{Path: sinkFile}))
	got, err := os.ReadFile(sinkFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("from-cat"), got)
}

func TestCommandSink_Tee(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses tee")
	}
	dir := t.TempDir()
	srcPath := writeSourceFile(t, []byte("tee-data"))
	dstDir := filepath.Join(dir, "bk")
	outFile := filepath.Join(dir, "tee.out")

	mgr := New(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "t")
	require.NoError(t, err)

	sink := &CommandSink{Name: "tee", Args: []string{outFile}}
	require.NoError(t, mgr.Restore(context.Background(), bp, sink))
	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("tee-data"), got)
}

func TestMySQLDumpArgs(t *testing.T) {
	args := MySQLDumpArgs(MySQLConfig{
		Host: "db.example.com", Port: 3307, User: "u", Database: "app",
		ExtraArgs: []string{"--set-gtid-purged=OFF"},
	})
	assert.Equal(t, []string{
		"-h", "db.example.com", "-P", "3307", "-u", "u",
		"--single-transaction", "--routines", "--triggers", "--events",
		"--set-gtid-purged=OFF", "app",
	}, args)

	src := MySQLDump(MySQLConfig{Host: "h", User: "u", Password: "secret", Database: "d"})
	assert.Equal(t, "mysqldump", src.Name)
	assert.Contains(t, src.Env, "MYSQL_PWD=secret")

	all := MySQLDumpArgs(MySQLConfig{Host: "h", User: "u"})
	assert.Contains(t, all, "--all-databases")
}

func TestPostgresMongoArgs(t *testing.T) {
	pg := PostgresDumpArgs(PostgresConfig{Host: "h", Port: 5433, User: "u", Database: "db"})
	assert.Contains(t, pg, "-d")
	assert.Contains(t, pg, "db")

	mg := MongoDumpArgs(MongoConfig{URI: "mongodb://localhost", Database: "app"})
	assert.Contains(t, mg, "--archive")
	assert.Contains(t, mg, "--uri=mongodb://localhost")
	assert.Contains(t, mg, "--db=app")
}

func TestValidateConfigs(t *testing.T) {
	assert.Error(t, ValidateMySQLConfig(MySQLConfig{}))
	assert.NoError(t, ValidateMySQLConfig(MySQLConfig{User: "u"}))
	assert.Error(t, ValidatePostgresConfig(PostgresConfig{}))
	assert.NoError(t, ValidatePostgresConfig(PostgresConfig{Database: "db"}))
	assert.Error(t, ValidateMongoConfig(MongoConfig{}))
	assert.NoError(t, ValidateMongoConfig(MongoConfig{URI: "mongodb://x"}))
	assert.Error(t, ValidateSQLiteConfig(SQLiteConfig{}))
	assert.NoError(t, ValidateSQLiteConfig(SQLiteConfig{Path: "a.db"}))
	assert.Error(t, ValidateRedisConfig(RedisConfig{}))
	assert.NoError(t, ValidateRedisConfig(RedisConfig{Host: "h"}))
	assert.Error(t, ValidateClickHouseConfig(ClickHouseConfig{}))
	assert.NoError(t, ValidateClickHouseConfig(ClickHouseConfig{Database: "db"}))
	assert.Error(t, ValidateMSSQLConfig(MSSQLConfig{}))
	assert.NoError(t, ValidateMSSQLConfig(MSSQLConfig{Host: "h", Database: "db"}))
}

func TestDirectorySourceSink(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("bbb"), 0o644))

	dstDir := t.TempDir()
	mgr := New(&DirectorySource{Root: root}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "dir")
	require.NoError(t, err)

	out := t.TempDir()
	require.NoError(t, mgr.Restore(context.Background(), bp, &DirectorySink{Dest: out}))
	got, err := os.ReadFile(filepath.Join(out, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("aaa"), got)
	got, err = os.ReadFile(filepath.Join(out, "sub", "b.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("bbb"), got)
}

func TestHelperArgs(t *testing.T) {
	assert.Equal(t, []string{"db.sqlite", ".dump"}, SQLiteDumpArgs(SQLiteConfig{Path: "db.sqlite"}))
	rd := RedisDumpArgs(RedisConfig{Host: "h", Port: 6380, Password: "p"})
	assert.Contains(t, rd, "--rdb")
	assert.Contains(t, rd, "-")
	ch := ClickHouseDumpArgs(ClickHouseConfig{Database: "analytics"})
	assert.Contains(t, strings.Join(ch, " "), "BACKUP DATABASE")
	ms := MSSQLDumpArgs(MSSQLConfig{Host: "h", Port: 1433, Database: "db", User: "u"})
	assert.Contains(t, ms, "-S")
}

func TestFileSource_EmptyPath(t *testing.T) {
	_, err := (&FileSource{}).Open(context.Background())
	assert.Error(t, err)
}

func TestFileDestination_EmptyDir(t *testing.T) {
	_, err := (&FileDestination{}).Write(context.Background(), "x", strings.NewReader("data"))
	assert.Error(t, err)
}
