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
