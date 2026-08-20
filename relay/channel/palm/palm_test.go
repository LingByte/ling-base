// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package palm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/palm"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestPalmAdaptor_GetChannelName(t *testing.T) {
	a := palm.Adaptor{}
	assert.Equal(t, "google palm", a.GetChannelName())
}

func TestPalmAdaptor_GetModelList(t *testing.T) {
	a := palm.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestPalmAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := palm.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "PaLM-2",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "PaLM-2",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestPalmAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := palm.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "PaLM-2",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestPalmAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := palm.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "PaLM-2",
		},
	}
	req := dto.EmbeddingRequest{}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestPalmAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := palm.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "PaLM-2",
		},
	}
	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
