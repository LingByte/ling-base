// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package memory implements an in-process event bus with sync/async
// dispatch, wildcard topic matching, and middleware support.
package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/ling-base/eventbus"
)

// DispatchMode controls how events are delivered to handlers.
type DispatchMode int

const (
	// Sync dispatches events synchronously — Publish blocks until all
	// handlers complete. Errors from one handler stop delivery to
	// subsequent handlers.
	Sync DispatchMode = iota

	// Async dispatches events in goroutines — Publish returns immediately
	// after enqueuing. Handler errors are collected but not returned to
	// the publisher.
	Async

	// Parallel dispatches events to all handlers concurrently in
	// separate goroutines, then waits for all to complete. Publish
	// blocks until all handlers finish.
	Parallel
)

// Bus is an in-memory event bus.
type Bus struct {
	mu         sync.RWMutex
	subs       map[string][]*subscription
	mode       DispatchMode
	middleware []eventbus.Middleware
	metrics    eventbus.MetricsCollector
	closed     atomic.Bool
	bufferSize int // async buffer size (0 = unbuffered)
	asyncQueue chan *pendingEvent
	asyncWg    sync.WaitGroup
}

type subscription struct {
	id      string
	topic   string
	handler eventbus.Handler
}

type pendingEvent struct {
	ctx context.Context
	e   *eventbus.Event
}

// Option configures the Bus.
type Option func(*Bus)

// WithDispatchMode sets the dispatch mode (default: Sync).
func WithDispatchMode(mode DispatchMode) Option {
	return func(b *Bus) { b.mode = mode }
}

// WithMiddleware adds middleware applied to all handlers.
func WithMiddleware(mw ...eventbus.Middleware) Option {
	return func(b *Bus) { b.middleware = append(b.middleware, mw...) }
}

// WithAsyncBufferSize sets the buffer size for async dispatch.
// Only effective with Async mode. Default: 1024.
func WithAsyncBufferSize(n int) Option {
	return func(b *Bus) { b.bufferSize = n }
}

// New creates a new in-memory event bus.
func New(opts ...Option) *Bus {
	b := &Bus{
		subs:       make(map[string][]*subscription),
		mode:       Sync,
		bufferSize: 1024,
	}
	for _, opt := range opts {
		opt(b)
	}
	if b.mode == Async {
		b.asyncQueue = make(chan *pendingEvent, b.bufferSize)
		b.startAsyncWorkers()
	}
	return b
}

// startAsyncWorkers launches goroutines to process the async queue.
func (b *Bus) startAsyncWorkers() {
	const numWorkers = 4
	for i := 0; i < numWorkers; i++ {
		b.asyncWg.Add(1)
		go func() {
			defer b.asyncWg.Done()
			for pe := range b.asyncQueue {
				b.dispatch(pe.ctx, pe.e)
			}
		}()
	}
}

// Publish sends an event to all matching subscribers.
func (b *Bus) Publish(ctx context.Context, e *eventbus.Event) error {
	if b.closed.Load() {
		return eventbus.ErrClosed
	}
	if e == nil {
		return fmt.Errorf("eventbus: event is nil")
	}
	if e.ID == "" {
		e.ID = eventbus.GenerateID()
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}

	b.metrics.RecordPublish()

	switch b.mode {
	case Async:
		b.metrics.RecordPending()
		select {
		case b.asyncQueue <- &pendingEvent{ctx: ctx, e: e}:
			return nil
		default:
			b.metrics.RecordFailed()
			return fmt.Errorf("eventbus: async queue full")
		}
	case Parallel:
		return b.dispatchParallel(ctx, e)
	default:
		return b.dispatch(ctx, e)
	}
}

// dispatch sends the event to all matching subscribers synchronously.
func (b *Bus) dispatch(ctx context.Context, e *eventbus.Event) error {
	subs := b.matchSubs(e.Name)
	if len(subs) == 0 {
		return nil
	}

	for _, s := range subs {
		handler := s.handler
		if len(b.middleware) > 0 {
			handler = eventbus.ApplyMiddleware(handler, b.middleware...)
		}
		b.metrics.RecordPending()
		start := time.Now()
		err := handler(ctx, e)
		b.metrics.RecordDelivered(time.Since(start))
		if err != nil {
			b.metrics.RecordFailed()
			if b.mode == Sync {
				return fmt.Errorf("eventbus: handler %q for %q failed: %w", s.id, e.Name, err)
			}
		}
	}
	return nil
}

// dispatchParallel sends the event to all matching subscribers concurrently.
func (b *Bus) dispatchParallel(ctx context.Context, e *eventbus.Event) error {
	subs := b.matchSubs(e.Name)
	if len(subs) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for _, s := range subs {
		handler := s.handler
		if len(b.middleware) > 0 {
			handler = eventbus.ApplyMiddleware(handler, b.middleware...)
		}
		wg.Add(1)
		go func(s *subscription, h eventbus.Handler) {
			defer wg.Done()
			b.metrics.RecordPending()
			start := time.Now()
			err := h(ctx, e)
			b.metrics.RecordDelivered(time.Since(start))
			if err != nil {
				b.metrics.RecordFailed()
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("eventbus: handler %q for %q failed: %w", s.id, e.Name, err)
				}
				errMu.Unlock()
			}
		}(s, handler)
	}
	wg.Wait()
	return firstErr
}

// matchSubs returns all subscriptions whose topic matches the event name.
func (b *Bus) matchSubs(name string) []*subscription {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var matched []*subscription
	for _, subs := range b.subs {
		for _, s := range subs {
			if eventbus.TopicMatches(s.topic, name) {
				matched = append(matched, s)
			}
		}
	}
	return matched
}

// Subscribe registers a handler for the given topic pattern.
// Returns a Subscription that can be used with Unsubscribe.
func (b *Bus) Subscribe(topic string, handler eventbus.Handler) eventbus.Subscription {
	if topic == "" {
		topic = "*"
	}
	id := fmt.Sprintf("sub-%d-%d", time.Now().UnixNano(), len(b.subs))
	s := &subscription{id: id, topic: topic, handler: handler}

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], s)
	b.mu.Unlock()

	b.metrics.RecordSubscribe()
	return eventbus.NewSubscription(id, topic)
}

// Unsubscribe removes a subscription.
func (b *Bus) Unsubscribe(sub eventbus.Subscription) error {
	if sub == nil {
		return fmt.Errorf("eventbus: subscription is nil")
	}
	topic := sub.Topic()
	id := sub.ID()

	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subs[topic]
	for i, s := range subs {
		if s.id == id {
			b.subs[topic] = append(subs[:i], subs[i+1:]...)
			if len(b.subs[topic]) == 0 {
				delete(b.subs, topic)
			}
			b.metrics.RecordUnsubscribe()
			return nil
		}
	}
	return fmt.Errorf("eventbus: subscription %s not found", id)
}

// Close shuts down the bus. For async mode, it drains the queue.
func (b *Bus) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}
	if b.asyncQueue != nil {
		close(b.asyncQueue)
		b.asyncWg.Wait()
	}
	return nil
}

// Metrics returns a snapshot of bus metrics.
func (b *Bus) Metrics() eventbus.Metrics {
	return b.metrics.Snapshot()
}

// SubscriberCount returns the number of subscribers for a topic pattern.
func (b *Bus) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[topic])
}

// Topics returns all registered topic patterns.
func (b *Bus) Topics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	topics := make([]string, 0, len(b.subs))
	for t := range b.subs {
		topics = append(topics, t)
	}
	return topics
}
