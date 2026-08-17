// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package local

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/ling-base/stores"
)

// ──────────────────────────────────────────────
// Bucket management tests
// ──────────────────────────────────────────────

func TestManager_CreateBucket(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	err := s.CreateBucket(&stores.CreateBucketRequest{
		Name:   "test-bucket",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Duplicate should fail.
	if err := s.CreateBucket(&stores.CreateBucketRequest{Name: "test-bucket"}); err == nil {
		t.Fatal("expected error for duplicate bucket")
	}

	// Invalid name should fail.
	if err := s.CreateBucket(&stores.CreateBucketRequest{Name: "x"}); err == nil {
		t.Fatal("expected error for short bucket name")
	}
}

func TestManager_CreateBucket_InvalidRequest(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if err := s.CreateBucket(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
	if err := s.CreateBucket(&stores.CreateBucketRequest{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestManager_ListBuckets(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "bucket-a"})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "bucket-b"})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "other-c"})

	// List all.
	resp, err := s.ListBuckets(nil)
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	if len(resp.Buckets) != 3 {
		t.Errorf("buckets = %d, want 3", len(resp.Buckets))
	}

	// Filter by prefix.
	resp, _ = s.ListBuckets(&stores.ListBucketsRequest{Prefix: "bucket"})
	if len(resp.Buckets) != 2 {
		t.Errorf("filtered buckets = %d, want 2", len(resp.Buckets))
	}

	// Limit.
	resp, _ = s.ListBuckets(&stores.ListBucketsRequest{MaxKeys: 1})
	if len(resp.Buckets) != 1 {
		t.Errorf("limited buckets = %d, want 1", len(resp.Buckets))
	}
}

func TestManager_DeleteBucket(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "to-delete"})

	if err := s.DeleteBucket("to-delete"); err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}

	// Second delete should fail.
	if err := s.DeleteBucket("to-delete"); err == nil {
		t.Fatal("expected error for deleting non-existent bucket")
	}
}

func TestManager_DeleteBucket_NotEmpty(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "non-empty"})
	s.UploadFile("non-empty", "file.txt", bytes.NewReader([]byte("data")), 4)

	if err := s.DeleteBucket("non-empty"); err == nil {
		t.Fatal("expected error for deleting non-empty bucket")
	}
}

func TestManager_GetBucketInfo(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{
		Name:      "info-bucket",
		Region:    "eu-west-1",
		IsPrivate: true,
		Tags:      map[string]string{"env": "test"},
	})

	info, err := s.GetBucketInfo("info-bucket")
	if err != nil {
		t.Fatalf("GetBucketInfo: %v", err)
	}
	if info.Name != "info-bucket" {
		t.Errorf("Name = %q", info.Name)
	}
	if info.Region != "eu-west-1" {
		t.Errorf("Region = %q", info.Region)
	}
	if !info.IsPrivate {
		t.Error("IsPrivate should be true")
	}
	if info.Tags["env"] != "test" {
		t.Errorf("Tags = %v", info.Tags)
	}
}

func TestManager_GetBucketInfo_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.GetBucketInfo("nope")
	if err == nil {
		t.Fatal("expected error for non-existent bucket")
	}
}

func TestManager_SetBucketPrivate(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "priv-bucket", IsPrivate: false})

	if err := s.SetBucketPrivate("priv-bucket", true); err != nil {
		t.Fatalf("SetBucketPrivate: %v", err)
	}

	info, _ := s.GetBucketInfo("priv-bucket")
	if !info.IsPrivate {
		t.Error("IsPrivate should be true after SetBucketPrivate")
	}
}

func TestManager_SetBucketPrivate_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if err := s.SetBucketPrivate("nope", true); err == nil {
		t.Fatal("expected error for non-existent bucket")
	}
}

func TestManager_GetBucketDomains(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "domain-bucket"})

	domains, err := s.GetBucketDomains("domain-bucket")
	if err != nil {
		t.Fatalf("GetBucketDomains: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("domains = %v, want empty", domains)
	}
}

func TestManager_GetBucketDomains_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.GetBucketDomains("nope")
	if err == nil {
		t.Fatal("expected error for non-existent bucket")
	}
}

// ──────────────────────────────────────────────
// Object management tests
// ──────────────────────────────────────────────

func TestManager_UploadAndListFiles(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "files-bucket"})

	// Upload files.
	s.UploadFile("files-bucket", "a.txt", bytes.NewReader([]byte("aaa")), 3)
	s.UploadFile("files-bucket", "b.txt", bytes.NewReader([]byte("bbb")), 3)
	s.UploadFile("files-bucket", "sub/c.txt", bytes.NewReader([]byte("ccc")), 3)

	// List all.
	resp, err := s.ListFiles("files-bucket", nil)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(resp.Files) != 3 {
		t.Errorf("files = %d, want 3", len(resp.Files))
	}

	// List with prefix.
	resp, _ = s.ListFiles("files-bucket", &stores.ListFilesRequest{Prefix: "sub/"})
	if len(resp.Files) != 1 {
		t.Errorf("prefixed files = %d, want 1", len(resp.Files))
	}

	// List with delimiter (virtual directories).
	resp, _ = s.ListFiles("files-bucket", &stores.ListFilesRequest{Delimiter: "/"})
	if len(resp.CommonPrefixes) != 1 {
		t.Errorf("common prefixes = %v, want [sub/]", resp.CommonPrefixes)
	}
	if resp.CommonPrefixes[0] != "sub/" {
		t.Errorf("common prefix = %q, want %q", resp.CommonPrefixes[0], "sub/")
	}
	// Top-level files should be a.txt and b.txt (not sub/c.txt).
	if len(resp.Files) != 2 {
		t.Errorf("top-level files = %d, want 2", len(resp.Files))
	}

	// List with limit.
	resp, _ = s.ListFiles("files-bucket", &stores.ListFilesRequest{Limit: 2})
	if len(resp.Files) != 2 {
		t.Errorf("limited files = %d, want 2", len(resp.Files))
	}
	if !resp.IsTruncated {
		t.Error("IsTruncated should be true")
	}
}

func TestManager_UploadFile_BucketNotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	err := s.UploadFile("nope", "file.txt", bytes.NewReader([]byte("x")), 1)
	if err == nil {
		t.Fatal("expected error for non-existent bucket")
	}
}

func TestManager_UploadFile_SizeMismatch(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "size-bucket"})
	err := s.UploadFile("size-bucket", "file.txt", bytes.NewReader([]byte("short")), 100)
	if err == nil {
		t.Fatal("expected error for size mismatch")
	}
}

func TestManager_GetFileInfo(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "info-file-bucket"})
	s.UploadFile("info-file-bucket", "test.txt", bytes.NewReader([]byte("hello")), 5)

	info, err := s.GetFileInfo("info-file-bucket", "test.txt")
	if err != nil {
		t.Fatalf("GetFileInfo: %v", err)
	}
	if info.Key != "test.txt" {
		t.Errorf("Key = %q", info.Key)
	}
	if info.Size != 5 {
		t.Errorf("Size = %d, want 5", info.Size)
	}
	if info.ContentType != "text/plain; charset=utf-8" && info.ContentType != "text/plain" {
		t.Errorf("ContentType = %q", info.ContentType)
	}
}

func TestManager_GetFileInfo_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "ne-bucket"})
	_, err := s.GetFileInfo("ne-bucket", "nope.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestManager_DeleteFile(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "del-bucket"})
	s.UploadFile("del-bucket", "file.txt", bytes.NewReader([]byte("data")), 4)

	if err := s.DeleteFile("del-bucket", "file.txt"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	// Verify gone.
	_, err := s.GetFileInfo("del-bucket", "file.txt")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestManager_CopyFile(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "src-bucket"})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "dst-bucket"})
	s.UploadFile("src-bucket", "original.txt", bytes.NewReader([]byte("copy me")), 7)

	err := s.CopyFile(&stores.CopyObjectRequest{
		SrcBucket:  "src-bucket",
		SrcKey:     "original.txt",
		DestBucket: "dst-bucket",
		DestKey:    "copied.txt",
	})
	if err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	// Verify destination.
	info, err := s.GetFileInfo("dst-bucket", "copied.txt")
	if err != nil {
		t.Fatalf("GetFileInfo dst: %v", err)
	}
	if info.Size != 7 {
		t.Errorf("Size = %d, want 7", info.Size)
	}

	// Verify source still exists.
	_, err = s.GetFileInfo("src-bucket", "original.txt")
	if err != nil {
		t.Fatal("source should still exist after copy")
	}
}

func TestManager_CopyFile_NilRequest(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if err := s.CopyFile(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestManager_CopyFile_SrcNotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "a"})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "b"})
	err := s.CopyFile(&stores.CopyObjectRequest{
		SrcBucket: "a", SrcKey: "nope",
		DestBucket: "b", DestKey: "x",
	})
	if err == nil {
		t.Fatal("expected error for non-existent source")
	}
}

func TestManager_MoveFile(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "move-bucket"})
	s.UploadFile("move-bucket", "old.txt", bytes.NewReader([]byte("move me")), 7)

	err := s.MoveFile(&stores.CopyObjectRequest{
		SrcBucket:  "move-bucket",
		SrcKey:     "old.txt",
		DestBucket: "move-bucket",
		DestKey:    "new.txt",
	})
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}

	// Verify destination exists.
	_, err = s.GetFileInfo("move-bucket", "new.txt")
	if err != nil {
		t.Fatal("destination should exist after move")
	}

	// Verify source is gone.
	_, err = s.GetFileInfo("move-bucket", "old.txt")
	if err == nil {
		t.Fatal("source should not exist after move")
	}
}

func TestManager_GetFileURL(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "url-bucket"})
	s.UploadFile("url-bucket", "file.txt", bytes.NewReader([]byte("x")), 1)

	url, err := s.GetFileURL("url-bucket", "file.txt", time.Hour)
	if err != nil {
		t.Fatalf("GetFileURL: %v", err)
	}
	if !strings.Contains(url, "url-bucket") {
		t.Errorf("URL = %q, should contain bucket name", url)
	}
	if !strings.Contains(url, "file.txt") {
		t.Errorf("URL = %q, should contain key", url)
	}
}

func TestManager_GetFileURL_NotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	s.CreateBucket(&stores.CreateBucketRequest{Name: "url-ne-bucket"})
	_, err := s.GetFileURL("url-ne-bucket", "nope.txt", time.Hour)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestManager_ListFiles_BucketNotExist(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	_, err := s.ListFiles("nope", nil)
	if err == nil {
		t.Fatal("expected error for non-existent bucket")
	}
}

// ──────────────────────────────────────────────
// Interface compliance tests
// ──────────────────────────────────────────────

func TestStore_ImplementsObjectStorageManager(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	// Verify Store implements ObjectStorageManager.
	var _ stores.ObjectStorageManager = s

	// Verify helper functions.
	if !stores.SupportsManagement(s) {
		t.Error("SupportsManagement should return true for local Store")
	}
	if stores.AsManager(s) == nil {
		t.Error("AsManager should return non-nil for local Store")
	}
}

func TestStore_NotMultipartUploader(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	if stores.SupportsMultipart(s) {
		t.Error("local Store should not support multipart upload")
	}
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"ab", true},                           // too short
		{"valid-bucket", false},                // valid
		{"a" + strings.Repeat("b", 62), false}, // exactly 63 chars
		{"a" + strings.Repeat("b", 63), true},  // too long
		{"has..dots", true},                    // consecutive dots
		{"has//slashes", true},                 // consecutive slashes
	}
	for _, tt := range tests {
		err := validateBucketName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateBucketName(%q) err = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
