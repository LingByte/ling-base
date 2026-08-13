// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

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
	c, err := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
}

func TestAudioCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.SubmitCensorAudio(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty audioURL")
	}
}

func TestAudioCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestAudioCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.SubmitCensorAudio(ctx, "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}
