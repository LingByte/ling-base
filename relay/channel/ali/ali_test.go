// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ali_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/ali"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestAliAdaptor_GetChannelName(t *testing.T) {
	a := ali.Adaptor{}
	assert.Equal(t, "ali", a.GetChannelName())
}

func TestAliAdaptor_GetModelList(t *testing.T) {
	a := ali.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestAliAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := ali.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "qwen-max",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "qwen-max",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAliAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := ali.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "qwen-max",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestAliAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := ali.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "text-embedding-v1",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "text-embedding-v1",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestAliAdaptor_ConvertOpenAIResponsesRequest_PassThrough(t *testing.T) {
	a := ali.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeResponses,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "qwen-max",
		},
	}
	req := dto.OpenAIResponsesRequest{
		Model: "qwen-max",
	}
	result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestAliAdaptor_ConvertAudioRequest(t *testing.T) {
	a := ali.Adaptor{}
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
