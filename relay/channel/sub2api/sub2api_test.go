// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sub2api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/sub2api"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestSub2apiAdaptor_GetChannelName(t *testing.T) {
	a := sub2api.Adaptor{}
	assert.Equal(t, "sub2api", a.GetChannelName())
}

func TestSub2apiAdaptor_GetModelList(t *testing.T) {
	a := sub2api.Adaptor{}
	models := a.GetModelList()
	// sub2api ModelList is empty by design (models fetched dynamically)
	_ = models
}

func TestSub2apiAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := sub2api.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-4o",
		},
	}
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-4o",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}
	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestSub2apiAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := sub2api.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-4o",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestSub2apiAdaptor_ConvertClaudeRequest_PassThrough(t *testing.T) {
	a := sub2api.Adaptor{}
	info := &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "claude-3",
		},
	}
	req := &dto.ClaudeRequest{
		Model:     "claude-3",
		MaxTokens: ptrUint(1024),
	}
	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func ptrUint(v uint) *uint {
	return &v
}
