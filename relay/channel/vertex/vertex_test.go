// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package vertex_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/vertex"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/setting"
)

func TestVertexAdaptor_ConvertGeminiRequest_DelegatesAndRemovesFunctionResponseID(t *testing.T) {
	tests := []struct {
		name            string
		removeIDEnabled bool
	}{
		{name: "with removeID enabled", removeIDEnabled: true},
		{name: "without removeID", removeIDEnabled: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &vertex.Adaptor{}
			info := common.NewRelayInfo()
			info.ChannelMeta = &common.ChannelMeta{}

			// Build a Gemini request with a functionResponse that has an ID.
			funcRespID := jsonRaw(`"fn-123"`)
			req := &dto.GeminiChatRequest{
				Contents: []dto.GeminiChatContent{
					{
						Role: "user",
						Parts: []dto.GeminiPart{
							{Text: "call the function"},
						},
					},
					{
						Role: "function",
						Parts: []dto.GeminiPart{
							{
								FunctionResponse: &dto.GeminiFunctionResponse{
									Name:     "get_weather",
									Response: map[string]interface{}{"temp": 72},
									ID:       funcRespID,
								},
							},
						},
					},
				},
			}

			// Set the setting.
			gs := setting.GetGeminiSettings()
			prev := gs.RemoveFunctionResponseIdEnabled
			gs.RemoveFunctionResponseIdEnabled = tc.removeIDEnabled
			defer func() { gs.RemoveFunctionResponseIdEnabled = prev }()

			result, err := a.ConvertGeminiRequest(context.Background(), info, req)
			require.NoError(t, err)

			outReq, ok := result.(*dto.GeminiChatRequest)
			require.True(t, ok)

			// The functionResponse should be present.
			fnResp := outReq.Contents[1].Parts[0].FunctionResponse
			require.NotNil(t, fnResp)
			assert.Equal(t, "get_weather", fnResp.Name)

			if tc.removeIDEnabled {
				// ID should have been removed (set to nil).
				assert.Nil(t, fnResp.ID)
			} else {
				// ID should be preserved.
				assert.NotNil(t, fnResp.ID)
			}
		})
	}
}

func TestVertexAdaptor_GetChannelName(t *testing.T) {
	a := &vertex.Adaptor{}
	assert.Equal(t, "vertex-ai", a.GetChannelName())
}

func TestVertexAdaptor_ConvertEmbeddingRequest_PassThrough(t *testing.T) {
	a := &vertex.Adaptor{}
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

func TestVertexAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := &vertex.Adaptor{}
	req := dto.RerankRequest{
		Model:     "test-model",
		Query:     "hello",
		Documents: []any{"doc1", "doc2"},
	}
	result, err := a.ConvertRerankRequest(context.Background(), relaymode.RelayModeRerank, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestVertexAdaptor_ConvertAudioRequest(t *testing.T) {
	a := &vertex.Adaptor{}
	info := common.NewRelayInfo()
	info.ChannelMeta = &common.ChannelMeta{
		ApiKey:            "test-key",
		UpstreamModelName: "test-model",
	}
	req := dto.AudioRequest{
		Model: "test-model",
		Input: "hello",
	}
	reader, err := a.ConvertAudioRequest(context.Background(), info, req)
	require.NoError(t, err)
	assert.NotNil(t, reader)
}

// jsonRaw is a helper to create json.RawMessage from a string.
func jsonRaw(s string) []byte { return []byte(s) }
