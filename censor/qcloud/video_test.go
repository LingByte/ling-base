// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qcloud

import (
	"context"
	"testing"
)

func TestNewVideoCensor_EmptyCredentials(t *testing.T) {
	_, err := NewVideoCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewVideoCensor_Valid(t *testing.T) {
	c, err := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
}

func TestVideoCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}
}

func TestVideoCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestVideoCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.SubmitCensorVideo(ctx, "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_Submit_APIError(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestVideoCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetCensorResult(ctx, "task-456")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_GetResult_APIError(t *testing.T) {
	c, _ := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "task-456")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestNewVideoCensor_CustomBizType(t *testing.T) {
	c, err := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk", BizType: "custom_biz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.bizType != "custom_biz" {
		t.Errorf("bizType = %q, want custom_biz", c.bizType)
	}
}

func TestNewVideoCensor_CustomRegion(t *testing.T) {
	c, err := NewVideoCensor(Config{SecretID: "sid", SecretKey: "sk", Region: "ap-shanghai"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.region != "ap-shanghai" {
		t.Errorf("region = %q, want ap-shanghai", c.region)
	}
}
