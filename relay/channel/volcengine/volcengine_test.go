// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package volcengine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/volcengine"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestVolcengineAdaptor_GetChannelName(t *testing.T) {
	a := volcengine.Adaptor{}
	assert.Equal(t, "volcengine", a.GetChannelName())
}

func TestVolcengineAdaptor_GetModelList(t *testing.T) {
	a := volcengine.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestVolcengineAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := volcengine.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "Doubao-pro-32k",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "Doubao-pro-32k",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestVolcengineAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := volcengine.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "Doubao-pro-32k",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestVolcengineAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := volcengine.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "Doubao-embedding",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "Doubao-embedding",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestVolcengineAdaptor_ConvertOpenAIResponsesRequest_PassThrough(t *testing.T) {
	a := volcengine.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeResponses,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "Doubao-pro-32k",
		},
	}
	req := dto.OpenAIResponsesRequest{
		Model: "Doubao-pro-32k",
	}
	result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestVolcengineAdaptor_ConvertImageRequest_PassThrough(t *testing.T) {
	a := volcengine.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeImagesGenerations,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "doubao-seedream-4-0-250828",
		},
	}
	req := dto.ImageRequest{
		Model:  "doubao-seedream-4-0-250828",
		Prompt: "a cat",
	}
	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestVolcengineAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := volcengine.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}
