// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ks3

import "testing"

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
