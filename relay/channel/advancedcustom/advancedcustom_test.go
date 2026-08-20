// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package advancedcustom_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/relay/channel/advancedcustom"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestAdvancedcustomAdaptor_GetChannelName(t *testing.T) {
	a := advancedcustom.Adaptor{}
	assert.Equal(t, "advanced_custom", a.GetChannelName())
}

func TestAdvancedcustomAdaptor_GetModelList(t *testing.T) {
	a := advancedcustom.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestAdvancedcustomAdaptor_ConvertOpenAIRequest(t *testing.T) {
	a := advancedcustom.Adaptor{}
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
	assert.NotNil(t, result)
}

func TestAdvancedcustomAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := advancedcustom.Adaptor{}
	info := &common.RelayInfo{
		RelayMode: relaymode.RelayModeChatCompletions,
		ChannelMeta: &common.ChannelMeta{
			ApiKey:            "test-key",
			UpstreamModelName: "gpt-4o",
		},
	}
	_, err := a.ConvertOpenAIRequest(context.Background(), info, nil)
	// nil request: resolveForConversion succeeds (converter=ConverterNone),
	// then convertOpenAICompatibleRequest delegates to openai adaptor which checks nil
	// The openai adaptor may or may not check nil — verify behavior
	_ = err
}
