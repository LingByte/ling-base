// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package mq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeDelivery is a test Delivery implementation.
type fakeDelivery struct {
	msg         *Message
	acked       bool
	nacked      bool
	rejected    bool
	nackRequeue bool
	rejRequeue  bool
	redelivered bool
	tag         uint64
}

func (f *fakeDelivery) Message() *Message       { return f.msg }
func (f *fakeDelivery) Body() []byte            { return f.msg.Body }
func (f *fakeDelivery) Headers() map[string]any { return f.msg.Headers }
func (f *fakeDelivery) RoutingKey() string      { return f.msg.RoutingKey }
func (f *fakeDelivery) Exchange() string        { return f.msg.Exchange }
func (f *fakeDelivery) Redelivered() bool       { return f.redelivered }
func (f *fakeDelivery) DeliveryTag() uint64     { return f.tag }
func (f *fakeDelivery) Ack() error              { f.acked = true; return nil }
func (f *fakeDelivery) Nack(requeue bool) error {
	f.nacked = true
	f.nackRequeue = requeue
	return nil
}
func (f *fakeDelivery) Reject(requeue bool) error {
	f.rejected = true
	f.rejRequeue = requeue
	return nil
}

func newFakeDelivery(body []byte) *fakeDelivery {
	return &fakeDelivery{
		msg: &Message{Body: body, RoutingKey: "test.key", Exchange: "test.ex"},
		tag: 1,
	}
}

// ============================================================
// Delivery tests
// ============================================================

func TestFakeDelivery_Ack(t *testing.T) {
	d := newFakeDelivery([]byte("hello"))
	assert.NoError(t, d.Ack())
	assert.True(t, d.acked)
	assert.False(t, d.nacked)
}

func TestFakeDelivery_Nack(t *testing.T) {
	d := newFakeDelivery([]byte("hello"))
	assert.NoError(t, d.Nack(true))
	assert.True(t, d.nacked)
	assert.True(t, d.nackRequeue)
}

func TestFakeDelivery_Reject(t *testing.T) {
	d := newFakeDelivery([]byte("hello"))
	assert.NoError(t, d.Reject(false))
	assert.True(t, d.rejected)
	assert.False(t, d.rejRequeue)
}

// ============================================================
// Middleware tests
// ============================================================

func TestRecoverMiddleware_NoPanic(t *testing.T) {
	called := false
	h := RecoverMiddleware(nil)(func(ctx context.Context, d Delivery) error {
		called = true
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.True(t, called)
}

func TestRecoverMiddleware_Panic(t *testing.T) {
	h := RecoverMiddleware(nil)(func(ctx context.Context, d Delivery) error {
		panic("boom")
	})
	err := h(context.Background(), newFakeDelivery(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
	assert.Contains(t, err.Error(), "boom")
}

func TestLoggingMiddleware(t *testing.T) {
	called := false
	h := LoggingMiddleware(nil)(func(ctx context.Context, d Delivery) error {
		called = true
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.True(t, called)
}

func TestMetricsMiddleware_Success(t *testing.T) {
	mc := NewMetricsCollector()
	h := MetricsMiddleware(mc)(func(ctx context.Context, d Delivery) error {
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	snap := mc.Snapshot()
	assert.Equal(t, int64(1), snap.Consumed)
	assert.Equal(t, int64(0), snap.Errors)
}

func TestMetricsMiddleware_Error(t *testing.T) {
	mc := NewMetricsCollector()
	h := MetricsMiddleware(mc)(func(ctx context.Context, d Delivery) error {
		return errors.New("fail")
	})
	assert.Error(t, h(context.Background(), newFakeDelivery(nil)))
	snap := mc.Snapshot()
	assert.Equal(t, int64(1), snap.Consumed)
	assert.Equal(t, int64(1), snap.Errors)
}

func TestMetricsMiddleware_Redelivered(t *testing.T) {
	mc := NewMetricsCollector()
	d := newFakeDelivery(nil)
	d.redelivered = true
	h := MetricsMiddleware(mc)(func(ctx context.Context, d Delivery) error {
		return nil
	})
	assert.NoError(t, h(context.Background(), d))
	snap := mc.Snapshot()
	assert.Equal(t, int64(1), snap.Consumed)
	assert.Equal(t, int64(1), snap.Redelivered)
}

func TestRetryMiddleware_SuccessFirstTry(t *testing.T) {
	calls := 0
	h := RetryMiddleware(DefaultRetryConfig())(func(ctx context.Context, d Delivery) error {
		calls++
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.Equal(t, 1, calls)
}

func TestRetryMiddleware_Exhausted(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxAttempts: 3, Backoff: 1 * time.Millisecond}
	h := RetryMiddleware(cfg)(func(ctx context.Context, d Delivery) error {
		calls++
		return errors.New("always fail")
	})
	err := h(context.Background(), newFakeDelivery(nil))
	assert.Error(t, err)
	assert.Equal(t, 3, calls)
	assert.Contains(t, err.Error(), "retry exhausted")
}

func TestRetryMiddleware_SucceedsOnSecondAttempt(t *testing.T) {
	calls := 0
	cfg := RetryConfig{MaxAttempts: 3, Backoff: 1 * time.Millisecond}
	h := RetryMiddleware(cfg)(func(ctx context.Context, d Delivery) error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.Equal(t, 2, calls)
}

func TestRetryMiddleware_ContextCancelled(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 10, Backoff: 100 * time.Millisecond}
	h := RetryMiddleware(cfg)(func(ctx context.Context, d Delivery) error {
		return errors.New("fail")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := h(ctx, newFakeDelivery(nil))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestRetryMiddleware_BackoffFn(t *testing.T) {
	calls := 0
	cfg := RetryConfig{
		MaxAttempts: 3,
		BackoffFn:   func(attempt int) time.Duration { return time.Duration(attempt) * time.Millisecond },
	}
	h := RetryMiddleware(cfg)(func(ctx context.Context, d Delivery) error {
		calls++
		return errors.New("fail")
	})
	_ = h(context.Background(), newFakeDelivery(nil))
	assert.Equal(t, 3, calls)
}

func TestRetryMiddleware_Defaults(t *testing.T) {
	// MaxAttempts=0 → default 1
	h := RetryMiddleware(RetryConfig{MaxAttempts: 0})(func(ctx context.Context, d Delivery) error {
		return errors.New("fail")
	})
	err := h(context.Background(), newFakeDelivery(nil))
	assert.Error(t, err)
}

func TestDeadLetterMiddleware_NoError(t *testing.T) {
	processed := false
	dlqCalled := false
	h := DeadLetterMiddleware(func(ctx context.Context, d Delivery) error {
		dlqCalled = true
		return nil
	})(func(ctx context.Context, d Delivery) error {
		processed = true
		return nil
	})
	d := newFakeDelivery(nil)
	assert.NoError(t, h(context.Background(), d))
	assert.True(t, processed)
	assert.False(t, dlqCalled)
	assert.False(t, d.acked)
}

func TestDeadLetterMiddleware_Error(t *testing.T) {
	dlqCalled := false
	h := DeadLetterMiddleware(func(ctx context.Context, d Delivery) error {
		dlqCalled = true
		return nil
	})(func(ctx context.Context, d Delivery) error {
		return errors.New("fail")
	})
	d := newFakeDelivery(nil)
	assert.NoError(t, h(context.Background(), d))
	assert.True(t, dlqCalled)
	assert.True(t, d.acked)
}

func TestDeadLetterMiddleware_NilDLQHandler(t *testing.T) {
	h := DeadLetterMiddleware(nil)(func(ctx context.Context, d Delivery) error {
		return errors.New("fail")
	})
	d := newFakeDelivery(nil)
	err := h(context.Background(), d)
	assert.Error(t, err)
	assert.False(t, d.acked)
}

func TestChain_Empty(t *testing.T) {
	called := false
	h := Chain()(func(ctx context.Context, d Delivery) error {
		called = true
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.True(t, called)
}

func TestChain_Order(t *testing.T) {
	var order []string
	mw1 := func(next Handler) Handler {
		return func(ctx context.Context, d Delivery) error {
			order = append(order, "mw1-before")
			err := next(ctx, d)
			order = append(order, "mw1-after")
			return err
		}
	}
	mw2 := func(next Handler) Handler {
		return func(ctx context.Context, d Delivery) error {
			order = append(order, "mw2-before")
			err := next(ctx, d)
			order = append(order, "mw2-after")
			return err
		}
	}
	h := Chain(mw1, mw2)(func(ctx context.Context, d Delivery) error {
		order = append(order, "handler")
		return nil
	})
	assert.NoError(t, h(context.Background(), newFakeDelivery(nil)))
	assert.Equal(t, []string{
		"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after",
	}, order)
}

// ============================================================
// Options tests
// ============================================================

func TestDefaultExchangeOptions(t *testing.T) {
	opts := DefaultExchangeOptions()
	assert.Equal(t, "topic", opts.Kind)
	assert.True(t, opts.Durable)
	assert.False(t, opts.AutoDelete)
}

func TestDefaultQueueOptions(t *testing.T) {
	opts := DefaultQueueOptions()
	assert.True(t, opts.Durable)
	assert.False(t, opts.AutoDelete)
	assert.False(t, opts.Exclusive)
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.Backoff)
}

// ============================================================
// Metrics tests
// ============================================================

func TestMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RecordPublish()
	mc.RecordPublish()
	mc.RecordConsume()
	mc.RecordAck()
	mc.RecordNack()
	mc.RecordReject()
	mc.RecordRedelivered()
	mc.RecordError()

	snap := mc.Snapshot()
	assert.Equal(t, int64(2), snap.Published)
	assert.Equal(t, int64(1), snap.Consumed)
	assert.Equal(t, int64(1), snap.Acked)
	assert.Equal(t, int64(1), snap.Nacked)
	assert.Equal(t, int64(1), snap.Rejected)
	assert.Equal(t, int64(1), snap.Redelivered)
	assert.Equal(t, int64(1), snap.Errors)
}
