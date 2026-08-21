// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package jimeng_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/jimeng"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestJimengAdaptor_GetChannelName(t *testing.T) {
	a := jimeng.Adaptor{}
	assert.Equal(t, "jimeng", a.GetChannelName())
}

func TestJimengAdaptor_GetModelList(t *testing.T) {
	a := jimeng.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestJimengAdaptor_ConvertImageRequest(t *testing.T) {
	a := jimeng.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "jimeng_high_aes_general_v21_L",
		},
	}
	req := dto.ImageRequest{
		Model:  "jimeng_high_aes_general_v21_L",
		Prompt: "a cat",
	}
	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestJimengAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := jimeng.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "jimeng_high_aes_general_v21_L",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "jimeng_high_aes_general_v21_L",
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestJimengAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := jimeng.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "jimeng_high_aes_general_v21_L",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestJimengAdaptor_ConvertEmbeddingRequest_Unsupported(t *testing.T) {
	a := jimeng.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "jimeng_high_aes_general_v21_L",
		},
	}
	_, err := a.ConvertEmbeddingRequest(context.Background(), info, dto.EmbeddingRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
