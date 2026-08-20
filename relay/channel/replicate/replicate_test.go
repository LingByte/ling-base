// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package replicate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/replicate"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestReplicateAdaptor_GetChannelName(t *testing.T) {
	a := replicate.Adaptor{}
	assert.Equal(t, "replicate", a.GetChannelName())
}

func TestReplicateAdaptor_GetModelList(t *testing.T) {
	a := replicate.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestReplicateAdaptor_ConvertOpenAIRequest_Unsupported(t *testing.T) {
	a := replicate.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, &dto.GeneralOpenAIRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestReplicateAdaptor_ConvertClaudeRequest_Unsupported(t *testing.T) {
	a := replicate.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertClaudeRequest(context.Background(), info, &dto.ClaudeRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestReplicateAdaptor_ConvertEmbeddingRequest_Unsupported(t *testing.T) {
	a := replicate.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	_, err := a.ConvertEmbeddingRequest(context.Background(), info, dto.EmbeddingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
