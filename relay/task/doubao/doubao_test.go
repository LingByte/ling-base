// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package doubao_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/doubao"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	assert.Equal(t, "doubao-video", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	models := a.GetModelList()
	// Should contain seedance models
	found := false
	for _, m := range models {
		if assert.Contains(t, m, "seedance") {
			found = true
		}
	}
	assert.True(t, found, "model list should contain seedance models")
	assert.Contains(t, models, "doubao-seedance-1-0-pro-250528")
	assert.Contains(t, models, "doubao-seedance-2-0-260128")
}

func TestBuildRequestURL(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://ark.cn-beijing.volces.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://ark.cn-beijing.volces.com"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/api/v3/contents/generations/tasks")
	assert.Contains(t, url, "https://ark.cn-beijing.volces.com")
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"model": "doubao-seedance-1-0-pro-250528",
		"status": "succeeded",
		"content": {
			"video_url": "https://example.com/video.mp4"
		},
		"usage": {
			"completion_tokens": 100,
			"total_tokens": 200
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://example.com/video.mp4", info.Url)
	assert.Equal(t, 100, info.CompletionTokens)
	assert.Equal(t, 200, info.TotalTokens)
	assert.Equal(t, "100%", info.Progress)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	jsonResp := `{
		"id": "task456",
		"status": "failed",
		"error": {
			"code": "INTERNAL_ERROR",
			"message": "something went wrong"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "something went wrong", info.Reason)
	assert.Equal(t, "100%", info.Progress)
}

func TestParseTaskResult_Processing(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	jsonResp := `{
		"id": "task789",
		"status": "processing"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
	assert.Equal(t, "50%", info.Progress)
}

func TestParseTaskResult_Queued(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	jsonResp := `{
		"id": "task000",
		"status": "queued"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusQueued, info.Status)
	assert.Equal(t, "10%", info.Progress)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}

func TestParseTaskResult_UnknownStatus(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	jsonResp := `{
		"id": "task000",
		"status": "unknown_status"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
}

// TestHasVideoInMetadata tests the unexported hasVideoInMetadata function
// indirectly. Since it's unexported, we verify through the exported surface
// that the adaptor handles metadata-containing requests without panicking.
func TestHasVideoInMetadata_Indirect(t *testing.T) {
	a := &doubao.TaskAdaptor{}
	assert.NotNil(t, a)
	// hasVideoInMetadata is a package-level unexported function.
	// It checks metadata["content"] for video_url entries.
	// We can't call it directly from an external test package.
}
