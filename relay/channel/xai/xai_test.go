// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package xai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/xai"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestXAIAdaptor_ConvertOpenAIResponsesRequest_FillsModelFromInfo(t *testing.T) {
	tests := []struct {
		name           string
		requestModel   string
		upstreamModel  string
		wantModel      string
	}{
		{
			name:          "empty model filled from info.UpstreamModelName",
			requestModel:  "",
			upstreamModel: "grok-3",
			wantModel:     "grok-3",
		},
		{
			name:          "non-empty model preserved",
			requestModel:  "grok-4",
			upstreamModel: "grok-3",
			wantModel:     "grok-4",
		},
	}

	a := &xai.Adaptor{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := common.NewRelayInfo()
			info.ChannelMeta = &common.ChannelMeta{}
			info.UpstreamModelName = tc.upstreamModel

			req := dto.OpenAIResponsesRequest{
				Model: tc.requestModel,
			}

			result, err := a.ConvertOpenAIResponsesRequest(context.Background(), info, req)
			require.NoError(t, err)

			outReq, ok := result.(dto.OpenAIResponsesRequest)
			require.True(t, ok)
			assert.Equal(t, tc.wantModel, outReq.Model)
		})
	}
}

func TestXAIAdaptor_GetModelList(t *testing.T) {
	a := &xai.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	// The model list includes grok models.
	found := false
	for _, m := range models {
		if m == "grok-3" || m == "grok-3-mini" {
			found = true
			break
		}
	}
	assert.True(t, found, "model list should contain a grok model")
}

func TestXAIAdaptor_GetChannelName(t *testing.T) {
	a := &xai.Adaptor{}
	assert.Equal(t, "xai", a.GetChannelName())
}
