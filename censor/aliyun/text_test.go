// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aliyun

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
	if !strings.Contains(err.Error(), "AccessKeyID and AccessKeySecret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewTextCensor_Valid(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.service != textService {
		t.Errorf("service = %q, want %q", c.service, textService)
	}
}

func TestNewTextCensor_CustomEndpoint(t *testing.T) {
	c, err := NewTextCensor(Config{
		AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: "green-cip.cn-beijing.aliyuncs.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should be initialized")
	}
}

func TestTextCensor_CensorText_CanceledContext(t *testing.T) {
	c, _ := NewTextCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.CensorText(ctx, "hello")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
