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
