// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package submodel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/submodel"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestSubmodelAdaptor_GetChannelName(t *testing.T) {
	a := submodel.Adaptor{}
	assert.Equal(t, "submodel", a.GetChannelName())
}

func TestSubmodelAdaptor_GetModelList(t *testing.T) {
	a := submodel.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestSubmodelAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := submodel.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "deepseek-ai/DeepSeek-R1",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "deepseek-ai/DeepSeek-R1",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestSubmodelAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := submodel.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "deepseek-ai/DeepSeek-R1",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestSubmodelAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := submodel.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestSubmodelAdaptor_ConvertEmbeddingRequest_Unsupported(t *testing.T) {
	a := submodel.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertEmbeddingRequest(context.Background(), info, dto.EmbeddingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
