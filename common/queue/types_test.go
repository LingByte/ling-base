// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package queue

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStatus_String(t *testing.T) {
	assert.Equal(t, "pending", StatusPending.String())
	assert.Equal(t, "running", StatusRunning.String())
	assert.Equal(t, "success", StatusSuccess.String())
}

func TestTaskStatus_IsTerminal(t *testing.T) {
	assert.True(t, StatusSuccess.IsTerminal())
	assert.True(t, StatusFailed.IsTerminal())
	assert.True(t, StatusCanceled.IsTerminal())
	assert.False(t, StatusPending.IsTerminal())
	assert.False(t, StatusRunning.IsTerminal())
	assert.False(t, StatusRetry.IsTerminal())
}

func TestEncodeDecodePayload(t *testing.T) {
	type MyPayload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	raw, err := EncodePayload(MyPayload{Name: "alice", Age: 30})
	require.NoError(t, err)

	task := &Task{Payload: raw}
	v, err := DecodePayload[MyPayload](task)
	require.NoError(t, err)
	assert.Equal(t, "alice", v.Name)
	assert.Equal(t, 30, v.Age)
}

func TestEncodePayload_JSON(t *testing.T) {
	raw, err := EncodePayload(map[string]int{"a": 1})
	require.NoError(t, err)

	var m map[string]int
	err = json.Unmarshal(raw, &m)
	require.NoError(t, err)
	assert.Equal(t, 1, m["a"])
}
