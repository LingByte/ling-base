// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package suno_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/suno"
)

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &suno.TaskAdaptor{}
	assert.Equal(t, "suno", a.GetChannelName())
}

func TestValidateRequestAndSetAction_EmptyAction(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.Action = ""

	taskErr := a.ValidateRequestAndSetAction(context.Background(), info)
	require.NotNil(t, taskErr)
}

func TestValidateRequestAndSetAction_Music(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.Action = "MUSIC"

	taskErr := a.ValidateRequestAndSetAction(context.Background(), info)
	require.Nil(t, taskErr)
	assert.Equal(t, "MUSIC", info.Action)
}

func TestValidateRequestAndSetAction_Lyrics(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.Action = "LYRICS"

	taskErr := a.ValidateRequestAndSetAction(context.Background(), info)
	require.Nil(t, taskErr)
	assert.Equal(t, "LYRICS", info.Action)
}

func TestValidateRequestAndSetAction_Invalid(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.Action = "INVALID"

	taskErr := a.ValidateRequestAndSetAction(context.Background(), info)
	require.NotNil(t, taskErr)
}

func TestValidateRequestAndSetAction_LowercaseNormalizes(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.Action = "music"

	taskErr := a.ValidateRequestAndSetAction(context.Background(), info)
	require.Nil(t, taskErr)
	assert.Equal(t, "MUSIC", info.Action)
}

func TestBuildRequestBody(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()

	body, err := a.BuildRequestBody(context.Background(), info)
	require.NoError(t, err)
	assert.Nil(t, body) // library mode passthrough
}

func TestBuildRequestURL(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://api.suno.com"
	info.Action = "MUSIC"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "/suno/submit/MUSIC")
	assert.Contains(t, url, "https://api.suno.com")
}

func TestDoResponse_Success(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()
	info.PublicTaskID = "pub-task-1"

	body := `{"code": 200, "message": "", "data": "suno-task-123"}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	taskID, _, taskErr := a.DoResponse(context.Background(), resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "suno-task-123", taskID)
}

func TestDoResponse_ErrorCode(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()

	body := `{"code": 400, "message": "bad request", "data": ""}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	_, _, taskErr := a.DoResponse(context.Background(), resp, info)
	require.NotNil(t, taskErr)
}

func TestDoResponse_InvalidJSON(t *testing.T) {
	a := &suno.TaskAdaptor{}
	info := testRelayInfo()

	body := `invalid json`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	_, _, taskErr := a.DoResponse(context.Background(), resp, info)
	require.NotNil(t, taskErr)
}

func TestParseTaskResult_NotApplicable(t *testing.T) {
	a := &suno.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte(`{}`))
	require.Error(t, err)
}
