// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package openrouter_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/channel/openrouter"
)

func TestOpenRouter_GetChannelName(t *testing.T) {
	a := openrouter.New("test-key", "")
	assert.Equal(t, "openrouter", a.GetChannelName())
}

func TestOpenRouter_GetModelList(t *testing.T) {
	a := openrouter.New("test-key", "")
	// OpenRouter exposes an empty model list (models are routed dynamically);
	// the call must succeed without panicking.
	_ = a.GetModelList()
}

func TestOpenRouter_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := openrouter.New("test-key", "https://openrouter.ai/api")
	info := common.NewRelayInfo()
	info.ChannelType = a.ChannelType

	stream := true
	req := &dto.GeneralOpenAIRequest{
		Model:    "openai/gpt-4o",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
		Stream:   &stream,
		StreamOptions: &dto.StreamOptions{
			IncludeUsage: true,
		},
	}

	out, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	require.NotNil(t, out)

	// Pass-through: the same request pointer is returned.
	assert.Same(t, req, out)

	// OpenRouter is not OpenAI/Azure, so StreamOptions should be cleared.
	assert.Nil(t, req.StreamOptions)
}

func TestOpenRouter_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := openrouter.New("test-key", "")
	info := common.NewRelayInfo()
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestOpenRouter_SetupRequestHeader_SpecialHeaders(t *testing.T) {
	a := openrouter.New("my-secret-key", "https://openrouter.ai/api")
	info := common.NewRelayInfo()
	info.ChannelType = a.ChannelType
	info.ApiKey = a.APIKey

	header := http.Header{}
	err := a.SetupRequestHeader(context.Background(), &header, info)
	require.NoError(t, err)

	// Bearer auth (inherited from OpenAI adaptor).
	assert.Equal(t, "Bearer my-secret-key", header.Get("Authorization"))
	assert.Equal(t, "application/json", header.Get("Content-Type"))

	// OpenRouter-specific headers.
	assert.Equal(t, "https://lingbyte.dev", header.Get("HTTP-Referer"))
	assert.Equal(t, "ling-base", header.Get("X-OpenRouter-Title"))
}

func TestOpenRouter_New_DefaultBaseURL(t *testing.T) {
	a := openrouter.New("key", "")
	assert.Equal(t, "https://openrouter.ai/api", a.BaseURL)

	b := openrouter.New("key", "https://custom.example.com")
	assert.Equal(t, "https://custom.example.com", b.BaseURL)
}
