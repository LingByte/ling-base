// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package eventbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Event(t *testing.T) {
	e := New("user.created", map[string]string{"id": "123"})
	assert.Equal(t, "user.created", e.Name)
	assert.NotEmpty(t, e.ID)
	assert.False(t, e.Time.IsZero())
	assert.Equal(t, 1, e.Attempt)
}

func TestNewWithSource(t *testing.T) {
	e := NewWithSource("test", "myservice", "payload")
	assert.Equal(t, "myservice", e.Source)
	assert.Equal(t, "payload", e.Payload)
}

func TestEvent_WithHeader(t *testing.T) {
	e := New("test", nil)
	e.WithHeader("trace-id", "abc123").WithHeader("source", "api")
	assert.Equal(t, "abc123", e.Headers["trace-id"])
	assert.Equal(t, "api", e.Headers["source"])
}

func TestEvent_String(t *testing.T) {
	e := New("test.event", nil)
	s := e.String()
	assert.Contains(t, s, "test.event")
	assert.Contains(t, s, e.ID)
}

func TestTopicMatches_Exact(t *testing.T) {
	assert.True(t, TopicMatches("user.created", "user.created"))
	assert.False(t, TopicMatches("user.created", "user.deleted"))
}

func TestTopicMatches_SingleWildcard(t *testing.T) {
	assert.True(t, TopicMatches("user.*", "user.created"))
	assert.True(t, TopicMatches("user.*", "user.deleted"))
	assert.False(t, TopicMatches("user.*", "user.profile.updated"))
}

func TestTopicMatches_MultiWildcard(t *testing.T) {
	assert.True(t, TopicMatches("user.>", "user.created"))
	assert.True(t, TopicMatches("user.>", "user.profile.updated"))
	assert.True(t, TopicMatches("user.>", "user.a.b.c"))
	assert.False(t, TopicMatches("user.>", "order.created"))
}

func TestTopicMatches_StarAll(t *testing.T) {
	assert.True(t, TopicMatches("*", "anything"))
	assert.True(t, TopicMatches("*", "user.created"))
}

func TestTopicMatches_PartialWildcard(t *testing.T) {
	assert.True(t, TopicMatches("*.created", "user.created"))
	assert.True(t, TopicMatches("*.created", "order.created"))
	assert.False(t, TopicMatches("*.created", "user.deleted"))
}

func TestApplyMiddleware(t *testing.T) {
	var order []string
	mw1 := func(next Handler) Handler {
		return func(ctx context.Context, e *Event) error {
			order = append(order, "mw1-before")
			err := next(ctx, e)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(next Handler) Handler {
		return func(ctx context.Context, e *Event) error {
			order = append(order, "mw2-before")
			err := next(ctx, e)
			order = append(order, "mw2-after")
			return err
		}
	}
	handler := func(ctx context.Context, e *Event) error {
		order = append(order, "handler")
		return nil
	}

	wrapped := ApplyMiddleware(handler, mw1, mw2)
	_ = wrapped(nil, &Event{Name: "test"})

	assert.Equal(t, []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}, order)
}

func TestMetricsCollector(t *testing.T) {
	var m MetricsCollector
	m.RecordPublish()
	m.RecordPublish()
	m.RecordPending()
	m.RecordPending()
	m.RecordDelivered(10 * 1e6) // 10ms
	m.RecordDelivered(30 * 1e6) // 30ms
	m.RecordFailed()

	snap := m.Snapshot()
	assert.Equal(t, int64(2), snap.Published)
	assert.Equal(t, int64(2), snap.Delivered)
	assert.Equal(t, int64(1), snap.Failed)
	assert.Equal(t, int64(0), snap.Pending) // 2 pending - 2 delivered = 0
	assert.Equal(t, int64(20*1e6), int64(snap.AvgLatency))
}

func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateID()
		assert.False(t, ids[id], "duplicate ID: %s", id)
		ids[id] = true
	}
}

func TestIsClosed(t *testing.T) {
	assert.True(t, IsClosed(ErrClosed))
	assert.False(t, IsClosed(nil))
}
