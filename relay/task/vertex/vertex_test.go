// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package vertex_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/task/vertex"
	"github.com/LingByte/ling-base/relay/task/taskmodel"
)

// validCredentialsJSON is a minimal JSON for vertex2.Credentials used by
// BuildRequestURL (only ProjectID is needed for URL construction).
const validCredentialsJSON = `{
	"project_id": "test-project",
	"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDdQ\n-----END PRIVATE KEY-----\n",
	"client_email": "test@test-project.iam.gserviceaccount.com"
}`

func testRelayInfo() *common.RelayInfo {
	return &common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{},
	}
}

func TestGetChannelName(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	assert.Equal(t, "vertex", a.GetChannelName())
}

func TestGetModelList(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	models := a.GetModelList()
	assert.Contains(t, models, "veo-3.0-generate-001")
	assert.Contains(t, models, "veo-3.0-fast-generate-001")
	assert.Contains(t, models, "veo-3.1-generate-preview")
	assert.Contains(t, models, "veo-3.1-fast-generate-preview")
}

func TestBuildRequestURL(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	a.Init(&common.RelayInfo{
		ChannelMeta: &common.ChannelMeta{
			ChannelBaseUrl: "https://us-central1-aiplatform.googleapis.com",
			ApiKey:         validCredentialsJSON,
		},
	})
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://us-central1-aiplatform.googleapis.com"
	info.ApiKey = validCredentialsJSON
	info.UpstreamModelName = "veo-3.0-generate-001"

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Contains(t, url, "predictLongRunning")
	assert.Contains(t, url, "veo-3.0-generate-001")
	assert.Contains(t, url, "test-project")
}

func TestBuildRequestURL_InvalidCredentials(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	info := testRelayInfo()
	info.ChannelBaseUrl = "https://us-central1-aiplatform.googleapis.com"
	info.ApiKey = "invalid-json"
	info.UpstreamModelName = "veo-3.0-generate-001"

	_, err := a.BuildRequestURL(info)
	require.Error(t, err)
}

func TestParseTaskResult_InProgress(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	jsonResp := `{
		"name": "projects/test-project/locations/us-central1/models/veo-3.0-generate-001/operations/op123",
		"done": false
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusInProgress, info.Status)
	assert.Equal(t, "50%", info.Progress)
}

func TestParseTaskResult_Success_WithVideoBytes(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	jsonResp := `{
		"name": "projects/test-project/locations/us-central1/models/veo-3.0-generate-001/operations/op123",
		"done": true,
		"response": {
			"videos": [
				{
					"mimeType": "video/mp4",
					"bytesBase64Encoded": "SGVsbG8gV29ybGQ="
				}
			]
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Equal(t, "100%", info.Progress)
	assert.Contains(t, info.Url, "data:video/mp4;base64,")
}

func TestParseTaskResult_Success_WithBytesBase64Encoded(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	jsonResp := `{
		"name": "projects/test-project/locations/us-central1/models/veo-3.0-generate-001/operations/op123",
		"done": true,
		"response": {
			"bytesBase64Encoded": "SGVsbG8gV29ybGQ=",
			"encoding": "mp4"
		}
	}`

	info, err := a.ParseTaskResult([]byte(jsonResp))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, taskmodel.TaskStatusSuccess, info.Status)
	assert.Contains(t, info.Url, "data:video/mp4;base64,")
}

func TestParseTaskResult_Error(t *testing.T) {
	a := &vertex.TaskAdaptor{}
	jsonResp := `{
		"name": "projects/test-project/locations/us-central1/models/veo-3.0-generate-001/operations/op123",
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
	a := &vertex.TaskAdaptor{}
	_, err := a.ParseTaskResult([]byte("invalid json"))
	require.Error(t, err)
}
