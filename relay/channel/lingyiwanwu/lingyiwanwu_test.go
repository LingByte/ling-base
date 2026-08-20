// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package lingyiwanwu_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/channel/lingyiwanwu"
)

func TestLingYiWanWu_GetChannelName(t *testing.T) {
	a := lingyiwanwu.New("test-key", "")
	assert.Equal(t, "lingyiwanwu", a.GetChannelName())
}

func TestLingYiWanWu_GetModelList(t *testing.T) {
	a := lingyiwanwu.New("test-key", "")
	models := a.GetModelList()
	require.NotEmpty(t, models)
	assert.Contains(t, models, "yi-large")
	assert.Contains(t, models, "yi-medium")
}

func TestLingYiWanWu_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := lingyiwanwu.New("test-key", "https://api.lingyiwanwu.com")
	info := common.NewRelayInfo()
	info.ChannelType = a.ChannelType

	stream := true
	req := &dto.GeneralOpenAIRequest{
		Model:    "yi-large",
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
		Stream:   &stream,
		StreamOptions: &dto.StreamOptions{
			IncludeUsage: true,
		},
	}

	out, err := a.ConvertOpenAIRequest(context.Background(), info, req)
	require.NoError(t, err)
	require.NotNil(t, out)

	// Pass-through: the same request pointer is returned.
	assert.Same(t, req, out)

	// lingyiwanwu is not OpenAI/Azure, so StreamOptions should be cleared.
	assert.Nil(t, req.StreamOptions)
}

func TestLingYiWanWu_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := lingyiwanwu.New("test-key", "")
	info := common.NewRelayInfo()
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestLingYiWanWu_New_DefaultBaseURL(t *testing.T) {
	a := lingyiwanwu.New("key", "")
	assert.Equal(t, "https://api.lingyiwanwu.com", a.BaseURL)

	b := lingyiwanwu.New("key", "https://custom.example.com")
	assert.Equal(t, "https://custom.example.com", b.BaseURL)
}
