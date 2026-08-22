//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2026 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package runner

import (
	"context"
	"testing"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/internal/flow/processor"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/plugin"
	"github.com/LingByte/ling-base/agentkit/session"
	sessioninmemory "github.com/LingByte/ling-base/agentkit/session/inmemory"
	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/require"
)

type summaryMessageCapturingAgent struct {
	name       string
	addSummary bool
	messages   []compat.Message
}

func (a *summaryMessageCapturingAgent) Info() agent.Info {
	return agent.Info{Name: a.name}
}

func (a *summaryMessageCapturingAgent) SubAgents() []agent.Agent {
	return nil
}

func (a *summaryMessageCapturingAgent) FindSubAgent(string) agent.Agent {
	return nil
}

func (a *summaryMessageCapturingAgent) Tools() []tool.Tool {
	return nil
}

func (a *summaryMessageCapturingAgent) Run(
	ctx context.Context,
	inv *agent.Invocation,
) (<-chan *event.Event, error) {
	if a.addSummary {
		events := inv.Session.Events
		last := events[len(events)-1]
		filterKey := inv.GetEventFilterKey()
		if inv.Session.Summaries == nil {
			inv.Session.Summaries = make(map[string]*session.Summary)
		}
		inv.Session.Summaries[filterKey] = &session.Summary{
			Summary:   "covered history",
			UpdatedAt: last.Timestamp,
			Boundary: session.NewSummaryBoundaryWithEventID(
				filterKey,
				last.Timestamp,
				last.ID,
			),
		}
	}

	req := &compat.Request{}
	processor.NewContentRequestProcessor(
		processor.WithAddSessionSummary(a.addSummary),
	).ProcessRequest(ctx, inv, req, nil)
	a.messages = append([]compat.Message(nil), req.Messages...)

	ch := make(chan *event.Event)
	close(ch)
	return ch, nil
}

func TestRunner_Run_SeedHistoryRespectsMidRunSummary(t *testing.T) {
	firstToolCall := compat.ToolCall{
		Type: "function",
		ID:   "call-1",
		Function: compat.FunctionDefinitionParam{
			Name:      "lookup-first",
			Arguments: []byte(`{"query":"current"}`),
		},
	}
	secondToolCall := compat.ToolCall{
		Type: "function",
		ID:   "call-2",
		Function: compat.FunctionDefinitionParam{
			Name:      "lookup-second",
			Arguments: []byte(`{"query":"context"}`),
		},
	}
	rewrittenToolTranscript := []compat.Message{
		compat.NewUserMessage("current context"),
		{
			Role:      compat.RoleAssistant,
			Content:   "calling first lookup",
			ToolCalls: []compat.ToolCall{firstToolCall},
		},
		compat.NewToolMessage("call-1", "lookup-first", `{"answer":"first"}`),
		{
			Role:      compat.RoleAssistant,
			Content:   "calling second lookup",
			ToolCalls: []compat.ToolCall{secondToolCall},
		},
		compat.NewToolMessage("call-2", "lookup-second", `{"answer":"second"}`),
		compat.NewUserMessage("rewritten"),
	}
	tests := []struct {
		name       string
		run        func(Runner) (<-chan *event.Event, error)
		want       []compat.Message
		addSummary bool
		eventHook  plugin.EventHook
	}{
		{
			name: "seed history is ordinary history at the summary cutoff",
			run: func(r Runner) (<-chan *event.Event, error) {
				return RunWithMessages(
					context.Background(),
					r,
					"user",
					"session",
					[]compat.Message{
						compat.NewUserMessage("current"),
						compat.NewAssistantMessage("old answer"),
						compat.NewUserMessage("current"),
					},
				)
			},
			want: []compat.Message{
				compat.NewUserMessage("current"),
			},
			addSummary: true,
		},
		{
			name: "rewritten current turn remains after the summary cutoff",
			run: func(r Runner) (<-chan *event.Event, error) {
				return r.Run(
					context.Background(),
					"user",
					"session",
					compat.NewUserMessage("original"),
					agent.WithMessages([]compat.Message{
						compat.NewUserMessage("old question"),
						compat.NewAssistantMessage("old answer"),
						compat.NewUserMessage("original"),
					}),
					agent.WithUserMessageRewriter(func(
						context.Context,
						*agent.UserMessageRewriteArgs,
					) ([]compat.Message, error) {
						return []compat.Message{
							compat.NewUserMessage("current context"),
							compat.NewUserMessage("rewritten"),
						}, nil
					}),
				)
			},
			want: []compat.Message{
				compat.NewUserMessage("current context"),
				compat.NewUserMessage("rewritten"),
			},
			addSummary: true,
		},
		{
			name: "seeded tool round remains covered by the summary",
			run: func(r Runner) (<-chan *event.Event, error) {
				return RunWithMessages(
					context.Background(),
					r,
					"user",
					"session",
					[]compat.Message{
						compat.NewUserMessage("old question"),
						{
							Role:      compat.RoleAssistant,
							Content:   "calling old lookup",
							ToolCalls: []compat.ToolCall{firstToolCall},
						},
						compat.NewToolMessage(
							"call-1",
							"lookup-first",
							`{"answer":"old"}`,
						),
						compat.NewUserMessage("current"),
					},
				)
			},
			want: []compat.Message{
				compat.NewUserMessage("current"),
			},
			addSummary: true,
		},
		{
			name: "rewritten current turn preserves its tool transcript",
			run: func(r Runner) (<-chan *event.Event, error) {
				return r.Run(
					context.Background(),
					"user",
					"session",
					compat.NewUserMessage("original"),
					agent.WithMessages([]compat.Message{
						compat.NewUserMessage("old question"),
						compat.NewAssistantMessage("old answer"),
						compat.NewUserMessage("original"),
					}),
					agent.WithUserMessageRewriter(func(
						context.Context,
						*agent.UserMessageRewriteArgs,
					) ([]compat.Message, error) {
						return rewrittenToolTranscript, nil
					}),
				)
			},
			want:       rewrittenToolTranscript,
			addSummary: true,
		},
		{
			name: "event replacement without id retains seed provenance",
			run: func(r Runner) (<-chan *event.Event, error) {
				return RunWithMessages(
					context.Background(),
					r,
					"user",
					"session",
					[]compat.Message{
						compat.NewUserMessage("old question"),
						compat.NewAssistantMessage("old answer"),
						compat.NewUserMessage("current"),
					},
				)
			},
			want: []compat.Message{
				compat.NewUserMessage("current"),
			},
			addSummary: true,
			eventHook: func(
				_ context.Context,
				_ *agent.Invocation,
				evt *event.Event,
			) (*event.Event, error) {
				replacement := *evt
				replacement.ID = ""
				return &replacement, nil
			},
		},
		{
			name: "seed history is unchanged without a summary cutoff",
			run: func(r Runner) (<-chan *event.Event, error) {
				return RunWithMessages(
					context.Background(),
					r,
					"user",
					"session",
					[]compat.Message{
						compat.NewUserMessage("old question"),
						compat.NewAssistantMessage("old answer"),
						compat.NewUserMessage("current"),
					},
				)
			},
			want: []compat.Message{
				compat.NewUserMessage("old question"),
				compat.NewAssistantMessage("old answer"),
				compat.NewUserMessage("current"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := &summaryMessageCapturingAgent{
				name:       "capture",
				addSummary: tt.addSummary,
			}
			opts := []Option{
				WithSessionService(sessioninmemory.NewSessionService()),
			}
			if tt.eventHook != nil {
				opts = append(opts, WithPlugins(&testPlugin{
					name: "replace-event",
					reg: func(registry *plugin.Registry) {
						registry.OnEvent(tt.eventHook)
					},
				}))
			}
			r := NewRunner("app", capture, opts...)
			events, err := tt.run(r)
			require.NoError(t, err)
			for range events {
			}
			if tt.addSummary {
				require.Len(t, capture.messages, len(tt.want)+1)
				require.Equal(t, compat.RoleSystem, capture.messages[0].Role)
				require.Contains(t, capture.messages[0].Content, "covered history")
				require.Equal(t, tt.want, capture.messages[1:])
				return
			}
			require.Equal(t, tt.want, capture.messages)
		})
	}
}

func TestRunner_Run_RewrittenCurrentTurnSurvivesSummaryInExistingSession(
	t *testing.T,
) {
	capture := &summaryMessageCapturingAgent{name: "capture"}
	r := NewRunner(
		"app",
		capture,
		WithSessionService(sessioninmemory.NewSessionService()),
	)
	events, err := r.Run(
		context.Background(),
		"user",
		"session",
		compat.NewUserMessage("old question"),
	)
	require.NoError(t, err)
	for range events {
	}

	rewritten := []compat.Message{
		compat.NewUserMessage("current context"),
		compat.NewAssistantMessage("current acknowledgement"),
		compat.NewUserMessage("rewritten"),
	}
	capture.addSummary = true
	events, err = r.Run(
		context.Background(),
		"user",
		"session",
		compat.NewUserMessage("original"),
		agent.WithUserMessageRewriter(func(
			context.Context,
			*agent.UserMessageRewriteArgs,
		) ([]compat.Message, error) {
			return rewritten, nil
		}),
	)
	require.NoError(t, err)
	for range events {
	}

	require.Len(t, capture.messages, len(rewritten)+1)
	require.Equal(t, compat.RoleSystem, capture.messages[0].Role)
	require.Contains(t, capture.messages[0].Content, "covered history")
	require.Equal(t, rewritten, capture.messages[1:])
}
