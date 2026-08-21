// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package baidu_v2_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/baidu_v2"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestBaiduV2Adaptor_GetChannelName(t *testing.T) {
	a := baidu_v2.Adaptor{}
	assert.Equal(t, "volcengine", a.GetChannelName())
}

func TestBaiduV2Adaptor_GetModelList(t *testing.T) {
	a := baidu_v2.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestBaiduV2Adaptor_ConvertClaudeRequest(t *testing.T) {
	a := baidu_v2.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "ernie-4.0-8k",
		},
	}
	req := &dto.ClaudeRequest{
		Model:    "ernie-4.0-8k",
		MaxTokens: ptrUint(1024),
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBaiduV2Adaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := baidu_v2.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "ernie-4.0-8k",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "ernie-4.0-8k",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBaiduV2Adaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := baidu_v2.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeImagesGenerations,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestBaiduV2Adaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := baidu_v2.Adaptor{}
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
