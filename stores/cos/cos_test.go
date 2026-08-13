// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cos

import "testing"

func TestNew(t *testing.T) {
	s := New(Config{SecretID: "id", SecretKey: "key", Region: "ap-guangzhou", BucketName: "bkt"})
	if s.cfg.BucketName != "bkt" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL(t *testing.T) {
	s := New(Config{Region: "ap-guangzhou", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.cos.ap-guangzhou.myqcloud.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestValidate_NotConfigured(t *testing.T) {
	s := New(Config{})
	if err := s.validate(); err == nil {
		t.Fatal("expected error for unconfigured COS")
	}
}

func TestValidate_PartialConfig(t *testing.T) {
	s := New(Config{SecretID: "id"})
	if err := s.validate(); err == nil {
		t.Fatal("expected error for partial COS config")
	}
}

func TestValidate_Valid(t *testing.T) {
	s := New(Config{SecretID: "id", SecretKey: "key", Region: "ap-guangzhou"})
	if err := s.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
