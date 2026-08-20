// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package kling_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	constant "github.com/LingByte/ling-base/relay/constant"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/kling"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &kling.TaskAdaptor{}
	assert.Equal(t, "kling", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &kling.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "kling-v1")
	assert.Contains(t, models, "kling-v1-6")
	assert.Contains(t, models, "kling-v2-master")
}

func TestBuildRequestURL_Generate(t *testing.T) {
	a := &kling.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.kling.com"
	info.Action = constant.TaskActionGenerate
	info.ApiKey = "ak|sk"
	a.Init(info)

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/v1/videos/image2video")
	assert.Contains(t, url, "https://api.kling.com")
}

func TestBuildRequestURL_TextGenerate(t *testing.T) {
	a := &kling.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.kling.com"
	info.Action = constant.TaskActionTextGenerate
	info.ApiKey = "ak|sk"
	a.Init(info)

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/v1/videos/text2video")
	assert.Contains(t, url, "https://api.kling.com")
}

func TestBuildRequestURL_NewAPIRelay(t *testing.T) {
	a := &kling.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.kling.com"
	info.Action = constant.TaskActionGenerate
	info.ApiKey = "sk-test-key"
	a.Init(info)

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/kling/v1/videos/image2video")
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &kling.TaskAdaptor{}
	jsonResp := `{
		"code": 0,
		"message": "",
		"task_id": "task123",
		"data": {
			"task_id": "task123",
			"task_status": "succeed",
			"task_result": {
				"videos": [
					{"id": "v1", "url": "https://example.com/video.mp4", "duration": "5"}
				]
			},
			"final_unit_deduction": "10"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://example.com/video.mp4", info.Url)
	assert.Equal(t, "task123", info.TaskID)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &kling.TaskAdaptor{}
	jsonResp := `{
		"code": 1000,
		"message": "insufficient credit",
		"data": {
			"task_id": "task456",
			"task_status": "failed",
			"task_status_msg": "credit error"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, 1000, info.Code)
}

func TestParseTaskResult_Processing(t *testing.T) {
	a := &kling.TaskAdaptor{}
	jsonResp := `{
		"code": 0,
		"data": {
			"task_id": "task789",
			"task_status": "processing"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
}

func TestParseTaskResult_Submitted(t *testing.T) {
	a := &kling.TaskAdaptor{}
	jsonResp := `{
		"code": 0,
		"data": {
			"task_id": "task000",
			"task_status": "submitted"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSubmitted, info.Status)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &kling.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}

func TestParseTaskResult_UnknownStatus(t *testing.T) {
	a := &kling.TaskAdaptor{}
	jsonResp := `{
		"code": 0,
		"data": {
			"task_id": "task000",
			"task_status": "unknown_status"
		}
	}`

	_, err := a.ParseTaskResult([]byte(jsonResp))
	require.Error(t, err)
}

// TestCreateJWTTokenWithKey tests JWT generation indirectly via BuildRequestHeader,
// since createJWTTokenWithKey is unexported.
func TestCreateJWTTokenWithKey(t *testing.T) {
	a := &kling.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.kling.com",
			ApiKey:         "access_key|secret_key",
		},
	})

	req, err := http.NewRequest("POST", "https://api.kling.com/v1/videos/text2video", nil)
	require.NoError(t, err)

	info := testRelayInfo()
	info.ApiKey = "access_key|secret_key"

	err = a.BuildRequestHeader(context.Background(), req, info)
	require.NoError(t, err)

	auth := req.Header.Get("Authorization")
	assert.NotEmpty(t, auth)
	assert.Contains(t, auth, "Bearer ")
	assert.NotEqual(t, "Bearer ", auth) // should have a token after Bearer
}

func TestCreateJWTTokenWithKey_NewAPIRelay(t *testing.T) {
	a := &kling.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.kling.com",
			ApiKey:         "sk-test-key",
		},
	})

	req, err := http.NewRequest("POST", "https://api.kling.com/v1/videos/text2video", nil)
	require.NoError(t, err)

	info := testRelayInfo()
	info.ApiKey = "sk-test-key"

	err = a.BuildRequestHeader(context.Background(), req, info)
	require.NoError(t, err)

	auth := req.Header.Get("Authorization")
	assert.Equal(t, "Bearer sk-test-key", auth)
}
