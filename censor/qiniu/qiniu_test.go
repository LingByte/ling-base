// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"net/http"
	"strings"
	"testing"
)

func TestNewMAC_EmptyAccessKey(t *testing.T) {
	_, err := newMAC(Config{SecretKey: "sk"})
	if err == nil {
		t.Fatal("expected error for empty AccessKey")
	}
	if !strings.Contains(err.Error(), "AccessKey and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewMAC_EmptySecretKey(t *testing.T) {
	_, err := newMAC(Config{AccessKey: "ak"})
	if err == nil {
		t.Fatal("expected error for empty SecretKey")
	}
	if !strings.Contains(err.Error(), "AccessKey and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewMAC_EmptyBoth(t *testing.T) {
	_, err := newMAC(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewMAC_Valid(t *testing.T) {
	mac, err := newMAC(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mac == nil {
		t.Fatal("MAC should be initialized")
	}
}

func TestDefaultHost_Default(t *testing.T) {
	if got := defaultHost(""); got != "ai.qiniuapi.com" {
		t.Errorf("defaultHost(\"\") = %q, want ai.qiniuapi.com", got)
	}
}

func TestDefaultHost_Custom(t *testing.T) {
	if got := defaultHost("custom.example.com"); got != "custom.example.com" {
		t.Errorf("defaultHost(\"custom.example.com\") = %q, want custom.example.com", got)
	}
}

func TestDefaultClient_Default(t *testing.T) {
	c := defaultClient(nil)
	if c != http.DefaultClient {
		t.Error("defaultClient(nil) should return http.DefaultClient")
	}
}

func TestDefaultClient_Custom(t *testing.T) {
	custom := &http.Client{}
	c := defaultClient(custom)
	if c != custom {
		t.Error("defaultClient should return the provided client when non-nil")
	}
}
