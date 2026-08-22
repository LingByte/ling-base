//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package inprocess

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
)

func TestReplyAccumulatorConsumeDeltaFullAndError(t *testing.T) {
	t.Parallel()

	var acc replyAccumulator
	acc.consume(nil)
	acc.consume(&event.Event{})
	acc.consume(&event.Event{
		Response: &compat.Response{
			Object: compat.ObjectTypeChatCompletionChunk,
			Choices: []compat.Choice{{
				Delta: compat.Message{Content: "hello "},
			}},
		},
	})
	acc.consume(&event.Event{
		Response: &compat.Response{
			Object: compat.ObjectTypeChatCompletionChunk,
			Choices: []compat.Choice{{
				Delta: compat.Message{Content: "world"},
			}},
		},
	})
	require.Equal(t, "hello world", acc.text)

	acc.consume(&event.Event{
		Response: &compat.Response{
			Object: compat.ObjectTypeChatCompletion,
			Choices: []compat.Choice{{
				Message: compat.NewAssistantMessage("full"),
			}},
		},
	})
	require.Equal(t, "full", acc.text)

	acc.consume(&event.Event{
		Response: &compat.Response{
			Error: &compat.ResponseError{
				Message: "stream failed",
			},
		},
	})
	require.ErrorContains(t, acc.err, "stream failed")
}
