// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package codex_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/codex"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestCodexAdaptor_GetChannelName(t *testing.T) {
	a := codex.Adaptor{}
	assert.Equal(t, "codex", a.GetChannelName())
}

func TestCodexAdaptor_GetModelList(t *testing.T) {
	a := codex.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestCodexAdaptor_ConvertOpenAIResponsesRequest(t *testing.T) {
	a := codex.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeResponses,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-5.5",
		},
	}
	req := dto.OpenAIResponsesRequest{
		Model: "gpt-5.5",
	}
	result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
	require.NoError(t, err)

	outReq, ok := result.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	// Codex backend requires the instructions field to be present; it defaults
	// to an empty string when not provided.
	assert.NotEmpty(t, outReq.Instructions)
	// store must be false for non-compact responses.
	assert.Equal(t, "false", string(outReq.Store))
	// max_output_tokens and temperature are removed for non-compact responses.
	assert.Nil(t, outReq.MaxOutputTokens)
	assert.Nil(t, outReq.Temperature)
}

func TestCodexAdaptor_ConvertOpenAIRequest_Unsupported(t *testing.T) {
	a := codex.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-5.5",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, &dto.GeneralOpenAIRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestCodexAdaptor_ConvertEmbeddingRequest_Unsupported(t *testing.T) {
	a := codex.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-5.5",
		},
	}
	_, err := a.ConvertEmbeddingRequest(context.Background(), info, dto.EmbeddingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
