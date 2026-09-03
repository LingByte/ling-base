// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewTextCensor_EmptyCredentials(t *testing.T) {
	_, err := NewTextCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if !strings.Contains(err.Error(), "AccessKey and SecretKey") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewTextCensor_Valid(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
	if c.host != "ai.qiniuapi.com" {
		t.Errorf("host = %q, want ai.qiniuapi.com", c.host)
	}
}

func TestNewTextCensor_CustomHost(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk", Host: "custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.host != "custom.example.com" {
		t.Errorf("host = %q, want custom.example.com", c.host)
	}
}

func TestNewTextCensor_CustomClient(t *testing.T) {
	custom := &http.Client{}
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk", Client: custom})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.client != custom {
		t.Error("client should be the custom client")
	}
}

func TestNewTextCensor_DefaultClient(t *testing.T) {
	c, err := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.client == nil {
		t.Fatal("client should default to a non-nil client")
	}
	if c.client.Timeout != 30*time.Second {
		t.Errorf("client timeout = %v, want 30s", c.client.Timeout)
	}
}

func TestTextCensor_CensorText_CanceledContext(t *testing.T) {
	c, _ := NewTextCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.CensorText(ctx, "hello")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestTextCensor_CensorText_Success(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"pass","scenes":{"antispam":{"suggestion":"pass","details":[{"label":"normal","score":0.99,"description":"clean content"}]}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, err := NewTextCensor(testConfig(srv, client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := c.CensorText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "pass" {
		t.Errorf("suggestion = %q, want pass", result.Suggestion)
	}
	if result.Label != "normal" {
		t.Errorf("label = %q, want normal", result.Label)
	}
	if result.Score != 0.99 {
		t.Errorf("score = %v, want 0.99", result.Score)
	}
	if result.Details != "clean content" {
		t.Errorf("details = %q, want clean content", result.Details)
	}
}

func TestTextCensor_CensorText_SuccessNoScenes(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"block"}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewTextCensor(testConfig(srv, client))
	result, err := c.CensorText(context.Background(), "bad text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "block" {
		t.Errorf("suggestion = %q, want block", result.Suggestion)
	}
	if result.Label != "" {
		t.Errorf("label = %q, want empty", result.Label)
	}
}

func TestTextCensor_CensorText_NilResult(t *testing.T) {
	body := `{"code":0,"message":"OK"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewTextCensor(testConfig(srv, client))
	result, err := c.CensorText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "" {
		t.Errorf("suggestion = %q, want empty", result.Suggestion)
	}
}

func TestTextCensor_CensorText_HTTPError(t *testing.T) {
	body := `{"code":500,"message":"internal error"}`
	srv, client := newTestServer(http.StatusInternalServerError, body)
	defer srv.Close()

	c, _ := NewTextCensor(testConfig(srv, client))
	_, err := c.CensorText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error should contain 'HTTP 500', got: %v", err)
	}
}

func TestTextCensor_CensorText_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewTextCensor(testConfig(srv, client))
	_, err := c.CensorText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}

func TestTextCensor_CensorText_ScenesNoDetails(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"review","scenes":{"antispam":{"suggestion":"review"}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewTextCensor(testConfig(srv, client))
	result, err := c.CensorText(context.Background(), "review text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "review" {
		t.Errorf("suggestion = %q, want review", result.Suggestion)
	}
	if result.Label != "" {
		t.Errorf("label = %q, want empty (no details)", result.Label)
	}
}
