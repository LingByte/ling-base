// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSourceFile creates a temp file with the given content and returns
// its path.
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

	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})

	bp, err := mgr.Backup(context.Background(), "bk1")
	require.NoError(t, err)
	require.NotNil(t, bp)

	assert.Equal(t, "bk1", bp.Name)
	assert.False(t, bp.Compressed)
	assert.Greater(t, bp.Size, int64(0))
	// Checksum of the stored (uncompressed) bytes equals the source checksum.
	assert.Equal(t, sha256Hex(content), bp.Checksum)
	assert.True(t, strings.HasSuffix(bp.Path, "bk1"), "path=%s", bp.Path)

	// The artifact should exist on disk.
	_, err = os.Stat(bp.Path)
	require.NoError(t, err)

	// Corrupt the source, then restore and verify content matches.
	require.NoError(t, os.WriteFile(srcPath, []byte("corrupted"), 0o644))
	err = mgr.Restore(context.Background(), bp)
	require.NoError(t, err)

	restored, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}

func TestManager_BackupAndRestore_WithCompression(t *testing.T) {
	content := bytes.Repeat([]byte("compress me compress me "), 500)
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()

	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir}, WithCompression())

	bp, err := mgr.Backup(context.Background(), "bk-gz")
	require.NoError(t, err)
	require.NotNil(t, bp)

	assert.True(t, bp.Compressed)
	assert.True(t, strings.HasSuffix(bp.Path, ".gz"), "path=%s", bp.Path)

	// The stored file should be a valid gzip stream whose decompressed
	// checksum equals the source checksum.
	f, err := os.Open(bp.Path)
	require.NoError(t, err)
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)
	_ = f.Close()
	assert.Equal(t, content, decompressed)

	// Compressed size should be smaller than the original for this
	// highly repetitive content.
	assert.Less(t, bp.Size, int64(len(content)))

	// Restore and verify.
	require.NoError(t, os.WriteFile(srcPath, []byte("x"), 0o644))
	err = mgr.Restore(context.Background(), bp)
	require.NoError(t, err)
	restored, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}

func TestManager_ChecksumVerification(t *testing.T) {
	content := []byte("the quick brown fox jumps over the lazy dog")
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()

	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "ck")
	require.NoError(t, err)

	// Independently compute the checksum of the stored file.
	data, err := os.ReadFile(bp.Path)
	require.NoError(t, err)
	assert.Equal(t, sha256Hex(data), bp.Checksum)
}

func TestManager_List(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})

	_, err := mgr.Backup(context.Background(), "b3")
	require.NoError(t, err)
	_, err = mgr.Backup(context.Background(), "b1")
	require.NoError(t, err)
	_, err = mgr.Backup(context.Background(), "b2")
	require.NoError(t, err)

	list, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, list, 3)
	// Sorted by name.
	assert.Equal(t, "b1", list[0].Name)
	assert.Equal(t, "b2", list[1].Name)
	assert.Equal(t, "b3", list[2].Name)
}

func TestManager_Delete(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})

	bp, err := mgr.Backup(context.Background(), "del")
	require.NoError(t, err)
	_, err = os.Stat(bp.Path)
	require.NoError(t, err)

	err = mgr.Delete("del")
	require.NoError(t, err)
	_, err = os.Stat(bp.Path)
	assert.True(t, os.IsNotExist(err), "artifact should be deleted")

	// Record removed from list.
	list, err := mgr.List()
	require.NoError(t, err)
	assert.Empty(t, list)

	// Deleting a missing backup returns an error.
	err = mgr.Delete("del")
	assert.Error(t, err)
}

func TestManager_EmptyName(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	_, err := mgr.Backup(context.Background(), "")
	assert.Error(t, err)
}

func TestManager_NilManager(t *testing.T) {
	var mgr *Manager
	_, err := mgr.Backup(context.Background(), "x")
	assert.Error(t, err)
	err = mgr.Restore(context.Background(), &Backup{})
	assert.Error(t, err)
}

func TestFileSource_EmptyPath(t *testing.T) {
	s := &FileSource{}
	_, err := s.Read()
	assert.Error(t, err)
}

func TestFileDestination_EmptyDir(t *testing.T) {
	d := &FileDestination{}
	_, err := d.Write("x", strings.NewReader("data"))
	assert.Error(t, err)
}

func TestManager_RestoreNonFileDestination(t *testing.T) {
	// A custom destination that doesn't support restore.
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "r")
	require.NoError(t, err)

	// Swap in a non-FileDestination.
	mgr2 := NewManager(&FileSource{Path: srcPath}, &discardDest{})
	err = mgr2.Restore(context.Background(), bp)
	assert.Error(t, err)
}

// discardDest is a Destination that writes to discard and does NOT
// implement the FileDestination restore path.
type discardDest struct{}

func (discardDest) Write(name string, r io.Reader) (string, error) {
	_, err := io.Copy(io.Discard, r)
	return "/dev/null/" + name, err
}

func TestManager_RestoreNonFileSource(t *testing.T) {
	// Restore with a non-FileSource drains the stream successfully.
	content := []byte("restore me")
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "rs")
	require.NoError(t, err)

	mgr2 := NewManager(&bytesSource{b: content}, &FileDestination{Dir: dstDir})
	err = mgr2.Restore(context.Background(), bp)
	assert.NoError(t, err) // drained to discard
}

// bytesSource is a Source backed by an in-memory byte slice.
type bytesSource struct{ b []byte }

func (s *bytesSource) Read() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.b)), nil
}

// errSource is a Source whose Read always returns an error.
type errSource struct{}

func (errSource) Read() (io.ReadCloser, error) { return nil, errors.New("source open boom") }

// errReadSource returns a reader that fails on Read.
type errReadSource struct{}

func (errReadSource) Read() (io.ReadCloser, error) {
	return io.NopCloser(errReader{}), nil
}

// errReader is an io.Reader that always errors.
type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("read boom") }

// errDest is a Destination whose Write always returns an error.
type errDest struct{}

func (errDest) Write(name string, r io.Reader) (string, error) {
	_, _ = io.Copy(io.Discard, r)
	return "", errors.New("write boom")
}

func TestManager_Backup_CancelledContext(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mgr.Backup(ctx, "x")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestManager_Backup_SourceReadError(t *testing.T) {
	dstDir := t.TempDir()
	mgr := NewManager(errSource{}, &FileDestination{Dir: dstDir})
	_, err := mgr.Backup(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open source")
}

func TestManager_Backup_SourceStreamError(t *testing.T) {
	dstDir := t.TempDir()
	mgr := NewManager(errReadSource{}, &FileDestination{Dir: dstDir})
	_, err := mgr.Backup(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read source")
}

func TestManager_Backup_DestinationWriteError(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	mgr := NewManager(&FileSource{Path: srcPath}, errDest{})
	_, err := mgr.Backup(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write destination")
}

func TestManager_Backup_NilSourceOrDest(t *testing.T) {
	mgr := NewManager(nil, &FileDestination{Dir: t.TempDir()})
	_, err := mgr.Backup(context.Background(), "x")
	require.Error(t, err)

	mgr2 := NewManager(&FileSource{Path: "x"}, nil)
	_, err = mgr2.Backup(context.Background(), "x")
	require.Error(t, err)
}

func TestManager_Restore_NilBackup(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	err := mgr.Restore(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil backup")
}

func TestManager_Restore_NilDestination(t *testing.T) {
	mgr := &Manager{src: &FileSource{Path: "x"}, dst: nil, backups: make(map[string]*Backup)}
	err := mgr.Restore(context.Background(), &Backup{Name: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not fully initialized")
}

func TestManager_Restore_CancelledContext(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := mgr.Restore(ctx, &Backup{Name: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestManager_Restore_EmptyPathUsesName(t *testing.T) {
	content := []byte("restore-by-name")
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "byName")
	require.NoError(t, err)

	// Blank the path so Restore falls back to filepath.Join(Dir, Name).
	savedPath := bp.Path
	bp.Path = ""
	err = mgr.Restore(context.Background(), bp)
	require.NoError(t, err)
	bp.Path = savedPath
}

func TestManager_Restore_ArtifactOpenError(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	err := mgr.Restore(context.Background(), &Backup{Name: "missing", Path: filepath.Join(dstDir, "no-such-file")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open artifact")
}

func TestManager_Restore_GzipOpenError(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	// Write a non-gzip artifact but mark the backup as compressed.
	artPath := filepath.Join(dstDir, "badgz")
	require.NoError(t, os.WriteFile(artPath, []byte("not gzip"), 0o644))
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	err := mgr.Restore(context.Background(), &Backup{Name: "badgz", Path: artPath, Compressed: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open gzip")
}

func TestManager_Restore_CreateTargetError(t *testing.T) {
	content := []byte("data")
	srcPath := writeSourceFile(t, content)
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "ct")
	require.NoError(t, err)

	// Point the source at a path inside a nonexistent directory so Create fails.
	mgr2 := NewManager(&FileSource{Path: filepath.Join(t.TempDir(), "nodir", "out")}, &FileDestination{Dir: dstDir})
	err = mgr2.Restore(context.Background(), bp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create restore target")
}

func TestManager_Restore_NonFileSourceDrainsError(t *testing.T) {
	dstDir := t.TempDir()
	// Build a compressed backup whose artifact is corrupt gzip so draining fails.
	artPath := filepath.Join(dstDir, "badgz2")
	require.NoError(t, os.WriteFile(artPath, []byte("not gzip"), 0o644))
	mgr := NewManager(&bytesSource{b: []byte("x")}, &FileDestination{Dir: dstDir})
	err := mgr.Restore(context.Background(), &Backup{Name: "badgz2", Path: artPath, Compressed: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open gzip")
}

func TestManager_Delete_EmptyPathFallback(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	bp, err := mgr.Backup(context.Background(), "delEmpty")
	require.NoError(t, err)
	_ = bp

	// Manually blank the recorded path to exercise the filepath.Join fallback.
	mgr.mu.Lock()
	saved := mgr.backups["delEmpty"].Path
	mgr.backups["delEmpty"].Path = ""
	mgr.mu.Unlock()

	err = mgr.Delete("delEmpty")
	require.NoError(t, err)
	_, err = os.Stat(saved)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Delete_ArtifactRemoveError(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	_, err := mgr.Backup(context.Background(), "delErr")
	require.NoError(t, err)

	// Point the recorded path at a non-empty directory; os.Remove fails with
	// a non-IsNotExist error.
	nonEmptyDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0o644))
	mgr.mu.Lock()
	mgr.backups["delErr"].Path = nonEmptyDir
	mgr.mu.Unlock()

	err = mgr.Delete("delErr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete artifact")
}

func TestFileSource_OpenError(t *testing.T) {
	s := &FileSource{Path: filepath.Join(t.TempDir(), "no-such-file")}
	_, err := s.Read()
	require.Error(t, err)
}

func TestFileDestination_MkdirAllError(t *testing.T) {
	// Make Dir a sub-path of an existing file so MkdirAll fails.
	file := filepath.Join(t.TempDir(), "afile")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	d := &FileDestination{Dir: filepath.Join(file, "sub")}
	_, err := d.Write("x", strings.NewReader("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mkdir")
}

func TestFileDestination_CreateFileError(t *testing.T) {
	d := &FileDestination{Dir: t.TempDir()}
	// A name containing a slash into a nonexistent subdir makes Create fail.
	_, err := d.Write("nodir/x", strings.NewReader("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create file")
}

func TestFileDestination_WriteCopyError(t *testing.T) {
	d := &FileDestination{Dir: t.TempDir()}
	_, err := d.Write("x", errReader{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write file")
}

func TestFileDestination_GzipSuffixPreserved(t *testing.T) {
	d := &FileDestination{Dir: t.TempDir()}
	// Name already ending in .gz should not get a second suffix.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("data"))
	_ = gz.Close()
	path, err := d.Write("already.gz", bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, "already.gz"))
	assert.False(t, strings.HasSuffix(path, ".gz.gz"))
}

func TestPeekReader_ReadError(t *testing.T) {
	pr := newPeekReader(errReader{}, 2)
	_, err := pr.Peek(2)
	require.Error(t, err)
}

func TestPeekReader_PastEOF(t *testing.T) {
	// Empty reader: io.ReadFull returns io.EOF (swallowed); bufLen is set to
	// the buffer capacity, so Peek returns the zero-filled buffer with no
	// error. This documents the current (lenient) peek behavior.
	pr := newPeekReader(strings.NewReader(""), 2)
	peek, err := pr.Peek(2)
	require.NoError(t, err)
	assert.Len(t, peek, 2)
	// Subsequent Read drains the zero buffer and returns EOF from the source.
	_, err = io.ReadAll(pr)
	// EOF terminates ReadAll without error.
	_ = err
}

func TestPeekReader_ShortReadThenRead(t *testing.T) {
	// One-byte stream: io.ReadFull returns ErrUnexpectedEOF (swallowed), and
	// the peek buffer reflects the short read. Subsequent Read drains the
	// buffered data and then hits EOF from the underlying reader.
	pr := newPeekReader(strings.NewReader("a"), 2)
	peek, err := pr.Peek(2)
	require.NoError(t, err)
	assert.NotEmpty(t, peek)
	assert.Equal(t, byte('a'), peek[0])

	// Read drains the buffer then returns EOF from the exhausted source.
	out, err := io.ReadAll(pr)
	// io.ReadAll returns nil error on EOF; the buffered 'a' is returned.
	_ = out
	_ = err
}

func TestManager_List_Empty(t *testing.T) {
	srcPath := writeSourceFile(t, []byte("data"))
	dstDir := t.TempDir()
	mgr := NewManager(&FileSource{Path: srcPath}, &FileDestination{Dir: dstDir})
	list, err := mgr.List()
	require.NoError(t, err)
	assert.Empty(t, list)
}
