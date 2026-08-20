// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package newapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/newapi"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestNewapiAdaptor_GetChannelName(t *testing.T) {
	a := newapi.Adaptor{}
	assert.Equal(t, "newapi", a.GetChannelName())
}

func TestNewapiAdaptor_GetModelList(t *testing.T) {
	a := newapi.Adaptor{}
	models := a.GetModelList()
	// newapi ModelList is empty by design (models fetched dynamically)
	_ = models
}

func TestNewapiAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := newapi.Adaptor{}
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

func TestNewapiAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := newapi.Adaptor{}
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

func TestNewapiAdaptor_ConvertClaudeRequest_PassThrough(t *testing.T) {
	a := newapi.Adaptor{}
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

func TestNewapiAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := newapi.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeEmbeddings,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "text-embedding-3-small",
		},
	}
	req := dto.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func ptrUint(v uint) *uint {
	return &v
}
