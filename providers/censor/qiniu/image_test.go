// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"context"
	"net/http"
	"strings"
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

func TestNewImageCensor_CustomHost(t *testing.T) {
	c, err := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk", Host: "custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.host != "custom.example.com" {
		t.Errorf("host = %q, want custom.example.com", c.host)
	}
}

func TestNewImageCensor_CustomClient(t *testing.T) {
	custom := &http.Client{}
	c, err := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk", Client: custom})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.client != custom {
		t.Error("client should be the custom client")
	}
}

func TestImageCensor_CensorImage_EmptyURL(t *testing.T) {
	c, _ := NewImageCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.CensorImage(context.Background(), "")
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

func TestImageCensor_CensorImage_Success(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"block","scenes":{"pulp":{"suggestion":"block","label":"porn","score":0.95}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	ic, err := NewImageCensor(testConfig(srv, client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "block" {
		t.Errorf("suggestion = %q, want block", result.Suggestion)
	}
	if result.Label != "porn" {
		t.Errorf("label = %q, want porn", result.Label)
	}
	if result.Score != 0.95 {
		t.Errorf("score = %v, want 0.95", result.Score)
	}
}

func TestImageCensor_CensorImage_SuccessNoScenes(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"pass"}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	ic, _ := NewImageCensor(testConfig(srv, client))
	result, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "pass" {
		t.Errorf("suggestion = %q, want pass", result.Suggestion)
	}
	if result.Label != "" {
		t.Errorf("label = %q, want empty", result.Label)
	}
}

func TestImageCensor_CensorImage_NilResult(t *testing.T) {
	body := `{"code":0,"message":"OK"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	ic, _ := NewImageCensor(testConfig(srv, client))
	result, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "" {
		t.Errorf("suggestion = %q, want empty", result.Suggestion)
	}
}

func TestImageCensor_CensorImage_HTTPError(t *testing.T) {
	body := `{"code":403,"message":"forbidden"}`
	srv, client := newTestServer(http.StatusForbidden, body)
	defer srv.Close()

	ic, _ := NewImageCensor(testConfig(srv, client))
	_, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should contain 'HTTP 403', got: %v", err)
	}
}

func TestImageCensor_CensorImage_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	ic, _ := NewImageCensor(testConfig(srv, client))
	_, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}

func TestImageCensor_CensorImage_ScenesEmptyLabel(t *testing.T) {
	body := `{"code":0,"message":"OK","result":{"suggestion":"pass","scenes":{"pulp":{"suggestion":"pass","label":"","score":0}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	ic, _ := NewImageCensor(testConfig(srv, client))
	result, err := ic.CensorImage(context.Background(), "https://example.com/img.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Suggestion != "pass" {
		t.Errorf("suggestion = %q, want pass", result.Suggestion)
	}
	if result.Label != "" {
		t.Errorf("label = %q, want empty", result.Label)
	}
}
