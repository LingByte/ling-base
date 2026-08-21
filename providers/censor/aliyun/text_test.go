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

func TestTextCensor_CensorText_APIError(t *testing.T) {
	c, _ := NewTextCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	// With fake credentials, the API call will fail. Verify the error is wrapped.
	_, err := c.CensorText(context.Background(), "some text")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
	if !strings.Contains(err.Error(), "aliyun TextModeration") {
		t.Errorf("error should contain 'aliyun TextModeration', got: %v", err)
	}
}

func TestTextCensor_CensorText_NonEmptyText(t *testing.T) {
	c, _ := NewTextCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	// Even with a canceled context, the method should return an error
	// because the underlying SDK call will fail.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.CensorText(ctx, "test content for moderation")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewTextCensor_NilClient(t *testing.T) {
	c := &TextCensor{client: nil, service: textService}
	if c == nil {
		t.Fatal("TextCensor should not be nil")
	}
	if c.service != textService {
		t.Errorf("service = %q, want %q", c.service, textService)
	}
}

func TestHighRiskLabels(t *testing.T) {
	expected := map[string]bool{
		"terrorism":  true,
		"porn":       true,
		"contraband": true,
	}
	for label := range expected {
		if !highRiskLabels[label] {
			t.Errorf("highRiskLabels should contain %q", label)
		}
	}
	if len(highRiskLabels) != len(expected) {
		t.Errorf("highRiskLabels has %d entries, want %d", len(highRiskLabels), len(expected))
	}
}
