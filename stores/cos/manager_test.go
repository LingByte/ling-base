// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cos

import (
	"testing"

	"github.com/LingByte/ling-base/stores"
)

// TestStore_ImplementsInterfaces verifies that the COS Store implements
// both ObjectStorageManager and MultipartUploader. These are compile-time
// checks — no live COS connection is needed.
func TestStore_ImplementsInterfaces(t *testing.T) {
	s := New(Config{
		SecretID:   "ak",
		SecretKey:  "sk",
		Region:     "ap-guangzhou",
		BucketName: "test-bucket",
	})

	// Compile-time interface checks.
	var _ stores.ObjectStorageManager = s
	var _ stores.MultipartUploader = s

	// Runtime helper checks.
	if !stores.SupportsManagement(s) {
		t.Error("SupportsManagement should return true for COS Store")
	}
	if stores.AsManager(s) == nil {
		t.Error("AsManager should return non-nil for COS Store")
	}
	if !stores.SupportsMultipart(s) {
		t.Error("SupportsMultipart should return true for COS Store")
	}
	if stores.AsMultipartUploader(s) == nil {
		t.Error("AsMultipartUploader should return non-nil for COS Store")
	}
}

func TestResolveBucket(t *testing.T) {
	s := New(Config{BucketName: "default-bucket"})

	if got := s.resolveBucket(""); got != "default-bucket" {
		t.Errorf("resolveBucket('') = %q, want %q", got, "default-bucket")
	}
	if got := s.resolveBucket("custom"); got != "custom" {
		t.Errorf("resolveBucket('custom') = %q, want %q", got, "custom")
	}
}

func TestCreateBucket_InvalidRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if err := s.CreateBucket(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
	if err := s.CreateBucket(&stores.CreateBucketRequest{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCopyFile_NilRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if err := s.CopyFile(nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestInitiateMultipartUpload_NilRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if _, err := s.InitiateMultipartUpload("bucket", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestUploadPart_NilRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if _, err := s.UploadPart("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCompleteMultipartUpload_NilRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if _, err := s.CompleteMultipartUpload("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestListParts_NilRequest(t *testing.T) {
	s := New(Config{SecretID: "ak", SecretKey: "sk", Region: "ap-guangzhou"})
	if _, err := s.ListParts("bucket", "key", nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}
