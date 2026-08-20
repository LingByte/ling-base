// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gemini_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/gemini"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestGeminiAdaptor_ConvertOpenAIRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *dto.GeneralOpenAIRequest
		check   func(t *testing.T, result any)
	}{
		{
			name: "messages to contents",
			request: &dto.GeneralOpenAIRequest{
				Model: "gemini-2.0-flash",
				Messages: []dto.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi"},
					{Role: "user", Content: "How are you?"},
				},
			},
			check: func(t *testing.T, result any) {
				geminiReq, ok := result.(*dto.GeminiChatRequest)
				require.True(t, ok, "result should be *dto.GeminiChatRequest")
				assert.NotEmpty(t, geminiReq.Contents)
				// User messages map to "user" role, assistant to "model".
				assert.Equal(t, "user", geminiReq.Contents[0].Role)
				assert.Equal(t, "model", geminiReq.Contents[1].Role)
				assert.Equal(t, "user", geminiReq.Contents[2].Role)
			},
		},
		{
			name: "system message to systemInstruction",
			request: &dto.GeneralOpenAIRequest{
				Model: "gemini-2.0-flash",
				Messages: []dto.Message{
					{Role: "system", Content: "You are a helpful assistant."},
					{Role: "user", Content: "Hello"},
				},
			},
			check: func(t *testing.T, result any) {
				geminiReq, ok := result.(*dto.GeminiChatRequest)
				require.True(t, ok)
				require.NotNil(t, geminiReq.SystemInstructions)
				// System message should not appear in contents.
				for _, c := range geminiReq.Contents {
					assert.NotEqual(t, "system", c.Role)
				}
			},
		},
	}

	a := gemini.New("test-key")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := common.NewRelayInfo()
			info.ChannelMeta = &common.ChannelMeta{}
			result, err := a.ConvertOpenAIRequest(context.Background(), info, tc.request)
			require.NoError(t, err)
			require.NotNil(t, result)
			tc.check(t, result)
		})
	}
}

func TestGeminiAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := gemini.New("test-key")
	_, err := a.ConvertOpenAIRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestGeminiAdaptor_ConvertGeminiRequest(t *testing.T) {
	a := gemini.New("test-key")
	info := common.NewRelayInfo()
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{Text: "hi"}}},
		},
	}
	result, err := a.ConvertGeminiRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Same(t, req, result)
}

func TestGeminiAdaptor_ConvertGeminiRequest_Nil(t *testing.T) {
	a := gemini.New("test-key")
	_, err := a.ConvertGeminiRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestGeminiAdaptor_ConvertImageRequest_Error(t *testing.T) {
	a := gemini.New("test-key")
	_, err := a.ConvertImageRequest(context.Background(), common.NewRelayInfo(), dto.ImageRequest{})
	require.Error(t, err)
}

func TestGeminiAdaptor_GetRequestURL(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		isStream    bool
		wantSuffix  string
	}{
		{
			name:       "chat model generateContent",
			model:      "gemini-2.0-flash",
			isStream:   false,
			wantSuffix: ":generateContent",
		},
		{
			name:       "stream model streamGenerateContent",
			model:      "gemini-2.0-flash",
			isStream:   true,
			wantSuffix: ":streamGenerateContent?alt=sse",
		},
		{
			name:       "imagen model predict",
			model:      "imagen-4.0-generate-001",
			isStream:   false,
			wantSuffix: ":predict",
		},
		{
			name:       "embedding model embedContent",
			model:      "text-embedding-001",
			isStream:   false,
			wantSuffix: ":embedContent",
		},
		{
			name:       "gemini-embedding model embedContent",
			model:      "gemini-embedding-001",
			isStream:   false,
			wantSuffix: ":embedContent",
		},
	}

	a := gemini.New("test-key")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := common.NewRelayInfo()
			info.ChannelMeta = &common.ChannelMeta{}
			info.UpstreamModelName = tc.model
			info.IsStream = tc.isStream
			url, err := a.GetRequestURL(info)
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(url, tc.wantSuffix),
				"expected URL to end with %q, got %q", tc.wantSuffix, url)
		})
	}
}

func TestGeminiAdaptor_GetModelList(t *testing.T) {
	a := gemini.New("test-key")
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "gemini-2.0-flash")
}
