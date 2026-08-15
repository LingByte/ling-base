// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecoverMiddleware_NoPanic(t *testing.T) {
	called := false
	h := RecoverMiddleware()(func(ctx context.Context, e *Event) error {
		called = true
		return nil
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRecoverMiddleware_Panic(t *testing.T) {
	h := RecoverMiddleware()(func(ctx context.Context, e *Event) error {
		panic("boom")
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestRetryMiddleware_Success(t *testing.T) {
	var calls int32
	h := RetryMiddleware(3, time.Millisecond)(func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestRetryMiddleware_FailsAfterRetries(t *testing.T) {
	var calls int32
	h := RetryMiddleware(3, time.Millisecond)(func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("always fails")
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.Error(t, err)
	assert.Equal(t, int32(4), atomic.LoadInt32(&calls)) // 1 + 3 retries
}

func TestRetryMiddleware_SucceedsOnRetry(t *testing.T) {
	var calls int32
	h := RetryMiddleware(3, time.Millisecond)(func(ctx context.Context, e *Event) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestRetryMiddleware_AttemptIncremented(t *testing.T) {
	var lastAttempt int
	h := RetryMiddleware(2, time.Millisecond)(func(ctx context.Context, e *Event) error {
		lastAttempt = e.Attempt
		return errors.New("fail")
	})
	_ = h(context.Background(), &Event{Name: "test"})
	assert.Equal(t, 3, lastAttempt) // attempts 1, 2, 3
}

func TestRetryMiddleware_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	h := RetryMiddleware(5, 10*time.Millisecond)(func(ctx context.Context, e *Event) error {
		return errors.New("fail")
	})
	err := h(ctx, &Event{Name: "test"})
	assert.Error(t, err)
}

func TestDeadLetterMiddleware_Success(t *testing.T) {
	dlqCalled := false
	dlqHandler := func(ctx context.Context, e *Event) error {
		dlqCalled = true
		return nil
	}
	h := DeadLetterMiddleware(dlqHandler)(func(ctx context.Context, e *Event) error {
		return nil
	})
	err := h(context.Background(), &Event{Name: "test"})
	assert.NoError(t, err)
	assert.False(t, dlqCalled) // DLQ not called on success
}

func TestDeadLetterMiddleware_Failure(t *testing.T) {
	var dlqEvent *Event
	dlqHandler := func(ctx context.Context, e *Event) error {
		dlqEvent = e
		return nil
	}
	h := DeadLetterMiddleware(dlqHandler)(func(ctx context.Context, e *Event) error {
		return errors.New("handler failed")
	})
	err := h(context.Background(), &Event{Name: "test.event", ID: "evt-1"})
	assert.Error(t, err)
	assert.NotNil(t, dlqEvent)
	assert.Equal(t, "test.event", dlqEvent.Name)
	assert.Equal(t, "handler failed", dlqEvent.Headers["x-error"])
	assert.Equal(t, "true", dlqEvent.Headers["x-dlq"])
}

func TestLoggingMiddleware(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)
	h := LoggingMiddleware(logger)(func(ctx context.Context, e *Event) error {
		return nil
	})
	err := h(context.Background(), &Event{Name: "test.event"})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "dispatching")
	assert.Contains(t, output, "handler ok")
}

func TestLoggingMiddleware_Error(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf, "", 0)
	h := LoggingMiddleware(logger)(func(ctx context.Context, e *Event) error {
		return errors.New("boom")
	})
	_ = h(context.Background(), &Event{Name: "test.event"})
	assert.Contains(t, buf.String(), "handler failed")
	assert.Contains(t, buf.String(), "boom")
}

func TestHandlerMetrics(t *testing.T) {
	m := NewHandlerMetrics()
	h := MetricsMiddleware(m)(func(ctx context.Context, e *Event) error {
		return nil
	})
	_ = h(context.Background(), &Event{Name: "topic1"})
	_ = h(context.Background(), &Event{Name: "topic1"})

	errH := MetricsMiddleware(m)(func(ctx context.Context, e *Event) error {
		return errors.New("fail")
	})
	_ = errH(context.Background(), &Event{Name: "topic2"})

	calls, errors, _ := m.Get("topic1")
	assert.Equal(t, int64(2), calls)
	assert.Equal(t, int64(0), errors)

	calls2, errors2, _ := m.Get("topic2")
	assert.Equal(t, int64(1), calls2)
	assert.Equal(t, int64(1), errors2)

	topics := m.Topics()
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
}

func TestMiddlewareChain(t *testing.T) {
	var logBuf strings.Builder
	logger := log.New(&logBuf, "", 0)
	metrics := NewHandlerMetrics()

	handler := func(ctx context.Context, e *Event) error {
		return nil
	}

	wrapped := ApplyMiddleware(handler,
		LoggingMiddleware(logger),
		RecoverMiddleware(),
		MetricsMiddleware(metrics),
	)

	err := wrapped(context.Background(), &Event{Name: "chain.test"})
	assert.NoError(t, err)
	calls, _, _ := metrics.Get("chain.test")
	assert.Equal(t, int64(1), calls)
	assert.Contains(t, logBuf.String(), "chain.test")
}

func TestRetryMiddleware_WithBackoff(t *testing.T) {
	var calls int32
	start := time.Now()
	h := RetryMiddleware(2, 50*time.Millisecond)(func(ctx context.Context, e *Event) error {
		atomic.AddInt32(&calls, 1)
		return fmt.Errorf("fail")
	})
	_ = h(context.Background(), &Event{Name: "test"})
	elapsed := time.Since(start)
	// 2 retries × 50ms backoff = ~100ms minimum
	assert.GreaterOrEqual(t, elapsed, 90*time.Millisecond)
}
