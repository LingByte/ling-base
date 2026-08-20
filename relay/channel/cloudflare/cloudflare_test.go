// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cloudflare_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/cloudflare"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestCloudflareAdaptor_GetChannelName(t *testing.T) {
	a := cloudflare.Adaptor{}
	assert.Equal(t, "cloudflare", a.GetChannelName())
}

func TestCloudflareAdaptor_GetModelList(t *testing.T) {
	a := cloudflare.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestCloudflareAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := cloudflare.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "@cf/meta/llama-3.1-8b-instruct",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "@cf/meta/llama-3.1-8b-instruct",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestCloudflareAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := cloudflare.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestCloudflareAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := cloudflare.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
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

func TestCloudflareAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := cloudflare.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestCloudflareAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := cloudflare.Adaptor{}
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
