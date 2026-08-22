//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestStreamModeFilter_Disabled_AllowsEvents(t *testing.T) {
	f := NewStreamModeFilter(false, nil)

	require.False(t, f.Allows(nil))
	require.True(t, f.Allows(&event.Event{}))
	require.True(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletion)))
}

func TestStreamModeFilter_EmptyModes_AllowsOnlyErrors(t *testing.T) {
	f := NewStreamModeFilter(true, nil)

	require.False(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletion)))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphExecution)))
	require.True(t, f.Allows(eventWithError()))
}

func TestStreamModeFilter_Messages(t *testing.T) {
	f := NewStreamModeFilter(true, []agent.StreamMode{agent.StreamModeMessages})

	require.True(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletionChunk)))
	require.True(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletion)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphExecution)))
}

func TestStreamModeFilter_Updates(t *testing.T) {
	f := NewStreamModeFilter(true, []agent.StreamMode{agent.StreamModeUpdates})

	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphExecution)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphChannelUpdate)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphStateUpdate)))
	require.True(t, f.Allows(eventWithObject(compat.ObjectTypeStateUpdate)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletion)))
}

func TestStreamModeFilter_Checkpoints(t *testing.T) {
	f := NewStreamModeFilter(
		true,
		[]agent.StreamMode{agent.StreamModeCheckpoints},
	)

	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphCheckpoint)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphCheckpointCreated)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphCheckpointCommitted)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphCheckpointInterrupt)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphNodeStart)))
}

func TestStreamModeFilter_Tasks(t *testing.T) {
	f := NewStreamModeFilter(true, []agent.StreamMode{agent.StreamModeTasks})

	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphBarrier)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeBarrier)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeExecution)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeStart)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeComplete)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeError)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphPregelStep)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphPregelPlanning)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphPregelExecution)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphPregelUpdate)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphStateUpdate)))
}

func TestStreamModeFilter_Custom(t *testing.T) {
	f := NewStreamModeFilter(true, []agent.StreamMode{agent.StreamModeCustom})

	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeCustom)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphNodeStart)))
}

func TestStreamModeFilter_Debug(t *testing.T) {
	f := NewStreamModeFilter(true, []agent.StreamMode{agent.StreamModeDebug})

	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphCheckpointCommitted)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphPregelStep)))
	require.True(t, f.Allows(eventWithObject(ObjectTypeGraphNodeStart)))
	require.True(t, f.Allows(eventWithError()))
	require.False(t, f.Allows(eventWithObject(ObjectTypeGraphStateUpdate)))
	require.False(t, f.Allows(eventWithObject(compat.ObjectTypeChatCompletion)))
}

func eventWithObject(object string) *event.Event {
	return &event.Event{Response: &compat.Response{Object: object}}
}

func eventWithError() *event.Event {
	return &event.Event{
		Response: &compat.Response{
			Object: compat.ObjectTypeError,
			Error:  &compat.ResponseError{Message: "boom"},
		},
	}
}
