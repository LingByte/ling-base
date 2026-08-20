// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package minimax_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel/minimax"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
)

func TestMiniMaxAdaptor_ConvertImageRequest(t *testing.T) {
	a := &minimax.Adaptor{}
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeImagesGenerations
	info.OriginModelName = "image-01"

	n := uint(2)
	req := dto.ImageRequest{
		Model:  "image-01",
		Prompt: "A beautiful sunset",
		N:      &n,
		Size:   "1024x1024",
	}

	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)

	mmReq, ok := result.(minimax.MiniMaxImageRequest)
	require.True(t, ok, "result should be MiniMaxImageRequest")
	assert.Equal(t, "image-01", mmReq.Model)
	assert.Equal(t, "A beautiful sunset", mmReq.Prompt)
	assert.Equal(t, 2, mmReq.N)
	assert.Equal(t, "1:1", mmReq.AspectRatio)
}

func TestMiniMaxAdaptor_ConvertImageRequest_DefaultModel(t *testing.T) {
	a := &minimax.Adaptor{}
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeImagesGenerations
	info.OriginModelName = ""

	req := dto.ImageRequest{
		Model:  "",
		Prompt: "test",
	}

	result, err := a.ConvertImageRequest(context.Background(), info, req)
	require.NoError(t, err)

	mmReq, ok := result.(minimax.MiniMaxImageRequest)
	require.True(t, ok)
	assert.Equal(t, "image-01", mmReq.Model)
}

func TestMiniMaxAdaptor_ConvertImageRequest_WrongRelayMode(t *testing.T) {
	a := &minimax.Adaptor{}
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeChatCompletions

	_, err := a.ConvertImageRequest(context.Background(), info, dto.ImageRequest{})
	require.Error(t, err)
}

func TestMiniMaxAdaptor_ConvertAudioRequest(t *testing.T) {
	a := &minimax.Adaptor{}
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeAudioSpeech
	info.OriginModelName = "speech-02-turbo"

	speed := 1.0
	req := dto.AudioRequest{
		Model:          "speech-02-turbo",
		Input:          "Hello world",
		Voice:          "male-qn-qingse",
		ResponseFormat: "mp3",
		Speed:          &speed,
	}

	result, err := a.ConvertAudioRequest(context.Background(), info, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The result should be JSON bytes of MiniMaxTTSRequest.
	var ttsReq minimax.MiniMaxTTSRequest
	data, readErr := io.ReadAll(result)
	require.NoError(t, readErr)
	err = json.Unmarshal(data, &ttsReq)
	require.NoError(t, err)
	assert.Equal(t, "speech-02-turbo", ttsReq.Model)
	assert.Equal(t, "Hello world", ttsReq.Text)
	assert.Equal(t, "male-qn-qingse", ttsReq.VoiceSetting.VoiceID)
	assert.Equal(t, 1.0, ttsReq.VoiceSetting.Speed)
	assert.Equal(t, "mp3", ttsReq.OutputFormat)
}

func TestMiniMaxAdaptor_ConvertAudioRequest_WrongRelayMode(t *testing.T) {
	a := &minimax.Adaptor{}
	info := common.NewRelayInfo()
	info.RelayMode = relaymode.RelayModeChatCompletions

	_, err := a.ConvertAudioRequest(context.Background(), info, dto.AudioRequest{})
	require.Error(t, err)
}

func TestMiniMaxAdaptor_GetModelList(t *testing.T) {
	a := &minimax.Adaptor{}
	models := a.GetModelList()
	assert.NotEmpty(t, models)
}

func TestMiniMaxAdaptor_GetChannelName(t *testing.T) {
	a := &minimax.Adaptor{}
	assert.Equal(t, "minimax", a.GetChannelName())
}
