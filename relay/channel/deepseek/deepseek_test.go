// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package deepseek_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/deepseek"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestDeepSeekAdaptor_ConvertOpenAIResponsesRequest_ThinkingSuffix(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		hasChannelMeta bool
		wantModel    string
		wantEffort   string
	}{
		{
			name:           "v4-flash-none suffix sets disabled effort",
			model:          "deepseek-v4-flash-none",
			hasChannelMeta: true,
			wantModel:      "deepseek-v4-flash",
			wantEffort:     "none",
		},
		{
			name:           "v4-pro-max suffix sets max effort",
			model:          "deepseek-v4-pro-max",
			hasChannelMeta: true,
			wantModel:      "deepseek-v4-pro",
			wantEffort:     "max",
		},
		{
			name:           "no suffix leaves model unchanged",
			model:          "deepseek-chat",
			hasChannelMeta: true,
			wantModel:      "deepseek-chat",
			wantEffort:     "",
		},
	}

	a := &deepseek.Adaptor{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := common.NewRelayInfo()
			if tc.hasChannelMeta {
				info.ChannelMeta = &common.ChannelMeta{}
				info.UpstreamModelName = tc.model
			}

			req := dto.OpenAIResponsesRequest{
				Model: tc.model,
			}

			result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
			require.NoError(t, err)

			outReq, ok := result.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			assert.Equal(t, tc.wantModel, outReq.Model)

			if tc.wantEffort != "" {
				require.NotNil(t, outReq.Reasoning)
				assert.Equal(t, tc.wantEffort, outReq.Reasoning.Effort)
			}
		})
	}
}

func TestDeepSeekAdaptor_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := &deepseek.Adaptor{}
	info := common.NewRelayInfo()
	// No channel meta → no thinking suffix processing, pure pass-through.
	req := &dto.GeneralOpenAIRequest{
		Model: "deepseek-chat",
		Messages: []dto.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Same(t, req, result)
}

func TestDeepSeekAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := &deepseek.Adaptor{}
	_, err := a.ConvertOpenAIRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestDeepSeekAdaptor_GetModelList(t *testing.T) {
	a := &deepseek.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "deepseek-chat")
	assert.Contains(t, models, "deepseek-reasoner")
}

func TestDeepSeekAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := &deepseek.Adaptor{}
	info := common.NewRelayInfo()
	info.ChannelMeta = &common.ChannelMeta{
		ApiKey:            "test-key",
		UpstreamModelName: "test-model",
	}
	req := dto.EmbeddingRequest{
		Model: "test-model",
		Input: "hello",
	}
	result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestDeepSeekAdaptor_GetChannelName(t *testing.T) {
	a := &deepseek.Adaptor{}
	assert.Equal(t, "deepseek", a.GetChannelName())
}
