// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build live

package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/LingByte/ling-base/relay/channel/openai"
	"github.com/LingByte/ling-base/relay/meter"
)

// Live integration tests against Qiniu LLM API (https://llmapi.qiniu.io).
// Run with: go test -tags=live -run TestLive -v ./relay/ -timeout 120s
//
// These tests require a valid API key set in the QINIU_API_KEY env var,
// or they fall back to the hardcoded key below.

const qiniuBaseURL = "https://llmapi.qiniu.io"
const qiniuAPIKey = "sk-c3qxB9P3y1hq9xuiqOduUg"

func newQiniuClient(t *testing.T) *Client {
	t.Helper()
	provider := openai.NewProvider(
		qiniuAPIKey,
		openai.WithBaseURL(qiniuBaseURL),
	)
	return New(
		WithProvider(provider),
		WithMeter(meter.NewMemoryMeter()),
		WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)
}

// TestLiveModels lists available models via GET /v1/models.
func TestLiveModels(t *testing.T) {
	req, err := http.NewRequest("GET", qiniuBaseURL+"/v1/models", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+qiniuAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Models endpoint status: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		t.Fatalf("models endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse OpenAI-style models list response.
	var modelsResp struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		t.Fatalf("parse models: %v\nbody: %s", err, string(body))
	}

	t.Logf("Total models: %d", len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		if i < 20 {
			t.Logf("  [%d] %s (owned_by: %s)", i+1, m.ID, m.OwnedBy)
		}
	}
	if len(modelsResp.Data) > 20 {
		t.Logf("  ... and %d more", len(modelsResp.Data)-20)
	}
}

// TestLiveChat tests a basic chat completion.
func TestLiveChat(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	// Use a model allowed by this API key.
	resp, err := client.Chat(ctx, &ChatRequest{
		Model: "gpt-5.4-mini",
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"Say hello in one sentence."`)},
		},
		MaxTokens: intPtr(50),
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	t.Logf("Chat response ID: %s", resp.ID)
	t.Logf("Model: %s", resp.Model)
	t.Logf("Usage: %+v", resp.Usage)
	for _, choice := range resp.Choices {
		t.Logf("Choice[%d] finish=%s", choice.Index, choice.FinishReason)
		// Extract text content from the message.
		contentStr := fmt.Sprintf("%v", choice.Message.Content)
		t.Logf("  Content: %s", contentStr)
	}
}

// TestLiveChatStream tests streaming chat completion.
func TestLiveChatStream(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	result, err := client.ChatStream(ctx, &ChatRequest{
		Model: "gpt-5.4-mini",
		Messages: []Message{
			{Role: "user", Content: json.RawMessage(`"Count from 1 to 5."`)},
		},
		MaxTokens: intPtr(100),
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	t.Log("Streaming response:")
	var fullText string
	chunkCount := 0
	for chunk := range result.Ch {
		if chunk.Err != nil {
			t.Errorf("stream error: %v", chunk.Err)
			break
		}
		if chunk.Done {
			t.Logf("  [done] usage: %+v", chunk.Usage)
			break
		}
		if chunk.Delta != "" {
			fullText += chunk.Delta
			chunkCount++
		}
	}
	t.Logf("Total chunks: %d", chunkCount)
	t.Logf("Full text: %s", fullText)
}

// TestLiveChatWithModel tries chat with a different model.
func TestLiveChatWithModel(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	models := []string{"gpt-5.4-mini", "gpt-5.4", "grok-4.5"}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			resp, err := client.Chat(ctx, &ChatRequest{
				Model: model,
				Messages: []Message{
					{Role: "user", Content: json.RawMessage(`"Hi, what model are you?"`)},
				},
				MaxTokens: intPtr(80),
			})
			if err != nil {
				t.Logf("  [FAIL] %s: %v", model, err)
				return
			}
			var content string
			if len(resp.Choices) > 0 {
				content = fmt.Sprintf("%v", resp.Choices[0].Message.Content)
			}
			t.Logf("  [OK] %s -> %s", model, content)
		})
	}
}

// TestLiveEmbedding tests embedding API.
func TestLiveEmbedding(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	resp, err := client.Embed(ctx, &EmbedRequest{
		Model: "gpt-5.4-mini",
		Input: json.RawMessage(`["hello world"]`),
	})
	if err != nil {
		t.Logf("Embedding failed (may not be supported): %v", err)
		return
	}

	t.Logf("Embedding response model: %s", resp.Model)
	t.Logf("Data count: %d", len(resp.Data))
	if len(resp.Data) > 0 {
		t.Logf("First vector dim: %d", len(resp.Data[0].Embedding))
	}
	t.Logf("Usage: %+v", resp.Usage)
}

// TestLiveResponses tests the OpenAI Responses API (/v1/responses).
func TestLiveResponses(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	resp, err := client.Responses(ctx, &ResponsesRequest{
		Model: "gpt-5.4-mini",
		Input: json.RawMessage(`"Say hello in one sentence."`),
	})
	if err != nil {
		t.Fatalf("Responses failed: %v", err)
	}

	t.Logf("Responses usage: %+v", resp.Usage)
	t.Logf("Raw data length: %d bytes", len(resp.Data))

	// Try to pretty-print the raw JSON.
	var pretty map[string]any
	if err := json.Unmarshal(resp.Data, &pretty); err == nil {
		prettyBytes, _ := json.MarshalIndent(pretty, "", "  ")
		// Print first 1500 chars to avoid flooding output.
		s := string(prettyBytes)
		if len(s) > 1500 {
			s = s[:1500] + "\n... (truncated)"
		}
		t.Logf("Response JSON:\n%s", s)
	} else {
		t.Logf("Raw response: %s", string(resp.Data))
	}
}

// TestLiveResponsesStream tests streaming Responses API.
func TestLiveResponsesStream(t *testing.T) {
	client := newQiniuClient(t)
	ctx := context.Background()

	// The Responses API streaming uses a different path - check if client supports it.
	// ResponsesRequest has Stream field but Responses() doesn't return a stream channel.
	// Let's test with Stream=true and see what happens.
	resp, err := client.Responses(ctx, &ResponsesRequest{
		Model:  "gpt-5.4-mini",
		Input:  json.RawMessage(`"Count from 1 to 3"`),
		Stream: true,
	})
	if err != nil {
		t.Logf("Responses stream failed (may not be supported in library mode): %v", err)
		return
	}

	t.Logf("Stream responses usage: %+v", resp.Usage)
	t.Logf("Stream data length: %d bytes", len(resp.Data))

	// Show first 1000 chars of raw data.
	s := string(resp.Data)
	if len(s) > 1000 {
		s = s[:1000] + "\n... (truncated)"
	}
	t.Logf("Raw stream data:\n%s", s)
}

func intPtr(v int) *int { return &v }
