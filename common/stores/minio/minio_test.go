// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package minio

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New(Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk", Bucket: "bkt", UseSSL: true})
	if s.cfg.Bucket != "bkt" {
		t.Errorf("bucket = %q", s.cfg.Bucket)
	}
	if !s.cfg.UseSSL {
		t.Error("UseSSL should be true")
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "mybucket", UseSSL: false})
	got := s.PublicURL("file.txt")
	want := "http://minio.local:9000/mybucket/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_SSL(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "mybucket", UseSSL: true})
	got := s.PublicURL("file.txt")
	want := "https://minio.local:9000/mybucket/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_BaseURL(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "bkt", BaseURL: "https://cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/bkt/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestSetBaseURL(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "bkt"})
	s.SetBaseURL("https://new.cdn.com")
	got := s.PublicURL("file.txt")
	want := "https://new.cdn.com/bkt/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk", Bucket: "bkt", UseSSL: true, BaseURL: "https://cdn.com"}
	s := New(cfg)
	if s.cfg.Endpoint != "localhost:9000" || s.cfg.AccessKey != "ak" || s.cfg.SecretKey != "sk" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
	if s.cfg.Bucket != "bkt" || !s.cfg.UseSSL || s.baseURL != "https://cdn.com" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_BaseURL_TrailingSlash(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "bkt", BaseURL: "https://cdn.example.com/"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/bkt/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestSetBaseURL_OverridesInitial(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", Bucket: "bkt", BaseURL: "https://old.com"})
	s.SetBaseURL("https://new.cdn.com")
	got := s.PublicURL("file.txt")
	want := "https://new.cdn.com/bkt/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestClient_ValidConfig(t *testing.T) {
	s := New(Config{Endpoint: "minio.local:9000", AccessKey: "ak", SecretKey: "sk", UseSSL: true})
	c, err := s.client()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_InvalidEndpoint(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	c, err := s.client()
	if err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
	if c != nil {
		t.Errorf("expected nil client on error, got %v", c)
	}
}

func TestRead_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if _, _, err := s.Read("key"); err == nil {
		t.Fatal("expected error for Read with invalid endpoint")
	}
}

func TestWrite_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if err := s.Write("key", nil); err == nil {
		t.Fatal("expected error for Write with invalid endpoint")
	}
}

func TestDelete_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if err := s.Delete("key"); err == nil {
		t.Fatal("expected error for Delete with invalid endpoint")
	}
}

func TestExists_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.Exists("key"); err == nil {
		t.Fatal("expected error for Exists with invalid endpoint")
	}
}

func TestSignedURL_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.SignedURL("key", time.Hour); err == nil {
		t.Fatal("expected error for SignedURL with invalid endpoint")
	}
}

func TestPresignUpload_ClientError(t *testing.T) {
	s := New(Config{Endpoint: "http://[::1:invalid", AccessKey: "ak", SecretKey: "sk"})
	if _, err := s.PresignUpload("key", "image/png", time.Hour); err == nil {
		t.Fatal("expected error for PresignUpload with invalid endpoint")
	}
}
