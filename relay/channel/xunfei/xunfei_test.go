// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package xunfei_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/xunfei"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestXunfeiAdaptor_GetChannelName(t *testing.T) {
	a := xunfei.Adaptor{}
	assert.Equal(t, "xunfei", a.GetChannelName())
}

func TestXunfeiAdaptor_GetModelList(t *testing.T) {
	a := xunfei.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestXunfeiAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := xunfei.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "SparkDesk-v4.0",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "SparkDesk-v4.0",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestXunfeiAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := xunfei.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "SparkDesk-v4.0",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestXunfeiAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := xunfei.Adaptor{}
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
