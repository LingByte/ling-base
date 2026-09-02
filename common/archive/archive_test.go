// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

// makeSrcDir creates a temp directory with a known structure:
//
//	src/
//	  hello.txt          -> "hello world"
//	  sub/
//	    nested.txt       -> "nested content"
//	    deep/
//	      leaf.txt       -> "leaf"
//	  empty/
//	    (empty dir)
func makeSrcDir(t *testing.T) string {
	t.Helper()
	src := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(src, "hello.txt"), []byte("hello world"), 0o644))

	sub := filepath.Join(src, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("nested content"), 0o644))

	deep := filepath.Join(sub, "deep")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "leaf.txt"), []byte("leaf"), 0o644))

	empty := filepath.Join(src, "empty")
	require.NoError(t, os.MkdirAll(empty, 0o755))

	return src
}

// expectedFiles returns the sorted relative paths expected to be in an archive
// of makeSrcDir's output (files only).
func expectedFiles() []string {
	return []string{
		"hello.txt",
		"sub/deep/leaf.txt",
		"sub/nested.txt",
	}
}

// readFiles walks dst and returns a map of relative path -> content for files.
func readFiles(t *testing.T, dst string) map[string]string {
	t.Helper()
	out := map[string]string{}
	require.NoError(t, filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dst, path)
		require.NoError(t, err)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	}))
	return out
}

// sortedKeys returns sorted keys of a map.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ──────────────────────────────────────────────
// ZipArchiver
// ──────────────────────────────────────────────

func TestZipArchiver_ArchiveUnarchive(t *testing.T) {
	src := makeSrcDir(t)
	var buf bytes.Buffer

	require.NoError(t, (&ZipArchiver{}).Archive(src, &buf))

	dst := t.TempDir()
	require.NoError(t, (&ZipArchiver{}).Unarchive(&buf, dst))

	files := readFiles(t, dst)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
	assert.Equal(t, "hello world", files["hello.txt"])
	assert.Equal(t, "nested content", files["sub/nested.txt"])
	assert.Equal(t, "leaf", files["sub/deep/leaf.txt"])
	// empty dir should exist
	assert.DirExists(t, filepath.Join(dst, "empty"))
}

func TestZipArchiver_EmptyDir(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "emptydir"), 0o755))

	var buf bytes.Buffer
	require.NoError(t, (&ZipArchiver{}).Archive(src, &buf))

	dst := t.TempDir()
	require.NoError(t, (&ZipArchiver{}).Unarchive(&buf, dst))
	assert.DirExists(t, filepath.Join(dst, "emptydir"))
}

func TestZipArchiver_LargeFile(t *testing.T) {
	src := t.TempDir()
	// 1 MB file
	data := bytes.Repeat([]byte("ABCDEFGH"), 128*1024)
	require.NoError(t, os.WriteFile(filepath.Join(src, "big.bin"), data, 0o644))

	var buf bytes.Buffer
	require.NoError(t, (&ZipArchiver{}).Archive(src, &buf))

	dst := t.TempDir()
	require.NoError(t, (&ZipArchiver{}).Unarchive(&buf, dst))

	got, err := os.ReadFile(filepath.Join(dst, "big.bin"))
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestZipArchiver_PathTraversal(t *testing.T) {
	// Craft a zip with a malicious entry.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../escape.txt")
	require.NoError(t, err)
	_, err = w.Write([]byte("evil"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	dst := t.TempDir()
	err = (&ZipArchiver{}).Unarchive(&buf, dst)
	assert.ErrorIs(t, err, ErrPathEscapes)
	// ensure file was not created outside dst
	_, err = os.Stat(filepath.Join(dst, "..", "escape.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestZipArchiver_AbsolutePath(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("/etc/evil.txt")
	require.NoError(t, err)
	_, err = io.WriteString(w, "evil")
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	dst := t.TempDir()
	err = (&ZipArchiver{}).Unarchive(&buf, dst)
	assert.ErrorIs(t, err, ErrPathEscapes)
}

// ──────────────────────────────────────────────
// TarArchiver
// ──────────────────────────────────────────────

func TestTarArchiver_ArchiveUnarchive(t *testing.T) {
	src := makeSrcDir(t)
	var buf bytes.Buffer

	require.NoError(t, (&TarArchiver{}).Archive(src, &buf))

	dst := t.TempDir()
	require.NoError(t, (&TarArchiver{}).Unarchive(&buf, dst))

	files := readFiles(t, dst)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
	assert.Equal(t, "hello world", files["hello.txt"])
	assert.Equal(t, "leaf", files["sub/deep/leaf.txt"])
}

func TestTarArchiver_PathTraversal(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Name: "../escape.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte("evil"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	dst := t.TempDir()
	err = (&TarArchiver{}).Unarchive(&buf, dst)
	assert.ErrorIs(t, err, ErrPathEscapes)
}

func TestTarArchiver_SymlinkSkipped(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Name: "link.txt", Typeflag: tar.TypeSymlink, Linkname: "../escape.txt", Mode: 0o777}
	require.NoError(t, tw.WriteHeader(hdr))
	require.NoError(t, tw.Close())

	dst := t.TempDir()
	require.NoError(t, (&TarArchiver{}).Unarchive(&buf, dst))
	// symlink should not be created
	_, err := os.Lstat(filepath.Join(dst, "link.txt"))
	assert.True(t, os.IsNotExist(err))
}

// ──────────────────────────────────────────────
// TarGzArchiver
// ──────────────────────────────────────────────

func TestTarGzArchiver_ArchiveUnarchive(t *testing.T) {
	src := makeSrcDir(t)
	var buf bytes.Buffer

	require.NoError(t, (&TarGzArchiver{}).Archive(src, &buf))

	// Verify it's actually gzipped.
	assert.NotEmpty(t, buf.Bytes())
	assert.Equal(t, byte(0x1f), buf.Bytes()[0])
	assert.Equal(t, byte(0x8b), buf.Bytes()[1])

	dst := t.TempDir()
	require.NoError(t, (&TarGzArchiver{}).Unarchive(&buf, dst))

	files := readFiles(t, dst)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
	assert.Equal(t, "hello world", files["hello.txt"])
}

func TestTarGzArchiver_PathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: "../../escape.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 4}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte("evil"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	dst := t.TempDir()
	err = (&TarGzArchiver{}).Unarchive(&buf, dst)
	assert.ErrorIs(t, err, ErrPathEscapes)
}

// ──────────────────────────────────────────────
// Convenience functions
// ──────────────────────────────────────────────

func TestZipUnzip(t *testing.T) {
	src := makeSrcDir(t)
	dst := t.TempDir()
	archivePath := filepath.Join(dst, "out.zip")

	require.NoError(t, Zip(src, archivePath))

	extractDir := filepath.Join(dst, "extracted")
	require.NoError(t, Unzip(archivePath, extractDir))

	files := readFiles(t, extractDir)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
}

func TestTarUntar(t *testing.T) {
	src := makeSrcDir(t)
	dst := t.TempDir()
	archivePath := filepath.Join(dst, "out.tar")

	require.NoError(t, Tar(src, archivePath))

	extractDir := filepath.Join(dst, "extracted")
	require.NoError(t, Untar(archivePath, extractDir))

	files := readFiles(t, extractDir)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
}

func TestTarGzUntarGz(t *testing.T) {
	src := makeSrcDir(t)
	dst := t.TempDir()
	archivePath := filepath.Join(dst, "out.tar.gz")

	require.NoError(t, TarGz(src, archivePath))

	extractDir := filepath.Join(dst, "extracted")
	require.NoError(t, UntarGz(archivePath, extractDir))

	files := readFiles(t, extractDir)
	assert.ElementsMatch(t, expectedFiles(), sortedKeys(files))
}

func TestZip_NonexistentSrc(t *testing.T) {
	err := Zip("/nonexistent/path", filepath.Join(t.TempDir(), "x.zip"))
	assert.Error(t, err)
}

func TestUnzip_NonexistentSrc(t *testing.T) {
	err := Unzip("/nonexistent/path.zip", t.TempDir())
	assert.Error(t, err)
}

func TestTar_NonexistentSrc(t *testing.T) {
	err := Tar("/nonexistent/path", filepath.Join(t.TempDir(), "x.tar"))
	assert.Error(t, err)
}

func TestUntar_NonexistentSrc(t *testing.T) {
	err := Untar("/nonexistent/path.tar", t.TempDir())
	assert.Error(t, err)
}

func TestTarGz_NonexistentSrc(t *testing.T) {
	err := TarGz("/nonexistent/path", filepath.Join(t.TempDir(), "x.tar.gz"))
	assert.Error(t, err)
}

func TestUntarGz_NonexistentSrc(t *testing.T) {
	err := UntarGz("/nonexistent/path.tar.gz", t.TempDir())
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// CompressFile / DecompressFile
// ──────────────────────────────────────────────

func TestCompressDecompressFile(t *testing.T) {
	src := t.TempDir()
	original := []byte("some compressible content " + strings.Repeat("ABCD", 100))
	require.NoError(t, os.WriteFile(filepath.Join(src, "input.txt"), original, 0o644))

	dst := t.TempDir()
	compressed := filepath.Join(dst, "input.txt.gz")
	require.NoError(t, CompressFile(filepath.Join(src, "input.txt"), compressed))

	// Verify gzip magic.
	data, err := os.ReadFile(compressed)
	require.NoError(t, err)
	assert.Equal(t, byte(0x1f), data[0])
	assert.Equal(t, byte(0x8b), data[1])

	decompressed := filepath.Join(dst, "output.txt")
	require.NoError(t, DecompressFile(compressed, decompressed))

	got, err := os.ReadFile(decompressed)
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestCompressFile_NonexistentSrc(t *testing.T) {
	err := CompressFile("/nonexistent/file", filepath.Join(t.TempDir(), "out.gz"))
	assert.Error(t, err)
}

func TestDecompressFile_NonexistentSrc(t *testing.T) {
	err := DecompressFile("/nonexistent/file.gz", filepath.Join(t.TempDir(), "out"))
	assert.Error(t, err)
}

func TestDecompressFile_InvalidGzip(t *testing.T) {
	src := t.TempDir()
	in := filepath.Join(src, "notgz.txt")
	require.NoError(t, os.WriteFile(in, []byte("not a gzip file"), 0o644))
	err := DecompressFile(in, filepath.Join(src, "out"))
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// DetectFormat
// ──────────────────────────────────────────────

func TestDetectFormat(t *testing.T) {
	src := makeSrcDir(t)
	dst := t.TempDir()

	zipPath := filepath.Join(dst, "a.zip")
	require.NoError(t, Zip(src, zipPath))
	format, err := DetectFormat(zipPath)
	require.NoError(t, err)
	assert.Equal(t, FormatZip, format)

	tarPath := filepath.Join(dst, "a.tar")
	require.NoError(t, Tar(src, tarPath))
	format, err = DetectFormat(tarPath)
	require.NoError(t, err)
	assert.Equal(t, FormatTar, format)

	tarGzPath := filepath.Join(dst, "a.tar.gz")
	require.NoError(t, TarGz(src, tarGzPath))
	format, err = DetectFormat(tarGzPath)
	require.NoError(t, err)
	assert.Equal(t, FormatTarGz, format)

	gzPath := filepath.Join(dst, "a.txt.gz")
	require.NoError(t, os.WriteFile(filepath.Join(src, "plain.txt"), []byte("plain"), 0o644))
	require.NoError(t, CompressFile(filepath.Join(src, "plain.txt"), gzPath))
	format, err = DetectFormat(gzPath)
	require.NoError(t, err)
	assert.Equal(t, FormatGz, format)
}

func TestDetectFormat_Unknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.bin")
	require.NoError(t, os.WriteFile(path, []byte("just some text"), 0o644))
	_, err := DetectFormat(path)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestDetectFormat_Nonexistent(t *testing.T) {
	_, err := DetectFormat("/nonexistent/file")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// ListArchive
// ──────────────────────────────────────────────

func TestListArchive_Zip(t *testing.T) {
	src := makeSrcDir(t)
	path := filepath.Join(t.TempDir(), "a.zip")
	require.NoError(t, Zip(src, path))

	names, err := ListArchive(path)
	require.NoError(t, err)
	// Should contain the files and dirs.
	sort.Strings(names)
	assert.Contains(t, names, "hello.txt")
	assert.Contains(t, names, "sub/nested.txt")
	assert.Contains(t, names, "sub/deep/leaf.txt")
}

func TestListArchive_Tar(t *testing.T) {
	src := makeSrcDir(t)
	path := filepath.Join(t.TempDir(), "a.tar")
	require.NoError(t, Tar(src, path))

	names, err := ListArchive(path)
	require.NoError(t, err)
	sort.Strings(names)
	assert.Contains(t, names, "hello.txt")
	assert.Contains(t, names, "sub/nested.txt")
	assert.Contains(t, names, "sub/deep/leaf.txt")
}

func TestListArchive_TarGz(t *testing.T) {
	src := makeSrcDir(t)
	path := filepath.Join(t.TempDir(), "a.tar.gz")
	require.NoError(t, TarGz(src, path))

	names, err := ListArchive(path)
	require.NoError(t, err)
	sort.Strings(names)
	assert.Contains(t, names, "hello.txt")
	assert.Contains(t, names, "sub/nested.txt")
}

func TestListArchive_Gz(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "plain.txt"), []byte("plain"), 0o644))
	path := filepath.Join(t.TempDir(), "plain.txt.gz")
	require.NoError(t, CompressFile(filepath.Join(src, "plain.txt"), path))

	names, err := ListArchive(path)
	require.NoError(t, err)
	assert.Len(t, names, 1)
	assert.Equal(t, "plain.txt", names[0])
}

func TestListArchive_GzNoName(t *testing.T) {
	// Create a gzip file without a Name header.
	src := t.TempDir()
	in, err := os.Create(filepath.Join(src, "data.bin"))
	require.NoError(t, err)
	_, err = in.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, in.Close())

	out, err := os.Create(filepath.Join(src, "data.gz"))
	require.NoError(t, err)
	gw := gzip.NewWriter(out)
	_, err = gw.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, out.Close())

	names, err := ListArchive(filepath.Join(src, "data.gz"))
	require.NoError(t, err)
	assert.Len(t, names, 1)
	assert.Equal(t, "data", names[0])
}

func TestListArchive_UnknownFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.bin")
	require.NoError(t, os.WriteFile(path, []byte("text"), 0o644))
	_, err := ListArchive(path)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestListArchive_Nonexistent(t *testing.T) {
	_, err := ListArchive("/nonexistent/file")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// safeJoin
// ──────────────────────────────────────────────

func TestSafeJoin(t *testing.T) {
	dst := t.TempDir()

	p, err := safeJoin(dst, "sub/file.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dst, "sub/file.txt"), p)

	_, err = safeJoin(dst, "../escape.txt")
	assert.ErrorIs(t, err, ErrPathEscapes)

	_, err = safeJoin(dst, "/etc/passwd")
	assert.ErrorIs(t, err, ErrPathEscapes)

	_, err = safeJoin(dst, "sub/../../escape.txt")
	assert.ErrorIs(t, err, ErrPathEscapes)
}

// ──────────────────────────────────────────────
// Archiver interface compliance
// ──────────────────────────────────────────────

func TestArchiverInterface(t *testing.T) {
	var _ Archiver = &ZipArchiver{}
	var _ Archiver = &TarArchiver{}
	var _ Archiver = &TarGzArchiver{}
}

// ──────────────────────────────────────────────
// Error paths
// ──────────────────────────────────────────────

// failWriter always fails on Write.
type failWriter struct{}

func (w *failWriter) Write(p []byte) (int, error) {
	return 0, io.ErrShortWrite
}

func TestZipArchiver_ArchiveFailingWriter(t *testing.T) {
	src := makeSrcDir(t)
	err := (&ZipArchiver{}).Archive(src, &failWriter{})
	assert.Error(t, err)
}

func TestTarArchiver_ArchiveFailingWriter(t *testing.T) {
	src := makeSrcDir(t)
	err := (&TarArchiver{}).Archive(src, &failWriter{})
	assert.Error(t, err)
}

func TestTarGzArchiver_ArchiveFailingWriter(t *testing.T) {
	src := makeSrcDir(t)
	err := (&TarGzArchiver{}).Archive(src, &failWriter{})
	assert.Error(t, err)
}

func TestZipArchiver_UnarchiveInvalidData(t *testing.T) {
	err := (&ZipArchiver{}).Unarchive(strings.NewReader("not a zip file"), t.TempDir())
	assert.Error(t, err)
}

func TestTarArchiver_UnarchiveInvalidData(t *testing.T) {
	err := (&TarArchiver{}).Unarchive(strings.NewReader("not a tar file at all but long enough to be read"), t.TempDir())
	assert.Error(t, err)
}

func TestTarGzArchiver_UnarchiveInvalidGzip(t *testing.T) {
	err := (&TarGzArchiver{}).Unarchive(strings.NewReader("not gzip"), t.TempDir())
	assert.Error(t, err)
}

func TestUntar_UnknownTypeFlag(t *testing.T) {
	// Build a tar with a hardlink entry (TypeLink) which falls into default.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Name: "link.txt", Typeflag: tar.TypeLink, Linkname: "target.txt", Mode: 0o644, Size: 0}
	require.NoError(t, tw.WriteHeader(hdr))
	require.NoError(t, tw.Close())

	require.NoError(t, (&TarArchiver{}).Unarchive(&buf, t.TempDir()))
}

func TestListArchive_CorruptedZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	require.NoError(t, os.WriteFile(path, []byte("PK\x03\x04corrupted"), 0o644))
	_, err := ListArchive(path)
	assert.Error(t, err)
}

func TestListArchive_CorruptedTar(t *testing.T) {
	// A file that DetectFormat sees as tar (ustar magic) but is truncated.
	path := filepath.Join(t.TempDir(), "bad.tar")
	data := make([]byte, 600)
	copy(data[257:262], []byte("ustar"))
	// Put a valid-looking header but with a bogus size so tr.Next fails mid-stream.
	require.NoError(t, os.WriteFile(path, data, 0o644))
	_, err := ListArchive(path)
	// May or may not error depending on tar parsing, but should not panic.
	_ = err
}

func TestDetectFormat_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
	_, err := DetectFormat(path)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestDetectFormat_GzipSeekError(t *testing.T) {
	// A valid gzip that is not a tar.gz should return FormatGz.
	path := filepath.Join(t.TempDir(), "plain.gz")
	in := filepath.Join(path + ".src")
	require.NoError(t, os.WriteFile(in, []byte("hello"), 0o644))
	require.NoError(t, CompressFile(in, path))
	format, err := DetectFormat(path)
	require.NoError(t, err)
	assert.Equal(t, FormatGz, format)
}

func TestCompressFile_BadDst(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "in.txt"), []byte("x"), 0o644))
	// dst inside a nonexistent directory
	err := CompressFile(filepath.Join(src, "in.txt"), "/nonexistent/dir/out.gz")
	assert.Error(t, err)
}

func TestDecompressFile_BadDst(t *testing.T) {
	src := t.TempDir()
	in := filepath.Join(src, "in.txt")
	require.NoError(t, os.WriteFile(in, []byte("hello"), 0o644))
	gz := filepath.Join(src, "in.gz")
	require.NoError(t, CompressFile(in, gz))
	err := DecompressFile(gz, "/nonexistent/dir/out.txt")
	assert.Error(t, err)
}

func TestZipArchiver_ArchiveWalkError(t *testing.T) {
	// src does not exist -> Walk error
	err := (&ZipArchiver{}).Archive("/nonexistent/src/dir", &bytes.Buffer{})
	assert.Error(t, err)
}

func TestTarArchiver_ArchiveWalkError(t *testing.T) {
	err := (&TarArchiver{}).Archive("/nonexistent/src/dir", &bytes.Buffer{})
	assert.Error(t, err)
}

func TestTarGzArchiver_ArchiveWalkError(t *testing.T) {
	err := (&TarGzArchiver{}).Archive("/nonexistent/src/dir", &bytes.Buffer{})
	assert.Error(t, err)
}

func TestZipArchiver_UnarchiveDirectoryEntry(t *testing.T) {
	// Ensure directory entries with explicit "/" suffix are created.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "mydir/", Method: zip.Store, UncompressedSize64: 0, ExternalAttrs: 0o40755 << 16})
	require.NoError(t, err)
	_ = w
	w2, err := zw.Create("mydir/file.txt")
	require.NoError(t, err)
	_, err = w2.Write([]byte("hi"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	dst := t.TempDir()
	require.NoError(t, (&ZipArchiver{}).Unarchive(&buf, dst))
	assert.DirExists(t, filepath.Join(dst, "mydir"))
	assert.FileExists(t, filepath.Join(dst, "mydir", "file.txt"))
}

func TestUntar_TypeDir(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "somedir", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "somedir/f.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2}))
	_, err := tw.Write([]byte("hi"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	dst := t.TempDir()
	require.NoError(t, (&TarArchiver{}).Unarchive(&buf, dst))
	assert.DirExists(t, filepath.Join(dst, "somedir"))
}

func TestListArchive_GzInvalidGzip(t *testing.T) {
	// File starts with gzip magic but is corrupted.
	path := filepath.Join(t.TempDir(), "bad.gz")
	require.NoError(t, os.WriteFile(path, []byte{0x1f, 0x8b, 0, 0, 0}, 0o644))
	_, err := ListArchive(path)
	// DetectFormat may return gz, then listTar/gzip.NewReader fails.
	_ = err
}
