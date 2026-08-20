// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ai360_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/channel/ai360"
)

func TestAI360_GetChannelName(t *testing.T) {
	a := ai360.New("test-key", "https://api.360.cn")
	assert.Equal(t, "ai360", a.GetChannelName())
}

func TestAI360_GetModelList(t *testing.T) {
	a := ai360.New("test-key", "")
	// ai360 shares the OpenAI model list; it should be non-nil.
	// (The list may be empty in this build, but the call must not panic.)
	_ = a.GetModelList()
}

func TestAI360_ConvertOpenAIRequest_PassThrough(t *testing.T) {
	a := ai360.New("test-key", "https://api.360.cn")
	info := common.NewRelayInfo()
	info.ChannelType = a.ChannelType

	stream := true
	req := &dto.GeneralOpenAIRequest{
		Model:    "gpt-4o-mini",
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

	// ai360 is not OpenAI/Azure, so StreamOptions should be cleared.
	assert.Nil(t, req.StreamOptions)
}

func TestAI360_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := ai360.New("test-key", "")
	info := common.NewRelayInfo()
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	require.Error(t, err)
}

func TestAI360_SetupRequestHeader_BearerAuth(t *testing.T) {
	a := ai360.New("my-secret-key", "https://api.360.cn")
	info := common.NewRelayInfo()
	info.ChannelType = a.ChannelType
	info.ApiKey = a.APIKey

	header := http.Header{}
	err := a.SetupRequestHeader(context.Background(), &header, info)
	require.NoError(t, err)

	assert.Equal(t, "Bearer my-secret-key", header.Get("Authorization"))
	assert.Equal(t, "application/json", header.Get("Content-Type"))
}

func TestAI360_New_DefaultBaseURL(t *testing.T) {
	a := ai360.New("key", "")
	assert.Equal(t, "https://api.360.cn", a.BaseURL)

	b := ai360.New("key", "https://custom.example.com")
	assert.Equal(t, "https://custom.example.com", b.BaseURL)
}
