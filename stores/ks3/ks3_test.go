// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ks3

import (
	"context"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	s := New(Config{Endpoint: "ks3-cn-beijing.ksyuncs.com", Region: "BEIJING", BucketName: "bkt"})
	if s.cfg.BucketName != "bkt" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Endpoint: "https://ks3-cn-beijing.ksyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.ks3-cn-beijing.ksyuncs.com/file.txt"
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
	cfg := Config{Endpoint: "ks3-cn-beijing.ksyuncs.com", Region: "BEIJING", AccessKeyID: "ak", AccessKeySecret: "sk", BucketName: "bkt", Domain: "https://cdn.com"}
	s := New(cfg)
	if s.cfg.Endpoint != "ks3-cn-beijing.ksyuncs.com" || s.cfg.Region != "BEIJING" || s.cfg.BucketName != "bkt" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
	if s.cfg.AccessKeyID != "ak" || s.cfg.AccessKeySecret != "sk" || s.cfg.Domain != "https://cdn.com" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_KeyWithLeadingSlash(t *testing.T) {
	s := New(Config{Endpoint: "https://ks3-cn-beijing.ksyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("/file.txt")
	want := "https://mybucket.ks3-cn-beijing.ksyuncs.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_Endpoint_HTTPScheme(t *testing.T) {
	s := New(Config{Endpoint: "http://ks3-cn-beijing.ksyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.ks3-cn-beijing.ksyuncs.com/file.txt"
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

func TestClient(t *testing.T) {
	s := New(Config{Endpoint: "ks3-cn-beijing.ksyuncs.com", Region: "BEIJING", AccessKeyID: "ak", AccessKeySecret: "sk", BucketName: "bkt"})
	c := s.client()
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestCheckConnectivity_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := CheckConnectivity(ctx, "ks3-cn-beijing.ksyuncs.com", "BEIJING", "ak", "sk", "bkt")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got %v", err)
	}
}
