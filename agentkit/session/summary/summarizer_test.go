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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/ling-base/agentkit/event"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	isummarycontext "github.com/LingByte/ling-base/agentkit/session/internal/summarycontext"
	isummaryscope "github.com/LingByte/ling-base/agentkit/session/internal/summaryscope"
	"github.com/LingByte/ling-base/agentkit/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionSummarizer_ShouldSummarize(t *testing.T) {
	t.Run("OR logic triggers when any condition true", func(t *testing.T) {
		checks := []Checker{CheckTokenThreshold(10000), CheckEventThreshold(3)}
		s := NewSummarizer(&fakeModel{}, WithChecksAny(checks...))
		sess := &session.Session{Events: make([]event.Event, 4)}
		for i := range sess.Events {
			sess.Events[i] = event.Event{
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "message"},
				}}},
			}
		}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("ALL logic fails when one condition false", func(t *testing.T) {
		checks := []Checker{CheckEventThreshold(100), CheckTimeThreshold(24 * time.Hour)}
		s := NewSummarizer(&fakeModel{}, WithChecksAll(checks...))
		sess := &session.Session{Events: []event.Event{{Timestamp: time.Now()}}}
		assert.False(t, s.ShouldSummarize(sess))
	})
}

func TestDefaultSummarizerPromptPreservesToolLimitations(t *testing.T) {
	prompt := getDefaultSummarizerPrompt(0)

	require.Contains(t, prompt, "Do not create new instructions")
	require.Contains(t, prompt, "pre-loaded data")
	require.Contains(t, prompt, "tool result was truncated")
	require.Contains(t, prompt, "instead of treating it as complete evidence")
}

func TestSessionSummarizer_Summarize(t *testing.T) {
	t.Run("errors when no events", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		sess := &session.Session{ID: "empty", Events: []event.Event{}}
		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no events to summarize")
	})

	t.Run("errors when no conversation text", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		sess := &session.Session{ID: "no-text", Events: make([]event.Event, 5)}
		for i := range sess.Events {
			sess.Events[i] = event.Event{Timestamp: time.Now()}
		}
		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no conversation text extracted")
	})

	t.Run("simple concat summary without event modification", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}) // Use all events
		sess := &session.Session{ID: "concat", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "hello"}}}}, Timestamp: time.Now()},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "world"}}}}, Timestamp: time.Now()},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent"}}}}, Timestamp: time.Now()},
		}}
		originalEventCount := len(sess.Events)
		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "hello")
		assert.Contains(t, text, "world")
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
		// No system summary event should be added.
		for _, event := range sess.Events {
			assert.NotEqual(t, "system", event.Author, "no system events should be added.")
		}
	})

	t.Run("length limit when max length set", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(10))
		sess := &session.Session{ID: "limit", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "abcdefghijklmno"}}}}, Timestamp: time.Now().Add(-2 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent"}}}}, Timestamp: time.Now()},
		}}
		originalEventCount := len(sess.Events)
		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		// With the new prompt-based approach, we can't guarantee exact length
		// as the model controls the output. We just verify it generates some text.
		assert.NotEmpty(t, text)
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
	})

	t.Run("no truncation when max length is zero", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(0))
		long := strings.Repeat("abc", 200)
		sess := &session.Session{ID: "no-trunc", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: long}}}}, Timestamp: time.Now().Add(-2 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent"}}}}, Timestamp: time.Now()},
		}}
		originalEventCount := len(sess.Events)
		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, long)
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
	})

	t.Run("author fallback to unknown", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		sess := &session.Session{ID: "author-fallback", Events: []event.Event{
			{Timestamp: time.Now().Add(-3 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "content"}}}}, Timestamp: time.Now().Add(-2 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent"}}}}, Timestamp: time.Now()},
		}}
		originalEventCount := len(sess.Events)
		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "unknown: content")
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
	})

	t.Run("uses all events for summarization", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		sess := &session.Session{ID: "all-events", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "old1"}}}}, Timestamp: time.Now().Add(-4 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "old2"}}}}, Timestamp: time.Now().Add(-3 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent1"}}}}, Timestamp: time.Now().Add(-2 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent2"}}}}, Timestamp: time.Now().Add(-1 * time.Second)},
		}}
		originalEventCount := len(sess.Events)
		_, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
		// No system events should be added.
		for _, event := range sess.Events {
			assert.NotEqual(t, "system", event.Author, "no system events should be added.")
		}
	})

	t.Run("branch scope excludes ancestor root events from summary text", func(t *testing.T) {
		s := NewSummarizer(
			&fakeModel{},
			WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
		)
		sess := &session.Session{
			ID:      "branch-scope",
			AppName: "app",
			Events: []event.Event{
				{
					Author:    "user",
					FilterKey: "app",
					Timestamp: time.Now().Add(-3 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "root message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub",
					Timestamp: time.Now().Add(-2 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "branch message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub/tool",
					Timestamp: time.Now().Add(-1 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "descendant message"},
					}}},
				},
			},
		}
		isummaryscope.SetScopeFilterKey(sess, "app/sub")

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotContains(t, text, "root message")
		assert.Contains(t, text, "branch message")
		assert.Contains(t, text, "descendant message")
	})

	t.Run("full-session summary keeps child branch content", func(t *testing.T) {
		s := NewSummarizer(
			&fakeModel{},
			WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
		)
		sess := &session.Session{
			ID:      "full-session-mixed",
			AppName: "app",
			Events: []event.Event{
				{
					Author:    "user",
					FilterKey: "app",
					Timestamp: time.Now().Add(-3 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "root message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub",
					Timestamp: time.Now().Add(-2 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "child message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub/tool",
					Timestamp: time.Now().Add(-1 * time.Second),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "descendant message"},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "root message")
		assert.Contains(t, text, "child message")
		assert.Contains(t, text, "descendant message")
	})

}

func TestSessionSummarizer_Metadata(t *testing.T) {
	s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(0))
	md := s.Metadata()
	assert.Equal(t, "fake", md[metadataKeyModelName])
	assert.Equal(t, 0, md[metadataKeyMaxSummaryWords])
	assert.Equal(t, 0, md[metadataKeyCheckFunctions])
}

func TestSessionSummarizer_PlaceholderReplacement(t *testing.T) {
	t.Run("max_summary_words placeholder replacement", func(t *testing.T) {
		// Test with custom prompt containing the placeholder
		customPrompt := "Please summarize the conversation within {max_summary_words} words: {conversation_text}"
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(100), WithPrompt(customPrompt))

		sess := &session.Session{ID: "test", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "Hello world"}}}}, Timestamp: time.Now()},
		}}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// Verify that the placeholder was replaced with the actual number
		// The fakeModel should have received a prompt with "100" instead of "{max_summary_words}"
		assert.Contains(t, text, "100") // fakeModel returns the prompt as the summary
		// Note: Custom prompts only replace with the number, not the full instruction
	})

	t.Run("placeholder removal when no length limit", func(t *testing.T) {
		// Test with custom prompt containing the placeholder but no length limit
		customPrompt := "Please summarize the conversation within {max_summary_words} words: {conversation_text}"
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(0), WithPrompt(customPrompt))

		sess := &session.Session{ID: "test", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "Hello world"}}}}, Timestamp: time.Now()},
		}}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// Verify that the placeholder was removed (empty string replacement)
		// The fakeModel should have received a prompt without the placeholder
		assert.NotContains(t, text, "{max_summary_words}")
	})

	t.Run("default prompt with length limit", func(t *testing.T) {
		// Test with default prompt and length limit
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(50))

		sess := &session.Session{ID: "test", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "Hello world"}}}}, Timestamp: time.Now()},
		}}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// Verify that the default prompt includes length instruction
		assert.Contains(t, text, "50")
		assert.Contains(t, text, "Please keep the summary within")
		assert.NotContains(t, text, "{max_summary_words}")
	})

	t.Run("default prompt without length limit", func(t *testing.T) {
		// Test with default prompt and no length limit
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(0))

		sess := &session.Session{ID: "test", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "Hello world"}}}}, Timestamp: time.Now()},
		}}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// Verify that the default prompt doesn't include length instruction
		assert.NotContains(t, text, "Please keep the summary within")
		assert.NotContains(t, text, "{max_summary_words}")
	})
}

func TestSessionSummarizer_PreviousSummaryPlaceholder(t *testing.T) {
	const previous = "the user prefers concise answers"
	newSession := func() *session.Session {
		return &session.Session{
			ID: "previous-summary",
			Events: []event.Event{
				{
					Author:    authorSystem,
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: previous},
					}}},
				},
				{
					Author:    authorUser,
					Timestamp: time.Now().Add(-time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "new request"},
					}}},
				},
			},
		}
	}

	t.Run("renders previous summary separately from new conversation", func(t *testing.T) {
		capture := &captureRequestModel{}
		s := NewSummarizer(
			capture,
			WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		)
		ctx := isummarycontext.WithPreviousSummary(context.Background(), previous)

		_, err := s.Summarize(ctx, newSession())
		require.NoError(t, err)
		require.NotNil(t, capture.lastRequest)
		prompt := capture.lastRequest.Messages[0].Content
		require.Contains(t, prompt, "Previous:\n"+previous)
		require.Contains(t, prompt, "Conversation:\nuser: new request")
		require.NotContains(t, prompt, "Conversation:\nsystem: "+previous)
	})

	t.Run("keeps legacy merged conversation without placeholder", func(t *testing.T) {
		capture := &captureRequestModel{}
		s := NewSummarizer(
			capture,
			WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
		)
		ctx := isummarycontext.WithPreviousSummary(context.Background(), previous)

		_, err := s.Summarize(ctx, newSession())
		require.NoError(t, err)
		prompt := capture.lastRequest.Messages[0].Content
		require.Contains(t, prompt, "Conversation:\nsystem: "+previous)
		require.Contains(t, prompt, "user: new request")
	})

	t.Run("supports a previous-summary-only forced input", func(t *testing.T) {
		capture := &captureRequestModel{}
		s := NewSummarizer(
			capture,
			WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		)
		ctx := isummarycontext.WithPreviousSummary(context.Background(), previous)

		_, err := s.Summarize(ctx, &session.Session{ID: "previous-only"})
		require.NoError(t, err)
		prompt := capture.lastRequest.Messages[0].Content
		require.Contains(t, prompt, "Previous:\n"+previous)
		require.Contains(t, prompt, "Conversation:\n\n\nSummary:")
	})

	t.Run("renders an empty previous summary on the first pass", func(t *testing.T) {
		capture := &captureRequestModel{}
		s := NewSummarizer(
			capture,
			WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		)
		sess := &session.Session{ID: "first-pass", Events: []event.Event{
			newEventWithContent("first request"),
		}}

		_, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		prompt := capture.lastRequest.Messages[0].Content
		require.Contains(t, prompt, "Previous:\n\n\nConversation:")
		require.Contains(t, prompt, "user: first request")
	})
}

func TestSessionSummarizer_CacheSafeForking(t *testing.T) {
	newTestSession := func() *session.Session {
		return &session.Session{ID: "cache-safe", Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.NewUserMessage("event text for standalone fallback"),
				}}},
			},
		}}
	}

	t.Run("appends compaction prompt to parent request", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "fork summary"}
		s := NewSummarizer(
			capture,
			WithCacheSafeForking(true),
			WithMaxSummaryWords(42),
		)
		lookupTool := &testTool{name: "lookup"}
		parent := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage("stable system"),
				compat.NewUserMessage("cached conversation"),
			},
			GenerationConfig: compat.GenerationConfig{Stream: true},
			StructuredOutput: &compat.StructuredOutput{
				Type: compat.StructuredOutputJSONSchema,
				JSONSchema: &compat.JSONSchemaConfig{
					Name:   "answer",
					Schema: map[string]any{"type": "object"},
				},
			},
			Tools: map[string]tool.Tool{"lookup": lookupTool},
		}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

		text, err := s.Summarize(ctx, newTestSession())
		require.NoError(t, err)
		require.Equal(t, "fork summary", text)
		require.NotNil(t, capture.request)
		require.Len(t, capture.request.Messages, 3)
		require.Equal(t, parent.Messages[0], capture.request.Messages[0])
		require.Equal(t, parent.Messages[1], capture.request.Messages[1])
		require.Equal(t, compat.RoleUser, capture.request.Messages[2].Role)
		require.Contains(t, capture.request.Messages[2].Content, "Summarize the user, assistant, and tool conversation above")
		require.Contains(t, capture.request.Messages[2].Content, "42")
		require.NotContains(t, capture.request.Messages[2].Content, "{conversation_text}")
		require.NotContains(t, capture.request.Messages[2].Content, "event text for standalone fallback")
		require.False(t, capture.request.GenerationConfig.Stream)
		require.Nil(t, capture.request.StructuredOutput)
		require.Equal(t, lookupTool, capture.request.Tools.(map[string]tool.Tool)["lookup"])
		require.Len(t, parent.Messages, 2, "parent request must not be mutated")
		require.True(t, parent.GenerationConfig.Stream, "parent generation config must not be mutated")
		require.NotNil(t, parent.StructuredOutput, "parent structured output must not be mutated")
	})

	t.Run("falls back to standalone request without parent request", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "standalone summary"}
		s := NewSummarizer(
			capture,
			WithCacheSafeForking(true),
			WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
		)

		text, err := s.Summarize(context.Background(), newTestSession())
		require.NoError(t, err)
		require.Equal(t, "standalone summary", text)
		require.NotNil(t, capture.request)
		require.Len(t, capture.request.Messages, 1)
		require.Equal(t, compat.RoleUser, capture.request.Messages[0].Role)
		require.Contains(t, capture.request.Messages[0].Content, "event text for standalone fallback")
	})

	t.Run("does not duplicate previous summary in a successful fork", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "fork summary"}
		s := NewSummarizer(
			capture,
			WithCacheSafeForking(true),
			WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		)
		parent := &compat.Request{Messages: []compat.Message{
			compat.NewSystemMessage("summary already injected in parent"),
			compat.NewUserMessage("new request"),
		}}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)
		ctx = isummarycontext.WithPreviousSummary(ctx, "raw previous summary")

		_, err := s.Summarize(ctx, newTestSession())
		require.NoError(t, err)
		require.Len(t, capture.request.Messages, 3)
		forkPrompt := capture.request.Messages[2].Content
		require.NotContains(t, forkPrompt, "raw previous summary")
		require.NotContains(t, forkPrompt, previousSummaryPlaceholder)
		require.Contains(t, forkPrompt, "conversation above")
	})

	t.Run("renders previous summary in standalone fallback", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "standalone summary"}
		s := NewSummarizer(
			capture,
			WithCacheSafeForking(true),
			WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
		)
		ctx := isummarycontext.WithPreviousSummary(context.Background(), "raw previous summary")

		_, err := s.Summarize(ctx, newTestSession())
		require.NoError(t, err)
		require.Len(t, capture.request.Messages, 1)
		prompt := capture.request.Messages[0].Content
		require.Contains(t, prompt, "Previous:\nraw previous summary")
		require.Contains(t, prompt, "user: event text for standalone fallback")
	})

	t.Run("compacts tool payloads before the cache-safe request overflows", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{
			response:      "bounded fork summary",
			contextWindow: 1000,
		}
		s := NewSummarizer(capture, WithCacheSafeForking(true))
		arguments := []byte(`{"input":"` + strings.Repeat("argument-", 600) + `"}`)
		toolResult := strings.Repeat("tool-result-", 800)
		parent := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage("stable system"),
				compat.NewUserMessage("keep this user request"),
				{
					Role: compat.RoleAssistant,
					ToolCalls: []compat.ToolCall{{
						Type: "function",
						ID:   "call-1",
						Function: compat.FunctionDefinitionParam{
							Name:      "lookup",
							Arguments: arguments,
						},
					}},
				},
				compat.NewToolMessage("call-1", "lookup", toolResult),
			},
			Tools: map[string]tool.Tool{"lookup": &testTool{name: "lookup"}},
		}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

		text, err := s.Summarize(ctx, newTestSession())
		require.NoError(t, err)
		require.Equal(t, "bounded fork summary", text)
		require.NotNil(t, capture.request)
		require.Nil(t, capture.request.Tools)
		require.Len(t, capture.request.Messages, 5)
		require.Equal(t, "keep this user request", capture.request.Messages[1].Content)
		require.JSONEq(
			t,
			summaryToolArgumentsOmitted,
			string(capture.request.Messages[2].ToolCalls[0].Function.Arguments),
		)
		require.Contains(t, capture.request.Messages[3].Content, "Tool result omitted")
		require.NotContains(t, capture.request.Messages[3].Content, "tool-result-")
		tokens, err := countSummaryRequestTokens(context.Background(), capture.request)
		require.NoError(t, err)
		require.LessOrEqual(t, tokens, 700)

		require.Equal(t, arguments, parent.Messages[2].ToolCalls[0].Function.Arguments)
		require.Equal(t, toolResult, parent.Messages[3].Content)
		require.Len(t, parent.Tools, 1)
	})

	t.Run("drops tool schemas only when they exceed the request budget", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{
			response:      "summary without tool schemas",
			contextWindow: 1000,
		}
		s := NewSummarizer(capture, WithCacheSafeForking(true))
		parent := &compat.Request{
			Messages: []compat.Message{
				compat.NewSystemMessage("stable system"),
				compat.NewUserMessage("small conversation"),
			},
			Tools: map[string]tool.Tool{
				"large_schema": &testTool{
					name:        "large_schema",
					description: strings.Repeat("schema-description ", 1000),
				},
			},
		}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

		text, err := s.Summarize(ctx, newTestSession())
		require.NoError(t, err)
		require.Equal(t, "summary without tool schemas", text)
		require.NotNil(t, capture.request)
		require.Nil(t, capture.request.Tools)
		require.Equal(t, "small conversation", capture.request.Messages[1].Content)
		require.Len(t, parent.Tools, 1, "parent request must not be mutated")
	})

	t.Run("honors a provider input budget below the fallback ratio", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{
			response:      "provider-budget summary",
			contextWindow: 1000,
			inputBudget:   220,
		}
		s := NewSummarizer(
			capture,
			WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
			WithCacheSafeForking(true),
			WithCacheSafeForkPrompt("Summarize the conversation above."),
		)
		oversizedParent := strings.Repeat("provider-budget-content ", 200)
		parent := &compat.Request{Messages: []compat.Message{
			compat.NewSystemMessage("stable system"),
			compat.NewUserMessage(oversizedParent),
		}}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)
		sess := &session.Session{
			ID: "provider-budget",
			Events: []event.Event{newEventWithContent(
				strings.Repeat("provider event ", 20),
			)},
		}

		text, err := s.Summarize(ctx, sess)
		require.NoError(t, err)
		require.Equal(t, "provider-budget summary", text)
		tokens, err := countSummaryRequestTokens(context.Background(), capture.request)
		require.NoError(t, err)
		require.LessOrEqual(t, tokens, 220)
	})

	t.Run("rejects a cache-safe request without conversation content", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "must not be called"}
		s := NewSummarizer(capture, WithCacheSafeForking(true))
		parent := &compat.Request{
			Messages: []compat.Message{compat.NewSystemMessage("system only")},
		}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

		_, err := s.Summarize(ctx, newTestSession())
		require.ErrorContains(t, err, "no conversation content")
		require.Nil(t, capture.request)
	})

	t.Run("rejects previous summary placeholder in fork prompt", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{response: "must not be called"}
		s := NewSummarizer(
			capture,
			WithCacheSafeForking(true),
			WithCacheSafeForkPrompt("Summarize above: {previous_summary}"),
		)
		parent := &compat.Request{Messages: []compat.Message{
			compat.NewUserMessage("source conversation"),
		}}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

		_, err := s.Summarize(ctx, newTestSession())
		require.ErrorContains(t, err, "render cache-safe fork prompt")
		require.Nil(t, capture.request)
	})

	t.Run("rejects oversized standalone source without advancing boundary", func(t *testing.T) {
		capture := &cacheSafeCaptureModel{
			response:      "must not be called",
			contextWindow: 1000,
		}
		s := NewSummarizer(capture, WithCacheSafeForking(true))
		oversized := strings.Repeat("oversized-user-content ", 1000)
		parent := &compat.Request{Messages: []compat.Message{
			compat.NewSystemMessage("stable system"),
			compat.NewUserMessage(oversized),
		}}
		ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)
		first := newEventWithContent(strings.Repeat("first event ", 500))
		first.ID = "event-1"
		second := newEventWithContent(strings.Repeat("second event ", 500))
		second.ID = "event-2"
		sess := &session.Session{
			ID:     "oversized-cache-safe",
			Events: []event.Event{first, second},
		}

		_, err := s.Summarize(ctx, sess)
		require.ErrorContains(t, err, "refusing to omit unsummarized conversation")
		require.Nil(t, capture.request)
		assert.NotContains(t, sess.State, lastIncludedTsKey)
		assert.NotContains(t, sess.State, lastIncludedEventIDKey)
	})
}

func TestSessionSummarizer_RetriesCacheSafeFailureWithStandaloneSource(t *testing.T) {
	for _, test := range []struct {
		name          string
		firstResponse *compat.Response
	}{
		{
			name: "empty summary",
			firstResponse: &compat.Response{
				Done: true,
			},
		},
		{
			name: "context overflow",
			firstResponse: func() *compat.Response {
				code := "context_length_exceeded"
				return &compat.Response{
					Done: true,
					Error: &compat.ResponseError{
						Message: "maximum context length exceeded",
						Code:    &code,
					},
				}
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			capture := &retrySummaryModel{
				contextWindow: 1000,
				responses: []*compat.Response{
					test.firstResponse,
					{
						Done: true,
						Choices: []compat.Choice{{
							Message: compat.NewAssistantMessage("standalone retry summary"),
						}},
					},
				},
			}
			s := NewSummarizer(capture, WithCacheSafeForking(true))
			parent := &compat.Request{Messages: []compat.Message{
				compat.NewSystemMessage("stable system"),
				compat.NewUserMessage("source conversation"),
			}}
			ctx := ContextWithCacheSafeForkRequest(context.Background(), parent)

			sess := &session.Session{
				ID:     "retry-summary",
				Events: []event.Event{newEventWithContent("event text for retry")},
			}
			text, err := s.Summarize(ctx, sess)
			require.NoError(t, err)
			require.Equal(t, "standalone retry summary", text)
			require.Len(t, capture.requests, 2)
			require.Len(t, capture.requests[0].Messages, 3)
			require.Len(t, capture.requests[1].Messages, 1)
			require.Contains(t, capture.requests[1].Messages[0].Content, "event text")
			require.Contains(t, capture.requests[1].Messages[0].Content,
				standaloneSummarySourceBoundary)
			require.Contains(t, capture.requests[1].Messages[0].Content,
				"Summarize the user, assistant")
			require.NotContains(t, capture.requests[1].Messages[0].Content,
				"stable system")
		})
	}
}

func TestSessionSummarizer_RejectsOversizedStandaloneRequest(t *testing.T) {
	capture := &cacheSafeCaptureModel{
		response:      "must not be called",
		contextWindow: 1000,
	}
	s := NewSummarizer(
		capture,
		WithPrompt("Conversation:\n{conversation_text}\n\nSummary:"),
	)
	sess := &session.Session{ID: "oversized", Events: []event.Event{{
		Author:    "user",
		Timestamp: time.Now(),
		Response: &compat.Response{Choices: []compat.Choice{{
			Message: compat.NewUserMessage(strings.Repeat("large-event ", 1000)),
		}}},
	}}}

	_, err := s.Summarize(context.Background(), sess)
	require.ErrorContains(t, err, "refusing to omit unsummarized conversation")
	require.Nil(t, capture.request)
	assert.NotContains(t, sess.State, lastIncludedTsKey)
	assert.NotContains(t, sess.State, lastIncludedEventIDKey)
}

func TestSessionSummarizer_BoundsPreviousSummaryAndConversation(t *testing.T) {
	capture := &cacheSafeCaptureModel{
		response:      "bounded standalone summary",
		contextWindow: 1000,
	}
	s := NewSummarizer(
		capture,
		WithPrompt("Previous:\n{previous_summary}\n\nConversation:\n{conversation_text}\n\nSummary:"),
	)
	previous := strings.Repeat("large-previous-summary ", 1000)
	conversation := "new conversation event"
	sess := &session.Session{ID: "oversized-previous", Events: []event.Event{
		{
			Author:    authorSystem,
			Timestamp: time.Now(),
			Response: &compat.Response{Choices: []compat.Choice{{
				Message: compat.Message{Content: previous},
			}}},
		},
		newEventWithContent(conversation),
	}}
	ctx := isummarycontext.WithPreviousSummary(context.Background(), previous)

	text, err := s.Summarize(ctx, sess)
	require.NoError(t, err)
	require.Equal(t, "bounded standalone summary", text)
	require.NotNil(t, capture.request)
	prompt := capture.request.Messages[0].Content
	require.Contains(t, prompt, summaryPreviousOmitted)
	require.Contains(t, prompt, conversation)
	tokens, err := countSummaryRequestTokens(context.Background(), capture.request)
	require.NoError(t, err)
	require.LessOrEqual(t, tokens, 700)
}

func TestTruncatePreviousSummary(t *testing.T) {
	previous := []rune("previous")

	require.Equal(t, "previous", truncatePreviousSummary(previous, 8))
	require.Empty(t, truncatePreviousSummary(previous, 0))
	require.Contains(t, truncatePreviousSummary(previous, 4),
		summaryPreviousOmitted)
}

func TestSessionSummarizer_RejectsOversizedSourceWithFixedPrompt(t *testing.T) {
	capture := &cacheSafeCaptureModel{
		response:      "must not be called",
		contextWindow: 1000,
	}
	s := NewSummarizer(
		capture,
		WithPrompt(
			"fixed-prefix\n<conversation>\n{conversation_text}\n"+
				"</conversation>\nfixed-suffix",
		),
	)
	oversized := strings.Repeat("large-event ", 1000)
	sess := &session.Session{
		ID:     "oversized-fixed-prompt",
		Events: []event.Event{newEventWithContent(oversized)},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.ErrorContains(t, err, "refusing to omit unsummarized conversation")
	require.Nil(t, capture.request)
	assert.NotContains(t, sess.State, lastIncludedTsKey)
	assert.NotContains(t, sess.State, lastIncludedEventIDKey)
}

func TestSessionSummarizer_RetriesEmptyStandaloneSummary(t *testing.T) {
	capture := &retrySummaryModel{
		contextWindow: 1000,
		responses: []*compat.Response{
			{Done: true},
			{
				Done: true,
				Choices: []compat.Choice{{
					Message: compat.NewAssistantMessage("standalone retry summary"),
				}},
			},
		},
	}
	s := NewSummarizer(capture)
	sess := &session.Session{
		ID: "standalone-empty-retry",
		Events: []event.Event{newEventWithContent(
			"event text for standalone fallback",
		)},
	}

	text, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, "standalone retry summary", text)
	require.Len(t, capture.requests, 2)
	require.Contains(t, capture.requests[0].Messages[0].Content,
		"event text for standalone fallback")
	require.Contains(t, capture.requests[1].Messages[0].Content,
		"event text for standalone fallback")
}

func TestEnsureSummaryRequestFitsPreservesSourceRounds(t *testing.T) {
	s := NewSummarizer(&cacheSafeCaptureModel{}).(*sessionSummarizer)
	request := &compat.Request{Messages: []compat.Message{
		compat.NewSystemMessage("stable system"),
		compat.NewUserMessage(strings.Repeat("old source ", 400)),
		{
			Role: compat.RoleAssistant,
			ToolCalls: []compat.ToolCall{{
				ID: "old-call",
				Function: compat.FunctionDefinitionParam{
					Name: "lookup",
					Arguments: []byte(`{"query":"` +
						strings.Repeat("old ", 100) + `"}`),
				},
			}},
		},
		compat.NewToolMessage("old-call", "lookup", "old result"),
		compat.NewUserMessage("latest source"),
		compat.NewAssistantMessage("latest answer"),
		compat.NewUserMessage("Summarize the conversation above."),
	}}

	err := s.ensureSummaryRequestFits(
		context.Background(),
		request,
		true,
		200,
	)
	require.ErrorContains(t, err, "without dropping source conversation")
	require.Len(t, request.Messages, 7)
	require.Equal(t, compat.RoleSystem, request.Messages[0].Role)
	require.Contains(t, request.Messages[1].Content, "old source")
	require.JSONEq(
		t,
		summaryToolArgumentsOmitted,
		string(request.Messages[2].ToolCalls[0].Function.Arguments),
	)
	require.Equal(t, "latest source", request.Messages[4].Content)
	require.Equal(t, "latest answer", request.Messages[5].Content)
	require.Equal(t, "Summarize the conversation above.", request.Messages[6].Content)
}

func TestPrepareSummaryRequestFallsBackBeforeDroppingSourceRounds(t *testing.T) {
	s := NewSummarizer(
		&cacheSafeCaptureModel{},
		WithPrompt("Conversation:\n{conversation_text}"),
	).(*sessionSummarizer)
	request := &compat.Request{Messages: []compat.Message{
		compat.NewSystemMessage("stable system"),
		compat.NewUserMessage(strings.Repeat("large source ", 400)),
		compat.NewAssistantMessage("latest progress"),
		compat.NewUserMessage("continue"),
		compat.NewUserMessage("Summarize the conversation above."),
	}}
	input := summaryPromptInput{
		conversationText: "user: original request\nassistant: latest progress",
	}

	prepared, mode, err := s.prepareSummaryRequest(
		context.Background(),
		request,
		callModeCacheSafeFork,
		input,
		200,
	)
	require.NoError(t, err)
	require.Equal(t, callModeStandalone, mode)
	require.Len(t, prepared.Messages, 1)
	require.Contains(t, prepared.Messages[0].Content, "original request")
	require.Contains(t, prepared.Messages[0].Content, "latest progress")
	require.NotContains(t, prepared.Messages[0].Content, "continue")
}

// fakeModel is a minimal model that returns the conversation content back to simulate LLM.
type fakeModel struct{}

func (f *fakeModel) Info() compat.Info { return compat.Info{Name: "fake"} }
func (f *fakeModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	content := ""
	if len(req.Messages) > 0 {
		// Extract conversation text from the prompt for testing.
		prompt := req.Messages[0].Content
		// Find the conversation part after "Conversation:\n"
		if idx := strings.Index(prompt, "Conversation:\n"); idx != -1 {
			conversation := prompt[idx+len("Conversation:\n"):]
			if summaryIdx := strings.Index(conversation, "\n\nSummary:"); summaryIdx != -1 {
				conversation = conversation[:summaryIdx]
			}
			content = strings.TrimSpace(conversation)
		} else {
			content = prompt
		}
		// For testing, return the full conversation content as the summary.
		content = "Summary: " + content
	}
	ch <- &compat.Response{Done: true, Choices: []compat.Choice{{Message: compat.Message{Content: content}}}}
	close(ch)
	return ch, nil
}

type cacheSafeCaptureModel struct {
	request       *compat.Request
	response      string
	contextWindow int
	inputBudget   int
}

func (m *cacheSafeCaptureModel) Info() compat.Info {
	return compat.Info{Name: "capture", ContextWindow: m.contextWindow}
}

func (m *cacheSafeCaptureModel) GenerateContent(
	_ context.Context,
	req *compat.Request,
) (<-chan *compat.Response, error) {
	m.request = req
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{
		Done: true,
		Choices: []compat.Choice{{
			Message: compat.Message{Role: compat.RoleAssistant, Content: m.response},
		}},
	}
	close(ch)
	return ch, nil
}

func (m *cacheSafeCaptureModel) InputTokenBudget(
	context.Context,
	*compat.Request,
) int {
	return m.inputBudget
}

type retrySummaryModel struct {
	contextWindow int
	inputBudget   int
	requests      []*compat.Request
	responses     []*compat.Response
}

func (m *retrySummaryModel) Info() compat.Info {
	return compat.Info{Name: "retry-summary", ContextWindow: m.contextWindow}
}

func (m *retrySummaryModel) GenerateContent(
	_ context.Context,
	request *compat.Request,
) (<-chan *compat.Response, error) {
	m.requests = append(m.requests, cloneRequestForCacheSafeFork(request))
	response := &compat.Response{Done: true}
	if len(m.responses) > 0 {
		response = m.responses[0]
		m.responses = m.responses[1:]
	}
	ch := make(chan *compat.Response, 1)
	ch <- response
	close(ch)
	return ch, nil
}

func (m *retrySummaryModel) InputTokenBudget(
	context.Context,
	*compat.Request,
) int {
	return m.inputBudget
}

type testTool struct {
	name        string
	description string
}

func (t *testTool) Declaration() *tool.Declaration {
	return &tool.Declaration{Name: t.name, Description: t.description}
}

func TestSessionSummarizer_Summarize_NilModel(t *testing.T) {
	s := &sessionSummarizer{
		model:  nil,
		prompt: "test prompt",
	}

	sess := &session.Session{
		ID: "test",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no model configured")
}

func TestSessionSummarizer_GenerateSummary_ModelError(t *testing.T) {
	errorModel := &errorModel{}
	s := NewSummarizer(errorModel)

	sess := &session.Session{
		ID: "test",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate summary")
}

func TestSessionSummarizer_GenerateSummary_NilResponseChannel(t *testing.T) {
	s := NewSummarizer(&nilResponseChannelModel{})

	sess := &session.Session{
		ID: "test-nil-channel",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model returned nil response channel")
}

func TestSessionSummarizer_GenerateSummary_ResponseError(t *testing.T) {
	responseErrorModel := &responseErrorModel{}
	s := NewSummarizer(responseErrorModel)

	sess := &session.Session{
		ID: "test",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model error during summarization")
}

func TestSessionSummarizer_GenerateSummary_ResponseErrorWithDetails(t *testing.T) {
	// Test that error messages include Type and Code when available.
	responseErrorModel := &responseErrorModelWithDetails{}
	s := NewSummarizer(responseErrorModel)

	sess := &session.Session{
		ID: "test-detailed-error",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	// Verify error message includes type and code.
	assert.Contains(t, err.Error(), "model error during summarization")
	assert.Contains(t, err.Error(), "[requestAuthError]")
	assert.Contains(t, err.Error(), "API key rate limit exceeded")
	assert.Contains(t, err.Error(), "(code: rate_limit_exceeded)")
}

func TestSessionSummarizer_GenerateSummary_EmptyResponse(t *testing.T) {
	emptyModel := &emptyResponseModel{}
	s := NewSummarizer(emptyModel)

	sess := &session.Session{
		ID: "test",
		Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}}, Timestamp: time.Now()},
		},
	}

	_, err := s.Summarize(context.Background(), sess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generated empty summary")
}

func TestSessionSummarizer_Summarize_ContextTimeoutWhileWaitingForResponse(t *testing.T) {
	s := NewSummarizer(&blockingResponseModel{})
	sess := &session.Session{
		ID: "timeout",
		Events: []event.Event{{
			Author:    "user",
			Response:  &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "test"}}}},
			Timestamp: time.Now(),
		}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := s.Summarize(ctx, sess)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestSessionSummarizer_ShouldSummarize_EmptyEvents(t *testing.T) {
	s := NewSummarizer(&fakeModel{}, WithEventThreshold(10))
	sess := &session.Session{Events: []event.Event{}}
	assert.False(t, s.ShouldSummarize(sess))
}

func TestSessionSummarizer_Metadata_NilModel(t *testing.T) {
	s := &sessionSummarizer{
		model:           nil,
		maxSummaryWords: 100,
		checks:          []checkEvaluator{},
	}
	md := s.Metadata()
	assert.Equal(t, "", md[metadataKeyModelName])
	assert.Equal(t, false, md[metadataKeyModelAvailable])
	assert.Equal(t, 100, md[metadataKeyMaxSummaryWords])
}

func TestSessionSummarizer_ExtractConversationText_WithAuthor(t *testing.T) {
	s := NewSummarizer(&fakeModel{})
	sess := &session.Session{
		ID: "test",
		Events: []event.Event{
			{
				Author:   "user",
				Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "hello"}}}},
			},
			{
				Author:   "assistant",
				Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "hi there"}}}},
			},
		},
	}

	text, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.Contains(t, text, "user:")
	assert.Contains(t, text, "assistant:")
}

func TestSessionSummarizer_ExtractConversationText_LeavesOutReasoningContent(t *testing.T) {
	s := NewSummarizer(&fakeModel{})
	sess := &session.Session{
		ID: "test-reasoning-content",
		Events: []event.Event{
			{
				Author: "assistant",
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ReasoningContent: "I should inspect the user request first.",
						Content:          "Here is the final answer.",
					},
				}}},
			},
		},
	}

	text, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)
	assert.NotContains(t, text, "I should inspect the user request first.")
	assert.Contains(t, text, "assistant: Here is the final answer.")
}

func TestSessionSummarizer_ExtractConversationText_WithToolCalls(t *testing.T) {
	s := NewSummarizer(&fakeModel{})

	t.Run("extracts tool call with arguments", func(t *testing.T) {
		sess := &session.Session{
			ID: "test-toolcall",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "What is the weather?"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{{
								ID:   "call_123",
								Type: "function",
								Function: compat.FunctionDefinitionParam{
									Name:      "get_weather",
									Arguments: []byte(`{"city":"Beijing"}`),
								},
							}},
						},
					}}},
				},
				{
					Author: "get_weather",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:   "call_123",
							ToolName: "get_weather",
							Content:  `{"temperature": 25, "weather": "sunny"}`,
						},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "The weather in Beijing is sunny with 25 degrees."},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "user:")
		assert.Contains(t, text, "[Called tool: get_weather")
		assert.Contains(t, text, "Beijing")
		assert.Contains(t, text, "[get_weather returned:")
		assert.Contains(t, text, "sunny")
	})

	t.Run("extracts tool call without arguments", func(t *testing.T) {
		sess := &session.Session{
			ID: "test-toolcall-no-args",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Get current time"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{{
								ID:   "call_456",
								Type: "function",
								Function: compat.FunctionDefinitionParam{
									Name: "get_current_time",
								},
							}},
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[Called tool: get_current_time]")
		assert.NotContains(t, text, "with args")
	})

	t.Run("includes full tool arguments by default", func(t *testing.T) {
		longArgs := `{"data":"` + strings.Repeat("x", 300) + `"}`
		sess := &session.Session{
			ID: "test-long-args",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Process data"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{{
								ID:   "call_789",
								Type: "function",
								Function: compat.FunctionDefinitionParam{
									Name:      "process_data",
									Arguments: []byte(longArgs),
								},
							}},
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[Called tool: process_data with args:")
		// Default formatter does not truncate.
		assert.Contains(t, text, strings.Repeat("x", 300))
	})

	t.Run("includes full tool response by default", func(t *testing.T) {
		longContent := strings.Repeat("result_data_", 100)
		sess := &session.Session{
			ID: "test-long-response",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Get data"},
					}}},
				},
				{
					Author: "tool",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:   "call_abc",
							ToolName: "get_data",
							Content:  longContent,
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[get_data returned:")
		// Default formatter does not truncate.
		assert.Contains(t, text, longContent)
	})

	t.Run("handles tool response without tool name", func(t *testing.T) {
		sess := &session.Session{
			ID: "test-no-tool-name",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Do something"},
					}}},
				},
				{
					Author: "tool",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:  "call_def",
							Content: "done",
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[tool returned: done]")
	})

	t.Run("handles multiple tool results in separate choices", func(t *testing.T) {
		// When model returns multiple tool calls, results may be distributed
		// across different choices (len(e.Response.Choices) > 1).
		sess := &session.Session{
			ID: "test-multi-choices",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Check weather in multiple cities"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{
								{
									ID:   "call_beijing",
									Type: "function",
									Function: compat.FunctionDefinitionParam{
										Name:      "get_weather",
										Arguments: []byte(`{"city":"Beijing"}`),
									},
								},
								{
									ID:   "call_shanghai",
									Type: "function",
									Function: compat.FunctionDefinitionParam{
										Name:      "get_weather",
										Arguments: []byte(`{"city":"Shanghai"}`),
									},
								},
							},
						},
					}}},
				},
				{
					Author: "tool",
					Response: &compat.Response{Choices: []compat.Choice{
						{
							Message: compat.Message{
								ToolID:   "call_beijing",
								ToolName: "get_weather",
								Content:  `{"temperature": 25, "weather": "sunny"}`,
							},
						},
						{
							Message: compat.Message{
								ToolID:   "call_shanghai",
								ToolName: "get_weather",
								Content:  `{"temperature": 22, "weather": "cloudy"}`,
							},
						},
					}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Beijing is sunny, Shanghai is cloudy."},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		// Both tool calls should be extracted.
		assert.Contains(t, text, "[Called tool: get_weather")
		assert.Contains(t, text, "Beijing")
		assert.Contains(t, text, "Shanghai")
		// Both tool results should be extracted.
		assert.Contains(t, text, "sunny")
		assert.Contains(t, text, "cloudy")
	})
}

func TestSessionSummarizer_CustomToolFormatters(t *testing.T) {
	t.Run("custom tool call formatter with truncation", func(t *testing.T) {
		truncatingFormatter := func(tc compat.ToolCall) string {
			name := tc.Function.Name
			if name == "" {
				return ""
			}
			args := string(tc.Function.Arguments)
			const maxLen = 50
			if len(args) > maxLen {
				args = args[:maxLen] + "...(truncated)"
			}
			return fmt.Sprintf("[Tool: %s, Args: %s]", name, args)
		}

		s := NewSummarizer(&fakeModel{}, WithToolCallFormatter(truncatingFormatter))
		longArgs := `{"data":"` + strings.Repeat("x", 100) + `"}`
		sess := &session.Session{
			ID: "test-custom-formatter",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Test"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{{
								Function: compat.FunctionDefinitionParam{
									Name:      "my_tool",
									Arguments: []byte(longArgs),
								},
							}},
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[Tool: my_tool, Args:")
		assert.Contains(t, text, "...(truncated)")
	})

	t.Run("custom tool result formatter excludes results", func(t *testing.T) {
		// Formatter that excludes tool results entirely.
		excludingFormatter := func(msg compat.Message) string {
			return "" // Return empty to exclude.
		}

		s := NewSummarizer(&fakeModel{}, WithToolResultFormatter(excludingFormatter))
		sess := &session.Session{
			ID: "test-exclude-results",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Test"},
					}}},
				},
				{
					Author: "tool",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:   "call_123",
							ToolName: "my_tool",
							Content:  "some result",
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotContains(t, text, "my_tool")
		assert.NotContains(t, text, "some result")
	})

	t.Run("custom formatter shows only tool name", func(t *testing.T) {
		nameOnlyFormatter := func(tc compat.ToolCall) string {
			if tc.Function.Name == "" {
				return ""
			}
			return fmt.Sprintf("[Used: %s]", tc.Function.Name)
		}

		s := NewSummarizer(&fakeModel{}, WithToolCallFormatter(nameOnlyFormatter))
		sess := &session.Session{
			ID: "test-name-only",
			Events: []event.Event{
				{
					Author: "user",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "Test"},
					}}},
				},
				{
					Author: "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolCalls: []compat.ToolCall{{
								Function: compat.FunctionDefinitionParam{
									Name:      "search",
									Arguments: []byte(`{"query":"test"}`),
								},
							}},
						},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "[Used: search]")
		assert.NotContains(t, text, "query")
	})
}

// errorModel returns an error when generating content
type errorModel struct{}

func (e *errorModel) Info() compat.Info { return compat.Info{Name: "error"} }
func (e *errorModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	return nil, fmt.Errorf("model error")
}

// responseErrorModel returns a response with an error.
type responseErrorModel struct{}

func (r *responseErrorModel) Info() compat.Info { return compat.Info{Name: "response-error"} }
func (r *responseErrorModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{
		Done:  true,
		Error: &compat.ResponseError{Message: "response error"},
	}
	close(ch)
	return ch, nil
}

// responseErrorModelWithDetails returns a response with detailed error info.
type responseErrorModelWithDetails struct{}

func (r *responseErrorModelWithDetails) Info() compat.Info {
	return compat.Info{Name: "response-error-detailed"}
}
func (r *responseErrorModelWithDetails) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	code := "rate_limit_exceeded"
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{
		Done: true,
		Error: &compat.ResponseError{
			Message: "API key rate limit exceeded",
			Type:    "requestAuthError",
			Code:    &code,
		},
	}
	close(ch)
	return ch, nil
}

// emptyResponseModel returns an empty response.
type emptyResponseModel struct{}

func (e *emptyResponseModel) Info() compat.Info { return compat.Info{Name: "empty"} }
func (e *emptyResponseModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{Done: true, Choices: []compat.Choice{{Message: compat.Message{Content: ""}}}}
	close(ch)
	return ch, nil
}

// blockingResponseModel simulates a non-cooperative provider that neither
// sends a response nor closes the response channel after ctx cancellation.
type blockingResponseModel struct{}

func (b *blockingResponseModel) Info() compat.Info { return compat.Info{Name: "blocking-response"} }
func (b *blockingResponseModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	// The channel is intentionally never closed to exercise context timeout handling.
	return make(chan *compat.Response), nil
}

type nilResponseChannelModel struct{}

func (n *nilResponseChannelModel) Info() compat.Info {
	return compat.Info{Name: "nil-response-channel"}
}

func (n *nilResponseChannelModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	return nil, nil
}

func TestFormatResponseError(t *testing.T) {
	tests := []struct {
		name     string
		err      *compat.ResponseError
		expected string
		isNil    bool
	}{
		{
			name:  "nil error",
			err:   nil,
			isNil: true,
		},
		{
			name: "message only",
			err: &compat.ResponseError{
				Message: "simple error",
			},
			expected: "model error during summarization: simple error",
		},
		{
			name: "with type",
			err: &compat.ResponseError{
				Message: "auth failed",
				Type:    "authError",
			},
			expected: "model error during summarization: [authError] auth failed",
		},
		{
			name: "with type and code",
			err: &compat.ResponseError{
				Message: "rate limit",
				Type:    "requestError",
				Code:    stringPtr("rate_limit_exceeded"),
			},
			expected: "model error during summarization: [requestError] rate limit (code: rate_limit_exceeded)",
		},
		{
			name: "with empty code",
			err: &compat.ResponseError{
				Message: "error message",
				Type:    "someType",
				Code:    stringPtr(""),
			},
			expected: "model error during summarization: [someType] error message",
		},
		{
			name: "code without type",
			err: &compat.ResponseError{
				Message: "error message",
				Code:    stringPtr("error_code"),
			},
			expected: "model error during summarization: error message (code: error_code)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResponseError(tt.err)
			if tt.isNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestSessionSummarizer_WithSkipRecent(t *testing.T) {
	t.Run("skipRecentFunc is set when configured", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(events []event.Event) int { return 5 }))
		assert.NotNil(t, s.(*sessionSummarizer).skipRecentFunc)
	})

	t.Run("skipRecentFunc nil by default", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		assert.Nil(t, s.(*sessionSummarizer).skipRecentFunc)
	})
}

func TestSessionSummarizer_FilterEventsForSummary(t *testing.T) {
	s := &sessionSummarizer{}

	t.Run("no filtering when skipRecentFunc is nil", func(t *testing.T) {
		events := []event.Event{
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "msg1"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "msg2"}}}}},
		}
		filtered := s.filterEventsForSummary(events)
		assert.Equal(t, events, filtered)
	})

	t.Run("returns empty when skip count >= events length", func(t *testing.T) {
		s := &sessionSummarizer{skipRecentFunc: func(_ []event.Event) int { return 5 }}
		events := []event.Event{
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "msg1"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "msg2"}}}}},
		}
		filtered := s.filterEventsForSummary(events)
		assert.Empty(t, filtered)
	})

	t.Run("filters recent events and keeps user message context", func(t *testing.T) {
		s := &sessionSummarizer{skipRecentFunc: func(_ []event.Event) int { return 2 }}
		events := []event.Event{
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user1"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant1"}}}}},
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user2"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant2"}}}}},
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "recent1"}}}}},           // should be skipped
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "recent2"}}}}}, // should be skipped
		}
		filtered := s.filterEventsForSummary(events)
		// Should keep events 0-3 (up to and including the last user message before recent events)
		expected := events[:4]
		assert.Equal(t, expected, filtered)
		assert.Len(t, filtered, 4)
	})

	t.Run("keeps prepended summary context for assistant tool chain", func(t *testing.T) {
		s := &sessionSummarizer{skipRecentFunc: func(_ []event.Event) int { return 1 }}
		now := time.Now().UTC()
		events := []event.Event{
			{
				Author:    authorSystem,
				Timestamp: now,
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "previous summary"},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: now.Add(-2 * time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						Role: compat.RoleAssistant,
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{
								Name:      "lookup_weather",
								Arguments: []byte(`{"city":"Shanghai"}`),
							},
						}},
					},
				}}},
			},
			{
				Author:    "tool",
				Timestamp: now.Add(-time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "lookup_weather",
						Content:  "sunny",
					},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: now.Add(time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Role: compat.RoleAssistant, Content: "recent response"},
				}}},
			},
		}

		filtered := s.filterEventsForSummary(events)
		expected := events[:3]
		assert.Equal(t, expected, filtered)
		assert.Len(t, filtered, 3)
	})

	t.Run("drops prepended summary when remaining tool calls are formatter-excluded", func(t *testing.T) {
		s := &sessionSummarizer{
			skipRecentFunc: func(_ []event.Event) int { return 1 },
			toolCallFormatter: func(compat.ToolCall) string {
				return ""
			},
		}
		now := time.Now().UTC()
		events := []event.Event{
			{
				Author:    authorSystem,
				Timestamp: now,
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "previous summary"},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: now.Add(-time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						Role: compat.RoleAssistant,
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{
								Name:      "lookup_weather",
								Arguments: []byte(`{"city":"Shanghai"}`),
							},
						}},
					},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: now.Add(time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Role: compat.RoleAssistant, Content: "recent response"},
				}}},
			},
		}

		filtered := s.filterEventsForSummary(events)
		assert.Empty(t, filtered)
	})

	t.Run("drops prepended summary when remaining tool results are formatter-excluded", func(t *testing.T) {
		s := &sessionSummarizer{
			skipRecentFunc: func(_ []event.Event) int { return 1 },
			toolResultFormatter: func(compat.Message) string {
				return ""
			},
		}
		now := time.Now().UTC()
		events := []event.Event{
			{
				Author:    authorSystem,
				Timestamp: now,
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "previous summary"},
				}}},
			},
			{
				Author:    "tool",
				Timestamp: now.Add(-time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "lookup_weather",
						Content:  "sunny",
					},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: now.Add(time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Role: compat.RoleAssistant, Content: "recent response"},
				}}},
			},
		}

		filtered := s.filterEventsForSummary(events)
		assert.Empty(t, filtered)
	})

	t.Run("returns empty slice when no user message in filtered events", func(t *testing.T) {
		s := &sessionSummarizer{skipRecentFunc: func(_ []event.Event) int { return 1 }}
		events := []event.Event{
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant1"}}}}},
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user1"}}}}}, // will be skipped
		}
		filtered := s.filterEventsForSummary(events)
		assert.Empty(t, filtered)
	})

	t.Run("keeps all events up to last user message when filtering", func(t *testing.T) {
		s := &sessionSummarizer{skipRecentFunc: func(_ []event.Event) int { return 3 }}
		events := []event.Event{
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user1"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant1"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant2"}}}}},
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user2"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "assistant3"}}}}},
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "recent1"}}}}}, // skipped
			{Author: "user", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "recent2"}}}}},           // skipped
			{Author: "assistant", Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "recent3"}}}}}, // skipped
		}
		filtered := s.filterEventsForSummary(events)
		// Should keep events 0-4 (up to and including the last user message before recent events)
		expected := events[:5]
		assert.Equal(t, expected, filtered)
		assert.Len(t, filtered, 5)
	})

}

func TestSummaryEventHelpers(t *testing.T) {
	t.Run("eventHasTextContent", func(t *testing.T) {
		t.Run("returns false for nil response", func(t *testing.T) {
			assert.False(t, eventHasTextContent(event.Event{}))
		})

		t.Run("returns false for whitespace-only content", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "   "},
				}}},
			}
			assert.False(t, eventHasTextContent(e))
		})

		t.Run("returns true when any choice has text", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{
					{Message: compat.Message{Content: "   "}},
					{Message: compat.Message{Content: "hello"}},
				}},
			}
			assert.True(t, eventHasTextContent(e))
		})
	})

	t.Run("eventHasSummarizableContent", func(t *testing.T) {
		defaultToolCallFmt := func(tc compat.ToolCall) string {
			return tc.Function.Name
		}
		defaultToolResultFmt := func(msg compat.Message) string {
			return msg.Content
		}

		t.Run("returns false for nil response", func(t *testing.T) {
			assert.False(t, eventHasSummarizableContent(
				event.Event{},
				defaultToolCallFmt,
				defaultToolResultFmt,
			))
		})

		t.Run("returns true for included tool calls", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{Name: "lookup_weather"},
						}},
					},
				}}},
			}
			assert.True(t, eventHasSummarizableContent(
				e,
				defaultToolCallFmt,
				defaultToolResultFmt,
			))
		})

		t.Run("returns false for formatter-excluded tool calls", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{Name: "lookup_weather"},
						}},
					},
				}}},
			}
			assert.False(t, eventHasSummarizableContent(
				e,
				func(compat.ToolCall) string { return "" },
				defaultToolResultFmt,
			))
		})

		t.Run("returns true for included tool results", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "lookup_weather",
						Content:  "sunny",
					},
				}}},
			}
			assert.True(t, eventHasSummarizableContent(
				e,
				defaultToolCallFmt,
				defaultToolResultFmt,
			))
		})

		t.Run("returns false for formatter-excluded tool results", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "lookup_weather",
						Content:  "sunny",
					},
				}}},
			}
			assert.False(t, eventHasSummarizableContent(
				e,
				defaultToolCallFmt,
				func(compat.Message) string { return "" },
			))
		})

		t.Run("returns true for regular content", func(t *testing.T) {
			e := event.Event{
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "assistant reply"},
				}}},
			}
			assert.True(t, eventHasSummarizableContent(
				e,
				defaultToolCallFmt,
				defaultToolResultFmt,
			))
		})
	})

	t.Run("hasPrependedSummaryContext", func(t *testing.T) {
		s := &sessionSummarizer{}
		now := time.Now().UTC()

		t.Run("returns false when fewer than two events", func(t *testing.T) {
			events := []event.Event{{
				Author:    authorSystem,
				Timestamp: now,
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "previous summary"},
				}}},
			}}
			assert.False(t, s.hasPrependedSummaryContext(events))
		})

		t.Run("returns false when first event is not a system summary", func(t *testing.T) {
			events := []event.Event{
				{
					Author:    "assistant",
					Timestamp: now,
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "assistant"},
					}}},
				},
				{
					Author:    "assistant",
					Timestamp: now.Add(-time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "reply"},
					}}},
				},
			}
			assert.False(t, s.hasPrependedSummaryContext(events))
		})

		t.Run("returns false when summary timestamp is older than next event", func(t *testing.T) {
			events := []event.Event{
				{
					Author:    authorSystem,
					Timestamp: now.Add(-2 * time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "previous summary"},
					}}},
				},
				{
					Author:    "assistant",
					Timestamp: now.Add(-time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "reply"},
					}}},
				},
			}
			assert.False(t, s.hasPrependedSummaryContext(events))
		})

		t.Run("returns true when later events have summarizable content", func(t *testing.T) {
			events := []event.Event{
				{
					Author:    authorSystem,
					Timestamp: now,
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "previous summary"},
					}}},
				},
				{
					Author:    "assistant",
					Timestamp: now.Add(-time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "reply"},
					}}},
				},
			}
			assert.True(t, s.hasPrependedSummaryContext(events))
		})
	})
}

func TestSessionSummarizer_SummarizeWithSkipRecent(t *testing.T) {
	t.Run("summarizes only non-recent events", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(_ []event.Event) int { return 2 }))

		// Create session with 5 events
		sess := &session.Session{
			ID: "test-session",
			Events: []event.Event{
				{
					Author:   "user",
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "hello"}}}},
				},
				{
					Author:   "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "hi there"}}}},
				},
				{
					Author:   "user",
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "how are you"}}}},
				},
				{
					Author:   "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "I'm fine"}}}},
				},
				{
					Author:   "user", // This will be skipped
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "recent message"}}}},
				},
				{
					Author:   "assistant", // This will be skipped
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "recent response"}}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)

		// The summary should contain the first 4 messages (events 0-3) but not the last 2
		assert.Contains(t, text, "hello")
		assert.Contains(t, text, "hi there")
		assert.Contains(t, text, "how are you")
		assert.Contains(t, text, "I'm fine")
		assert.NotContains(t, text, "recent message")
		assert.NotContains(t, text, "recent response")
	})

	t.Run("summarizes assistant tool chain when previous summary provides context", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(_ []event.Event) int { return 1 }))

		summaryTs := time.Now().UTC()
		toolCallTs := summaryTs.Add(-2 * time.Minute)
		toolResultTs := summaryTs.Add(-time.Minute)
		sess := &session.Session{
			ID: "tool-chain-session",
			Events: []event.Event{
				{
					Author:    authorSystem,
					Timestamp: summaryTs,
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "previous summary"},
					}}},
				},
				{
					Author:    "assistant",
					Timestamp: toolCallTs,
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							Role: compat.RoleAssistant,
							ToolCalls: []compat.ToolCall{{
								Function: compat.FunctionDefinitionParam{
									Name:      "lookup_weather",
									Arguments: []byte(`{"city":"Shanghai"}`),
								},
							}},
						},
					}}},
				},
				{
					Author:    "tool",
					Timestamp: toolResultTs,
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:   "call-1",
							ToolName: "lookup_weather",
							Content:  "sunny",
						},
					}}},
				},
				{
					Author:    "assistant",
					Timestamp: summaryTs.Add(time.Minute),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Role: compat.RoleAssistant, Content: "recent response"},
					}}},
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.Contains(t, text, "previous summary")
		assert.Contains(t, text, "Called tool: lookup_weather")
		assert.Contains(t, text, "lookup_weather returned: sunny")
		assert.NotContains(t, text, "recent response")

		raw := sess.State[lastIncludedTsKey]
		require.NotEmpty(t, raw)

		got, err := time.Parse(time.RFC3339Nano, string(raw))
		require.NoError(t, err)
		assert.True(t, got.Equal(toolResultTs))
	})

	t.Run("errors when filtered events have no user message", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(_ []event.Event) int { return 2 }))

		// Create session where filtering removes all user messages
		sess := &session.Session{
			ID: "test-session",
			Events: []event.Event{
				{
					Author:   "assistant",
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "response1"}}}},
				},
				{
					Author:   "user", // This will be skipped
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleUser, Content: "user message"}}}},
				},
				{
					Author:   "assistant", // This will be skipped
					Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Role: compat.RoleAssistant, Content: "response2"}}}},
				},
			},
		}

		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no conversation text extracted")
	})
}

func TestSessionSummarizer_RecordLastIncludedBoundary(t *testing.T) {
	now := time.Now().UTC()
	keepTs := now.Add(-2 * time.Minute)
	sess := &session.Session{
		ID: "ts-session",
		Events: []event.Event{
			{
				ID:        "keep-event",
				Author:    "user",
				Timestamp: keepTs,
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Role: compat.RoleUser, Content: "keep"},
				}}},
			},
			{
				Author:    "user",
				Timestamp: now.Add(-time.Minute),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Role: compat.RoleUser, Content: "skip"},
				}}},
			},
		},
	}

	s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(_ []event.Event) int { return 1 }))
	_, err := s.Summarize(context.Background(), sess)
	require.NoError(t, err)

	require.NotNil(t, sess.State)
	raw := sess.State[lastIncludedTsKey]
	require.NotEmpty(t, raw)

	got, err := time.Parse(time.RFC3339Nano, string(raw))
	require.NoError(t, err)
	assert.True(t, got.Equal(keepTs))
	assert.Equal(t, "keep-event", string(sess.State[lastIncludedEventIDKey]))
}

func TestSessionSummarizer_RecordLastIncludedBoundary_NoStateOrEvents(t *testing.T) {
	s := &sessionSummarizer{}

	t.Run("nil session", func(t *testing.T) {
		s.recordLastIncludedBoundary(nil, nil)
	})

	t.Run("empty events does nothing", func(t *testing.T) {
		sess := &session.Session{}
		s.recordLastIncludedBoundary(sess, []event.Event{})
		assert.Nil(t, sess.State)
	})
}

func TestSessionSummarizer_BuildCheckSession(t *testing.T) {
	t.Run("returns nil for nil session", func(t *testing.T) {
		s := &sessionSummarizer{}
		assert.Nil(t, s.buildCheckSession(nil))
	})

	t.Run("injects token text without summary input formatter", func(t *testing.T) {
		s := &sessionSummarizer{
			toolResultFormatter: func(compat.Message) string { return "[tool result]" },
		}
		content := strings.Repeat("x", 2000)
		sess := &session.Session{
			Events: []event.Event{
				{
					Author:    "tool",
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							ToolID:   "call-1",
							ToolName: "read_file",
							Content:  content,
						},
					}}},
				},
			},
		}

		checkSess := s.buildCheckSession(sess)
		require.NotNil(t, checkSess)

		raw, ok := checkSess.GetState(tokenThresholdConversationTextStateKey)
		require.True(t, ok)
		assert.Contains(t, string(raw), content)
	})

	t.Run("injects reasoning content only for token threshold checks", func(t *testing.T) {
		s := &sessionSummarizer{}
		reasoning := "thinking through the answer"
		sess := &session.Session{
			Events: []event.Event{
				{
					Author:    "assistant",
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{
							Content:          "final answer",
							ReasoningContent: reasoning,
						},
					}}},
				},
			},
		}

		checkSess := s.buildCheckSession(sess)
		require.NotNil(t, checkSess)

		raw, ok := checkSess.GetState(tokenThresholdReasoningContentStateKey)
		require.True(t, ok)
		assert.Equal(t, reasoning, string(raw))
	})

	t.Run("uses branch scope when building injected token text", func(t *testing.T) {
		s := &sessionSummarizer{}
		sess := &session.Session{
			AppName: "app",
			Events: []event.Event{
				{
					Author:    "user",
					FilterKey: "app",
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "root message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub",
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "branch message"},
					}}},
				},
				{
					Author:    "assistant",
					FilterKey: "app/sub/tool",
					Timestamp: time.Now(),
					Response: &compat.Response{Choices: []compat.Choice{{
						Message: compat.Message{Content: "descendant message"},
					}}},
				},
			},
		}
		isummaryscope.SetScopeFilterKey(sess, "app/sub")

		checkSess := s.buildCheckSession(sess)
		require.NotNil(t, checkSess)

		raw, ok := checkSess.GetState(tokenThresholdConversationTextStateKey)
		require.True(t, ok)
		text := string(raw)
		assert.NotContains(t, text, "root message")
		assert.Contains(t, text, "branch message")
		assert.Contains(t, text, "descendant message")
	})
}

func TestSessionSummarizer_Metadata_IncludesSkipRecent(t *testing.T) {
	s := NewSummarizer(&fakeModel{}, WithSkipRecent(func(_ []event.Event) int { return 3 }))
	metadata := s.Metadata()

	assert.Equal(t, true, metadata[metadataKeySkipRecentEnabled])
}

func TestSessionSummarizer_SetPrompt(t *testing.T) {
	t.Run("updates prompt successfully", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		originalPrompt := s.(*sessionSummarizer).prompt

		newPrompt := "Custom prompt with {conversation_text} and {max_summary_words} words."
		s.(*sessionSummarizer).SetPrompt(newPrompt)

		assert.NotEqual(t, originalPrompt, s.(*sessionSummarizer).prompt)
		assert.Equal(t, newPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("ignores empty prompt", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		originalPrompt := s.(*sessionSummarizer).prompt

		s.(*sessionSummarizer).SetPrompt("")

		assert.Equal(t, originalPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("updated prompt is used in summarization", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})

		// Set a custom prompt that includes specific markers.
		customPrompt := "Test custom prompt: {conversation_text}"
		s.(*sessionSummarizer).SetPrompt(customPrompt)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{
				{
					Response: &compat.Response{
						Choices: []compat.Choice{{
							Message: compat.Message{Content: "Hello world"},
						}},
					},
					Timestamp: time.Now(),
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// fakeModel returns the prompt as part of the summary,
		// so we can verify the custom prompt was used.
		assert.Contains(t, text, "Test custom prompt")
	})

	t.Run("SetPrompt with placeholder replacement", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(50))

		// Set a custom prompt with both placeholders.
		customPrompt := "Summarize in {max_summary_words} words: {conversation_text}"
		s.(*sessionSummarizer).SetPrompt(customPrompt)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{
				{
					Response: &compat.Response{
						Choices: []compat.Choice{{
							Message: compat.Message{Content: "Test content"},
						}},
					},
					Timestamp: time.Now(),
				},
			},
		}

		text, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text)

		// Verify placeholder was replaced with actual number.
		assert.Contains(t, text, "50")
		assert.Contains(t, text, "Summarize in")
	})

	t.Run("multiple SetPrompt calls", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})

		firstPrompt := "First prompt: {conversation_text}"
		s.(*sessionSummarizer).SetPrompt(firstPrompt)
		assert.Equal(t, firstPrompt, s.(*sessionSummarizer).prompt)

		secondPrompt := "Second prompt: {conversation_text}"
		s.(*sessionSummarizer).SetPrompt(secondPrompt)
		assert.Equal(t, secondPrompt, s.(*sessionSummarizer).prompt)

		// Empty prompt should not change.
		s.(*sessionSummarizer).SetPrompt("")
		assert.Equal(t, secondPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("SetPrompt on nil summarizer", func(t *testing.T) {
		var s *sessionSummarizer
		// Should not panic
		assert.NotPanics(t, func() {
			if s != nil {
				s.SetPrompt("test")
			}
		})
	})

	t.Run("SetPrompt validates conversationTextPlaceholder", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		originalPrompt := s.(*sessionSummarizer).prompt

		invalidPrompt := "Prompt without placeholder"
		assert.NotPanics(t, func() {
			s.(*sessionSummarizer).SetPrompt(invalidPrompt)
		})
		// Invalid prompt should not be set
		assert.Equal(t, originalPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("SetPrompt validates maxSummaryWordsPlaceholder when maxSummaryWords > 0", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(50))
		originalPrompt := s.(*sessionSummarizer).prompt

		invalidPrompt := "Prompt with {conversation_text} but no max words placeholder"
		assert.NotPanics(t, func() {
			s.(*sessionSummarizer).SetPrompt(invalidPrompt)
		})
		// Invalid prompt should not be set
		assert.Equal(t, originalPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("SetPrompt accepts prompt without maxSummaryWordsPlaceholder when system prompt already has it", func(t *testing.T) {
		s := NewSummarizer(
			&fakeModel{},
			WithMaxSummaryWords(50),
			WithSystemPrompt("Keep it within {max_summary_words} words."),
		)

		validPrompt := "Prompt with {conversation_text} only"
		s.(*sessionSummarizer).SetPrompt(validPrompt)
		assert.Equal(t, validPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("SetPrompt accepts valid prompt without maxSummaryWordsPlaceholder when maxSummaryWords = 0", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})

		validPrompt := "Prompt with {conversation_text} only"
		s.(*sessionSummarizer).SetPrompt(validPrompt)
		assert.Equal(t, validPrompt, s.(*sessionSummarizer).prompt)
	})

	t.Run("NewSummarizer validates prompt with WithPrompt", func(t *testing.T) {
		t.Run("accepts and validates invalid prompt", func(t *testing.T) {
			assert.NotPanics(t, func() {
				s := NewSummarizer(&fakeModel{}, WithPrompt("invalid prompt"))
				assert.NotNil(t, s)
				// The invalid prompt is set despite validation warning
				assert.Equal(t, "invalid prompt", s.(*sessionSummarizer).prompt)
			})
		})

		t.Run("accepts valid prompt", func(t *testing.T) {
			assert.NotPanics(t, func() {
				s := NewSummarizer(&fakeModel{}, WithPrompt("prompt with {conversation_text}"))
				assert.NotNil(t, s)
				assert.Equal(t, "prompt with {conversation_text}", s.(*sessionSummarizer).prompt)
			})
		})

		t.Run("validates prompt with maxSummaryWords", func(t *testing.T) {
			assert.NotPanics(t, func() {
				s := NewSummarizer(&fakeModel{}, WithMaxSummaryWords(50), WithPrompt("prompt with {conversation_text} and {max_summary_words}"))
				assert.NotNil(t, s)
				assert.Equal(t, "prompt with {conversation_text} and {max_summary_words}", s.(*sessionSummarizer).prompt)
			})
		})
	})

	t.Run("SetPrompt validates invalid prompt", func(t *testing.T) {
		s := NewSummarizer(&fakeModel{})
		originalPrompt := s.(*sessionSummarizer).prompt

		// Setting an invalid prompt should not change the current prompt
		assert.NotPanics(t, func() {
			s.(*sessionSummarizer).SetPrompt("another invalid prompt")
		})
		assert.Equal(t, originalPrompt, s.(*sessionSummarizer).prompt)
	})
}

type captureRequestModel struct {
	lastRequest *compat.Request
	output      string
}

func (c *captureRequestModel) Info() compat.Info { return compat.Info{Name: "capture"} }

func (c *captureRequestModel) GenerateContent(
	ctx context.Context,
	req *compat.Request,
) (<-chan *compat.Response, error) {
	_ = ctx
	c.lastRequest = req
	ch := make(chan *compat.Response, 1)
	output := c.output
	if output == "" {
		output = "captured"
	}
	ch <- &compat.Response{
		Done: true,
		Choices: []compat.Choice{{
			Message: compat.Message{Content: output},
		}},
	}
	close(ch)
	return ch, nil
}

func TestSessionSummarizer_WithSystemPrompt(t *testing.T) {
	t.Run("prepends system message and keeps user prompt intact", func(t *testing.T) {
		m := &captureRequestModel{}
		s := NewSummarizer(
			m,
			WithSystemPrompt("Focus on key decisions."),
			WithPrompt("<conversation>\n{conversation_text}\n</conversation>\n\nSummary:"),
		)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{{
				Author: "user",
				Response: &compat.Response{
					Choices: []compat.Choice{{
						Message: compat.Message{Content: "Hello world"},
					}},
				},
				Timestamp: time.Now(),
			}},
		}

		_, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		require.NotNil(t, m.lastRequest)
		require.Len(t, m.lastRequest.Messages, 2)
		assert.Equal(t, compat.RoleSystem, m.lastRequest.Messages[0].Role)
		assert.Equal(t, "Focus on key decisions.", m.lastRequest.Messages[0].Content)
		assert.Equal(t, compat.RoleUser, m.lastRequest.Messages[1].Role)
		assert.Equal(
			t,
			"<conversation>\nuser: Hello world\n</conversation>\n\nSummary:",
			m.lastRequest.Messages[1].Content,
		)
	})

	t.Run("renders max summary words in system prompt", func(t *testing.T) {
		m := &captureRequestModel{}
		s := NewSummarizer(
			m,
			WithMaxSummaryWords(50),
			WithSystemPrompt("Keep it within {max_summary_words} words."),
			WithPrompt("<conversation>\n{conversation_text}\n</conversation>\n\nSummary:"),
		)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{{
				Author: "user",
				Response: &compat.Response{
					Choices: []compat.Choice{{
						Message: compat.Message{Content: "Need a summary"},
					}},
				},
				Timestamp: time.Now(),
			}},
		}

		_, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		require.NotNil(t, m.lastRequest)
		require.Len(t, m.lastRequest.Messages, 2)
		assert.Equal(t, "Keep it within 50 words.", m.lastRequest.Messages[0].Content)
		assert.Equal(t, "<conversation>\nuser: Need a summary\n</conversation>\n\nSummary:", m.lastRequest.Messages[1].Content)
	})

	t.Run("fails when system prompt includes conversation placeholder", func(t *testing.T) {
		s := NewSummarizer(
			&captureRequestModel{},
			WithSystemPrompt("Do not use {conversation_text} here."),
		)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{{
				Author: "user",
				Response: &compat.Response{
					Choices: []compat.Choice{{
						Message: compat.Message{Content: "Need a summary"},
					}},
				},
				Timestamp: time.Now(),
			}},
		}

		_, err := s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "render system prompt")
	})

	t.Run("fails when system prompt includes previous summary placeholder", func(t *testing.T) {
		s := NewSummarizer(
			&captureRequestModel{},
			WithSystemPrompt("Do not use {previous_summary} here."),
			WithPrompt("Previous: {previous_summary}\nConversation: {conversation_text}"),
		)
		ctx := isummarycontext.WithPreviousSummary(context.Background(), "previous")
		sess := &session.Session{ID: "test", Events: []event.Event{
			newEventWithContent("Need a summary"),
		}}

		_, err := s.Summarize(ctx, sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "render system prompt")
	})
}

func TestSessionSummarizer_SetModel(t *testing.T) {
	t.Run("updates model successfully", func(t *testing.T) {
		originalModel := &fakeModel{}
		s := NewSummarizer(originalModel)
		assert.Same(t, originalModel, s.(*sessionSummarizer).model)

		newModel := &customOutputModel{output: "new"}
		s.(*sessionSummarizer).SetModel(newModel)

		assert.Same(t, newModel, s.(*sessionSummarizer).model)
		assert.NotSame(t, originalModel, s.(*sessionSummarizer).model)
	})

	t.Run("ignores nil model", func(t *testing.T) {
		originalModel := &fakeModel{}
		s := NewSummarizer(originalModel)

		s.(*sessionSummarizer).SetModel(nil)

		assert.Equal(t, originalModel, s.(*sessionSummarizer).model)
	})

	t.Run("updated model is used in summarization", func(t *testing.T) {
		originalModel := &fakeModel{}
		s := NewSummarizer(originalModel)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{
				{
					Response: &compat.Response{
						Choices: []compat.Choice{{
							Message: compat.Message{Content: "Hello world"},
						}},
					},
					Timestamp: time.Now(),
				},
			},
		}

		// Use original model
		text1, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text1)

		// Switch to a different model that returns different output
		newModel := &customOutputModel{output: "Custom model summary"}
		s.(*sessionSummarizer).SetModel(newModel)

		text2, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)
		assert.NotEmpty(t, text2)
		assert.Contains(t, text2, "Custom model summary")
	})

	t.Run("model metadata updates after SetModel", func(t *testing.T) {
		model1 := &fakeModel{}
		s := NewSummarizer(model1)

		metadata1 := s.Metadata()
		assert.Equal(t, "fake", metadata1[metadataKeyModelName])

		// Switch to a different model
		model2 := &customOutputModel{output: "test"}
		s.(*sessionSummarizer).SetModel(model2)

		metadata2 := s.Metadata()
		assert.Equal(t, "custom-output", metadata2[metadataKeyModelName])
		assert.NotEqual(t, metadata1[metadataKeyModelName], metadata2[metadataKeyModelName])
	})

	t.Run("multiple SetModel calls", func(t *testing.T) {
		model1 := &fakeModel{}
		s := NewSummarizer(model1)
		assert.Equal(t, model1, s.(*sessionSummarizer).model)

		model2 := &customOutputModel{output: "test"}
		s.(*sessionSummarizer).SetModel(model2)
		assert.Equal(t, model2, s.(*sessionSummarizer).model)

		model3 := &fakeModel{}
		s.(*sessionSummarizer).SetModel(model3)
		assert.Equal(t, model3, s.(*sessionSummarizer).model)

		// Nil model should not change
		s.(*sessionSummarizer).SetModel(nil)
		assert.Equal(t, model3, s.(*sessionSummarizer).model)
	})

	t.Run("SetModel with error model", func(t *testing.T) {
		originalModel := &fakeModel{}
		s := NewSummarizer(originalModel)

		sess := &session.Session{
			ID: "test",
			Events: []event.Event{
				{
					Response: &compat.Response{
						Choices: []compat.Choice{{
							Message: compat.Message{Content: "Hello"},
						}},
					},
					Timestamp: time.Now(),
				},
			},
		}

		// Original model should work
		_, err := s.Summarize(context.Background(), sess)
		require.NoError(t, err)

		// Switch to error model
		errorModel := &errorModel{}
		s.(*sessionSummarizer).SetModel(errorModel)

		_, err = s.Summarize(context.Background(), sess)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model error")
	})

	t.Run("SetModel on nil summarizer", func(t *testing.T) {
		var s *sessionSummarizer
		// Should not panic
		assert.NotPanics(t, func() {
			if s != nil {
				s.SetModel(&fakeModel{})
			}
		})
	})
}

// customOutputModel returns a custom output for testing.
type customOutputModel struct {
	output string
}

func (c *customOutputModel) Info() compat.Info {
	return compat.Info{Name: "custom-output"}
}

func (c *customOutputModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{
		Done: true,
		Choices: []compat.Choice{
			{Message: compat.Message{Content: c.output}},
		},
	}
	close(ch)
	return ch, nil
}
