// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package cohere_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/cohere"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestCohereAdaptor_GetChannelName(t *testing.T) {
	a := cohere.Adaptor{}
	assert.Equal(t, "cohere", a.GetChannelName())
}

func TestCohereAdaptor_GetModelList(t *testing.T) {
	a := cohere.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestCohereAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := cohere.Adaptor{}
	req := dto.RerankRequest{
		Model:     "rerank-english-v3.0",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestCohereAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := cohere.Adaptor{}
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

func TestCohereAdaptor_ConvertClaudeRequest_Unsupported(t *testing.T) {
	a := cohere.Adaptor{}
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
