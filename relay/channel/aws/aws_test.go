// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package aws_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/aws"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestAWSAdaptor_ConvertClaudeRequest_URLMediaSourcesToBase64(t *testing.T) {
	a := &aws.Adaptor{}
	info := common.NewRelayInfo()

	// Build a Claude request with a message containing a URL-type image source.
	// We use a data: URI so no network call is needed.
	mediaContent := []dto.ClaudeMediaMessage{
		{
			Type: "image",
			Source: &dto.ClaudeMessageSource{
				Type: "url",
				Url:  "data:image/png;base64,iVBORw0KGgo=",
			},
		},
		{
			Type: "text",
			Text: strPtr("Describe this image."),
		},
	}

	req := &dto.ClaudeRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: mediaContent},
		},
	}

	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)

	outReq, ok := result.(*dto.ClaudeRequest)
	require.True(t, ok)

	// The message should still have content that can be parsed.
	parsed, pErr := outReq.Messages[0].ParseContent()
	require.NoError(t, pErr)
	require.Len(t, parsed, 2)

	// The image source should now be base64, not url.
	imgMsg := parsed[0]
	require.NotNil(t, imgMsg.Source)
	assert.Equal(t, "base64", imgMsg.Source.Type)
	assert.Equal(t, "image/png", imgMsg.Source.MediaType)
	assert.NotEmpty(t, imgMsg.Source.Data)
	assert.Empty(t, imgMsg.Source.Url)
}

func TestAWSAdaptor_ConvertClaudeRequest_StringContentUnchanged(t *testing.T) {
	a := &aws.Adaptor{}
	info := common.NewRelayInfo()

	req := &dto.ClaudeRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "Hello, world!"},
		},
	}

	result, err := a.ConvertClaudeRequest(context.Background(), info, req)
	require.NoError(t, err)

	outReq, ok := result.(*dto.ClaudeRequest)
	require.True(t, ok)
	// String content should remain unchanged.
	assert.True(t, outReq.Messages[0].IsStringContent())
}

func TestAWSAdaptor_GetChannelName(t *testing.T) {
	a := &aws.Adaptor{}
	assert.Equal(t, "aws", a.GetChannelName())
}

func strPtr(s string) *string { return &s }
