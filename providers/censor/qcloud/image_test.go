// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qcloud

import (
	"context"
	"testing"
)

func TestNewImageCensor_EmptyCredentials(t *testing.T) {
	_, err := NewImageCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewImageCensor_Valid(t *testing.T) {
	c, err := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
}

func TestImageCensor_CensorImage_EmptyURL(t *testing.T) {
	c, _ := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.CensorImage(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty imageURL")
	}
}

func TestImageCensor_CensorImage_CanceledContext(t *testing.T) {
	c, _ := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.CensorImage(ctx, "https://example.com/img.png")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestImageCensor_CensorImage_APIError(t *testing.T) {
	c, _ := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.CensorImage(context.Background(), "https://example.com/img.png")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestNewImageCensor_CustomRegion(t *testing.T) {
	c, err := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk", Region: "ap-shanghai"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.region != "ap-shanghai" {
		t.Errorf("region = %q, want ap-shanghai", c.region)
	}
}

func TestNewImageCensor_DefaultRegion(t *testing.T) {
	c, err := NewImageCensor(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.region != defaultRegion {
		t.Errorf("region = %q, want %q", c.region, defaultRegion)
	}
}
