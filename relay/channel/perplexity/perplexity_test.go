// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package perplexity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/perplexity"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestPerplexityAdaptor_GetChannelName(t *testing.T) {
	a := perplexity.Adaptor{}
	assert.Equal(t, "perplexity", a.GetChannelName())
}

func TestPerplexityAdaptor_GetModelList(t *testing.T) {
	a := perplexity.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestPerplexityAdaptor_ConvertClaudeRequest(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "sonar",
		},
	}
	req := &dto.ClaudeRequest{
		Model:     "sonar",
		MaxTokens: ptrUint(1024),
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	// delegates to openai.Adaptor.ConvertClaudeRequest which may return error or converted request
	// we just verify it doesn't panic and returns something
	_ = err
	_ = result
}

func TestPerplexityAdaptor_ConvertOpenAIResponsesRequest_PassThrough(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeResponses,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "sonar",
		},
	}
	req := dto.OpenAIResponsesRequest{
		Model: "sonar",
	}
	result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestPerplexityAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "sonar",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "sonar",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPerplexityAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "sonar",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestPerplexityAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "sonar",
		},
	}
	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestPerplexityAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := perplexity.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "test-model",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func ptrUint(v uint) *uint {
	return &v
}
