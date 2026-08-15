// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aliyun

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
	c, err := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.service != audioService {
		t.Errorf("service = %q, want %q", c.service, audioService)
	}
}

func TestAudioCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.SubmitCensorAudio(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty audioURL")
	}
}

func TestAudioCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestAudioCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.SubmitCensorAudio(ctx, "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestAudioCensor_Submit_APIError(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.SubmitCensorAudio(context.Background(), "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestAudioCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.GetCensorResult(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestAudioCensor_GetResult_APIError(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKeyID: "ak", AccessKeySecret: "sk"})
	_, err := c.GetCensorResult(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error with fake credentials")
	}
}

func TestNewAudioCensor_CustomEndpoint(t *testing.T) {
	c, err := NewAudioCensor(Config{
		AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: "green-cip.cn-beijing.aliyuncs.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatal("client should be initialized")
	}
	if c.service != audioService {
		t.Errorf("service = %q, want %q", c.service, audioService)
	}
}
