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

func TestNewAudioCensor_CustomHost(t *testing.T) {
	c, err := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk", Host: "custom.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.host != "custom.example.com" {
		t.Errorf("host = %q, want custom.example.com", c.host)
	}
}

func TestNewAudioCensor_CustomClient(t *testing.T) {
	custom := &http.Client{}
	c, err := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk", Client: custom})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.client != custom {
		t.Error("client should be the custom client")
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

func TestAudioCensor_Submit_Success(t *testing.T) {
	body := `{"id":"job-audio-123"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	id, err := c.SubmitCensorAudio(context.Background(), "https://example.com/audio.mp3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "job-audio-123" {
		t.Errorf("id = %q, want job-audio-123", id)
	}
}

func TestAudioCensor_Submit_HTTPError(t *testing.T) {
	body := `{"error":"forbidden"}`
	srv, client := newTestServer(http.StatusForbidden, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	_, err := c.SubmitCensorAudio(context.Background(), "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should contain 'HTTP 403', got: %v", err)
	}
}

func TestAudioCensor_Submit_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	_, err := c.SubmitCensorAudio(context.Background(), "https://example.com/audio.mp3")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}

func TestAudioCensor_GetResult_CanceledContext(t *testing.T) {
	c, _ := NewAudioCensor(Config{AccessKey: "ak", SecretKey: "sk"})
	ctx, _ := contextWithCancel()
	_, err := c.GetCensorResult(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error with canceled context")
	}
}

func TestAudioCensor_GetResult_Success(t *testing.T) {
	body := `{"id":"task-123","status":"FINISHED","response":{"message":"done","result":{"suggestion":"block","scenes":{"antispam":{"cuts":[{"details":[{"label":"politics","score":0.92}]}]}}}}}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFinished {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFinished)
	}
	if snap.Suggestion != "block" {
		t.Errorf("suggestion = %q, want block", snap.Suggestion)
	}
	if snap.Label != "politics" {
		t.Errorf("label = %q, want politics", snap.Label)
	}
	if snap.Score != 0.92 {
		t.Errorf("score = %v, want 0.92", snap.Score)
	}
	if snap.Msg != "done" {
		t.Errorf("msg = %q, want done", snap.Msg)
	}
}

func TestAudioCensor_GetResult_Waiting(t *testing.T) {
	body := `{"id":"task-123","status":"WAITING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobWaiting {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobWaiting)
	}
}

func TestAudioCensor_GetResult_Doing(t *testing.T) {
	body := `{"id":"task-123","status":"DOING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobDoing {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobDoing)
	}
}

func TestAudioCensor_GetResult_Failed(t *testing.T) {
	body := `{"id":"task-123","status":"FAILED","error":"processing failed"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFailed {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFailed)
	}
	if snap.Error != "processing failed" {
		t.Errorf("error = %q, want processing failed", snap.Error)
	}
}

func TestAudioCensor_GetResult_FailedNoError(t *testing.T) {
	body := `{"id":"task-123","status":"FAILED"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Status != censor.JobFailed {
		t.Errorf("status = %q, want %q", snap.Status, censor.JobFailed)
	}
	if snap.Error != "qiniu audio job failed" {
		t.Errorf("error = %q, want 'qiniu audio job failed'", snap.Error)
	}
}

func TestAudioCensor_GetResult_UnknownStatus(t *testing.T) {
	body := `{"id":"task-123","status":"PREPARING"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
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

func TestAudioCensor_GetResult_FinishedNoResult(t *testing.T) {
	body := `{"id":"task-123","status":"FINISHED"}`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	snap, err := c.GetCensorResult(context.Background(), "task-123")
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

func TestAudioCensor_GetResult_HTTPError(t *testing.T) {
	body := `{"error":"server error"}`
	srv, client := newTestServer(http.StatusInternalServerError, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	_, err := c.GetCensorResult(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error should contain 'HTTP 500', got: %v", err)
	}
}

func TestAudioCensor_GetResult_InvalidJSON(t *testing.T) {
	body := `not valid json`
	srv, client := newTestServer(http.StatusOK, body)
	defer srv.Close()

	c, _ := NewAudioCensor(testConfig(srv, client))
	_, err := c.GetCensorResult(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("error should contain 'failed to parse response', got: %v", err)
	}
}
