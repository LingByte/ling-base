// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aliyun

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
	c, err := NewImageCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
}

func TestImageCensor_CensorImage_EmptyURL(t *testing.T) {
	c, _ := NewImageCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.CensorImage(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty imageURL")
	}
}
