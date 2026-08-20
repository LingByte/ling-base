// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gemini_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/gemini"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	assert.Equal(t, "gemini", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "veo-3.0-generate-001")
	assert.Contains(t, models, "veo-3.0-fast-generate-001")
	assert.Contains(t, models, "veo-3.1-generate-preview")
	assert.Contains(t, models, "veo-3.1-fast-generate-preview")
}

func TestBuildRequestURL(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://generativelanguage.googleapis.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://generativelanguage.googleapis.com"
	info.UpstreamModelName = "veo-3.0-generate-001"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "predictLongRunning")
	assert.Contains(t, url, "veo-3.0-generate-001")
	assert.Contains(t, url, "v1beta")
}

func TestParseTaskResult_InProgress(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	jsonResp := `{
		"name": "models/veo-3.0-generate-001/operations/op123",
		"done": false
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
	assert.Equal(t, "50%", info.Progress)
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	jsonResp := `{
		"name": "models/veo-3.0-generate-001/operations/op123",
		"done": true,
		"response": {
			"generateVideoResponse": {
				"generatedVideos": [
					{
						"video": {
							"uri": "https://example.com/video.mp4"
						}
					}
				]
			}
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "https://example.com/video.mp4", info.RemoteUrl)
}

func TestParseTaskResult_Error(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	jsonResp := `{
		"name": "models/veo-3.0-generate-001/operations/op123",
		"done": true,
		"error": {
			"message": "generation failed"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "generation failed", info.Reason)
	assert.Equal(t, "100%", info.Progress)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &gemini.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}
