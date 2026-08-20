// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package zhipu_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/zhipu"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestZhipuAdaptor_GetChannelName(t *testing.T) {
	a := zhipu.Adaptor{}
	assert.Equal(t, "zhipu", a.GetChannelName())
}

func TestZhipuAdaptor_GetModelList(t *testing.T) {
	a := zhipu.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestZhipuAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := zhipu.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "chatglm_turbo",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "chatglm_turbo",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestZhipuAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := zhipu.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "chatglm_turbo",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestZhipuAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := zhipu.Adaptor{}
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
