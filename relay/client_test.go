// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/gemini"
	"github.com/LingByte/ling-base/relay/channel/openai"
	"github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/constant"
	"github.com/LingByte/ling-base/relay/meter"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/relaykit/types"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
)

// ─── Mock helpers ───────────────────────────────────────────────

// mockJSONServer creates a test HTTP server that responds with the given JSON body.
func mockJSONServer(t *testing.T, responseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, responseBody)
	}))
}

// mockSSEServer creates a test HTTP server that responds with SSE data.
func mockSSEServer(t *testing.T, sseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
}

// mockBinaryServer creates a test HTTP server that responds with raw binary data.
func mockBinaryServer(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
}

// mockAdaptor is a minimal common.Adaptor implementation for testing.
// It passes through requests and returns the raw response body, allowing
// the Client to parse it. It supports audio (unlike the openai adaptor
// in library mode).
type mockAdaptor struct {
	baseURL string
}

func (a *mockAdaptor) Init(info *common.RelayInfo) {}
func (a *mockAdaptor) GetRequestURL(info *common.RelayInfo) (string, error) {
	base := a.baseURL
	if info.ChannelBaseUrl != "" {
		base = info.ChannelBaseUrl
	}
	base = strings.TrimSuffix(base, "/")
	switch info.RelayMode {
	case relaymode.RelayModeAudioSpeech,
		relaymode.RelayModeAudioTranscription,
		relaymode.RelayModeAudioTranslation:
		return base + "/v1/audio/speech", nil
	case relaymode.RelayModeRerank:
		return base + "/v1/rerank", nil
	case relaymode.RelayModeResponses:
		return base + "/v1/responses", nil
	case relaymode.RelayModeModerations:
		return base + "/v1/moderations", nil
	case relaymode.RelayModeCompletions:
		return base + "/v1/completions", nil
	default:
		return base + "/v1/chat/completions", nil
	}
}
func (a *mockAdaptor) SetupRequestHeader(ctx context.Context, header *http.Header, info *common.RelayInfo) error {
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}
func (a *mockAdaptor) ConvertOpenAIRequest(ctx context.Context, info *common.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertClaudeRequest(ctx context.Context, info *common.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertGeminiRequest(ctx context.Context, info *common.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertImageRequest(ctx context.Context, info *common.RelayInfo, request dto.ImageRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertEmbeddingRequest(ctx context.Context, info *common.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertAudioRequest(ctx context.Context, info *common.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	jsonData, _ := json.Marshal(request)
	return strings.NewReader(string(jsonData)), nil
}
func (a *mockAdaptor) ConvertRerankRequest(ctx context.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) ConvertOpenAIResponsesRequest(ctx context.Context, info *common.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}
func (a *mockAdaptor) DoRequest(ctx context.Context, info *common.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return nil, nil // Client falls back to its own HTTP client.
}
func (a *mockAdaptor) DoResponse(ctx context.Context, resp *http.Response, info *common.RelayInfo, w http.ResponseWriter) (usage any, err *types.NewAPIError) {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewError(readErr, types.ErrorCodeReadResponseBodyFailed)
	}

	switch info.RelayMode {
	case relaymode.RelayModeAudioSpeech,
		relaymode.RelayModeAudioTranscription,
		relaymode.RelayModeAudioTranslation:
		// Audio: write raw bytes, record request count.
		w.Write(body)
		return meter.Usage{RequestCount: 1}, nil
	case relaymode.RelayModeRerank:
		w.Write(body)
		var resp struct {
			Usage *dto.Usage `json:"usage"`
		}
		_ = json.Unmarshal(body, &resp)
		if resp.Usage != nil {
			return resp.Usage, nil
		}
		return dto.Usage{}, nil
	case relaymode.RelayModeResponses:
		w.Write(body)
		var resp struct {
			Usage *dto.Usage `json:"usage"`
		}
		_ = json.Unmarshal(body, &resp)
		if resp.Usage != nil {
			return resp.Usage, nil
		}
		return dto.Usage{}, nil
	case relaymode.RelayModeModerations:
		w.Write(body)
		return meter.Usage{RequestCount: 1}, nil
	default:
		if info.IsStream {
			// Stream: write raw SSE, parse usage from final chunk.
			w.Write(body)
			u := dto.Usage{}
			for _, line := range strings.Split(string(body), "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "data: ") {
					continue
				}
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					continue
				}
				var chunk struct {
					Usage *dto.Usage `json:"usage"`
				}
				if json.Unmarshal([]byte(data), &chunk) == nil && chunk.Usage != nil {
					u = *chunk.Usage
				}
			}
			return &u, nil
		}
		// Non-stream chat / completions: re-marshal and write.
		w.Write(body)
		var resp struct {
			Usage *dto.Usage `json:"usage"`
		}
		_ = json.Unmarshal(body, &resp)
		if resp.Usage != nil {
			return resp.Usage, nil
		}
		return dto.Usage{}, nil
	}
}
func (a *mockAdaptor) GetModelList() []string  { return []string{"test-model"} }
func (a *mockAdaptor) GetChannelName() string { return "mock" }

// newMockClient creates a Client backed by a mockAdaptor pointing at baseURL.
func newMockClient(baseURL string, m meter.Meter) *Client {
	return New(
		WithProvider(NewProvider("mock", constant.APITypeOpenAI, &mockAdaptor{baseURL: baseURL}, baseURL, "test-key")),
		WithMeter(m),
	)
}

// ─── 1. TestClient_ChatStream ───────────────────────────────────

func TestClient_ChatStream(t *testing.T) {
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"Hello"}}]}`,
		``,
		`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" world"}}]}`,
		``,
		`data: {"id":"chatcmpl-1","choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	server := mockSSEServer(t, sseBody)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	result, err := client.ChatStream(ctx, &ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Collect chunks.
	var deltas []string
	for chunk := range result.Ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		if chunk.Delta != "" {
			deltas = append(deltas, chunk.Delta)
		}
	}

	// Verify 2 content deltas.
	require.Len(t, deltas, 2)
	assert.Equal(t, "Hello", deltas[0])
	assert.Equal(t, " world", deltas[1])

	// Verify final usage.
	assert.Equal(t, 5, result.Usage.InputTokens)
	assert.Equal(t, 2, result.Usage.OutputTokens)
	assert.Equal(t, 7, result.Usage.TotalTokens)

	// Verify meter recorded usage.
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 5, stats.TotalUsage.InputTokens)
	assert.Equal(t, 2, stats.TotalUsage.OutputTokens)
}

// ─── 2. TestClient_Audio (TTS) ──────────────────────────────────

func TestClient_Audio(t *testing.T) {
	audioData := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00} // fake WAV header
	server := mockBinaryServer(t, audioData)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	resp, err := client.Audio(ctx, &AudioRequest{
		Model:          "tts-1",
		Input:          json.RawMessage(`"Hello world"`),
		Voice:          "alloy",
		ResponseFormat: "mp3",
	}, false)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify response body is the audio data.
	assert.Equal(t, audioData, resp.Data)

	// Verify meter recorded RequestCount.
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1, stats.TotalUsage.RequestCount)
}

// ─── 3. TestClient_Rerank ───────────────────────────────────────

func TestClient_Rerank(t *testing.T) {
	mockResp := `{"results":[{"index":0,"relevance_score":0.95},{"index":1,"relevance_score":0.3}],"usage":{"prompt_tokens":10,"total_tokens":10}}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	resp, err := client.Rerank(ctx, &RerankRequest{
		Model:     "rerank-english-v3.0",
		Query:     "What is AI?",
		Documents: []string{"AI is artificial intelligence.", "I like pizza."},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify results count.
	require.Len(t, resp.Results, 2)
	assert.Equal(t, 0, resp.Results[0].Index)
	assert.InDelta(t, 0.95, resp.Results[0].RelevanceScore, 0.001)
	assert.Equal(t, 1, resp.Results[1].Index)
	assert.InDelta(t, 0.3, resp.Results[1].RelevanceScore, 0.001)

	// Verify usage recorded.
	assert.Equal(t, 10, resp.Usage.InputTokens)
	assert.Equal(t, 10, resp.Usage.TotalTokens)

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 10, stats.TotalUsage.InputTokens)
}

// ─── 4. TestClient_Responses ────────────────────────────────────

func TestClient_Responses(t *testing.T) {
	mockResp := `{"id":"resp-1","model":"gpt-4o","output":[{"type":"message","content":[{"type":"output_text","text":"Hello!"}]}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	resp, err := client.Responses(ctx, &ResponsesRequest{
		Model: "gpt-4o",
		Input: json.RawMessage(`"Hi"`),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify data is non-empty.
	assert.NotEmpty(t, resp.Data)
	assert.Contains(t, string(resp.Data), "resp-1")

	// Verify usage.
	assert.Equal(t, 5, resp.Usage.InputTokens)
	assert.Equal(t, 3, resp.Usage.OutputTokens)
	assert.Equal(t, 8, resp.Usage.TotalTokens)

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
}

// ─── 5. TestClient_Moderations ──────────────────────────────────

func TestClient_Moderations(t *testing.T) {
	mockResp := `{"id":"modr-1","model":"text-moderation-latest","results":[{"flagged":false,"categories":{"hate":false},"category_scores":{"hate":0.01}}]}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	resp, err := client.Moderations(ctx, &ModerationsRequest{
		Model: "text-moderation-latest",
		Input: "some text to moderate",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify results.
	assert.Equal(t, "modr-1", resp.ID)
	assert.Equal(t, "text-moderation-latest", resp.Model)
	require.Len(t, resp.Results, 1)
	assert.False(t, resp.Results[0].Flagged)
	assert.False(t, resp.Results[0].Categories["hate"])
	assert.InDelta(t, 0.01, resp.Results[0].CategoryScores["hate"], 0.001)

	// Verify meter recorded RequestCount.
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1, stats.TotalUsage.RequestCount)
}

// ─── 6. TestClient_Completions ──────────────────────────────────

func TestClient_Completions(t *testing.T) {
	mockResp := `{"id":"cmpl-1","model":"text-davinci-003","choices":[{"text":"Hello world"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := newMockClient(server.URL, m)

	ctx := context.Background()
	resp, err := client.Completions(ctx, &CompletionsRequest{
		Model:  "text-davinci-003",
		Prompt: json.RawMessage(`"Say hello"`),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify response.
	assert.Equal(t, "text-davinci-003", resp.Model)
	require.Len(t, resp.Choices, 1)
	assert.Contains(t, string(resp.Choices[0]), "Hello world")

	// Verify usage.
	assert.Equal(t, 5, resp.Usage.InputTokens)
	assert.Equal(t, 2, resp.Usage.OutputTokens)
	assert.Equal(t, 7, resp.Usage.TotalTokens)

	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 5, stats.TotalUsage.InputTokens)
}

// ─── 7. TestClient_GeminiChat ───────────────────────────────────

func TestClient_GeminiChat(t *testing.T) {
	mockResp := `{"candidates":[{"content":{"parts":[{"text":"2+2=4"}],"role":"model"}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"totalTokenCount":8}}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := New(
		WithProvider(gemini.NewProvider("test-key", gemini.WithBaseURL(server.URL))),
		WithMeter(m),
	)

	ctx := context.Background()
	req := &GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "What is 2+2?"},
				},
			},
		},
	}
	resp, err := client.GeminiChat(ctx, req, "gemini-pro")
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify candidates.
	require.Len(t, resp.Candidates, 1)
	require.Len(t, resp.Candidates[0].Content.Parts, 1)
	assert.Equal(t, "2+2=4", resp.Candidates[0].Content.Parts[0].Text)
	assert.Equal(t, "model", resp.Candidates[0].Content.Role)

	// Verify usage extraction (input=5, output=3, total=8).
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 5, stats.TotalUsage.InputTokens)
	assert.Equal(t, 3, stats.TotalUsage.OutputTokens)
	assert.Equal(t, 8, stats.TotalUsage.TotalTokens)
}

// ─── 8. TestClient_MidjourneySubmit ─────────────────────────────

func TestClient_MidjourneySubmit(t *testing.T) {
	mockResp := `{"code":1,"description":"SUCCESS","result":"task123"}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := New(
		WithProvider(NewProvider("midjourney", constant.APITypeOpenAI, &mockAdaptor{baseURL: server.URL}, server.URL, "mj-secret")),
		WithMeter(m),
	)

	ctx := context.Background()
	resp, err := client.MidjourneySubmit(ctx, server.URL, "mj-secret", relaymode.RelayModeMidjourneyImagine, &MidjourneyRequest{
		Prompt: "a cat in space",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify code and result.
	assert.Equal(t, 1, resp.Code)
	assert.Equal(t, "SUCCESS", resp.Description)
	assert.Equal(t, "task123", resp.Result)

	// Verify meter recorded RequestCount.
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1, stats.TotalUsage.RequestCount)
}

// ─── 9. TestClient_MidjourneyFetch ──────────────────────────────

func TestClient_MidjourneyFetch(t *testing.T) {
	mockResp := `{"id":"task123","status":"SUCCESS","progress":"100%","imageUrl":"https://example.com/img.png"}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	client := New()

	ctx := context.Background()
	task, err := client.MidjourneyFetch(ctx, server.URL, "mj-secret", "task123")
	require.NoError(t, err)
	require.NotNil(t, task)

	// Verify status and imageUrl.
	assert.Equal(t, "task123", task.ID)
	assert.Equal(t, "SUCCESS", task.Status)
	assert.Equal(t, "100%", task.Progress)
	assert.Equal(t, "https://example.com/img.png", task.ImageUrl)
}

// ─── 10. TestClient_SubmitSunoTask ──────────────────────────────

func TestClient_SubmitSunoTask(t *testing.T) {
	mockResp := `{"code":200,"message":"success","data":"suno-task-123"}`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	m := meter.NewMemoryMeter()
	client := New(
		WithProvider(NewProvider("suno", constant.APITypeOpenAI, &mockAdaptor{baseURL: server.URL}, server.URL, "suno-key")),
		WithMeter(m),
	)

	ctx := context.Background()
	resp, err := client.SubmitSunoTask(ctx, server.URL, "suno-key", SunoActionMusic, &SunoSubmitRequest{
		GptDescriptionPrompt: "A happy song",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify code and data.
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, "suno-task-123", resp.Data)

	// Verify meter recorded RequestCount.
	stats, err := m.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalRecords)
	assert.Equal(t, 1, stats.TotalUsage.RequestCount)
}

// ─── 11. TestClient_FetchSunoTask ───────────────────────────────

func TestClient_FetchSunoTask(t *testing.T) {
	mockResp := `[{"task_id":"suno-task-123","status":"SUCCESS","finish_time":1700000000}]`

	server := mockJSONServer(t, mockResp)
	defer server.Close()

	client := New()

	ctx := context.Background()
	tasks, err := client.FetchSunoTask(ctx, server.URL, "suno-key", []string{"suno-task-123"})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Verify task status.
	assert.Equal(t, "suno-task-123", tasks[0].TaskID)
	assert.Equal(t, "SUCCESS", tasks[0].Status)
	assert.Equal(t, int64(1700000000), tasks[0].FinishTime)
}

// ─── 12. TestExtractUsage ───────────────────────────────────────

func TestExtractUsage(t *testing.T) {
	t.Run("meter.Usage direct return", func(t *testing.T) {
		input := meter.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
		result := extractUsage(input)
		assert.Equal(t, 10, result.InputTokens)
		assert.Equal(t, 5, result.OutputTokens)
		assert.Equal(t, 15, result.TotalTokens)
	})

	t.Run("dto.Usage conversion", func(t *testing.T) {
		input := dto.Usage{PromptTokens: 20, CompletionTokens: 8, TotalTokens: 28}
		result := extractUsage(input)
		// PromptTokens → InputTokens
		assert.Equal(t, 20, result.InputTokens)
		assert.Equal(t, 8, result.OutputTokens)
		assert.Equal(t, 28, result.TotalTokens)
	})

	t.Run("nil returns empty", func(t *testing.T) {
		result := extractUsage(nil)
		assert.Equal(t, meter.Usage{}, result)
	})

	t.Run("dto.GeminiUsageMetadata conversion", func(t *testing.T) {
		input := dto.GeminiUsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 3,
			TotalTokenCount:      8,
		}
		result := extractUsage(input)
		assert.Equal(t, 5, result.InputTokens)
		assert.Equal(t, 3, result.OutputTokens)
		assert.Equal(t, 8, result.TotalTokens)
	})

	t.Run("pointer meter.Usage", func(t *testing.T) {
		input := &meter.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}
		result := extractUsage(input)
		assert.Equal(t, 100, result.InputTokens)
		assert.Equal(t, 50, result.OutputTokens)
		assert.Equal(t, 150, result.TotalTokens)
	})

	t.Run("nil pointer dto.Usage", func(t *testing.T) {
		var input *dto.Usage
		result := extractUsage(input)
		assert.Equal(t, meter.Usage{}, result)
	})
}

// ─── 13. TestParseChatResponse ──────────────────────────────────

func TestParseChatResponse(t *testing.T) {
	body := []byte(`{
		"id": "chatcmpl-abc123",
		"model": "gpt-4o",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello there!"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15
		}
	}`)

	usage := meter.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	resp, err := parseChatResponse(body, "gpt-4o", "openai", usage)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify ID, Model, Choices, Usage.
	assert.Equal(t, "chatcmpl-abc123", resp.ID)
	assert.Equal(t, "gpt-4o", resp.Model)
	assert.Equal(t, "openai", resp.Provider)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, 0, resp.Choices[0].Index)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	assert.Equal(t, 10, resp.Usage.InputTokens)
	assert.Equal(t, 5, resp.Usage.OutputTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

// ─── 14. TestBuildRelayInfo ─────────────────────────────────────

func TestBuildRelayInfo(t *testing.T) {
	t.Run("openai provider", func(t *testing.T) {
		m := meter.NewMemoryMeter()
		client := New(
			WithProvider(openai.NewProvider("my-api-key", openai.WithBaseURL("https://api.example.com"))),
			WithMeter(m),
		)

		info := client.buildRelayInfo(relaymode.RelayModeChatCompletions, "gpt-4o", true)

		// Verify RelayMode, OriginModelName, UpstreamModelName, IsStream, ApiType.
		assert.Equal(t, relaymode.RelayModeChatCompletions, info.RelayMode)
		assert.Equal(t, "gpt-4o", info.OriginModelName)
		assert.Equal(t, "gpt-4o", info.UpstreamModelName)
		assert.True(t, info.IsStream)
		assert.Equal(t, constant.APITypeOpenAI, info.ApiType)

		// Verify ConfiguredProvider BaseURL/APIKey populated.
		assert.Equal(t, "https://api.example.com", info.ChannelBaseUrl)
		assert.Equal(t, "my-api-key", info.ApiKey)
		assert.Equal(t, "https://api.example.com", info.ChannelMeta.ChannelBaseUrl)
		assert.Equal(t, "my-api-key", info.ChannelMeta.ApiKey)
		assert.Equal(t, "gpt-4o", info.ChannelMeta.UpstreamModelName)
	})

	t.Run("gemini provider non-stream", func(t *testing.T) {
		client := New(
			WithProvider(gemini.NewProvider("gem-key", gemini.WithBaseURL("https://gemini.example.com"))),
		)

		info := client.buildRelayInfo(relaymode.RelayModeGemini, "gemini-pro", false)

		assert.Equal(t, relaymode.RelayModeGemini, info.RelayMode)
		assert.Equal(t, "gemini-pro", info.OriginModelName)
		assert.Equal(t, "gemini-pro", info.UpstreamModelName)
		assert.False(t, info.IsStream)
		assert.Equal(t, constant.APITypeGemini, info.ApiType)
		assert.Equal(t, "https://gemini.example.com", info.ChannelBaseUrl)
		assert.Equal(t, "gem-key", info.ApiKey)
	})

	t.Run("generic provider", func(t *testing.T) {
		adaptor := &mockAdaptor{baseURL: "https://custom.example.com"}
		client := New(
			WithProvider(NewProvider("custom", constant.APITypeOpenAI, adaptor, "https://custom.example.com", "custom-key")),
		)

		info := client.buildRelayInfo(relaymode.RelayModeCompletions, "custom-model", false)

		assert.Equal(t, relaymode.RelayModeCompletions, info.RelayMode)
		assert.Equal(t, "custom-model", info.OriginModelName)
		assert.Equal(t, "custom-model", info.UpstreamModelName)
		assert.False(t, info.IsStream)
		assert.Equal(t, "https://custom.example.com", info.ChannelBaseUrl)
		assert.Equal(t, "custom-key", info.ApiKey)
	})
}
