//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package trpcagent

import (
	"testing"

	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/stretchr/testify/require"
)

func TestMessageCollectorMergesAssistantContentDeltas(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				Role:    compat.RoleAssistant,
				Content: "hello ",
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "world"},
			FinishReason: stringPtr("stop"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Equal(t, compat.NewUserMessage("input"), messages[0])
	require.Equal(t, compat.NewAssistantMessage("hello world"), messages[1])
}

func TestMessageCollectorFlushesOpenStreamBeforeFullMessage(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "partial"},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Message: compat.NewAssistantMessage("final"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewUserMessage("input"), messages[0])
	require.Equal(t, compat.NewAssistantMessage("partial"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("final"), messages[2])
}

func TestMessageCollectorFlushesDoneEventWithoutFinishReason(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "done flush"},
		}},
	}))
	require.Len(t, collector.messagesList(), 1)
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Done: true,
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Equal(t, compat.NewAssistantMessage("done flush"), messages[1])
}

func TestMessageCollectorFlushAllKeepsOpenStreamsInStartOrder(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-a", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "first"},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-b", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "second"},
		}},
	}))
	collector.flushAll()
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewAssistantMessage("first"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("second"), messages[2])
}

func TestMessageCollectorMergesReasoningAndContentParts(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	firstText := "first"
	secondText := "second"
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ReasoningContent: "think",
				ContentParts: []compat.ContentPart{{
					Type: compat.ContentTypeText,
					Text: &firstText,
				}},
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ReasoningContent:   "ing",
				ReasoningSignature: "sig-1",
				ContentParts: []compat.ContentPart{{
					Type: compat.ContentTypeText,
					Text: &secondText,
				}},
			},
			FinishReason: stringPtr("stop"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Equal(t, compat.RoleAssistant, messages[1].Role)
	require.Equal(t, "thinking", messages[1].ReasoningContent)
	require.Equal(t, "sig-1", messages[1].ReasoningSignature)
	require.Len(t, messages[1].ContentParts, 2)
	require.Equal(t, "first", *messages[1].ContentParts[0].Text)
	require.Equal(t, "second", *messages[1].ContentParts[1].Text)
}

func TestMessageCollectorKeepsChoicesSeparate(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{
			{
				Index:        0,
				Delta:        compat.Message{Content: "choice zero"},
				FinishReason: stringPtr("stop"),
			},
			{
				Index:        1,
				Delta:        compat.Message{Content: "choice one"},
				FinishReason: stringPtr("stop"),
			},
		},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewAssistantMessage("choice zero"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("choice one"), messages[2])
}

func TestMessageCollectorKeepsBranchesSeparate(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "branch-a "},
		}},
	}, event.WithBranch("branch-a")))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "branch-b"},
			FinishReason: stringPtr("stop"),
		}},
	}, event.WithBranch("branch-b")))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "done"},
			FinishReason: stringPtr("stop"),
		}},
	}, event.WithBranch("branch-a")))
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewAssistantMessage("branch-b"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("branch-a done"), messages[2])
}

func TestMessageCollectorKeepsSameInvocationStreamsSeparateByResponseID(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		ID: "rsp-a",
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "alpha "},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		ID: "rsp-b",
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "beta"},
			FinishReason: stringPtr("stop"),
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		ID: "rsp-a",
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "omega"},
			FinishReason: stringPtr("stop"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewAssistantMessage("beta"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("alpha omega"), messages[2])
}

func TestMessageCollectorKeepsParallelChildStreamsSeparateByTriggerID(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	evtA := event.NewResponseEvent("child", "worker", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "alpha "},
		}},
	})
	evtA.ParentInvocationID = "parent"
	evtA.ParentMetadata = &event.ParentInvocationMetadata{TriggerType: event.TriggerTypeToolCall, TriggerID: "call-a"}
	collector.addEvent(evtA)
	evtB := event.NewResponseEvent("child", "worker", &compat.Response{
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "beta"},
			FinishReason: stringPtr("stop"),
		}},
	})
	evtB.ParentInvocationID = "parent"
	evtB.ParentMetadata = &event.ParentInvocationMetadata{TriggerType: event.TriggerTypeToolCall, TriggerID: "call-b"}
	collector.addEvent(evtB)
	evtA = event.NewResponseEvent("child", "worker", &compat.Response{
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "omega"},
			FinishReason: stringPtr("stop"),
		}},
	})
	evtA.ParentInvocationID = "parent"
	evtA.ParentMetadata = &event.ParentInvocationMetadata{TriggerType: event.TriggerTypeToolCall, TriggerID: "call-a"}
	collector.addEvent(evtA)
	messages := collector.messagesList()
	require.Len(t, messages, 3)
	require.Equal(t, compat.NewAssistantMessage("beta"), messages[1])
	require.Equal(t, compat.NewAssistantMessage("alpha omega"), messages[2])
}

func TestMessageCollectorGroupsChunksByResponseIDWhenInvocationMissing(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("", "", &compat.Response{
		ID: "rsp-1",
		Choices: []compat.Choice{{
			Delta: compat.Message{Content: "he"},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("", "", &compat.Response{
		ID: "rsp-1",
		Choices: []compat.Choice{{
			Delta:        compat.Message{Content: "llo"},
			FinishReason: stringPtr("stop"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Equal(t, compat.NewAssistantMessage("hello"), messages[1])
}

func TestMessageCollectorMergesToolCallDeltasByIndex(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	index := 0
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					Type:  "function",
					ID:    "call-1",
					Index: &index,
					Function: compat.FunctionDefinitionParam{
						Name:      "lookup",
						Arguments: []byte(`{"a"`),
					},
				}},
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					Index: &index,
					Function: compat.FunctionDefinitionParam{
						Arguments: []byte(`:1}`),
					},
				}},
			},
			FinishReason: stringPtr("tool_calls"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Len(t, messages[1].ToolCalls, 1)
	require.Equal(t, "call-1", messages[1].ToolCalls[0].ID)
	require.Equal(t, "lookup", messages[1].ToolCalls[0].Function.Name)
	require.Equal(t, []byte(`{"a":1}`), messages[1].ToolCalls[0].Function.Arguments)
}

func TestMessageCollectorMergesToolCallDeltasByID(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					ID: "call-1",
					Function: compat.FunctionDefinitionParam{
						Arguments: []byte(`{"q":12`),
					},
				}},
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					ID: "call-1",
					Function: compat.FunctionDefinitionParam{
						Arguments: []byte(`3}`),
					},
				}},
			},
			FinishReason: stringPtr("tool_calls"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Len(t, messages[1].ToolCalls, 1)
	require.Equal(t, "call-1", messages[1].ToolCalls[0].ID)
	require.Equal(t, []byte(`{"q":123}`), messages[1].ToolCalls[0].Function.Arguments)
}

func TestMessageCollectorMergesToolCallDeltasWhenLaterChunksOmitID(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					ID: "call-1",
					Function: compat.FunctionDefinitionParam{
						Name:      "lookup",
						Arguments: []byte(`{"q":12`),
					},
				}},
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{{
					Function: compat.FunctionDefinitionParam{
						Arguments: []byte(`3}`),
					},
				}},
			},
			FinishReason: stringPtr("tool_calls"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Len(t, messages[1].ToolCalls, 1)
	require.Equal(t, "call-1", messages[1].ToolCalls[0].ID)
	require.Equal(t, "lookup", messages[1].ToolCalls[0].Function.Name)
	require.Equal(t, []byte(`{"q":123}`), messages[1].ToolCalls[0].Function.Arguments)
}

func TestMessageCollectorKeepsToolCallPositionKeysSeparate(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{
					{
						Function: compat.FunctionDefinitionParam{
							Name:      "first",
							Arguments: []byte(`{"a"`),
						},
					},
					{
						Function: compat.FunctionDefinitionParam{
							Name:      "second",
							Arguments: []byte(`{"b"`),
						},
					},
				},
			},
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Delta: compat.Message{
				ToolCalls: []compat.ToolCall{
					{
						Function: compat.FunctionDefinitionParam{
							Arguments: []byte(`:1}`),
						},
					},
					{
						Function: compat.FunctionDefinitionParam{
							Arguments: []byte(`:2}`),
						},
					},
				},
			},
			FinishReason: stringPtr("tool_calls"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Len(t, messages[1].ToolCalls, 2)
	require.Equal(t, "first", messages[1].ToolCalls[0].Function.Name)
	require.Equal(t, []byte(`{"a":1}`), messages[1].ToolCalls[0].Function.Arguments)
	require.Equal(t, "second", messages[1].ToolCalls[1].Function.Name)
	require.Equal(t, []byte(`{"b":2}`), messages[1].ToolCalls[1].Function.Arguments)
}

func TestMessageCollectorIgnoresNilAndEmptyEvents(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(nil)
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", nil))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{}))
	require.Equal(t, []compat.Message{compat.NewUserMessage("input")}, collector.messagesList())
}

func TestMessageCollectorDeduplicatesAdjacentMessages(t *testing.T) {
	collector := newMessageCollector(compat.NewUserMessage("input"))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Message: compat.NewAssistantMessage("same"),
		}},
	}))
	collector.addEvent(event.NewResponseEvent("inv-1", "writer", &compat.Response{
		Choices: []compat.Choice{{
			Message: compat.NewAssistantMessage("same"),
		}},
	}))
	messages := collector.messagesList()
	require.Len(t, messages, 2)
	require.Equal(t, compat.NewAssistantMessage("same"), messages[1])
}
