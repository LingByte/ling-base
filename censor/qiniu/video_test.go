// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

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
	c, err := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
}

func TestVideoCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}
}

func TestVideoCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestVideoCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.SubmitCensorVideo(ctx, "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
