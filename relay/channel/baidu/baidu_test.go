// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package baidu_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/baidu"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestBaiduAdaptor_GetChannelName(t *testing.T) {
	a := baidu.Adaptor{}
	assert.Equal(t, "baidu", a.GetChannelName())
}

func TestBaiduAdaptor_GetModelList(t *testing.T) {
	a := baidu.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestBaiduAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := baidu.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "ERNIE-4.0-8K",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "ERNIE-4.0-8K",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBaiduAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := baidu.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "ERNIE-4.0-8K",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestBaiduAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := baidu.Adaptor{}
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

func TestBaiduAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := baidu.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "Embedding-V1",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "Embedding-V1",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestBaiduAdaptor_ConvertClaudeRequest_Unsupported(t *testing.T) {
	a := baidu.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertClaudeRequest(context.Background(), info, &dto.ClaudeRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestBaiduAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := baidu.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestBaiduAdaptor_ConvertAudioRequest(t *testing.T) {
	a := baidu.Adaptor{}
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
