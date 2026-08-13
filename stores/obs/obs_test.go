// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package obs

import "testing"

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
