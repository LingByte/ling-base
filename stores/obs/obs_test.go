// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package obs

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", Region: "cn-north-4", BucketName: "bkt"})
	if s.cfg.BucketName != "bkt" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://obs.cn-north-4.myhuaweicloud.com/mybucket/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_ProxyDomain(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", BucketName: "bkt", ProxyDomain: "https://cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", Region: "cn-north-4", AccessKeyID: "ak", AccessKeySecret: "sk", BucketName: "bkt", ProxyDomain: "https://cdn.com"}
	s := New(cfg)
	if s.cfg.Endpoint != "https://obs.cn-north-4.myhuaweicloud.com" || s.cfg.Region != "cn-north-4" || s.cfg.BucketName != "bkt" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
	if s.cfg.AccessKeyID != "ak" || s.cfg.AccessKeySecret != "sk" || s.cfg.ProxyDomain != "https://cdn.com" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_KeyWithLeadingSlash(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", BucketName: "mybucket"})
	got := s.PublicURL("/file.txt")
	want := "https://obs.cn-north-4.myhuaweicloud.com/mybucket/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_Endpoint_TrailingSlash(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com/", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://obs.cn-north-4.myhuaweicloud.com/mybucket/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestClient_ValidConfig(t *testing.T) {
	s := New(Config{Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", Region: "cn-north-4", AccessKeyID: "ak", AccessKeySecret: "sk"})
	c, err := s.client(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}
