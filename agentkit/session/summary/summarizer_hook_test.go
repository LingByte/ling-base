//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package summary

import (
	"context"
	"strings"
	"testing"

	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	isummarycontext "github.com/LingByte/ling-base/agentkit/session/internal/summarycontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type echoPromptModel struct {
	lastPrompt string
}

func (m *echoPromptModel) Info() compat.Info {
	return compat.Info{Name: "echo"}
}

func (m *echoPromptModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	if len(req.Messages) > 0 {
		m.lastPrompt = req.Messages[0].Content
	}
	ch <- &compat.Response{
		Done: true,
		Choices: []compat.Choice{{
			Message: compat.Message{Content: m.lastPrompt},
		}},
	}
	close(ch)
	return ch, nil
}

func newEventWithContent(content string) event.Event {
	return event.Event{
		Author: "user",
		Response: &compat.Response{
			Choices: []compat.Choice{{Message: compat.Message{Content: content}}},
		},
	}
}

func TestSessionSummarizer_PreHook_ModifiesInput(t *testing.T) {
	model := &echoPromptModel{}
	s := NewSummarizer(model, WithPreSummaryHook(func(in *PreSummaryHookContext) error {
		in.Text = "HOOKED_TEXT"
		return nil
	}))

	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Contains(t, summary, "HOOKED_TEXT")
	assert.NotContains(t, summary, "origin")
}

func TestSessionSummarizer_PreHook_ErrorBehavior(t *testing.T) {
	model := &echoPromptModel{}
	hookErr := assert.AnError

	t.Run("abort on error", func(t *testing.T) {
		s := NewSummarizer(model,
			WithPreSummaryHook(func(in *PreSummaryHookContext) error {
				return hookErr
			}),
			WithSummaryHookAbortOnError(true),
		)
		sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pre-summary hook failed")
	})

	t.Run("fallback on error", func(t *testing.T) {
		s := NewSummarizer(model,
			WithPreSummaryHook(func(in *PreSummaryHookContext) error {
				return hookErr
			}),
			WithSummaryHookAbortOnError(false),
		)
		sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
		summary, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, summary, "origin")
	})
}

func TestSessionSummarizer_PreHook_ModifiesEventsAndRebuildsText(t *testing.T) {
	model := &echoPromptModel{}
	s := NewSummarizer(model, WithPreSummaryHook(func(in *PreSummaryHookContext) error {
		in.Events = []event.Event{newEventWithContent("new-content")}
		in.Text = ""
		return nil
	}))

	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Contains(t, summary, "new-content")
	assert.NotContains(t, summary, "origin")
}

func TestSessionSummarizer_PreHook_ModifiesSeparatedPreviousSummary(t *testing.T) {
	mdl := &echoPromptModel{}
	var capturedEvents []event.Event
	s := NewSummarizer(
		mdl,
		WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		WithPreSummaryHook(func(in *PreSummaryHookContext) error {
			capturedEvents = append([]event.Event(nil), in.Events...)
			require.Equal(t, "previous", in.PreviousSummary)
			require.Equal(t, "user: new conversation", in.Text)
			in.PreviousSummary = "redacted previous"
			in.Text = "redacted conversation"
			return nil
		}),
	)
	sess := &session.Session{ID: "sess", Events: []event.Event{
		{
			Author: authorSystem,
			Response: &compat.Response{Choices: []compat.Choice{{
				Message: compat.Message{Content: "previous"},
			}}},
		},
		newEventWithContent("new conversation"),
	}}
	ctx := isummarycontext.WithPreviousSummary(context.Background(), "previous")

	result, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	require.Len(t, capturedEvents, 1)
	require.Equal(t, authorUser, capturedEvents[0].Author)
	require.Contains(t, result, "Previous:\nredacted previous")
	require.Contains(t, result, "Conversation:\nredacted conversation")
	require.NotContains(t, result, "new conversation")
}

func TestSessionSummarizer_PreHook_ClearsSeparatedConversation(t *testing.T) {
	mdl := &echoPromptModel{}
	s := NewSummarizer(
		mdl,
		WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		WithPreSummaryHook(func(in *PreSummaryHookContext) error {
			require.Equal(t, "previous", in.PreviousSummary)
			in.Events = nil
			in.Text = ""
			return nil
		}),
	)
	sess := &session.Session{ID: "sess", Events: []event.Event{
		{
			Author: authorSystem,
			Response: &compat.Response{Choices: []compat.Choice{{
				Message: compat.Message{Content: "previous"},
			}}},
		},
		newEventWithContent("sensitive conversation"),
	}}
	ctx := isummarycontext.WithPreviousSummary(context.Background(), "previous")

	result, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	require.Contains(t, result, "Previous:\nprevious")
	require.Contains(t, result, "Conversation:\n\n\nSummary:")
	require.NotContains(t, result, "sensitive conversation")
}

func TestSessionSummarizer_PostHook_ModifiesOutput(t *testing.T) {
	model := &echoPromptModel{}
	s := NewSummarizer(model, WithPostSummaryHook(func(in *PostSummaryHookContext) error {
		in.Summary = "POST_HOOKED"
		return nil
	}))

	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Equal(t, "POST_HOOKED", summary)
}

func TestSessionSummarizer_PreHook_ContextPropagation(t *testing.T) {
	model := &echoPromptModel{}
	type ctxKey string
	const contextKey ctxKey = "pre-hook-key"

	var capturedVal any
	s := NewSummarizer(model, WithPreSummaryHook(func(in *PreSummaryHookContext) error {
		capturedVal = in.Ctx.Value(contextKey)
		return nil
	}))

	ctx := context.WithValue(context.Background(), contextKey, "pre-ctx-value")
	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	_, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	assert.Equal(t, "pre-ctx-value", capturedVal)
}

func TestSessionSummarizer_PostHook_ContextPropagation(t *testing.T) {
	model := &echoPromptModel{}
	type ctxKey string
	const contextKey ctxKey = "post-hook-key"

	var capturedVal any
	s := NewSummarizer(model, WithPostSummaryHook(func(in *PostSummaryHookContext) error {
		capturedVal = in.Ctx.Value(contextKey)
		return nil
	}))

	ctx := context.WithValue(context.Background(), contextKey, "ctx-value")
	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	_, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	assert.Equal(t, "ctx-value", capturedVal)
}

func TestSessionSummarizer_PreHook_ContextPropagationToPostHook(t *testing.T) {
	model := &echoPromptModel{}
	type ctxKey string
	const preHookKey ctxKey = "pre-hook-injected-key"

	var capturedVal any
	s := NewSummarizer(model,
		WithPreSummaryHook(func(in *PreSummaryHookContext) error {
			// PreHook injects a new value into context.
			in.Ctx = context.WithValue(in.Ctx, preHookKey, "injected-by-pre-hook")
			return nil
		}),
		WithPostSummaryHook(func(in *PostSummaryHookContext) error {
			// PostHook should be able to read the value injected by PreHook.
			capturedVal = in.Ctx.Value(preHookKey)
			return nil
		}),
	)

	sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
	_, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Equal(t, "injected-by-pre-hook", capturedVal)
}

func TestSessionSummarizer_PostHook_ErrorBehavior(t *testing.T) {
	model := &echoPromptModel{}
	hookErr := assert.AnError

	t.Run("abort on error", func(t *testing.T) {
		s := NewSummarizer(model,
			WithPostSummaryHook(func(in *PostSummaryHookContext) error {
				return hookErr
			}),
			WithSummaryHookAbortOnError(true),
		)
		sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "post-summary hook failed")
	})

	t.Run("fallback on error", func(t *testing.T) {
		s := NewSummarizer(model,
			WithPostSummaryHook(func(in *PostSummaryHookContext) error {
				return hookErr
			}),
			WithSummaryHookAbortOnError(false),
		)
		sess := &session.Session{ID: "sess", Events: []event.Event{newEventWithContent("origin")}}
		summary, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, summary, "origin")
	})
}

type panicGenerateModel struct {
	contextWindow int
}

func (m *panicGenerateModel) Info() compat.Info {
	return compat.Info{
		Name:          "panic-generate",
		ContextWindow: m.contextWindow,
	}
}

func (m *panicGenerateModel) GenerateContent(
	ctx context.Context,
	req *compat.Request,
) (<-chan *compat.Response, error) {
	panic("GenerateContent should not be called")
}

type staticResponseModel struct {
	content string
}

func (m *staticResponseModel) Info() compat.Info {
	return compat.Info{Name: "static"}
}

func (m *staticResponseModel) GenerateContent(
	ctx context.Context,
	req *compat.Request,
) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{
		Done: true,
		Choices: []compat.Choice{{
			Message: compat.Message{Content: m.content},
		}},
	}
	close(ch)
	return ch, nil
}

func TestSessionSummarizer_ModelCallbacks_Before_ModifiesRequest(t *testing.T) {
	m := &echoPromptModel{}
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			ctx context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			if args != nil && args.Request != nil && len(args.Request.Messages) > 0 {
				args.Request.Messages[0].Content = "MODIFIED"
			}
			return nil, nil
		},
	)
	s := NewSummarizer(m, WithModelCallbacks(callbacks))

	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Contains(t, summary, "MODIFIED")
	assert.NotContains(t, summary, "origin")
}

func TestSessionSummarizer_ModelCallbacks_Before_RejectsOversizedRequest(
	t *testing.T,
) {
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			_ context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			args.Request.Messages = append(
				args.Request.Messages,
				compat.NewSystemMessage(strings.Repeat("callback-content ", 1000)),
			)
			return nil, nil
		},
	)
	s := NewSummarizer(
		&panicGenerateModel{contextWindow: 1000},
		WithModelCallbacks(callbacks),
	)
	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.ErrorContains(t, err, "no longer fits after before-model callbacks")
}

func TestSessionSummarizer_ModelCallbacks_Before_CustomResponseSkipsModel(t *testing.T) {
	var callbackInput string
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			ctx context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			callbackInput = args.Request.Messages[0].Content
			return &compat.BeforeModelResult{
				CustomResponse: &compat.Response{
					Done: true,
					Choices: []compat.Choice{{
						Message: compat.Message{Content: "FROM_CALLBACK"},
					}},
				},
			}, nil
		},
	)
	s := NewSummarizer(
		&panicGenerateModel{contextWindow: 1000},
		WithModelCallbacks(callbacks),
	)

	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Equal(t, "FROM_CALLBACK", summary)
	assert.Contains(t, callbackInput, "origin")
}

func TestSessionSummarizer_ModelCallbacks_Before_AppliesToFallbackRequest(
	t *testing.T,
) {
	const sentinel = "callback-applied"
	maxTokens := 77
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			_ context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			args.Request.GenerationConfig.MaxTokens = &maxTokens
			if args.Request.Headers == nil {
				args.Request.Headers = make(map[string]string)
			}
			args.Request.Headers["X-Summary-Callback"] = sentinel
			return nil, nil
		},
	)
	capture := &cacheSafeCaptureModel{
		response:      "summary",
		contextWindow: 1000,
	}
	s := NewSummarizer(
		capture,
		WithCacheSafeForking(true),
		WithModelCallbacks(callbacks),
	)
	oversizedParent := strings.Repeat("oversized-parent ", 1000)
	parent := &compat.Request{Messages: []compat.Message{
		compat.NewSystemMessage("stable system"),
		compat.NewUserMessage(oversizedParent),
	}}
	ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)
	sess := &session.Session{
		ID:     "fallback-callback",
		Events: []event.Event{newEventWithContent("event text for fallback")},
	}

	summary, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	require.Equal(t, "summary", summary)
	require.NotNil(t, capture.request)
	require.Len(t, capture.request.Messages, 1)
	require.Contains(t, capture.request.Messages[0].Content,
		"event text for fallback")
	require.NotNil(t, capture.request.GenerationConfig.MaxTokens)
	require.Equal(t, maxTokens,
		*capture.request.GenerationConfig.MaxTokens)
	require.Equal(t, sentinel,
		capture.request.Headers["X-Summary-Callback"])
}

func TestSessionSummarizer_ModelCallbacks_After_OverridesResponse(t *testing.T) {
	callbacks := compat.NewCallbacks().RegisterAfterModel(
		func(
			ctx context.Context,
			args *compat.AfterModelArgs,
		) (*compat.AfterModelResult, error) {
			return &compat.AfterModelResult{
				CustomResponse: &compat.Response{
					Done: true,
					Choices: []compat.Choice{{
						Message: compat.Message{Content: "OVERRIDE"},
					}},
				},
			}, nil
		},
	)
	s := NewSummarizer(
		&staticResponseModel{content: "ORIG"},
		WithModelCallbacks(callbacks),
	)

	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}
	summary, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Equal(t, "OVERRIDE", summary)
}

func TestSessionSummarizer_ModelCallbacks_ContextPropagationToPostHook(t *testing.T) {
	type ctxKey string
	const key ctxKey = "after-model-key"

	m := &staticResponseModel{content: "OK"}
	var captured any

	callbacks := compat.NewCallbacks().RegisterAfterModel(
		func(
			ctx context.Context,
			args *compat.AfterModelArgs,
		) (*compat.AfterModelResult, error) {
			return &compat.AfterModelResult{
				Context: context.WithValue(ctx, key, "value"),
			}, nil
		},
	)

	s := NewSummarizer(m,
		WithModelCallbacks(callbacks),
		WithPostSummaryHook(func(in *PostSummaryHookContext) error {
			captured = in.Ctx.Value(key)
			return nil
		}),
	)

	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}
	_, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Equal(t, "value", captured)
}

func TestSessionSummarizer_ModelCallbacks_Before_Error(t *testing.T) {
	callbacks := compat.NewCallbacks().RegisterBeforeModel(
		func(
			ctx context.Context,
			args *compat.BeforeModelArgs,
		) (*compat.BeforeModelResult, error) {
			return nil, assert.AnError
		},
	)

	s := NewSummarizer(
		&staticResponseModel{content: "OK"},
		WithModelCallbacks(callbacks),
	)

	sess := &session.Session{
		ID:     "sess",
		Events: []event.Event{newEventWithContent("origin")},
	}
	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before model callback failed")
}
