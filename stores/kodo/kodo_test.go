// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kodo

import (
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New(Config{AccessKey: "ak", SecretKey: "sk", BucketName: "bkt", Domain: "https://cdn.example.com"})
	if s.cfg.BucketName != "bkt" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL_EmptyDomain(t *testing.T) {
	s := New(Config{})
	if u := s.PublicURL("file.txt"); u != "" {
		t.Errorf("expected empty URL, got %q", u)
	}
}

func TestPublicURL_Public(t *testing.T) {
	s := New(Config{Domain: "cdn.example.com"})
	got := s.PublicURL("file.txt")
	if got == "" {
		t.Fatal("expected non-empty URL")
	}
	if !contains(got, "file.txt") {
		t.Errorf("URL should contain key, got %q", got)
	}
}

func TestPublicURL_Private(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com", Private: true, AccessKey: "ak", SecretKey: "sk"})
	got := s.PublicURL("file.txt")
	if got == "" {
		t.Fatal("expected non-empty signed URL")
	}
	// Private URLs contain a signature (e= parameter)
	if !contains(got, "e=") {
		t.Errorf("expected signed URL with e= param, got %q", got)
	}
}

func TestNormalizedDomain(t *testing.T) {
	s := New(Config{Domain: "cdn.example.com"})
	if d := s.normalizedDomain(); d != "http://cdn.example.com" {
		t.Errorf("normalizedDomain() = %q", d)
	}
	s2 := New(Config{Domain: "https://cdn.example.com"})
	if d := s2.normalizedDomain(); d != "https://cdn.example.com" {
		t.Errorf("normalizedDomain() = %q", d)
	}
}

func TestSignedURL_EmptyDomain(t *testing.T) {
	s := New(Config{})
	_, err := s.SignedURL("file.txt", 0)
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{AccessKey: "ak", SecretKey: "sk", BucketName: "bkt", Domain: "https://cdn.example.com", Private: true, Region: "z0"}
	s := New(cfg)
	if s.cfg.AccessKey != "ak" || s.cfg.SecretKey != "sk" || s.cfg.BucketName != "bkt" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
	if !s.cfg.Private || s.cfg.Region != "z0" || s.cfg.Domain != "https://cdn.example.com" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestMac(t *testing.T) {
	s := New(Config{AccessKey: "ak", SecretKey: "sk"})
	m := s.mac()
	if m == nil {
		t.Fatal("expected non-nil mac")
	}
}

func TestMakeConfig_HTTPS(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com", AccessKey: "ak", BucketName: "bkt"})
	cfg := s.makeConfig()
	if !cfg.UseHTTPS {
		t.Error("expected UseHTTPS=true for https domain")
	}
}

func TestMakeConfig_HTTP(t *testing.T) {
	s := New(Config{Domain: "http://cdn.example.com", AccessKey: "ak", BucketName: "bkt"})
	cfg := s.makeConfig()
	if cfg.UseHTTPS {
		t.Error("expected UseHTTPS=false for http domain")
	}
}

func TestMakeConfig_NoDomain(t *testing.T) {
	s := New(Config{AccessKey: "ak", BucketName: "bkt"})
	cfg := s.makeConfig()
	if cfg.UseHTTPS {
		t.Error("expected UseHTTPS=false when no domain")
	}
}

func TestUploadToken(t *testing.T) {
	s := New(Config{AccessKey: "ak", SecretKey: "sk", BucketName: "bkt"})
	tok := s.uploadToken()
	if tok == "" {
		t.Fatal("expected non-empty upload token")
	}
	if !contains(tok, ":") {
		t.Errorf("expected token to contain ':', got %q", tok)
	}
}

func TestPublicURL_HTTPSDomain(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com"})
	got := s.PublicURL("file.txt")
	if !contains(got, "https://cdn.example.com") {
		t.Errorf("expected https domain in URL, got %q", got)
	}
	if !contains(got, "file.txt") {
		t.Errorf("expected key in URL, got %q", got)
	}
}

func TestSignedURL_WithDomain(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com", AccessKey: "ak", SecretKey: "sk"})
	u, err := s.SignedURL("file.txt", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == "" {
		t.Fatal("expected non-empty signed URL")
	}
	if !contains(u, "e=") {
		t.Errorf("expected signed URL with e= param, got %q", u)
	}
}

func TestPresignUpload(t *testing.T) {
	s := New(Config{AccessKey: "ak", SecretKey: "sk", BucketName: "bkt"})
	du, err := s.PresignUpload("file.txt", "image/png", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if du == nil {
		t.Fatal("expected non-nil DirectUpload")
	}
	if du.Provider != "kodo" {
		t.Errorf("Provider = %q, want %q", du.Provider, "kodo")
	}
	if du.Method != http.MethodPost {
		t.Errorf("Method = %q, want %q", du.Method, http.MethodPost)
	}
	if du.Key != "file.txt" {
		t.Errorf("Key = %q, want %q", du.Key, "file.txt")
	}
	if du.Form["key"] != "file.txt" {
		t.Errorf("Form[key] = %q, want %q", du.Form["key"], "file.txt")
	}
	if du.Form["token"] == "" {
		t.Error("expected non-empty form token")
	}
	if du.FileField != "file" {
		t.Errorf("FileField = %q, want %q", du.FileField, "file")
	}
}

func TestPresignUpload_NoContentType(t *testing.T) {
	s := New(Config{AccessKey: "ak", SecretKey: "sk", BucketName: "bkt"})
	du, err := s.PresignUpload("file.txt", "", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := du.Headers["Content-Type"]; ok {
		t.Error("expected no Content-Type header when contentType empty")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[0:len(sub)] == sub || contains(s[1:], sub)))
}
