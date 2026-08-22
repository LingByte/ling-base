//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package plugin_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/plugin"
	"github.com/LingByte/ling-base/agentkit/tool"
)

type testPlugin struct {
	name string
	reg  func(r *plugin.Registry)
}

func (p *testPlugin) Name() string { return p.name }

func (p *testPlugin) Register(r *plugin.Registry) {
	if p.reg != nil {
		p.reg(r)
	}
}

type closerPlugin struct {
	name       string
	closedWith context.Context
	closeOrder *[]string
	closeErr   error
}

func (p *closerPlugin) Name() string { return p.name }

func (p *closerPlugin) Register(r *plugin.Registry) {}

func (p *closerPlugin) Close(ctx context.Context) error {
	p.closedWith = ctx
	if p.closeOrder != nil {
		*p.closeOrder = append(*p.closeOrder, p.name)
	}
	return p.closeErr
}

func TestNewManager_DuplicateName(t *testing.T) {
	p1 := &testPlugin{name: "p"}
	p2 := &testPlugin{name: "p"}
	_, err := plugin.NewManager(p1, p2)
	require.Error(t, err)
}

func TestNewManager_NilPlugin(t *testing.T) {
	_, err := plugin.NewManager(nil)
	require.Error(t, err)
}

func TestNewManager_EmptyName(t *testing.T) {
	p := &testPlugin{name: ""}
	_, err := plugin.NewManager(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name")
}

func TestMustNewManager_PanicsOnError(t *testing.T) {
	p := &testPlugin{name: ""}
	require.Panics(t, func() {
		_ = plugin.MustNewManager(p)
	})
}

func TestWithPlugins_AppendsRunPluginManager(t *testing.T) {
	var opts agent.RunOptions
	called := false
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.BeforeAgent(func(ctx context.Context, args *agent.BeforeAgentArgs) (*agent.BeforeAgentResult, error) {
				called = true
				return nil, nil
			})
		},
	}
	plugin.WithPlugins(p)(&opts)
	require.Len(t, opts.Plugins, 1)
	callbacks := opts.Plugins[0].AgentCallbacks()
	require.NotNil(t, callbacks)
	_, err := callbacks.RunBeforeAgent(context.Background(), &agent.BeforeAgentArgs{})
	require.NoError(t, err)
	require.True(t, called)
}

func TestWithPlugins_InvalidPluginDoesNotAppendRunPluginManager(t *testing.T) {
	var opts agent.RunOptions
	require.NotPanics(t, func() {
		plugin.WithPlugins(nil)(&opts)
	})
	require.Empty(t, opts.Plugins)
}

func TestManager_CallbackSetsNilWhenEmpty(t *testing.T) {
	m, err := plugin.NewManager()
	require.NoError(t, err)
	require.Nil(t, m.AgentCallbacks())
	require.Nil(t, m.ModelCallbacks())
	require.Nil(t, m.ToolCallbacks())

	e := &event.Event{}
	out, err := m.OnEvent(context.Background(), &agent.Invocation{}, e)
	require.NoError(t, err)
	require.Same(t, e, out)
	require.NoError(t, m.AfterRun(context.Background(), &plugin.AfterRunArgs{}))
	require.NoError(t, m.Close(context.Background()))
	require.NoError(t, m.Close(nil))
}

func TestManager_NilReceiver_IsSafe(t *testing.T) {
	var m *plugin.Manager
	require.Nil(t, m.AgentCallbacks())
	require.Nil(t, m.ModelCallbacks())
	require.Nil(t, m.ToolCallbacks())

	out, err := m.OnEvent(
		context.Background(),
		&agent.Invocation{},
		nil,
	)
	require.NoError(t, err)
	require.Nil(t, out)
	require.NoError(t, m.AfterRun(context.Background(), nil))
	require.NoError(t, m.Close(nil))
}

func TestManager_Close_ReverseOrderAndJoinErrors(t *testing.T) {
	const (
		errCloseP2 = "close err p2"
		errCloseP3 = "close err p3"
	)
	var closeOrder []string
	p1 := &closerPlugin{name: "p1", closeOrder: &closeOrder}
	p2Err := errors.New(errCloseP2)
	p2 := &closerPlugin{
		name:       "p2",
		closeOrder: &closeOrder,
		closeErr:   p2Err,
	}
	p3Err := errors.New(errCloseP3)
	p3 := &closerPlugin{
		name:       "p3",
		closeOrder: &closeOrder,
		closeErr:   p3Err,
	}

	m := plugin.MustNewManager(p1, p2, p3)
	err := m.Close(nil)
	require.Error(t, err)
	require.ErrorIs(t, err, p2Err)
	require.ErrorIs(t, err, p3Err)
	require.Contains(t, err.Error(), "plugin")
	require.Contains(t, err.Error(), "p2")
	require.Contains(t, err.Error(), "p3")

	require.Equal(t, []string{"p3", "p2", "p1"}, closeOrder)
	require.NotNil(t, p3.closedWith)
}

func TestManager_Close_SkipsNonCloser(t *testing.T) {
	var closeOrder []string
	p1 := &closerPlugin{name: "p1", closeOrder: &closeOrder}
	p2 := &testPlugin{name: "p2"}
	p3 := &closerPlugin{name: "p3", closeOrder: &closeOrder}

	m := plugin.MustNewManager(p1, p2, p3)
	err := m.Close(nil)
	require.NoError(t, err)
	require.Equal(t, []string{"p3", "p1"}, closeOrder)
}

func TestManager_ModelCallbacks_Order(t *testing.T) {
	var calls []string
	p1 := &testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.BeforeModel(func(
				ctx context.Context,
				args *compat.BeforeModelArgs,
			) (*compat.BeforeModelResult, error) {
				calls = append(calls, "p1")
				return nil, nil
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.BeforeModel(func(
				ctx context.Context,
				args *compat.BeforeModelArgs,
			) (*compat.BeforeModelResult, error) {
				calls = append(calls, "p2")
				return nil, nil
			})
		},
	}

	m := plugin.MustNewManager(p1, p2)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: &compat.Request{}},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p2"}, calls)
}

func TestManager_ModelCallbacks_EarlyExit(t *testing.T) {
	var calls []string
	p1 := &testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.BeforeModel(func(
				ctx context.Context,
				args *compat.BeforeModelArgs,
			) (*compat.BeforeModelResult, error) {
				calls = append(calls, "p1")
				return &compat.BeforeModelResult{
					CustomResponse: &compat.Response{Done: true},
				}, nil
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.BeforeModel(func(
				ctx context.Context,
				args *compat.BeforeModelArgs,
			) (*compat.BeforeModelResult, error) {
				calls = append(calls, "p2")
				return nil, nil
			})
		},
	}

	m := plugin.MustNewManager(p1, p2)
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)

	result, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: &compat.Request{}},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.CustomResponse)
	require.Equal(t, []string{"p1"}, calls)
}

func TestManager_AfterToolMessagesCanonicalizesReplacements(t *testing.T) {
	original := []compat.Message{
		compat.NewToolMessage("call-1", "search", "raw 1"),
		compat.NewToolMessage("call-2", "search", "raw 2"),
	}
	finishReason := "tool_calls"
	toolEvent := event.NewResponseEvent("inv", "agent", &compat.Response{
		Object: compat.ObjectTypeToolResponse,
		Choices: []compat.Choice{
			{Index: 10, Message: original[0], FinishReason: &finishReason},
			{Index: 11, Message: original[1], FinishReason: &finishReason},
		},
	})
	var secondHookMessages []compat.Message
	var secondHookChoices []compat.Choice
	m := plugin.MustNewManager(
		&testPlugin{
			name: "p1",
			reg: func(r *plugin.Registry) {
				r.AfterToolMessages(func(
					context.Context,
					*plugin.AfterToolMessagesArgs,
				) (*plugin.AfterToolMessagesResult, error) {
					return &plugin.AfterToolMessagesResult{
						ToolResultMessages: []compat.Message{
							compat.NewToolMessage("call-2", "search", "summary 2"),
							compat.NewToolMessage("call-1", "search", "summary 1"),
						},
					}, nil
				})
			},
		},
		&testPlugin{
			name: "p2",
			reg: func(r *plugin.Registry) {
				r.AfterToolMessages(func(
					_ context.Context,
					args *plugin.AfterToolMessagesArgs,
				) (*plugin.AfterToolMessagesResult, error) {
					secondHookMessages = append([]compat.Message(nil), args.ToolResultMessages...)
					secondHookChoices = append([]compat.Choice(nil), args.ToolResultEvent.Response.Choices...)
					return nil, nil
				})
			},
		},
	)

	args := &plugin.AfterToolMessagesArgs{
		ToolResultEvent:    toolEvent,
		Messages:           append([]compat.Message{compat.NewUserMessage("query")}, original...),
		ToolResultMessages: original,
	}
	result, err := m.AfterToolMessages(context.Background(), args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.ToolResultMessages, 2)
	require.Equal(t, "call-1", result.ToolResultMessages[0].ToolID)
	require.Equal(t, "summary 1", result.ToolResultMessages[0].Content)
	require.Equal(t, "call-2", result.ToolResultMessages[1].ToolID)
	require.Equal(t, "summary 2", result.ToolResultMessages[1].Content)
	require.Equal(t, result.ToolResultMessages, args.ToolResultMessages)
	require.Equal(t, result.ToolResultMessages, secondHookMessages)
	require.Equal(t, "summary 1", secondHookChoices[0].Message.Content)
	require.Equal(t, "summary 2", secondHookChoices[1].Message.Content)
	require.Equal(t, 10, toolEvent.Response.Choices[0].Index)
	require.Same(t, &finishReason, toolEvent.Response.Choices[0].FinishReason)
	require.Equal(t, "summary 1", toolEvent.Response.Choices[0].Message.Content)
	require.Equal(t, 11, toolEvent.Response.Choices[1].Index)
	require.Same(t, &finishReason, toolEvent.Response.Choices[1].FinishReason)
	require.Equal(t, "summary 2", toolEvent.Response.Choices[1].Message.Content)
}

func TestManager_AfterToolMessagesRejectsInvalidReplacement(t *testing.T) {
	m := plugin.MustNewManager(&testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.AfterToolMessages(func(
				context.Context,
				*plugin.AfterToolMessagesArgs,
			) (*plugin.AfterToolMessagesResult, error) {
				return &plugin.AfterToolMessagesResult{
					ToolResultMessages: []compat.Message{compat.NewAssistantMessage("bad")},
				}, nil
			})
		},
	})
	_, err := m.AfterToolMessages(context.Background(), &plugin.AfterToolMessagesArgs{
		ToolResultMessages: []compat.Message{
			compat.NewToolMessage("call-1", "search", "raw"),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing tool id")
}

func TestManager_AfterToolMessagesEdgeCases(t *testing.T) {
	var nilRegistry *plugin.Registry
	nilRegistry.AfterToolMessages(nil)
	(&plugin.Registry{}).AfterToolMessages(nil)

	var nilManager *plugin.Manager
	res, err := nilManager.AfterToolMessages(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, res)

	m := plugin.MustNewManager(&testPlugin{
		name: "empty",
		reg: func(r *plugin.Registry) {
			r.AfterToolMessages(func(
				context.Context,
				*plugin.AfterToolMessagesArgs,
			) (*plugin.AfterToolMessagesResult, error) {
				return &plugin.AfterToolMessagesResult{}, nil
			})
		},
	})
	res, err = m.AfterToolMessages(context.Background(), &plugin.AfterToolMessagesArgs{})
	require.NoError(t, err)
	require.Nil(t, res)

	hookErr := errors.New("hook failed")
	m = plugin.MustNewManager(&testPlugin{
		name: "hook-error",
		reg: func(r *plugin.Registry) {
			r.AfterToolMessages(func(
				context.Context,
				*plugin.AfterToolMessagesArgs,
			) (*plugin.AfterToolMessagesResult, error) {
				return &plugin.AfterToolMessagesResult{
					ToolResultMessages: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
				}, hookErr
			})
		},
	})
	res, err = m.AfterToolMessages(context.Background(), &plugin.AfterToolMessagesArgs{
		ToolResultMessages: []compat.Message{compat.NewToolMessage("call-1", "lookup", "raw")},
	})
	require.ErrorIs(t, err, hookErr)
	require.NotNil(t, res)

	t.Run("uses event messages when args messages are empty", func(t *testing.T) {
		toolEvent := event.NewResponseEvent("inv", "agent", &compat.Response{
			Object: compat.ObjectTypeToolResponse,
			Choices: []compat.Choice{
				{Index: 2, Delta: compat.NewToolMessage("call-1", "lookup", "raw delta")},
			},
		})
		m := plugin.MustNewManager(&testPlugin{
			name: "event-fallback",
			reg: func(r *plugin.Registry) {
				r.AfterToolMessages(func(
					_ context.Context,
					args *plugin.AfterToolMessagesArgs,
				) (*plugin.AfterToolMessagesResult, error) {
					require.Len(t, args.ToolResultMessages, 0)
					return &plugin.AfterToolMessagesResult{
						ToolResultMessages: []compat.Message{
							compat.NewToolMessage("call-1", "lookup", "summary delta"),
						},
					}, nil
				})
			},
		})
		args := &plugin.AfterToolMessagesArgs{
			ToolResultEvent: toolEvent,
			Messages: []compat.Message{
				compat.NewUserMessage("query"),
				compat.NewToolMessage("call-1", "lookup", "raw delta"),
			},
		}
		res, err := m.AfterToolMessages(context.Background(), args)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.ToolResultMessages, 1)
		require.Equal(t, "summary delta", res.ToolResultMessages[0].Content)
		require.Equal(t, "summary delta", toolEvent.Response.Choices[0].Delta.Content)
		require.Len(t, args.Messages, 2)
		require.Equal(t, "summary delta", args.Messages[1].Content)
	})

	cases := []struct {
		name         string
		original     []compat.Message
		replacements []compat.Message
		want         string
	}{
		{
			name:         "empty original",
			original:     nil,
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
			want:         "original tool result messages are empty",
		},
		{
			name:         "count mismatch",
			original:     []compat.Message{compat.NewToolMessage("call-1", "lookup", "raw")},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary"), compat.NewToolMessage("call-2", "lookup", "summary")},
			want:         "replacement count",
		},
		{
			name:         "replacement role",
			original:     []compat.Message{compat.NewToolMessage("call-1", "lookup", "raw")},
			replacements: []compat.Message{{Role: compat.RoleAssistant, ToolID: "call-1", Content: "summary"}},
			want:         "must use role",
		},
		{
			name:         "duplicate replacement",
			original:     []compat.Message{compat.NewToolMessage("call-1", "lookup", "raw"), compat.NewToolMessage("call-2", "lookup", "raw")},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary"), compat.NewToolMessage("call-1", "lookup", "summary again")},
			want:         "duplicate tool id",
		},
		{
			name:         "original missing id",
			original:     []compat.Message{{Role: compat.RoleTool, Content: "raw"}},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
			want:         "original tool message missing tool id",
		},
		{
			name:         "original role",
			original:     []compat.Message{{Role: compat.RoleAssistant, ToolID: "call-1", Content: "raw"}},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
			want:         "original for tool id",
		},
		{
			name:         "missing replacement",
			original:     []compat.Message{compat.NewToolMessage("call-1", "lookup", "raw")},
			replacements: []compat.Message{compat.NewToolMessage("call-2", "lookup", "summary")},
			want:         "replacement missing tool id",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := plugin.MustNewManager(&testPlugin{
				name: "invalid",
				reg: func(r *plugin.Registry) {
					r.AfterToolMessages(func(
						context.Context,
						*plugin.AfterToolMessagesArgs,
					) (*plugin.AfterToolMessagesResult, error) {
						return &plugin.AfterToolMessagesResult{
							ToolResultMessages: tc.replacements,
						}, nil
					})
				},
			})
			_, err := m.AfterToolMessages(context.Background(), &plugin.AfterToolMessagesArgs{
				ToolResultMessages: tc.original,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}

	eventCases := []struct {
		name         string
		eventChoices []compat.Choice
		replacements []compat.Message
		want         string
	}{
		{
			name: "choice missing id",
			eventChoices: []compat.Choice{{
				Message: compat.NewAssistantMessage("no tool id"),
			}},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
			want:         "original tool result choice missing tool id",
		},
		{
			name: "event choice missing replacement",
			eventChoices: []compat.Choice{{
				Message: compat.NewToolMessage("call-2", "lookup", "raw"),
			}},
			replacements: []compat.Message{compat.NewToolMessage("call-1", "lookup", "summary")},
			want:         "replacement missing tool id",
		},
		{
			name: "event replacement unknown",
			eventChoices: []compat.Choice{{
				Message: compat.NewToolMessage("call-1", "lookup", "raw"),
			}},
			replacements: []compat.Message{
				compat.NewToolMessage("call-1", "lookup", "summary"),
				compat.NewToolMessage("call-2", "lookup", "summary"),
			},
			want: "replacement contains unknown tool id",
		},
	}
	for _, tc := range eventCases {
		t.Run(tc.name, func(t *testing.T) {
			m := plugin.MustNewManager(&testPlugin{
				name: "event-invalid",
				reg: func(r *plugin.Registry) {
					r.AfterToolMessages(func(
						context.Context,
						*plugin.AfterToolMessagesArgs,
					) (*plugin.AfterToolMessagesResult, error) {
						return &plugin.AfterToolMessagesResult{
							ToolResultMessages: tc.replacements,
						}, nil
					})
				},
			})
			original := make([]compat.Message, 0, len(tc.replacements))
			for _, msg := range tc.replacements {
				original = append(original, compat.NewToolMessage(msg.ToolID, msg.ToolName, "raw"))
			}
			_, err := m.AfterToolMessages(context.Background(), &plugin.AfterToolMessagesArgs{
				ToolResultMessages: original,
				ToolResultEvent: event.NewResponseEvent("inv", "agent", &compat.Response{
					Object:  compat.ObjectTypeToolResponse,
					Choices: tc.eventChoices,
				}),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestManager_OnEvent_Order(t *testing.T) {
	var calls []string
	p1 := &testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.OnEvent(func(
				ctx context.Context,
				inv *agent.Invocation,
				e *event.Event,
			) (*event.Event, error) {
				calls = append(calls, "p1")
				return nil, nil
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.OnEvent(func(
				ctx context.Context,
				inv *agent.Invocation,
				e *event.Event,
			) (*event.Event, error) {
				calls = append(calls, "p2")
				return nil, nil
			})
		},
	}

	m := plugin.MustNewManager(p1, p2)
	inv := &agent.Invocation{}
	e := event.New("inv", "author")
	_, err := m.OnEvent(context.Background(), inv, e)
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p2"}, calls)
}

func TestManager_OnEvent_ErrorWrapsName(t *testing.T) {
	wantErr := errors.New("boom")
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.OnEvent(func(
				ctx context.Context,
				inv *agent.Invocation,
				e *event.Event,
			) (*event.Event, error) {
				return nil, wantErr
			})
		},
	}

	m := plugin.MustNewManager(p)
	_, err := m.OnEvent(
		context.Background(),
		&agent.Invocation{},
		&event.Event{},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin")
	require.Contains(t, err.Error(), "p")
}

func TestManager_OnEvent_ReplacementPropagates(t *testing.T) {
	const (
		tagOriginal = "orig"
		tagUpdated  = "updated"
	)
	var seen []string
	p1 := &testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.OnEvent(func(
				ctx context.Context,
				inv *agent.Invocation,
				e *event.Event,
			) (*event.Event, error) {
				updated := &event.Event{Tag: tagUpdated}
				return updated, nil
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.OnEvent(func(
				ctx context.Context,
				inv *agent.Invocation,
				e *event.Event,
			) (*event.Event, error) {
				if e != nil {
					seen = append(seen, e.Tag)
				}
				return nil, nil
			})
		},
	}

	m := plugin.MustNewManager(p1, p2)
	_, err := m.OnEvent(
		context.Background(),
		&agent.Invocation{},
		&event.Event{Tag: tagOriginal},
	)
	require.NoError(t, err)
	require.Equal(t, []string{tagUpdated}, seen)
}

func TestManager_AfterRun_Order(t *testing.T) {
	var calls []string
	p1 := &testPlugin{
		name: "p1",
		reg: func(r *plugin.Registry) {
			r.AfterRun(func(ctx context.Context, args *plugin.AfterRunArgs) error {
				calls = append(calls, "p1")
				return nil
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.AfterRun(func(ctx context.Context, args *plugin.AfterRunArgs) error {
				calls = append(calls, "p2")
				return nil
			})
		},
	}
	m := plugin.MustNewManager(p1, p2)
	err := m.AfterRun(context.Background(), &plugin.AfterRunArgs{
		Invocation:      &agent.Invocation{},
		CompletionEvent: event.New("inv", "runner"),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p2"}, calls)
}

func TestManager_AfterRun_ErrorWrapsName(t *testing.T) {
	wantErr := errors.New("boom")
	called := false
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.AfterRun(func(ctx context.Context, args *plugin.AfterRunArgs) error {
				return wantErr
			})
		},
	}
	p2 := &testPlugin{
		name: "p2",
		reg: func(r *plugin.Registry) {
			r.AfterRun(func(ctx context.Context, args *plugin.AfterRunArgs) error {
				called = true
				return nil
			})
		},
	}
	m := plugin.MustNewManager(p, p2)
	err := m.AfterRun(context.Background(), &plugin.AfterRunArgs{})
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Contains(t, err.Error(), "plugin")
	require.Contains(t, err.Error(), "p")
	require.True(t, called)
}

func TestManager_AgentCallbacks_WrapErrorWithName(t *testing.T) {
	wantErr := errors.New("boom")
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.BeforeAgent(func(
				ctx context.Context,
				args *agent.BeforeAgentArgs,
			) (*agent.BeforeAgentResult, error) {
				return nil, wantErr
			})
		},
	}

	m := plugin.MustNewManager(p)
	cb := m.AgentCallbacks()
	require.NotNil(t, cb)

	_, err := cb.RunBeforeAgent(
		context.Background(),
		&agent.BeforeAgentArgs{Invocation: &agent.Invocation{}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "p")
}

func TestManager_ToolCallbacks_WrapErrorWithName(t *testing.T) {
	wantErr := errors.New("boom")
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.BeforeTool(func(
				ctx context.Context,
				args *tool.BeforeToolArgs,
			) (*tool.BeforeToolResult, error) {
				return nil, wantErr
			})
		},
	}

	m := plugin.MustNewManager(p)
	cb := m.ToolCallbacks()
	require.NotNil(t, cb)

	_, err := cb.RunBeforeTool(
		context.Background(),
		&tool.BeforeToolArgs{ToolName: "t"},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "p")
}

func TestManager_ModelCallbacks_WrapAfterErrorWithName(t *testing.T) {
	wantErr := errors.New("boom")
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.AfterModel(func(
				ctx context.Context,
				args *compat.AfterModelArgs,
			) (*compat.AfterModelResult, error) {
				return nil, wantErr
			})
		},
	}

	m := plugin.MustNewManager(p)
	cb := m.ModelCallbacks()
	require.NotNil(t, cb)

	_, err := cb.RunAfterModel(
		context.Background(),
		&compat.AfterModelArgs{
			Request:  &compat.Request{},
			Response: &compat.Response{Done: true},
		},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "p")
}

func TestManager_ToolCallbacks_WrapAfterErrorWithName(t *testing.T) {
	wantErr := errors.New("boom")
	p := &testPlugin{
		name: "p",
		reg: func(r *plugin.Registry) {
			r.AfterTool(func(
				ctx context.Context,
				args *tool.AfterToolArgs,
			) (*tool.AfterToolResult, error) {
				return nil, wantErr
			})
		},
	}

	m := plugin.MustNewManager(p)
	cb := m.ToolCallbacks()
	require.NotNil(t, cb)

	_, err := cb.RunAfterTool(
		context.Background(),
		&tool.AfterToolArgs{
			ToolName:    "t",
			Declaration: nil,
			Arguments:   []byte("{}"),
			Result:      "x",
		},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "p")
}

func TestGlobalInstruction_PrependsSystemMessage(t *testing.T) {
	m := plugin.MustNewManager(plugin.NewGlobalInstruction("policy"))
	callbacks := m.ModelCallbacks()
	require.NotNil(t, callbacks)

	req := &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("hi")},
	}

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.NotEmpty(t, req.Messages)
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.Contains(t, req.Messages[0].Content, "policy")
}

func TestGlobalInstruction_NoMessages_AddsSystem(t *testing.T) {
	const instr = "policy"
	m := plugin.MustNewManager(plugin.NewGlobalInstruction(instr))
	callbacks := m.ModelCallbacks()

	req := &compat.Request{}
	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)

	require.Len(t, req.Messages, 1)
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.Equal(t, instr, req.Messages[0].Content)
}

func TestGlobalInstruction_EmptyInstruction_NoChange(t *testing.T) {
	m := plugin.MustNewManager(plugin.NewGlobalInstruction("  \n\t "))
	callbacks := m.ModelCallbacks()
	req := &compat.Request{
		Messages: []compat.Message{compat.NewUserMessage("hi")},
	}

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 1)
	require.Equal(t, compat.RoleUser, req.Messages[0].Role)
}

func TestGlobalInstruction_SystemEmptyContent_Sets(t *testing.T) {
	const instr = "policy"
	m := plugin.MustNewManager(plugin.NewGlobalInstruction(instr))
	callbacks := m.ModelCallbacks()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage(""),
			compat.NewUserMessage("hi"),
		},
	}

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 2)
	require.Equal(t, instr, req.Messages[0].Content)
}

func TestGlobalInstruction_SystemWithContent_Prepends(t *testing.T) {
	const (
		instr = "policy"
		old   = "old"
	)
	m := plugin.MustNewManager(plugin.NewGlobalInstruction(instr))
	callbacks := m.ModelCallbacks()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewSystemMessage(old),
			compat.NewUserMessage("hi"),
		},
	}

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.True(
		t,
		strings.HasPrefix(req.Messages[0].Content, instr),
	)
	require.Contains(t, req.Messages[0].Content, old)
}

func TestGlobalInstruction_FirstNonSystem_Prepends(t *testing.T) {
	const instr = "policy"
	m := plugin.MustNewManager(plugin.NewGlobalInstruction(instr))
	callbacks := m.ModelCallbacks()
	req := &compat.Request{
		Messages: []compat.Message{
			compat.NewUserMessage("hi"),
		},
	}

	_, err := callbacks.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: req},
	)
	require.NoError(t, err)
	require.Len(t, req.Messages, 2)
	require.Equal(t, compat.RoleSystem, req.Messages[0].Role)
	require.Equal(t, instr, req.Messages[0].Content)
	require.Equal(t, compat.RoleUser, req.Messages[1].Role)
}

func TestLoggingPlugin_Callbacks_DoNotError(t *testing.T) {
	m := plugin.MustNewManager(plugin.NewLogging())

	inv := &agent.Invocation{
		AgentName: "a",
		Model:     &staticModel{name: "m"},
	}

	agentCB := m.AgentCallbacks()
	require.NotNil(t, agentCB)
	before, err := agentCB.RunBeforeAgent(
		context.Background(),
		&agent.BeforeAgentArgs{Invocation: inv},
	)
	require.NoError(t, err)
	ctx := context.Background()
	if before != nil && before.Context != nil {
		ctx = before.Context
	}
	_, err = agentCB.RunAfterAgent(ctx, &agent.AfterAgentArgs{
		Invocation:        inv,
		FullResponseEvent: &event.Event{},
		Error:             nil,
	})
	require.NoError(t, err)

	modelCB := m.ModelCallbacks()
	require.NotNil(t, modelCB)
	invCtx := agent.NewInvocationContext(context.Background(), inv)
	beforeModel, err := modelCB.RunBeforeModel(
		invCtx,
		&compat.BeforeModelArgs{Request: &compat.Request{}},
	)
	require.NoError(t, err)
	var modelCtx context.Context = invCtx
	if beforeModel != nil && beforeModel.Context != nil {
		modelCtx = beforeModel.Context
	}
	_, err = modelCB.RunAfterModel(modelCtx, &compat.AfterModelArgs{
		Request:  &compat.Request{},
		Response: &compat.Response{Done: true},
		Error:    nil,
	})
	require.NoError(t, err)

	toolCB := m.ToolCallbacks()
	require.NotNil(t, toolCB)
	toolCtx := context.WithValue(
		context.Background(),
		tool.ContextKeyToolCallID{},
		"call",
	)
	beforeTool, err := toolCB.RunBeforeTool(
		toolCtx,
		&tool.BeforeToolArgs{
			ToolName:    "t",
			Declaration: nil,
			Arguments:   []byte("{}"),
		},
	)
	require.NoError(t, err)
	if beforeTool != nil && beforeTool.Context != nil {
		toolCtx = beforeTool.Context
	}
	_, err = toolCB.RunAfterTool(toolCtx, &tool.AfterToolArgs{
		ToolName:    "t",
		Declaration: nil,
		Arguments:   []byte("{}"),
		Result:      "ok",
		Error:       nil,
	})
	require.NoError(t, err)
}

func TestLoggingPlugin_NoInvocationContextAndErrorArgs(t *testing.T) {
	m := plugin.MustNewManager(plugin.NewLogging())
	cb := m.ModelCallbacks()
	require.NotNil(t, cb)

	before, err := cb.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: &compat.Request{}},
	)
	require.NoError(t, err)
	require.NotNil(t, before)

	afterCtx := context.Background()
	if before.Context != nil {
		afterCtx = before.Context
	}
	_, err = cb.RunAfterModel(afterCtx, &compat.AfterModelArgs{
		Request:  &compat.Request{},
		Response: &compat.Response{Done: true},
		Error:    errors.New("boom"),
	})
	require.NoError(t, err)

	toolCB := m.ToolCallbacks()
	require.NotNil(t, toolCB)
	_, err = toolCB.RunBeforeTool(
		context.Background(),
		&tool.BeforeToolArgs{
			ToolName:    "t",
			Declaration: nil,
			Arguments:   []byte("{}"),
		},
	)
	require.NoError(t, err)
}

func TestGlobalInstruction_NilRequest_IsSafe(t *testing.T) {
	m := plugin.MustNewManager(plugin.NewGlobalInstruction("policy"))
	cb := m.ModelCallbacks()
	require.NotNil(t, cb)

	_, err := cb.RunBeforeModel(
		context.Background(),
		&compat.BeforeModelArgs{Request: nil},
	)
	require.NoError(t, err)
}

type staticModel struct {
	name string
}

func (m *staticModel) GenerateContent(
	_ context.Context,
	_ *compat.Request,
) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{Done: true}
	close(ch)
	return ch, nil
}

func (m *staticModel) Info() compat.Info {
	return compat.Info{Name: m.name}
}

func TestRegistry_NilReceiver_IsSafe(t *testing.T) {
	var r *plugin.Registry
	r.BeforeAgent(nil)
	r.AfterAgent(nil)
	r.BeforeModel(nil)
	r.AfterModel(nil)
	r.BeforeTool(nil)
	r.AfterTool(nil)
	r.OnEvent(nil)
	r.AfterRun(nil)
}

func TestNewNamedLogging_EmptyName_UsesDefault(t *testing.T) {
	const defaultName = "logging"
	got := plugin.NewNamedLogging("")
	require.Equal(t, defaultName, got.Name())
}

func TestNewNamedGlobalInstruction_EmptyName_UsesDefault(t *testing.T) {
	const defaultName = "global_instruction"
	got := plugin.NewNamedGlobalInstruction("", "x")
	require.Equal(t, defaultName, got.Name())
}
