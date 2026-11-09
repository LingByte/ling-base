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

func TestValidate_MissingSecretKey(t *testing.T) {
	s := New(Config{SecretID: "id", Region: "ap-guangzhou"})
	if err := s.validate(); err == nil {
		t.Fatal("expected error when SecretKey missing")
	}
}

func TestValidate_MissingRegion(t *testing.T) {
	s := New(Config{SecretID: "id", SecretKey: "key"})
	if err := s.validate(); err == nil {
		t.Fatal("expected error when Region missing")
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{SecretID: "sid", SecretKey: "skey", Region: "ap-shanghai", BucketName: "bkt-1"}
	s := New(cfg)
	if s.cfg.SecretID != "sid" || s.cfg.SecretKey != "skey" || s.cfg.Region != "ap-shanghai" || s.cfg.BucketName != "bkt-1" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_NestedKey(t *testing.T) {
	s := New(Config{Region: "ap-guangzhou", BucketName: "mybucket"})
	got := s.PublicURL("a/b/c/file.txt")
	want := "https://mybucket.cos.ap-guangzhou.myqcloud.com/a/b/c/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_DifferentRegion(t *testing.T) {
	s := New(Config{Region: "ap-beijing", BucketName: "b"})
	got := s.PublicURL("x.png")
	want := "https://b.cos.ap-beijing.myqcloud.com/x.png"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestClient_InvalidConfig(t *testing.T) {
	s := New(Config{})
	c, err := s.client()
	if err == nil {
		t.Fatal("expected error for unconfigured COS client")
	}
	if c != nil {
		t.Errorf("expected nil client on error, got %v", c)
	}
}

func TestClient_ValidConfig(t *testing.T) {
	s := New(Config{SecretID: "id", SecretKey: "key", Region: "ap-guangzhou", BucketName: "bkt"})
	c, err := s.client()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestDelete_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if err := s.Delete("key"); err == nil {
		t.Fatal("expected error for Delete with unconfigured COS")
	}
}

func TestExists_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if _, err := s.Exists("key"); err == nil {
		t.Fatal("expected error for Exists with unconfigured COS")
	}
}

func TestRead_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if _, _, err := s.Read("key"); err == nil {
		t.Fatal("expected error for Read with unconfigured COS")
	}
}

func TestWrite_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if err := s.Write("key", nil); err == nil {
		t.Fatal("expected error for Write with unconfigured COS")
	}
}

func TestSignedURL_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if _, err := s.SignedURL("key", 0); err == nil {
		t.Fatal("expected error for SignedURL with unconfigured COS")
	}
}

func TestPresignUpload_InvalidConfig(t *testing.T) {
	s := New(Config{})
	if _, err := s.PresignUpload("key", "image/png", 0); err == nil {
		t.Fatal("expected error for PresignUpload with unconfigured COS")
	}
}
