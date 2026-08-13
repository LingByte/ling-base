// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package oss

import "testing"

func TestNew(t *testing.T) {
	s := New(Config{
		AccessKeyID:     "ak",
		AccessKeySecret: "sk",
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		BucketName:      "bucket",
	})
	if s.cfg.BucketName != "bucket" {
		t.Errorf("bucket = %q", s.cfg.BucketName)
	}
}

func TestPublicURL_Default(t *testing.T) {
	s := New(Config{Endpoint: "https://oss-cn-hangzhou.aliyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.oss-cn-hangzhou.aliyuncs.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_BaseURL(t *testing.T) {
	s := New(Config{Endpoint: "https://oss-cn-hangzhou.aliyuncs.com", BucketName: "bkt", BaseURL: "https://cdn.example.com"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestSetBaseURL(t *testing.T) {
	s := New(Config{Endpoint: "https://oss-cn-hangzhou.aliyuncs.com", BucketName: "bkt"})
	s.SetBaseURL("https://new.cdn.com")
	got := s.PublicURL("file.txt")
	want := "https://new.cdn.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestBucket_NotConfigured(t *testing.T) {
	s := New(Config{})
	_, err := s.bucket()
	if err == nil {
		t.Fatal("expected error for unconfigured OSS")
	}
}
