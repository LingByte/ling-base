// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package huggingface_test

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/relay/channel/huggingface"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/constant"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaults(t *testing.T) {
	a := huggingface.New("hf-key")
	assert.Equal(t, "hf-key", a.APIKey)
	assert.Equal(t, huggingface.DefaultBaseURL, a.BaseURL)
	assert.Equal(t, constant.ChannelTypeHuggingFace, a.ChannelType)
	assert.Equal(t, "huggingface", a.GetChannelName())
	assert.Empty(t, a.GetModelList())
}

func TestWithBaseURL(t *testing.T) {
	a := huggingface.New("hf-key", huggingface.WithBaseURL("https://example.test"))
	assert.Equal(t, "https://example.test", a.BaseURL)
}

func TestGetRequestURLChatCompletions(t *testing.T) {
	a := huggingface.New("hf-key")
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: huggingface.DefaultBaseURL,
		},
	}
	url, err := a.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://router.huggingface.co/v1/chat/completions", url)
}

func TestConvertOpenAIRequestPassthrough(t *testing.T) {
	a := huggingface.New("hf-key")
	req := &dto.GeneralOpenAIRequest{Model: "meta-llama/Meta-Llama-3-8B-Instruct"}
	got, err := a.ConvertOpenAIRequest(context.Background(), &common.RelayInfo{}, req)
	require.NoError(t, err)
	assert.Equal(t, req, got)
}

func TestNewProvider(t *testing.T) {
	p := huggingface.NewProvider("hf-key")
	assert.Equal(t, "huggingface", p.Name())
	assert.Equal(t, constant.APITypeHuggingFace, p.ApiType())
	assert.Equal(t, huggingface.DefaultBaseURL, p.BaseURL())
	assert.Equal(t, "hf-key", p.APIKey())
	require.NotNil(t, p.Adaptor())
}
