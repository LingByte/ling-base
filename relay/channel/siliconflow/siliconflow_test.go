// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package siliconflow_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/siliconflow"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestSiliconFlowAdaptor_ConvertImageRequest(t *testing.T) {
	tests := []struct {
		name         string
		request      dto.ImageRequest
		wantModel    string
		wantPrompt   string
		wantImageSize string
		wantBatchSize uint
	}{
		{
			name: "basic conversion with size and n",
			request: dto.ImageRequest{
				Model:  "black-forest-labs/FLUX.1-schnell",
				Prompt: "A cat in space",
				Size:   "1024x1024",
				N:      uintPtr(2),
			},
			wantModel:     "black-forest-labs/FLUX.1-schnell",
			wantPrompt:    "A cat in space",
			wantImageSize: "1024x1024",
			wantBatchSize: 2,
		},
		{
			name: "no n defaults to zero batch size",
			request: dto.ImageRequest{
				Model:  "InstantX/InstantID",
				Prompt: "portrait",
				Size:   "512x512",
			},
			wantModel:     "InstantX/InstantID",
			wantPrompt:    "portrait",
			wantImageSize: "512x512",
			wantBatchSize: 0,
		},
	}

	a := &siliconflow.Adaptor{}
	info := common.NewRelayInfo()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := a.ConvertImageRequest(context.Background(), info, tc.request)
			require.NoError(t, err)

			sfReq, ok := result.(*siliconflow.SFImageRequest)
			require.True(t, ok, "result should be *SFImageRequest")
			assert.Equal(t, tc.wantModel, sfReq.Model)
			assert.Equal(t, tc.wantPrompt, sfReq.Prompt)
			assert.Equal(t, tc.wantImageSize, sfReq.ImageSize)
			assert.Equal(t, tc.wantBatchSize, sfReq.BatchSize)
		})
	}
}

func TestSiliconFlowAdaptor_GetModelList(t *testing.T) {
	a := &siliconflow.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestSiliconFlowAdaptor_GetChannelName(t *testing.T) {
	a := &siliconflow.Adaptor{}
	assert.Equal(t, "siliconflow", a.GetChannelName())
}

func uintPtr(v uint) *uint { return &v }
