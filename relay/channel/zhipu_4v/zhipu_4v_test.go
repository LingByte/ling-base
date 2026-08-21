// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package zhipu_4v_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/zhipu_4v"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestZhipu4VAdaptor_GetChannelName(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	assert.Equal(t, "zhipu_4v", a.GetChannelName())
}

func TestZhipu4VAdaptor_GetModelList(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestZhipu4VAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "glm-4",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "glm-4",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestZhipu4VAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "glm-4",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestZhipu4VAdaptor_ConvertClaudeRequest_PassThrough(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "glm-4",
		},
	}
	req := &dto.ClaudeRequest{
		Model:     "glm-4",
		MaxTokens: ptrUint(1024),
	}
	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestZhipu4VAdaptor_ConvertImageRequest_PassThrough(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeImagesGenerations,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "glm-4",
		},
	}
	req := dto.ImageRequest{
		Model:  "glm-4",
		Prompt: "a cat",
	}
	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestZhipu4VAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "glm-4",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "glm-4",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestZhipu4VAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := zhipu_4v.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func ptrUint(v uint) *uint {
	return &v
}
