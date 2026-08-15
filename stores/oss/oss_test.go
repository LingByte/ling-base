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

func TestBucket_MissingSecret(t *testing.T) {
	s := New(Config{AccessKeyID: "ak", Endpoint: "oss-cn-hangzhou.aliyuncs.com"})
	_, err := s.bucket()
	if err == nil {
		t.Fatal("expected error when AccessKeySecret missing")
	}
}

func TestBucket_MissingEndpoint(t *testing.T) {
	s := New(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := s.bucket()
	if err == nil {
		t.Fatal("expected error when Endpoint missing")
	}
}

func TestNew_WithBaseURL(t *testing.T) {
	s := New(Config{Endpoint: "oss-cn-hangzhou.aliyuncs.com", BucketName: "bkt", BaseURL: "https://cdn.example.com"})
	if s.baseURL != "https://cdn.example.com" {
		t.Errorf("baseURL = %q, want %q", s.baseURL, "https://cdn.example.com")
	}
}

func TestNew_PreservesAllFields(t *testing.T) {
	cfg := Config{AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: "ep", BucketName: "bkt", BaseURL: "https://b.com"}
	s := New(cfg)
	if s.cfg.AccessKeyID != "ak" || s.cfg.AccessKeySecret != "sk" || s.cfg.Endpoint != "ep" || s.cfg.BucketName != "bkt" {
		t.Errorf("config not preserved: %+v", s.cfg)
	}
}

func TestPublicURL_HTTPScheme(t *testing.T) {
	s := New(Config{Endpoint: "http://oss-cn-hangzhou.aliyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.oss-cn-hangzhou.aliyuncs.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_NoScheme(t *testing.T) {
	s := New(Config{Endpoint: "oss-cn-hangzhou.aliyuncs.com", BucketName: "mybucket"})
	got := s.PublicURL("file.txt")
	want := "https://mybucket.oss-cn-hangzhou.aliyuncs.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestPublicURL_BaseURL_TrailingSlash(t *testing.T) {
	s := New(Config{Endpoint: "https://oss-cn-hangzhou.aliyuncs.com", BucketName: "bkt", BaseURL: "https://cdn.example.com/"})
	got := s.PublicURL("file.txt")
	want := "https://cdn.example.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestSetBaseURL_OverridesInitial(t *testing.T) {
	s := New(Config{Endpoint: "https://oss-cn-hangzhou.aliyuncs.com", BucketName: "bkt", BaseURL: "https://old.com"})
	s.SetBaseURL("https://new.cdn.com")
	got := s.PublicURL("file.txt")
	want := "https://new.cdn.com/file.txt"
	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}

func TestDelete_NotConfigured(t *testing.T) {
	s := New(Config{})
	if err := s.Delete("key"); err == nil {
		t.Fatal("expected error for Delete with unconfigured OSS")
	}
}

func TestExists_NotConfigured(t *testing.T) {
	s := New(Config{})
	if _, err := s.Exists("key"); err == nil {
		t.Fatal("expected error for Exists with unconfigured OSS")
	}
}

func TestRead_NotConfigured(t *testing.T) {
	s := New(Config{})
	if _, _, err := s.Read("key"); err == nil {
		t.Fatal("expected error for Read with unconfigured OSS")
	}
}

func TestWrite_NotConfigured(t *testing.T) {
	s := New(Config{})
	if err := s.Write("key", nil); err == nil {
		t.Fatal("expected error for Write with unconfigured OSS")
	}
}

func TestSignedURL_NotConfigured(t *testing.T) {
	s := New(Config{})
	if _, err := s.SignedURL("key", 0); err == nil {
		t.Fatal("expected error for SignedURL with unconfigured OSS")
	}
}

func TestPresignUpload_NotConfigured(t *testing.T) {
	s := New(Config{})
	if _, err := s.PresignUpload("key", "image/png", 0); err == nil {
		t.Fatal("expected error for PresignUpload with unconfigured OSS")
	}
}
