// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package tencent_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/tencent"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestTencentAdaptor_GetChannelName(t *testing.T) {
	a := tencent.Adaptor{}
	assert.Equal(t, "tencent", a.GetChannelName())
}

func TestTencentAdaptor_GetModelList(t *testing.T) {
	a := tencent.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestTencentAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := tencent.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "123456|secretId|secretKey",
			UpstreamModelName: "hunyuan-pro",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "hunyuan-pro",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTencentAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := tencent.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "123456|secretId|secretKey",
			UpstreamModelName: "hunyuan-pro",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestTencentAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := tencent.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestTencentAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := tencent.Adaptor{}
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

func TestTencentAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := tencent.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestTencentAdaptor_ConvertAudioRequest(t *testing.T) {
	a := tencent.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	req := dto.AudioRequest{
		Model: "test-model",
		Input: "hello",
	}
	reader, err := a.ConvertAudioRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, reader)
}
