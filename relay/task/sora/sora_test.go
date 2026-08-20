// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sora_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	constant "github.com/LingByte/ling-base/relay/constant"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/sora"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &sora.TaskAdaptor{}
	assert.Equal(t, "sora", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &sora.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "sora-2")
	assert.Contains(t, models, "sora-2-pro")
}

func TestBuildRequestBody(t *testing.T) {
	a := &sora.TaskAdaptor{}
	info := testRelayInfo()

	body, err := a.BuildRequestBody(context.Background(), info)
	require.NoError(t, err)
	assert.Nil(t, body) // library mode passthrough
}

func TestBuildRequestURL_Generate(t *testing.T) {
	a := &sora.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.openai.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.openai.com"
	info.Action = constant.TaskActionGenerate

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1/videos", url)
}

func TestBuildRequestURL_Remix(t *testing.T) {
	a := &sora.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://api.openai.com",
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.openai.com"
	info.Action = constant.TaskActionRemix
	info.OriginTaskID = "task-abc"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/v1/videos/task-abc/remix")
}

func TestParseTaskResult_Queued(t *testing.T) {
	a := &sora.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"status": "queued",
		"progress": 0
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusQueued, info.Status)
}

func TestParseTaskResult_Processing(t *testing.T) {
	a := &sora.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"status": "processing",
		"progress": 50
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
	assert.Equal(t, "50%", info.Progress)
}

func TestParseTaskResult_Completed(t *testing.T) {
	a := &sora.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"status": "completed",
		"progress": 100
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
}

func TestParseTaskResult_Failed(t *testing.T) {
	a := &sora.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"status": "failed",
		"error": {
			"message": "generation failed",
			"code": "ERR"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "generation failed", info.Reason)
}

func TestParseTaskResult_Cancelled(t *testing.T) {
	a := &sora.TaskAdaptor{}
	jsonResp := `{
		"id": "task123",
		"status": "cancelled"
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusFailure, info.Status)
	assert.Equal(t, "task failed", info.Reason)
}

func TestParseTaskResult_InvalidJSON(t *testing.T) {
	a := &sora.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}
