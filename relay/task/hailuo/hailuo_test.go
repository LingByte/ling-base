// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package hailuo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/hailuo"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	assert.Equal(t, "hailuo-video", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "MiniMax-Hailuo-2.3")
	assert.Contains(t, models, "T2V-01")
	assert.Contains(t, models, "I2V-01")
	assert.NotEmpty(t, models)
}

func TestBuildRequestURL(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.minimaxi.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.minimaxi.com"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/v1/video_generation")
	assert.Contains(t, url, "https://api.minimaxi.com")
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	jsonResp := `{
		"task_id": "task123",
		"status": "Success",
		"file_id": "file456",
		"base_resp": {
			"status_code": 0,
			"status_msg": ""
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "100%", info.Progress)
}

func TestParseTaskResult_Processing(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	jsonResp := `{
		"task_id": "task123",
		"status": "Processing",
		"base_resp": {
			"status_code": 0,
			"status_msg": ""
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
	assert.Equal(t, "50%", info.Progress)
}

func TestParseTaskResult_Preparing(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	jsonResp := `{
		"task_id": "task123",
		"status": "Preparing",
		"base_resp": {
			"status_code": 0,
			"status_msg": ""
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	jsonResp := `{
		"task_id": "task123",
		"status": "Fail",
		"base_resp": {
			"status_code": 1002,
			"status_msg": "rate limited"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "rate limited", info.Reason)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}

func TestParseTaskResult_UnknownStatus(t *testing.T) {
	a := &hailuo.TaskAdaptor{}
	jsonResp := `{
		"task_id": "task123",
		"status": "Unknown",
		"base_resp": {
			"status_code": 0,
			"status_msg": ""
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
}
