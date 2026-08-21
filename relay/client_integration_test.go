// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package relay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/common/circuitbreaker"
	"github.com/LingByte/ling-base/common/retry"
	"github.com/LingByte/ling-base/relay/channel/openai"
)

// These tests exercise the full HTTP round-trip pipeline of the relay
// Client using httptest mock servers. No external API is required.

// ─── 1. TestIntegration_Chat_RoundTrip ──────────────────────────

func TestIntegration_Chat_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-test",
			"model": "gpt-4",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "Hello!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "chatcmpl-test", resp.ID)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, "openai", resp.Provider)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	assert.Equal(t, 10, resp.Usage.InputTokens)
	assert.Equal(t, 5, resp.Usage.OutputTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

// ─── 2. TestIntegration_Chat_Error_400 ──────────────────────────

func TestIntegration_Chat_Error_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid request",
				"type":    "invalid_request_error",
				"code":    "bad_request",
			},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "invalid request")
}

// ─── 3. TestIntegration_Chat_Error_429 ──────────────────────────

func TestIntegration_Chat_Error_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "rate limit exceeded",
				"type":    "rate_limit_error",
			},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "429")
	assert.Contains(t, err.Error(), "rate limit")
}

// ─── 4. TestIntegration_Chat_Error_500 ──────────────────────────

func TestIntegration_Chat_Error_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "server error")
}

// ─── 5. TestIntegration_ChatStream_RoundTrip ────────────────────

func TestIntegration_ChatStream_RoundTrip(t *testing.T) {
	sseResponse := "data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n" +
		"data: {\"id\":\"1\",\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":2,\"total_tokens\":7}}\n\n" +
		"data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	result, err := client.ChatStream(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	var fullText string
	var finalTotal int
	var gotFinal bool
	for chunk := range result.Ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if chunk.Done {
			require.NotNil(t, chunk.Usage)
			finalTotal = chunk.Usage.TotalTokens
			gotFinal = true
			break
		}
		fullText += chunk.Delta
	}
	assert.Contains(t, fullText, "Hello")
	assert.Contains(t, fullText, "world")
	require.True(t, gotFinal, "expected a final done chunk")
	assert.Equal(t, 7, finalTotal)
}

// ─── 6. TestIntegration_Embed_RoundTrip ─────────────────────────

func TestIntegration_Embed_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"model": "text-embedding-3-small",
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float32{0.1, 0.2, 0.3},
				},
			},
			"usage": map[string]any{"prompt_tokens": 8, "total_tokens": 8},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Embed(context.Background(), &EmbedRequest{
		Model: "text-embedding-3-small",
		Input: json.RawMessage(`"hello"`),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "text-embedding-3-small", resp.Model)
	assert.Equal(t, "openai", resp.Provider)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, 0, resp.Data[0].Index)
	assert.InDeltaSlice(t, []float32{0.1, 0.2, 0.3}, resp.Data[0].Embedding, 0.0001)
	assert.Equal(t, 8, resp.Usage.InputTokens)
	assert.Equal(t, 8, resp.Usage.TotalTokens)
}

// ─── 7. TestIntegration_Image_RoundTrip ─────────────────────────

func TestIntegration_Image_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/images/generations", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data": []map[string]any{
				{
					"url":            "https://example.com/image.png",
					"revised_prompt": "a cat in space",
				},
			},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	n := 1
	resp, err := client.Image(context.Background(), &ImageRequest{
		Model:  "dall-e-3",
		Prompt: "a cat in space",
		N:      &n,
		Size:   "1024x1024",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "openai", resp.Provider)
	assert.Equal(t, int64(1700000000), resp.Created)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "https://example.com/image.png", resp.Data[0].URL)
	assert.Equal(t, "a cat in space", resp.Data[0].RevisedPrompt)
	assert.Equal(t, 1, resp.Usage.ImageCount)
}

// ─── 8. TestIntegration_Responses_RoundTrip ─────────────────────

func TestIntegration_Responses_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/responses", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "resp-test",
			"model": "gpt-4o",
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{"type": "output_text", "text": "Hello from responses!"},
					},
				},
			},
			"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	resp, err := client.Responses(context.Background(), &ResponsesRequest{
		Model: "gpt-4o",
		Input: json.RawMessage(`"Hi"`),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEmpty(t, resp.Data)
	assert.Contains(t, string(resp.Data), "resp-test")
	assert.Contains(t, string(resp.Data), "Hello from responses!")
	assert.Equal(t, 5, resp.Usage.InputTokens)
	assert.Equal(t, 3, resp.Usage.OutputTokens)
	assert.Equal(t, 8, resp.Usage.TotalTokens)
}

// ─── 9. TestIntegration_Retry_On_500 ────────────────────────────

func TestIntegration_Retry_On_500(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&attempts, 1)
		if cur < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "test",
			"model": "gpt-4",
			"choices": []map[string]any{
				{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": "ok"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(
		WithProvider(provider),
		WithRetry(
			retry.WithMaxAttempts(3),
			retry.WithNoBackoff(),
		),
	)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	assert.Equal(t, "test", resp.ID)
	assert.Equal(t, 2, resp.Usage.TotalTokens)
}

// ─── 10. TestIntegration_CircuitBreaker ─────────────────────────

func TestIntegration_CircuitBreaker(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	cb := circuitbreaker.New(circuitbreaker.Config{
		MaxRequests:       1,
		FailureThreshold:  0.5,
		MinRequests:       2,
		RecoveryTimeout:   60 * time.Second, // long enough that it stays open during the test
		SlidingWindowSize: 10,
	})

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(
		WithProvider(provider),
		WithCircuitBreaker(cb),
		WithRetry(retry.WithMaxAttempts(1)), // one attempt per Chat call
	)

	req := &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	}

	// Call 1: 1st failure recorded, breaker stays closed (windowLen=1 < MinRequests=2).
	_, err1 := client.Chat(context.Background(), req)
	require.Error(t, err1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))

	// Call 2: 2nd failure recorded, failure rate 1.0 >= 0.5 with windowLen=2 → trips Open.
	_, err2 := client.Chat(context.Background(), req)
	require.Error(t, err2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&hits))
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	// Call 3: breaker is Open → rejected immediately, server NOT hit.
	_, err3 := client.Chat(context.Background(), req)
	require.Error(t, err3)
	assert.Equal(t, int32(2), atomic.LoadInt32(&hits)) // no additional hit
	assert.True(t, errors.Is(err3, circuitbreaker.ErrCircuitOpen),
		"expected ErrCircuitOpen, got: %v", err3)
}

// ─── 11. TestIntegration_Fallback ───────────────────────────────

func TestIntegration_Fallback(t *testing.T) {
	hits := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		model := req["model"].(string)

		if model == "gpt-4" {
			w.WriteHeader(500)
			w.Write([]byte("model unavailable"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "test", "model": model,
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "ok"}, "finish_reason": "stop"}},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(
		WithProvider(provider),
		WithFallback(FallbackConfig{
			FallbackModels: []string{"gpt-3.5-turbo"},
		}),
	)

	resp, err := client.Chat(context.Background(), &ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "gpt-3.5-turbo", resp.Model)
	assert.Equal(t, int32(2), hits.Load())
}

// ─── 12. TestIntegration_Context_Cancellation ───────────────────

func TestIntegration_Context_Cancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is cancelled (i.e. the client gave up).
		<-r.Context().Done()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key", openai.WithBaseURL(server.URL))
	client := New(WithProvider(provider))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := client.Chat(ctx, &ChatRequest{
			Model:    "gpt-4",
			Messages: []Message{{Role: "user", Content: "hi"}},
		})
		errCh <- err
	}()

	cancel() // cancel the in-flight request

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled),
			"expected context.Canceled, got: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for cancelled request to return")
	}
}
