// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package minio

import (
	"testing"

	"github.com/LingByte/ling-base/stores"
)

// TestStore_ImplementsInterfaces verifies that the MinIO Store implements
// both ObjectStorageManager and MultipartUploader. These are compile-time
// checks — no live MinIO connection is needed.
func TestStore_ImplementsInterfaces(t *testing.T) {
	s := New(Config{
		Endpoint:  "localhost:9000",
		AccessKey: "ak",
		SecretKey: "sk",
		Bucket:    "test-bucket",
		UseSSL:    false,
	})

	// Compile-time interface checks.
	var _ stores.ObjectStorageManager = s
	var _ stores.MultipartUploader = s

	// Runtime helper checks.
	if !stores.SupportsManagement(s) {
		t.Error("SupportsManagement should return true for MinIO Store")
	}
	if stores.AsManager(s) == nil {
		t.Error("AsManager should return non-nil for MinIO Store")
	}
	if !stores.SupportsMultipart(s) {
		t.Error("SupportsMultipart should return true for MinIO Store")
	}
	if stores.AsMultipartUploader(s) == nil {
		t.Error("AsMultipartUploader should return non-nil for MinIO Store")
	}
}

func TestResolveBucket(t *testing.T) {
	s := New(Config{Bucket: "default-bucket"})

	if got := s.resolveBucket(""); got != "default-bucket" {
		t.Errorf("resolveBucket('') = %q, want %q", got, "default-bucket")
	}
	if got := s.resolveBucket("custom"); got != "custom" {
		t.Errorf("resolveBucket('custom') = %q, want %q", got, "custom")
	}
}

func TestCreateBucket_InvalidRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if err := s.CreateBucket(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
	if err := s.CreateBucket(&stores.CreateBucketRequest{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCopyFile_NilRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if err := s.CopyFile(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestInitiateMultipartUpload_NilRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.InitiateMultipartUpload("bucket", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestUploadPart_NilRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.UploadPart("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCompleteMultipartUpload_NilRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.CompleteMultipartUpload("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestListParts_NilRequest(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.ListParts("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}
