// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package s3

import "testing"

func TestNew(t *testing.T) {
	cfg := Config{
		Region:          "us-east-1",
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		BucketName:      "bucket",
	}
	s := New(cfg)
	if s.cfg.Region != "us-east-1" {
		t.Errorf("region = %q", s.cfg.Region)
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Region: "us-east-1", BucketName: "mybucket"})
	got := s.PublicURL("path/file.txt")
	want := "https://mybucket.s3.us-east-1.amazonaws.com/path/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_CustomDomain(t *testing.T) {
	s := New(Config{Domain: "https://cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_CustomDomain_NoScheme(t *testing.T) {
	s := New(Config{Domain: "cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_Endpoint_PathStyle(t *testing.T) {
	s := New(Config{Endpoint: "https://minio.local:9000", BucketName: "bkt", UsePathStyle: true})
	got := s.PublicURL("file.txt")
	want := "https://minio.local:9000/bkt/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_Endpoint_VirtualHost(t *testing.T) {
	s := New(Config{Endpoint: "https://s3.local:9000", BucketName: "bkt", UsePathStyle: false})
	got := s.PublicURL("file.txt")
	want := "https://s3.local:9000/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}
