// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"strings"
	"testing"
)

func TestNewTextCensor_EmptyCredentials(t *testing.T) {
	_, err := NewTextCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if !strings.Contains(err.Error(), "AccessKey and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewTextCensor_Valid(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
	if c.host != "ai.qiniuapi.com" {
		t.Errorf("host = %q, want ai.qiniuapi.com", c.host)
	}
}

func TestNewTextCensor_CustomHost(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk", Host: "custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.host != "custom.example.com" {
		t.Errorf("host = %q, want custom.example.com", c.host)
	}
}

func TestTextCensor_CensorText_CanceledContext(t *testing.T) {
	c, _ := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.CensorText(ctx, "hello")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
