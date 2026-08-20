// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package dify_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/dify"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestDifyAdaptor_GetChannelName(t *testing.T) {
	a := dify.Adaptor{}
	assert.Equal(t, "dify", a.GetChannelName())
}

func TestDifyAdaptor_GetModelList(t *testing.T) {
	a := dify.Adaptor{}
	models := a.GetModelList()
	// dify ModelList is empty by design (models fetched dynamically)
	_ = models
}

func TestDifyAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := dify.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "test-model",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "test-model",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDifyAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := dify.Adaptor{}
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

func TestDifyAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := dify.Adaptor{}
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
