// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tos

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	s := New(Config{Endpoint: "tos-cn-beijing.volces.com", Region: "cn-beijing", BucketName: "bkt"})
	if s.cfg.BucketName != "bkt" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Endpoint: "https://tos-cn-beijing.volces.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.tos-cn-beijing.volces.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_CustomDomain(t *testing.T) {
	s := New(Config{Domain: "cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_CustomDomain_WithScheme(t *testing.T) {
	s := New(Config{Domain: "http://cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "http://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{Endpoint: "tos-cn-beijing.volces.com", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk", BucketName: "bkt", Domain: "https://cdn.com"}
	s := New(cfg)
	if s.cfg.Endpoint != "tos-cn-beijing.volces.com" || s.cfg.Region != "cn-beijing" || s.cfg.BucketName != "bkt" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
	if s.cfg.AccessKeyID != "ak" || s.cfg.AccessKeySecret != "sk" || s.cfg.Domain != "https://cdn.com" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_KeyWithLeadingSlash(t *testing.T) {
	s := New(Config{Endpoint: "https://tos-cn-beijing.volces.com", BucketName: "mybucket"})
	got := s.PublicURL("/file.txt")
	want := "https://mybucket.tos-cn-beijing.volces.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_Endpoint_HTTPScheme(t *testing.T) {
	s := New(Config{Endpoint: "http://tos-cn-beijing.volces.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.tos-cn-beijing.volces.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_CustomDomain_TrailingSlash(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com/"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestClient_ValidConfig(t *testing.T) {
	s := New(Config{Endpoint: "tos-cn-beijing.volces.com", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk"})
	c, err := s.client(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestRead_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk"})
	if _, _, err := s.Read("key"); err == nil {
		t.Fatal("expected error for Read with empty endpoint")
	}
}

func TestWrite_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err := s.Write("key", nil); err == nil {
		t.Fatal("expected error for Write with empty endpoint")
	}
}

func TestDelete_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err := s.Delete("key"); err == nil {
		t.Fatal("expected error for Delete with empty endpoint")
	}
}

func TestExists_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "", Region: "cn-beijing", AccessKeyID: "ak", AccessKeySecret: "sk"})
	if _, err := s.Exists("key"); err == nil {
		t.Fatal("expected error for Exists with empty endpoint")
	}
}
