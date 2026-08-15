// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package qiniu

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/LingByte/ling-base/censor"
)

func TestNewVideoCensor_EmptyCredentials(t *testing.T) {
	_, err := NewVideoCensor(Config{})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNewVideoCensor_Valid(t *testing.T) {
	c, err := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.mac == nil {
		t.Fatal("client should be initialized")
	}
}

func TestNewVideoCensor_CustomHost(t *testing.T) {
	c, err := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk", Host: "custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.host != "custom.example.com" {
		t.Errorf("host = %q, want custom.example.com", c.host)
	}
}

func TestNewVideoCensor_CustomClient(t *testing.T) {
	custom := &http.Client{}
	c, err := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk", Client: custom})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.client != custom {
		t.Error("client should be the custom client")
	}
}

func TestVideoCensor_Submit_EmptyURL(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.SubmitCensorVideo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoURL")
	}
}

func TestVideoCensor_GetResult_EmptyTaskID(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	_, err := c.GetCensorResult(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
}

func TestVideoCensor_Submit_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.SubmitCensorVideo(ctx, "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_Submit_Success(t *testing.T) {
	body := `{"job":"job-video-456"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	id, err := c.SubmitCensorVideo(context.Background(), "https://example.com/video.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "job-video-456" {
		t.Errorf("id = %q, want job-video-456", id)
	}
}

func TestVideoCensor_Submit_HTTPError(t *testing.T) {
	body := `{"error":"forbidden"}`
	srv, client := newTestServer(http.StatusForbidden, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	_, err := c.SubmitCensorVideo(context.Background(), "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should contain 'HTTP 403', got: %v", err)
	}
}

func TestVideoCensor_Submit_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	_, err := c.SubmitCensorVideo(context.Background(), "https://example.com/video.mp4")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}

func TestVideoCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewVideoCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.GetCensorResult(ctx, "task-456")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestVideoCensor_GetResult_Success(t *testing.T) {
	body := `{"id":"task-456","status":"FINISHED","result":{"message":"done","result":{"suggestion":"review","scenes":{"pulp":{"cuts":[{"details":[{"label":"sexy","score":0.85}]}]}}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFinished {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFinished)
	}
	if snap.Suggestion != "review" {
		t.Errorf("suggestion = %q, want review", snap.Suggestion)
	}
	if snap.Label != "sexy" {
		t.Errorf("label = %q, want sexy", snap.Label)
	}
	if snap.Score != 0.85 {
		t.Errorf("score = %v, want 0.85", snap.Score)
	}
	if snap.Msg != "done" {
		t.Errorf("msg = %q, want done", snap.Msg)
	}
}

func TestVideoCensor_GetResult_Waiting(t *testing.T) {
	body := `{"id":"task-456","status":"WAITING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobWaiting {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobWaiting)
	}
}

func TestVideoCensor_GetResult_Doing(t *testing.T) {
	body := `{"id":"task-456","status":"DOING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobDoing {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobDoing)
	}
}

func TestVideoCensor_GetResult_Failed(t *testing.T) {
	body := `{"id":"task-456","status":"FAILED","error":"processing error"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFailed {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFailed)
	}
	if snap.Error != "processing error" {
		t.Errorf("error = %q, want processing error", snap.Error)
	}
}

func TestVideoCensor_GetResult_FailedNoError(t *testing.T) {
	body := `{"id":"task-456","status":"FAILED"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFailed {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFailed)
	}
	if snap.Error != "qiniu video job failed" {
		t.Errorf("error = %q, want 'qiniu video job failed'", snap.Error)
	}
}

func TestVideoCensor_GetResult_UnknownStatus(t *testing.T) {
	body := `{"id":"task-456","status":"PREPARING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobDoing {
		t.Errorf("status = %q, want %q (default for unknown)", snap.Status, censor.JobDoing)
	}
	if snap.Msg != "PREPARING" {
		t.Errorf("msg = %q, want PREPARING", snap.Msg)
	}
}

func TestVideoCensor_GetResult_FinishedNoResult(t *testing.T) {
	body := `{"id":"task-456","status":"FINISHED"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFinished {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFinished)
	}
	if snap.Suggestion != "" {
		t.Errorf("suggestion = %q, want empty", snap.Suggestion)
	}
}

func TestVideoCensor_GetResult_HTTPError(t *testing.T) {
	body := `{"error":"server error"}`
	srv, client := newTestServer(http.StatusInternalServerError, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	_, err := c.GetCensorResult(context.Background(), "task-456")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error should contain 'HTTP 500', got: %v", err)
	}
}

func TestVideoCensor_GetResult_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewVideoCensor(testConfig(srv, client))
	_, err := c.GetCensorResult(context.Background(), "task-456")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}
