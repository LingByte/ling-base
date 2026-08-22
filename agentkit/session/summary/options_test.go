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
	"time"

	"github.com/LingByte/ling-base/agentkit/event"
	"github.com/LingByte/ling-base/agentkit/internal/summarytrigger"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/LingByte/ling-base/agentkit/session"
	"github.com/stretchr/testify/assert"
)

func optionTestMessageEvent(content string, ts time.Time) event.Event {
	return event.Event{
		Timestamp: ts,
		Response: &compat.Response{Choices: []compat.Choice{{
			Message: compat.Message{Content: content},
		}}},
	}
}

func TestOptions(t *testing.T) {
	t.Run("WithPrompt", func(t *testing.T) {
		s := NewSummarizer(&testModel{}, WithPrompt("test"))
		sm, ok := s.(*sessionSummarizer)
		assert.True(t, ok)
		assert.Equal(t, "test", sm.prompt)
	})

	t.Run("WithSystemPrompt", func(t *testing.T) {
		s := NewSummarizer(&testModel{}, WithSystemPrompt("system"))
		sm, ok := s.(*sessionSummarizer)
		assert.True(t, ok)
		assert.Equal(t, "system", sm.systemPrompt)
	})

	t.Run("WithName", func(t *testing.T) {
		s := NewSummarizer(&testModel{}, WithName("  demo  "))
		sm, ok := s.(*sessionSummarizer)
		assert.True(t, ok)
		assert.Equal(t, "  demo  ", sm.name)
		assert.Equal(t, "  demo  ", sm.Metadata()[metadataKeySummarizerName])
	})

	t.Run("WithTokenThreshold", func(t *testing.T) {
		// Verify metadata increments and logic via isolated checks.
		s := NewSummarizer(&testModel{}, WithTokenThreshold(2))
		md := s.Metadata()
		assert.Equal(t, 1, md[metadataKeyCheckFunctions])

		sIso := NewSummarizer(&testModel{}, WithTokenThreshold(2))
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("a", 40)},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("b", 40)},
				}}},
			},
		}}
		assert.True(t, sIso.ShouldSummarize(sess))
	})

	t.Run("WithTokenThreshold uses request context when available", func(t *testing.T) {
		defer SetTokenCounter(nil)

		counter := &testContextTokenCounter{
			key:   "trace",
			value: "req-1",
		}
		SetTokenCounter(counter)

		s := NewSummarizer(&testModel{}, WithTokenThreshold(100))
		contextual, ok := s.(ContextAwareSummarizer)
		assert.True(t, ok)

		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "a"},
				}}},
			},
		}}

		assert.False(t, contextual.ShouldSummarizeWithContext(context.Background(), sess))
		assert.True(t, contextual.ShouldSummarizeWithContext(
			context.WithValue(context.Background(), "trace", "req-1"),
			sess,
		))
		assert.Equal(t, 1, counter.hit)
		assert.Equal(t, 1, counter.miss)
	})

	t.Run("WithTokenThreshold honors skipRecent-filtered input", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithSkipRecent(func(_ []event.Event) int { return 2 }),
			WithTokenThreshold(100),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "short"},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "reply"},
				}}},
			},
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("a", 800)},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("b", 800)},
				}}},
			},
		}}
		assert.False(t, s.ShouldSummarize(sess))
	})

	t.Run("WithTokenThreshold ignores summarizer tool result formatter", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolResultFormatter(func(compat.Message) string { return "[tool result]" }),
			WithTokenThreshold(100),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "tool",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "read_file",
						Content:  strings.Repeat("x", 2000),
					},
				}}},
			},
		}}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithTokenThreshold ignores summarizer tool call formatter", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolCallFormatter(func(compat.ToolCall) string { return "[tool call]" }),
			WithTokenThreshold(100),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{
								Name:      "read_file",
								Arguments: []byte(`{"content":"` + strings.Repeat("x", 2000) + `"}`),
							},
						}},
					},
				}}},
			},
		}}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithTokenThreshold skips empty summary input", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolResultFormatter(func(compat.Message) string { return "" }),
			WithTokenThreshold(100),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "tool",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "read_file",
						Content:  strings.Repeat("x", 2000),
					},
				}}},
			},
		}}
		assert.False(t, s.ShouldSummarize(sess))
	})

	t.Run("WithChecksAny token checker honors skipRecent-filtered input", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithSkipRecent(func(_ []event.Event) int { return 2 }),
			WithChecksAny(CheckTokenThreshold(100)),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "short"},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "reply"},
				}}},
			},
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("a", 800)},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("b", 800)},
				}}},
			},
		}}
		assert.False(t, s.ShouldSummarize(sess))
	})

	t.Run("WithChecksAny token checker ignores summarizer tool result formatter", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolResultFormatter(func(compat.Message) string { return "[tool result]" }),
			WithChecksAny(CheckTokenThreshold(100)),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "tool",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "read_file",
						Content:  strings.Repeat("x", 2000),
					},
				}}},
			},
		}}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithChecksAny token checker ignores summarizer tool call formatter", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolCallFormatter(func(compat.ToolCall) string { return "[tool call]" }),
			WithChecksAny(CheckTokenThreshold(100)),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolCalls: []compat.ToolCall{{
							Function: compat.FunctionDefinitionParam{
								Name:      "read_file",
								Arguments: []byte(`{"content":"` + strings.Repeat("x", 2000) + `"}`),
							},
						}},
					},
				}}},
			},
		}}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("effective empty input suppresses summarize even when custom checks pass", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithSkipRecent(func(events []event.Event) int { return len(events) }),
			WithChecksAny(func(*session.Session) bool { return true }),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: "hello"},
				}}},
			},
		}}
		assert.False(t, s.ShouldSummarize(sess))
	})

	t.Run("formatter-empty input suppresses event threshold", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithToolResultFormatter(func(compat.Message) string { return "" }),
			WithEventThreshold(0),
		)
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "tool",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{
						ToolID:   "call-1",
						ToolName: "read_file",
						Content:  "excluded",
					},
				}}},
			},
		}}
		assert.False(t, s.ShouldSummarize(sess))
	})

	t.Run("WithEventThreshold", func(t *testing.T) {
		s := NewSummarizer(&testModel{}, WithEventThreshold(2))
		md := s.Metadata()
		assert.Equal(t, 1, md[metadataKeyCheckFunctions])

		sIso := NewSummarizer(&testModel{}, WithEventThreshold(2))
		sess := &session.Session{Events: []event.Event{
			optionTestMessageEvent("one", time.Now()),
			optionTestMessageEvent("two", time.Now()),
			optionTestMessageEvent("three", time.Now()),
		}}
		assert.True(t, sIso.ShouldSummarize(sess))
	})

	t.Run("WithTimeThreshold", func(t *testing.T) {
		s := NewSummarizer(&testModel{}, WithTimeThreshold(10*time.Millisecond))
		md := s.Metadata()
		assert.Equal(t, 1, md[metadataKeyCheckFunctions])

		sIso := NewSummarizer(&testModel{}, WithTimeThreshold(10*time.Millisecond))
		older := time.Now().Add(-20 * time.Millisecond)
		sess := &session.Session{Events: []event.Event{
			optionTestMessageEvent("old", older),
		}}
		assert.True(t, sIso.ShouldSummarize(sess))
	})

	t.Run("WithChecksAll", func(t *testing.T) {
		checks := []Checker{CheckEventThreshold(1), CheckTokenThreshold(4)}
		s := NewSummarizer(&testModel{}, WithChecksAll(checks...))
		sess := &session.Session{Events: []event.Event{
			{
				Author:    "user",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("a", 40)},
				}}},
			},
			{
				Author:    "assistant",
				Timestamp: time.Now(),
				Response: &compat.Response{Choices: []compat.Choice{{
					Message: compat.Message{Content: strings.Repeat("b", 40)},
				}}},
			},
		}}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithChecksAny", func(t *testing.T) {
		checks := []Checker{CheckTokenThreshold(10000), CheckEventThreshold(3)}
		s := NewSummarizer(&testModel{}, WithChecksAny(checks...))
		sess := &session.Session{Events: make([]event.Event, 4)}
		for i := range sess.Events {
			sess.Events[i] = optionTestMessageEvent("event", time.Now())
		}
		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithChecksAny uses request gap observation", func(t *testing.T) {
		s := NewSummarizer(
			&testModel{},
			WithChecksAny(
				CheckEventThreshold(100),
				CheckTimeThreshold(20*time.Second),
			),
		)
		sess := &session.Session{Events: []event.Event{
			optionTestMessageEvent("event", time.Now()),
		}}
		summarytrigger.SetObservation(sess, summarytrigger.RequestGapObservation{
			Elapsed:   30 * time.Second,
			Available: true,
		})

		assert.True(t, s.ShouldSummarize(sess))
	})

	t.Run("WithMaxSummaryWords_MetadataAndLengthLimit", func(t *testing.T) {
		// Set a small max length and ensure metadata reflects it and length is limited in prompt.
		s := NewSummarizer(&testModel{}, WithMaxSummaryWords(50))
		md := s.Metadata()
		assert.Equal(t, 50, md[metadataKeyMaxSummaryWords])

		sess := &session.Session{ID: "sess-ml", Events: []event.Event{
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}}, Timestamp: time.Now().Add(-2 * time.Second)},
			{Response: &compat.Response{Choices: []compat.Choice{{Message: compat.Message{Content: "recent"}}}}, Timestamp: time.Now()},
		}}
		originalEventCount := len(sess.Events)
		text, err := s.Summarize(context.Background(), sess)
		assert.NoError(t, err)
		// Note: With the new prompt-based approach, we can't guarantee exact length
		// as the model controls the output. We just verify it generates some text.
		assert.NotEmpty(t, text)
		// Events should remain unchanged.
		assert.Equal(t, originalEventCount, len(sess.Events), "events should remain unchanged.")
	})

	t.Run("WithMaxSummaryWords_IgnoresNonPositive", func(t *testing.T) {
		// Non-positive should be ignored, default remains in metadata.
		s := NewSummarizer(&testModel{}, WithMaxSummaryWords(0))
		md := s.Metadata()
		// Default is 0 (no truncation).
		assert.Equal(t, 0, md[metadataKeyMaxSummaryWords])
	})
}

type testModel struct{}

func (t *testModel) Info() compat.Info { return compat.Info{Name: "test"} }
func (t *testModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	ch := make(chan *compat.Response, 1)
	ch <- &compat.Response{Done: true, Choices: []compat.Choice{{Message: compat.Message{Content: "ok"}}}}
	close(ch)
	return ch, nil
}
