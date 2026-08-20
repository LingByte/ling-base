// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mokaai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/mokaai"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestMokaaiAdaptor_GetChannelName(t *testing.T) {
	a := mokaai.Adaptor{}
	assert.Equal(t, "mokaai", a.GetChannelName())
}

func TestMokaaiAdaptor_GetModelList(t *testing.T) {
	a := mokaai.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestMokaaiAdaptor_ConvertOpenAIRequest_Embeddings(t *testing.T) {
	a := mokaai.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "m3e-base",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "m3e-base",
		Input: "hello",
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMokaaiAdaptor_ConvertOpenAIRequest_UnsupportedDefault(t *testing.T) {
	a := mokaai.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "m3e-base",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, &dto.GeneralOpenAIRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestMokaaiAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := mokaai.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "m3e-base",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "m3e-base",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestMokaaiAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := mokaai.Adaptor{}
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
