// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package coze_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/coze"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestCozeAdaptor_GetChannelName(t *testing.T) {
	a := coze.Adaptor{}
	assert.Equal(t, "coze", a.GetChannelName())
}

func TestCozeAdaptor_GetModelList(t *testing.T) {
	a := coze.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestCozeAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := coze.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "moonshot-v1-8k",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "moonshot-v1-8k",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCozeAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := coze.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "moonshot-v1-8k",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestCozeAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := coze.Adaptor{}
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

func TestCozeAdaptor_ConvertClaudeRequest_Unsupported(t *testing.T) {
	a := coze.Adaptor{}
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
