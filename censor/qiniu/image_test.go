// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"testing"
)

func TestNewImageCensor_EmptyCredentials(t *testing.T) {
	_, err := NewImageCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewImageCensor_Valid(t *testing.T) {
	c, err := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
}

func TestImageCensor_CensorImage_EmptyURL(t *testing.T) {
	c, _ := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.CensorImage(nil, "")
	if err == nil {
		t.Fatal("expected error for empty imageURL")
	}
}

func TestImageCensor_CensorImage_CanceledContext(t *testing.T) {
	c, _ := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.CensorImage(ctx, "https://example.com/img.png")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
