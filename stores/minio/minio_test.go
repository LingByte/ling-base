// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package minio

import "testing"

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
