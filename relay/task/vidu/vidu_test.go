// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package vidu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	constant "github.com/LingByte/ling-base/relay/constant"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/vidu"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	assert.Equal(t, "vidu", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "viduq2")
	assert.Contains(t, models, "vidu2.0")
	assert.Contains(t, models, "viduq1")
	assert.Contains(t, models, "vidu1.5")
}

func TestBuildRequestURL_TextGenerate(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.vidu.cn",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.vidu.cn"
	info.Action = constant.TaskActionTextGenerate

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/ent/v2/text2video")
}

func TestBuildRequestURL_Generate(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.vidu.cn"
	info.Action = constant.TaskActionGenerate

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/ent/v2/img2video")
}

func TestBuildRequestURL_FirstTailGenerate(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.vidu.cn"
	info.Action = constant.TaskActionFirstTailGenerate

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/ent/v2/start-end2video")
}

func TestBuildRequestURL_ReferenceGenerate(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.vidu.cn"
	info.Action = constant.TaskActionReferenceGenerate

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/ent/v2/reference2video")
}

func TestParseTaskResult_Success(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	jsonResp := `{
		"state": "success",
		"err_code": "",
		"creations": [
			{"id": "c1", "url": "https://example.com/video.mp4", "cover_url": "https://example.com/cover.jpg"}
		]
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "https://example.com/video.mp4", info.Url)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	jsonResp := `{
		"state": "failed",
		"err_code": "GENERATION_FAILED"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "GENERATION_FAILED", info.Reason)
}

func TestParseTaskResult_Processing(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	jsonResp := `{
		"state": "processing"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
}

func TestParseTaskResult_Created(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	jsonResp := `{
		"state": "created"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSubmitted, info.Status)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}

func TestParseTaskResult_UnknownState(t *testing.T) {
	a := &vidu.TaskAdaptor{}
	jsonResp := `{
		"state": "unknown_state"
	}`

	_, err := a.ParseTaskResult([]byte(jsonResp))
	require.Error(t, err)
}
