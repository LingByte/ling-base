// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kodo

import "testing"

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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[0:len(sub)] == sub || contains(s[1:], sub)))
}
