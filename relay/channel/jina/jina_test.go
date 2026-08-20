// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package jina_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/jina"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestJinaAdaptor_ConvertEmbeddingRequest_ClearsEncodingFormat(t *testing.T) {
	a := &jina.Adaptor{}
	info := common.NewRelayInfo()

	tests := []struct {
		name           string
		encodingFormat string
	}{
		{name: "base64 encoding format cleared", encodingFormat: "base64"},
		{name: "float encoding format cleared", encodingFormat: "float"},
		{name: "empty stays empty", encodingFormat: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := dto.EmbeddingRequest{
				Model:          "jina-clip-v1",
				Input:          "hello world",
				EncodingFormat: tc.encodingFormat,
			}

			result, err := a.ConvertEmbeddingRequest(context.Background(), info, req)
			require.NoError(t, err)

			outReq, ok := result.(dto.EmbeddingRequest)
			require.True(t, ok)
			assert.Empty(t, outReq.EncodingFormat, "EncodingFormat should be cleared")
		})
	}
}

func TestJinaAdaptor_ConvertRerankRequest_PassThrough(t *testing.T) {
	a := &jina.Adaptor{}

	req := dto.RerankRequest{
		Model:    "jina-reranker-v2-base-multilingual",
		Query:    "hello",
		Documents: []any{"world", "foo"},
	}

	result, err := a.ConvertRerankRequest(context.Background(), 0, req)
	require.NoError(t, err)
	assert.Equal(t, req, result)
}

func TestJinaAdaptor_GetModelList(t *testing.T) {
	a := &jina.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "jina-clip-v1")
}

func TestJinaAdaptor_GetChannelName(t *testing.T) {
	a := &jina.Adaptor{}
	assert.Equal(t, "jina", a.GetChannelName())
}
