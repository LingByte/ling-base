// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qcloud

import (
	"context"
	"testing"
)

func TestNewAudioCensor_EmptyCredentials(t *testing.T) {
	_, err := NewAudioCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewAudioCensor_Valid(t *testing.T) {
	c, err := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.bizType != "default" {
		t.Errorf("bizType = %q, want default", c.bizType)
	}
}

func TestAudioCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.SubmitCensorAudio(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty audioURL")
	}
}

func TestAudioCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestAudioCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.SubmitCensorAudio(ctx, "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestAudioCensor_Submit_APIError(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.SubmitCensorAudio(context.Background(), "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestAudioCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetCensorResult(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestAudioCensor_GetResult_APIError(t *testing.T) {
	c, _ := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestNewAudioCensor_CustomBizType(t *testing.T) {
	c, err := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk", BizType: "custom_biz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.bizType != "custom_biz" {
		t.Errorf("bizType = %q, want custom_biz", c.bizType)
	}
}

func TestNewAudioCensor_CustomRegion(t *testing.T) {
	c, err := NewAudioCensor(Config{SecretID: "sid", SecretKey: "sk", Region: "ap-shanghai"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.region != "ap-shanghai" {
		t.Errorf("region = %q, want ap-shanghai", c.region)
	}
}
