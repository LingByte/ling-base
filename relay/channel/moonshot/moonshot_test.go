// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package moonshot_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/moonshot"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestMoonshotAdaptor_ConvertClaudeRequest_DelegatesToClaude(t *testing.T) {
	a := &moonshot.Adaptor{}
	info := common.NewRelayInfo()

	req := &dto.ClaudeRequest{
		Model: "moonshot-v1-8k",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	// Claude pass-through returns the same request.
	assert.Same(t, req, result)
}

func TestMoonshotAdaptor_ConvertClaudeRequest_Nil(t *testing.T) {
	a := &moonshot.Adaptor{}
	_, err := a.ConvertClaudeRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestMoonshotAdaptor_ConvertImageRequest_DelegatesToOpenAI(t *testing.T) {
	a := &moonshot.Adaptor{}
	info := common.NewRelayInfo()

	req := dto.ImageRequest{
		Model:  "dall-e-3",
		Prompt: "A cat",
	}

	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)
	// OpenAI pass-through returns the same request.
	assert.Equal(t, req, result)
}

func TestMoonshotAdaptor_GetModelList(t *testing.T) {
	a := &moonshot.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "kimi-k2.5")
}

func TestMoonshotAdaptor_GetChannelName(t *testing.T) {
	a := &moonshot.Adaptor{}
	assert.Equal(t, "moonshot", a.GetChannelName())
}
