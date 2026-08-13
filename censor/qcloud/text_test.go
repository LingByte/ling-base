// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qcloud

import (
	"context"
	"strings"
	"testing"
)

func TestNewTextCensor_EmptyCredentials(t *testing.T) {
	_, err := NewTextCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if !strings.Contains(err.Error(), "SecretID and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewTextCensor_Valid(t *testing.T) {
	c, err := NewTextCensor(Config{SecretID: "sid", SecretKey: "sk", Region: "ap-shanghai"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.region != "ap-shanghai" {
		t.Errorf("region = %q, want ap-shanghai", c.region)
	}
}

func TestNewTextCensor_DefaultRegion(t *testing.T) {
	c, err := NewTextCensor(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.region != defaultRegion {
		t.Errorf("region = %q, want %q", c.region, defaultRegion)
	}
}

func TestTextCensor_CensorText_CanceledContext(t *testing.T) {
	c, _ := NewTextCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.CensorText(ctx, "hello")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
