// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package ollama_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/ollama"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestOllamaAdaptor_ConvertClaudeRequest_DelegatesToOpenAI(t *testing.T) {
	a := &ollama.Adaptor{}
	info := common.NewRelayInfo()

	req := &dto.ClaudeRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	// OpenAI's ConvertClaudeRequest is a pass-through.
	assert.Same(t, req, result)
}

func TestOllamaAdaptor_ConvertClaudeRequest_Nil(t *testing.T) {
	a := &ollama.Adaptor{}
	result, err := a.ConvertClaudeRequest(context.Background(), common.NewRelayInfo(), nil)
	// OpenAI's ConvertClaudeRequest is a pass-through, so nil input returns nil, nil.
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestOllamaAdaptor_GetModelList(t *testing.T) {
	a := &ollama.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestOllamaAdaptor_GetChannelName(t *testing.T) {
	a := &ollama.Adaptor{}
	assert.Equal(t, "ollama", a.GetChannelName())
}
