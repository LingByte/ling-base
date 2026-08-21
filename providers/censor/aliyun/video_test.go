// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aliyun

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
	c, err := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.service != videoService {
		t.Errorf("service = %q, want %q", c.service, videoService)
	}
}

func TestVideoCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}
}

func TestVideoCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestVideoCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.SubmitCensorVideo(ctx, "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_Submit_APIError(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestVideoCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetCensorResult(ctx, "task-456")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_GetResult_APIError(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.GetCensorResult(context.Background(), "task-456")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestNewVideoCensor_CustomEndpoint(t *testing.T) {
	c, err := NewVideoCensor(Config{
		AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: "green-cip.cn-beijing.aliyuncs.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.service != videoService {
		t.Errorf("service = %q, want %q", c.service, videoService)
	}
}
