// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package jimeng_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/jimeng"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	assert.Equal(t, "jimeng", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "jimeng_vgfm_t2v_l20")
}

func TestBuildRequestURL(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://visual.volcengineapi.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://visual.volcengineapi.com"
	info.ApiKey = "ak|sk"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "Action=CVSync2AsyncSubmitTask")
	assert.Contains(t, url, "Version=2022-08-31")
	assert.Contains(t, url, "https://visual.volcengineapi.com")
}

func TestBuildRequestURL_NewAPIRelay(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://visual.volcengineapi.com"
	info.ApiKey = "sk-test-key"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/jimeng/?Action=CVSync2AsyncSubmitTask")
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	jsonResp := `{
		"code": 10000,
		"data": {
			"status": "done",
			"video_url": "https://example.com/video.mp4"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "https://example.com/video.mp4", info.Url)
}

func TestParseTaskResult_Queued(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	jsonResp := `{
		"code": 10000,
		"data": {
			"status": "in_queue"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusQueued, info.Status)
	assert.Equal(t, "10%", info.Progress)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	jsonResp := `{
		"code": 50000,
		"message": "internal error",
		"data": {
			"status": ""
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, 50000, info.Code)
	assert.Equal(t, "internal error", info.Reason)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}

// TestSignRequest tests the unexported signRequest method indirectly via
// BuildRequestHeader, since signRequest is unexported.
func TestSignRequest_Indirect(t *testing.T) {
	a := &jimeng.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://visual.volcengineapi.com",
			ApiKey:         "test_access_key|test_secret_key",
		},
	})

	info := testRelayInfo()
	info.ChannelBaseUrl = "https://visual.volcengineapi.com"
	info.ApiKey = "test_access_key|test_secret_key"

	req, err := http.NewRequest("POST", "https://visual.volcengineapi.com/?Action=CVSync2AsyncSubmitTask&Version=2022-08-31", nil)
	require.NoError(t, err)

	err = a.BuildRequestHeader(nil, req, info)
	require.NoError(t, err)

	auth := req.Header.Get("Authorization")
	assert.NotEmpty(t, auth)
	assert.Contains(t, auth, "HMAC-SHA256")
	assert.NotEmpty(t, req.Header.Get("X-Date"))
	assert.NotEmpty(t, req.Header.Get("X-Content-Sha256"))
}
