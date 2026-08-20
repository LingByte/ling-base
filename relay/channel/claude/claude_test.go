// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package claude_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/claude"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func newClaudeInfo() *common.RelayInfo {
	info := common.NewRelayInfo()
	info.ChannelMeta = &common.ChannelMeta{}
	info.ConvOptions().Claude.DefaultMaxTokens = func(model string) int { return 1024 }
	return info
}

func boolPtr(v bool) *bool { return &v }

func TestClaudeAdaptor_ConvertOpenAIRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *dto.GeneralOpenAIRequest
		info    *common.RelayInfo
		check   func(t *testing.T, result any)
	}{
		{
			name: "user/assistant/system messages",
			info: newClaudeInfo(),
			request: &dto.GeneralOpenAIRequest{
				Model: "claude-3-5-sonnet-20241022",
				Messages: []dto.Message{
					{Role: "system", Content: "You are helpful."},
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there"},
					{Role: "user", Content: "How are you?"},
				},
			},
			check: func(t *testing.T, result any) {
				claudeReq, ok := result.(*dto.ClaudeRequest)
				require.True(t, ok, "result should be *dto.ClaudeRequest")
				assert.Equal(t, "claude-3-5-sonnet-20241022", claudeReq.Model)
				// System message should be extracted to the System field.
				assert.NotNil(t, claudeReq.System)
				// Non-system messages should be in Messages.
				assert.GreaterOrEqual(t, len(claudeReq.Messages), 3)
				assert.Equal(t, "user", claudeReq.Messages[0].Role)
				assert.Equal(t, "assistant", claudeReq.Messages[1].Role)
				assert.Equal(t, "user", claudeReq.Messages[2].Role)
			},
		},
		{
			name: "max_tokens default applied",
			info: newClaudeInfo(),
			request: &dto.GeneralOpenAIRequest{
				Model: "claude-3-5-sonnet-20241022",
				Messages: []dto.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			check: func(t *testing.T, result any) {
				claudeReq, ok := result.(*dto.ClaudeRequest)
				require.True(t, ok)
				require.NotNil(t, claudeReq.MaxTokens)
				assert.Equal(t, uint(1024), *claudeReq.MaxTokens)
			},
		},
		{
			name: "stream setting preserved",
			info: func() *common.RelayInfo {
				i := newClaudeInfo()
				i.IsStream = true
				return i
			}(),
			request: &dto.GeneralOpenAIRequest{
				Model:  "claude-3-5-sonnet-20241022",
				Stream: boolPtr(true),
				Messages: []dto.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			check: func(t *testing.T, result any) {
				claudeReq, ok := result.(*dto.ClaudeRequest)
				require.True(t, ok)
				require.NotNil(t, claudeReq.Stream)
				assert.True(t, *claudeReq.Stream)
			},
		},
	}

	a := claude.New("test-key")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := a.ConvertOpenAIRequest(context.Background(), tc.info, tc.request)
			require.NoError(t, err)
			require.NotNil(t, result)
			tc.check(t, result)
		})
	}
}

func TestClaudeAdaptor_ConvertOpenAIRequest_Nil(t *testing.T) {
	a := claude.New("test-key")
	_, err := a.ConvertOpenAIRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestClaudeAdaptor_ConvertClaudeRequest(t *testing.T) {
	a := claude.New("test-key")
	info := common.NewRelayInfo()
	req := &dto.ClaudeRequest{
		Model:    "claude-3-5-sonnet-20241022",
		Messages: []dto.ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.Same(t, req, result)
}

func TestClaudeAdaptor_ConvertClaudeRequest_Nil(t *testing.T) {
	a := claude.New("test-key")
	_, err := a.ConvertClaudeRequest(context.Background(), common.NewRelayInfo(), nil)
	require.Error(t, err)
}

func TestClaudeAdaptor_ConvertImageRequest_Unsupported(t *testing.T) {
	a := claude.New("test-key")
	_, err := a.ConvertImageRequest(context.Background(), common.NewRelayInfo(), dto.ImageRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image")
}

func TestClaudeAdaptor_ConvertAudioRequest_Unsupported(t *testing.T) {
	a := claude.New("test-key")
	_, err := a.ConvertAudioRequest(context.Background(), common.NewRelayInfo(), dto.AudioRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audio")
}

func TestClaudeAdaptor_GetModelList(t *testing.T) {
	a := claude.New("test-key")
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "claude-3-5-sonnet-20241022")
}

func TestClaudeAdaptor_GetChannelName(t *testing.T) {
	a := claude.New("test-key")
	assert.Equal(t, "claude", a.GetChannelName())
}
