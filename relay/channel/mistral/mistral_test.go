// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mistral_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/mistral"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestMistralAdaptor_GetChannelName(t *testing.T) {
	a := mistral.Adaptor{}
	assert.Equal(t, "mistral", a.GetChannelName())
}

func TestMistralAdaptor_GetModelList(t *testing.T) {
	a := mistral.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestMistralAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := mistral.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "mistral-large-latest",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "mistral-large-latest",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMistralAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := mistral.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "mistral-large-latest",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestMistralAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := mistral.Adaptor{}
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

func TestMistralAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := mistral.Adaptor{}
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

func TestMistralAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := mistral.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}
