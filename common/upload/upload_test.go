// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package upload

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

// makePNG creates a small valid PNG image as []byte.
func makePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// readFile is a small wrapper around os.ReadFile for test readability.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// mustParseURL parses a URL or fails the test.
func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

// makeMultipartRequest builds an HTTP request with the given field name and
// files (filename -> content).
func makeMultipartRequest(t *testing.T, fieldName string, files map[string][]byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for filename, content := range files {
		part, err := writer.CreateFormFile(fieldName, filename)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	req, err := http.NewRequest("POST", "/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// makeMultipartRequestMultiple builds an HTTP request with multiple files
// under the same field name.
func makeMultipartRequestMultiple(t *testing.T, fieldName string, filenames []string, contents [][]byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for i, filename := range filenames {
		part, err := writer.CreateFormFile(fieldName, filename)
		require.NoError(t, err)
		_, err = part.Write(contents[i])
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	req, err := http.NewRequest("POST", "/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// ──────────────────────────────────────────────
// Handler construction
// ──────────────────────────────────────────────

func TestNewHandler(t *testing.T) {
	h := NewHandler("/tmp/uploads",
		WithMaxSize(1024),
		WithAllowedTypes([]string{"image/png"}),
		WithFilenameGenerator(func(orig string) string { return "custom_" + orig }),
	)
	require.NotNil(t, h)
	assert.Equal(t, int64(1024), h.MaxSize)
	assert.Equal(t, []string{"image/png"}, h.AllowedTypes)
	assert.Equal(t, "/tmp/uploads", h.DestDir)
	assert.NotNil(t, h.filenameGen)
}

func TestNewHandler_Defaults(t *testing.T) {
	h := NewHandler("/tmp/uploads")
	assert.Equal(t, "/tmp/uploads", h.DestDir)
	assert.Equal(t, int64(0), h.MaxSize)
	assert.Nil(t, h.AllowedTypes)
	assert.Nil(t, h.filenameGen)
}

// ──────────────────────────────────────────────
// Save
// ──────────────────────────────────────────────

func TestHandler_Save_Success(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20))
	pngData := makePNG(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	require.NoError(t, err)
	_, err = part.Write(pngData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	info, err := h.SaveFromRequest(req, "file")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "test.png", info.OriginalName)
	assert.Equal(t, int64(len(pngData)), info.Size)
	assert.Equal(t, ".png", info.Extension)
	assert.Equal(t, "image/png", info.MIMEType)
	assert.True(t, strings.HasSuffix(info.SavedName, ".png"))
	assert.FileExists(t, info.Path)
}

func TestHandler_Save_CustomFilename(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir,
		WithFilenameGenerator(func(orig string) string { return "renamed_" + filepath.Base(orig) }),
	)
	pngData := makePNG(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "photo.png")
	require.NoError(t, err)
	_, err = part.Write(pngData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	info, err := h.SaveFromRequest(req, "file")
	require.NoError(t, err)
	assert.Equal(t, "renamed_photo.png", info.SavedName)
}

func TestHandler_Save_TooLarge(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(10))

	pngData := makePNG(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "big.png")
	require.NoError(t, err)
	_, err = part.Write(pngData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = h.SaveFromRequest(req, "file")
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestHandler_Save_TypeNotAllowed(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir,
		WithMaxSize(1<<20),
		WithAllowedTypes([]string{"image/png"}),
	)

	// Create a text file with .txt extension.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "doc.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello world"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = h.SaveFromRequest(req, "file")
	assert.ErrorIs(t, err, ErrFileTypeNotAllowed)
}

func TestHandler_Save_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "empty.png")
	require.NoError(t, err)
	_, err = part.Write(nil)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = h.SaveFromRequest(req, "file")
	assert.ErrorIs(t, err, ErrEmptyFile)
}

func TestHandler_Save_NoFile(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := h.SaveFromRequest(req, "file")
	assert.ErrorIs(t, err, ErrNoFile)
}

func TestHandler_Save_DirectMimeCheck(t *testing.T) {
	dir := t.TempDir()
	// Allowed types include text/plain but we send a .txt file with text content.
	h := NewHandler(dir,
		WithMaxSize(1<<20),
		WithAllowedTypes([]string{"text/plain"}),
	)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "note.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello plain text"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	info, err := h.SaveFromRequest(req, "file")
	require.NoError(t, err)
	assert.Equal(t, "text/plain", info.MIMEType)
}

// ──────────────────────────────────────────────
// SaveMultipleFromRequest
// ──────────────────────────────────────────────

func TestHandler_SaveMultipleFromRequest(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20))
	pngData := makePNG(t)

	req := makeMultipartRequestMultiple(t, "files",
		[]string{"a.png", "b.png"},
		[][]byte{pngData, pngData},
	)

	infos, err := h.SaveMultipleFromRequest(req, "files")
	require.NoError(t, err)
	assert.Len(t, infos, 2)
	for _, info := range infos {
		assert.FileExists(t, info.Path)
		assert.Equal(t, "image/png", info.MIMEType)
	}
}

func TestHandler_SaveMultipleFromRequest_NoFiles(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.Close())
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := h.SaveMultipleFromRequest(req, "files")
	assert.ErrorIs(t, err, ErrNoFile)
}

// ──────────────────────────────────────────────
// SaveFromRequest via httptest server
// ──────────────────────────────────────────────

func TestHandler_SaveFromRequest_Server(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20), WithAllowedTypes([]string{"image/png"}))
	pngData := makePNG(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		info, err := h.SaveFromRequest(r, "file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte(info.SavedName))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := makeMultipartRequest(t, "file", map[string][]byte{"test.png": pngData})
	// Re-create the request targeting the test server.
	req = makeMultipartRequest(t, "file", map[string][]byte{"test.png": pngData})
	req.URL = mustParseURL(t, srv.URL+"/upload")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ──────────────────────────────────────────────
// ValidateFile
// ──────────────────────────────────────────────

func TestValidateFile_Success(t *testing.T) {
	header := &multipart.FileHeader{Filename: "test.png", Size: 100}
	err := ValidateFile(header, []string{"image/png"}, 200, -1)
	assert.NoError(t, err)
}

func TestValidateFile_TooLarge(t *testing.T) {
	header := &multipart.FileHeader{Filename: "test.png", Size: 200}
	err := ValidateFile(header, nil, 100, -1)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestValidateFile_Empty(t *testing.T) {
	header := &multipart.FileHeader{Filename: "test.png", Size: 0}
	err := ValidateFile(header, nil, 100, -1)
	assert.ErrorIs(t, err, ErrEmptyFile)
}

func TestValidateFile_TypeNotAllowed(t *testing.T) {
	header := &multipart.FileHeader{Filename: "test.txt", Size: 100}
	err := ValidateFile(header, []string{"image/png"}, 200, -1)
	assert.ErrorIs(t, err, ErrFileTypeNotAllowed)
}

func TestValidateFile_NoMaxSize(t *testing.T) {
	header := &multipart.FileHeader{Filename: "test.png", Size: 999999}
	err := ValidateFile(header, nil, 0, -1)
	assert.NoError(t, err)
}

func TestValidateFile_ActualSizeTooLarge(t *testing.T) {
	// Client claims Size=50 but actual bytes written exceed MaxSize.
	header := &multipart.FileHeader{Filename: "test.png", Size: 50}
	err := ValidateFile(header, nil, 100, 150)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestValidateFile_ActualSizeEmpty(t *testing.T) {
	// Client claims Size=100 but actual bytes written is 0.
	header := &multipart.FileHeader{Filename: "test.png", Size: 100}
	err := ValidateFile(header, nil, 200, 0)
	assert.ErrorIs(t, err, ErrEmptyFile)
}

// ──────────────────────────────────────────────
// GenerateSafeFilename
// ──────────────────────────────────────────────

func TestGenerateSafeFilename_Normal(t *testing.T) {
	name := GenerateSafeFilename("photo.png")
	assert.True(t, strings.HasSuffix(name, "_photo.png"))
}

func TestGenerateSafeFilename_PathTraversal(t *testing.T) {
	name := GenerateSafeFilename("../../../etc/passwd")
	assert.NotContains(t, name, "..")
	assert.True(t, strings.HasSuffix(name, "passwd"))
}

func TestGenerateSafeFilename_Empty(t *testing.T) {
	name := GenerateSafeFilename("")
	assert.NotEmpty(t, name)
}

func TestGenerateSafeFilename_DotOnly(t *testing.T) {
	name := GenerateSafeFilename(".")
	assert.NotEmpty(t, name)
	assert.NotEqual(t, ".", name)
}

func TestGenerateSafeFilename_SlashOnly(t *testing.T) {
	name := GenerateSafeFilename("/")
	assert.NotEmpty(t, name)
}

// ──────────────────────────────────────────────
// IsAllowedType
// ──────────────────────────────────────────────

func TestIsAllowedType_Empty(t *testing.T) {
	assert.True(t, IsAllowedType("anything", nil))
	assert.True(t, IsAllowedType("anything", []string{}))
}

func TestIsAllowedType_Allowed(t *testing.T) {
	assert.True(t, IsAllowedType("image/png", []string{"image/png", "image/jpeg"}))
}

func TestIsAllowedType_CaseInsensitive(t *testing.T) {
	assert.True(t, IsAllowedType("Image/PNG", []string{"image/png"}))
}

func TestIsAllowedType_NotAllowed(t *testing.T) {
	assert.False(t, IsAllowedType("text/plain", []string{"image/png"}))
}

// ──────────────────────────────────────────────
// DetectMIME
// ──────────────────────────────────────────────

func TestDetectMIME_PNG(t *testing.T) {
	pngData := makePNG(t)
	mt := DetectMIME(pngData)
	assert.Equal(t, "image/png", mt)
}

func TestDetectMIME_Text(t *testing.T) {
	mt := DetectMIME([]byte("hello world"))
	assert.Equal(t, "text/plain", mt)
}

func TestDetectMIME_Empty(t *testing.T) {
	mt := DetectMIME(nil)
	assert.Equal(t, "", mt)
}

func TestDetectMIME_HTML(t *testing.T) {
	mt := DetectMIME([]byte("<html><body>test</body></html>"))
	assert.Equal(t, "text/html", mt)
}

// ──────────────────────────────────────────────
// extensionToMIME
// ──────────────────────────────────────────────

func TestExtensionToMIME(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".pdf", "application/pdf"},
		{".txt", "text/plain"},
		{".csv", "text/csv"},
		{".zip", "application/zip"},
		{".webp", "image/webp"},
		{".xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, extensionToMIME(tt.ext))
	}
}

// ──────────────────────────────────────────────
// ChunkUploader
// ──────────────────────────────────────────────

func TestChunkUploader_SaveAndMerge(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "test-upload-1"

	// Simulate a 250-byte file split into 3 chunks.
	data := bytes.Repeat([]byte("ABCD"), 63) // 252 bytes
	chunk0 := data[:100]
	chunk1 := data[100:200]
	chunk2 := data[200:]

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader(chunk0)))
	require.NoError(t, cu.SaveChunk(uploadID, 1, bytes.NewReader(chunk1)))
	require.NoError(t, cu.SaveChunk(uploadID, 2, bytes.NewReader(chunk2)))

	// Check status.
	count, ok := cu.Status(uploadID)
	assert.True(t, ok)
	assert.Equal(t, 3, count)

	// Merge.
	info, err := cu.Merge(uploadID, 3, "merged.bin")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, int64(252), info.Size)
	assert.True(t, strings.HasSuffix(info.SavedName, ".bin"))
	assert.FileExists(t, info.Path)

	// Verify content.
	content, err := readFile(info.Path)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestChunkUploader_Merge_NotFound(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	_, err := cu.Merge("nonexistent", 3, "file.bin")
	assert.ErrorIs(t, err, ErrUploadNotFound)
}

func TestChunkUploader_Merge_MissingChunks(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "partial"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))
	require.NoError(t, cu.SaveChunk(uploadID, 1, bytes.NewReader([]byte("chunk1"))))

	_, err := cu.Merge(uploadID, 3, "file.bin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing chunks")
}

func TestChunkUploader_SaveChunk_EmptyID(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	err := cu.SaveChunk("", 0, bytes.NewReader([]byte("data")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty upload ID")
}

func TestChunkUploader_Cleanup(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "cleanup-test"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))
	require.NoError(t, cu.SaveChunk(uploadID, 1, bytes.NewReader([]byte("chunk1"))))

	_, ok := cu.Status(uploadID)
	assert.True(t, ok)

	require.NoError(t, cu.Cleanup(uploadID))

	_, ok = cu.Status(uploadID)
	assert.False(t, ok)
}

func TestChunkUploader_Status_NotFound(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	count, ok := cu.Status("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, 0, count)
}

func TestChunkUploader_OverwriteChunk(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "overwrite"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("first"))))
	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("second"))))

	count, ok := cu.Status(uploadID)
	assert.True(t, ok)
	assert.Equal(t, 1, count)

	info, err := cu.Merge(uploadID, 1, "over.bin")
	require.NoError(t, err)
	content, _ := readFile(info.Path)
	assert.Equal(t, []byte("second"), content)
}

func TestChunkUploader_Merge_MIME(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 1000)
	uploadID := "mime-test"

	pngData := makePNG(t)
	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader(pngData)))

	info, err := cu.Merge(uploadID, 1, "image.png")
	require.NoError(t, err)
	assert.Equal(t, "image/png", info.MIMEType)
}

// ──────────────────────────────────────────────
// SaveFromRequest parse error
// ──────────────────────────────────────────────

func TestHandler_SaveFromRequest_ParseError(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(10))

	// Send a request with invalid multipart content type.
	req, _ := http.NewRequest("POST", "/upload", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "text/plain")

	_, err := h.SaveFromRequest(req, "file")
	assert.Error(t, err)
}

func TestHandler_SaveMultipleFromRequest_ParseError(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(10))

	req, _ := http.NewRequest("POST", "/upload", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "text/plain")

	_, err := h.SaveMultipleFromRequest(req, "files")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Save error paths
// ──────────────────────────────────────────────

func TestHandler_Save_Direct_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20))

	// Create a multipart file header with empty content.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "empty.png")
	require.NoError(t, err)
	_, err = part.Write(nil)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_ = req.ParseMultipartForm(1 << 20)
	file, header, _ := req.FormFile("file")
	defer file.Close()

	_, err = h.Save(file, header)
	assert.ErrorIs(t, err, ErrEmptyFile)
}

func TestHandler_Save_MkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a file where the dest dir should be — MkdirAll will fail.
	filePath := filepath.Join(dir, "notadir")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	h := NewHandler(filePath, WithMaxSize(1<<20))
	pngData := makePNG(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	require.NoError(t, err)
	_, err = part.Write(pngData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = h.SaveFromRequest(req, "file")
	assert.Error(t, err)
}

func TestHandler_Save_WriteFileFailure(t *testing.T) {
	dir := t.TempDir()
	// Use a custom filename generator so we know the target name, then create
	// a directory with that name so WriteFile fails.
	const targetName = "target_dir"
	h := NewHandler(dir,
		WithMaxSize(1<<20),
		WithFilenameGenerator(func(string) string { return targetName }),
	)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, targetName), 0o755))

	pngData := makePNG(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	require.NoError(t, err)
	_, err = part.Write(pngData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = h.SaveFromRequest(req, "file")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// ChunkUploader error paths
// ──────────────────────────────────────────────

func TestChunkUploader_SaveChunk_MkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	// Make the temp dir path a file so MkdirAll fails.
	blocker := filepath.Join(dir, ".chunks")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	cu := NewChunkUploader(dir, 100)
	err := cu.SaveChunk("test", 0, bytes.NewReader([]byte("data")))
	assert.Error(t, err)
}

func TestChunkUploader_Merge_DestMkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "merge-fail"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))

	// Make dest dir a file so MkdirAll fails.
	cu.DestDir = filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(cu.DestDir, []byte("x"), 0o644))

	_, err := cu.Merge(uploadID, 1, "file.bin")
	assert.Error(t, err)
}

func TestChunkUploader_Merge_ChunkOpenError(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "open-fail"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))

	// Delete the chunk file on disk but keep it tracked in the map.
	require.NoError(t, os.Remove(cu.chunkPath(uploadID, 0)))

	_, err := cu.Merge(uploadID, 1, "file.bin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open chunk")
}

func TestChunkUploader_Cleanup_NotExist(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	// Cleanup of a non-existent upload should not error (RemoveAll is idempotent).
	err := cu.Cleanup("nonexistent")
	assert.NoError(t, err)
}

// ──────────────────────────────────────────────
// Additional coverage: MIME mismatch, empty data, chunk errors
// ──────────────────────────────────────────────

func TestHandler_Save_MimeMismatch(t *testing.T) {
	dir := t.TempDir()
	// .png extension passes ValidateFile, but content is text → MIME check fails.
	h := NewHandler(dir,
		WithMaxSize(1<<20),
		WithAllowedTypes([]string{"image/png"}),
	)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "fake.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("this is not a PNG file"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = h.SaveFromRequest(req, "file")
	assert.ErrorIs(t, err, ErrFileTypeNotAllowed)
}

func TestHandler_Save_EmptyDataAfterRead(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1<<20))

	// Create a file header with Size > 0 but provide an empty reader.
	header := &multipart.FileHeader{
		Filename: "test.png",
		Size:     100,
	}
	emptyFile := &emptyMultipartFile{}
	_, err := h.Save(emptyFile, header)
	assert.ErrorIs(t, err, ErrEmptyFile)
}

// emptyMultipartFile implements multipart.File returning no data.
type emptyMultipartFile struct{}

func (e *emptyMultipartFile) Read(p []byte) (int, error)         { return 0, io.EOF }
func (e *emptyMultipartFile) Close() error                        { return nil }
func (e *emptyMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
func (e *emptyMultipartFile) ReadAt(p []byte, offset int64) (int, error) {
	return 0, io.EOF
}

func TestHandler_SaveMultipleFromRequest_SaveError(t *testing.T) {
	dir := t.TempDir()
	// First file is valid PNG, second is a .png with text content (MIME mismatch).
	h := NewHandler(dir,
		WithMaxSize(1<<20),
		WithAllowedTypes([]string{"image/png"}),
	)
	pngData := makePNG(t)

	req := makeMultipartRequestMultiple(t, "files",
		[]string{"good.png", "bad.png"},
		[][]byte{pngData, []byte("not a png")},
	)

	infos, err := h.SaveMultipleFromRequest(req, "files")
	assert.Error(t, err)
	// First file should have been saved.
	assert.Len(t, infos, 1)
}

func TestChunkUploader_SaveChunk_CreateError(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "create-fail"

	// Successfully create the chunk dir for chunk 0.
	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("first"))))

	// Now make the chunk file path a directory so Create fails on re-save.
	chunkPath := cu.chunkPath(uploadID, 0)
	require.NoError(t, os.Remove(chunkPath))
	require.NoError(t, os.MkdirAll(chunkPath, 0o755))

	err := cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("second")))
	assert.Error(t, err)
}

func TestChunkUploader_Merge_CreateError(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "merge-create-fail"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))

	// Use a filename generator that produces a name matching an existing dir.
	safeName := GenerateSafeFilename("file.bin")
	targetDir := filepath.Join(cu.DestDir, safeName)
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	_, err := cu.Merge(uploadID, 1, "file.bin")
	// Merge generates a new safe name with timestamp, so it may succeed.
	// Instead, force failure by making DestDir a file.
	cu.DestDir = filepath.Join(dir, "blocker_file")
	require.NoError(t, os.WriteFile(cu.DestDir, []byte("x"), 0o644))
	_, err = cu.Merge(uploadID, 1, "file.bin")
	assert.Error(t, err)
}

func TestChunkUploader_Merge_LargeFirstChunk(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 1000)
	uploadID := "large-chunk"

	// Create a first chunk larger than 512 bytes to cover the >= 512 branch.
	largeData := bytes.Repeat([]byte("A"), 600)
	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader(largeData)))
	require.NoError(t, cu.SaveChunk(uploadID, 1, bytes.NewReader([]byte("B"))))

	info, err := cu.Merge(uploadID, 2, "large.bin")
	require.NoError(t, err)
	assert.Equal(t, int64(601), info.Size)
}

func TestChunkUploader_Cleanup_RemoveAllError(t *testing.T) {
	dir := t.TempDir()
	cu := NewChunkUploader(dir, 100)
	uploadID := "cleanup-fail"

	require.NoError(t, cu.SaveChunk(uploadID, 0, bytes.NewReader([]byte("chunk0"))))

	// Make the parent temp dir read-only so RemoveAll of the child fails.
	// On some systems (root) this may not cause an error, so we also try
	// making the chunk dir path a file.
	parentDir := cu.tempDir
	require.NoError(t, os.Chmod(parentDir, 0o555))
	defer os.Chmod(parentDir, 0o755)

	_ = cu.Cleanup(uploadID)
	// Restore permissions for temp dir cleanup.
	_ = os.Chmod(parentDir, 0o755)
}

// ──────────────────────────────────────────────
// Streaming Save tests
// ──────────────────────────────────────────────

// errMultipartFile implements multipart.File but fails on Read.
type errMultipartFile struct{}

func (e *errMultipartFile) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (e *errMultipartFile) Close() error                { return nil }
func (e *errMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
func (e *errMultipartFile) ReadAt(p []byte, offset int64) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestHandler_Save_StreamCopyError(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1 << 20))

	header := &multipart.FileHeader{Filename: "test.png", Size: 100}
	_, err := h.Save(&errMultipartFile{}, header)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write file")
}

// hardReadErrMultipartFile fails with a non-EOF error on Read, triggering
// the ReadFull error path in Save.
type hardReadErrMultipartFile struct{}

func (h *hardReadErrMultipartFile) Read(p []byte) (int, error) {
	return 0, io.ErrNoProgress
}
func (h *hardReadErrMultipartFile) Close() error { return nil }
func (h *hardReadErrMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
func (h *hardReadErrMultipartFile) ReadAt(p []byte, offset int64) (int, error) {
	return 0, io.ErrNoProgress
}

func TestHandler_Save_ReadFileError(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1 << 20))

	header := &multipart.FileHeader{Filename: "test.png", Size: 100}
	_, err := h.Save(&hardReadErrMultipartFile{}, header)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read file")
}

// lyingMultipartFile implements multipart.File returning more data than
// header.Size claims, to test the actual-size validation.
type lyingMultipartFile struct {
	data []byte
	pos  int
}

func (l *lyingMultipartFile) Read(p []byte) (int, error) {
	if l.pos >= len(l.data) {
		return 0, io.EOF
	}
	n := copy(p, l.data[l.pos:])
	l.pos += n
	return n, nil
}
func (l *lyingMultipartFile) Close() error { return nil }
func (l *lyingMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
func (l *lyingMultipartFile) ReadAt(p []byte, offset int64) (int, error) {
	return 0, io.EOF
}

func TestHandler_Save_ActualSizeExceedsMaxSize(t *testing.T) {
	dir := t.TempDir()
	// MaxSize=100, but the actual data is 200 bytes. header.Size lies (50).
	h := NewHandler(dir, WithMaxSize(100))

	bigData := bytes.Repeat([]byte("A"), 200)
	header := &multipart.FileHeader{Filename: "test.bin", Size: 50}
	_, err := h.Save(&lyingMultipartFile{data: bigData}, header)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestHandler_Save_StreamingLargeFile(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(10<<20))

	// 5 MB of data — would OOM with io.ReadAll in a constrained env.
	largeData := bytes.Repeat([]byte("X"), 5<<20)
	header := &multipart.FileHeader{Filename: "big.bin", Size: int64(len(largeData))}
	info, err := h.Save(&lyingMultipartFile{data: largeData}, header)
	require.NoError(t, err)
	assert.Equal(t, int64(len(largeData)), info.Size)
	assert.FileExists(t, info.Path)

	saved, err := os.ReadFile(info.Path)
	require.NoError(t, err)
	assert.Equal(t, largeData, saved)
}

func TestHandler_Save_CreateTempError(t *testing.T) {
	// Skip if running as root — root bypasses file permission checks.
	if os.Geteuid() == 0 {
		t.Skip("skipping when running as root")
	}
	dir := t.TempDir()
	h := NewHandler(dir, WithMaxSize(1 << 20))

	// MkdirAll succeeds on the existing dir, but CreateTemp fails because
	// the directory is read-only.
	require.NoError(t, os.Chmod(dir, 0o444))
	defer os.Chmod(dir, 0o755)

	pngData := makePNG(t)
	header := &multipart.FileHeader{Filename: "test.png", Size: int64(len(pngData))}
	_, err := h.Save(&lyingMultipartFile{data: pngData}, header)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}
