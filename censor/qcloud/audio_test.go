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
