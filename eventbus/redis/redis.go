// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package redis implements a persistent event bus using Redis Streams.
//
// Events are published to Redis Streams keyed by topic. Subscribers
// consume from streams using consumer groups, enabling:
//
//   - Cross-process pub/sub (multiple instances share the bus)
//   - Persistence (events survive process restarts)
//   - At-least-once delivery (via consumer group pending entries list)
//   - Horizontal scaling (multiple consumers per group)
//
// Basic usage:
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	bus, _ := redis.New(rdb, "myapp:events",
//	    redis.WithConsumerGroup("worker-1"),
//	)
//	defer bus.Close()
//
//	bus.Subscribe("user.*", func(ctx context.Context, e *eventbus.Event) error {
//	    log.Printf("user event: %s", e.Name)
//	    return nil
//	})
//
//	bus.Start() // begins consuming in the background
//
//	bus.Publish(ctx, eventbus.New("user.created", userID))
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/LingByte/ling-base/eventbus"
)

// Client is the Redis command subset required by this package.
type Client interface {
	XAdd(ctx context.Context, a *goredis.XAddArgs) *goredis.StringCmd
	XReadGroup(ctx context.Context, a *goredis.XReadGroupArgs) *goredis.XStreamSliceCmd
	XAck(ctx context.Context, stream, group string, ids ...string) *goredis.IntCmd
	XLen(ctx context.Context, stream string) *goredis.IntCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) *goredis.StatusCmd
}

// Bus is a Redis Streams-backed event bus.
type Bus struct {
	client     Client
	prefix     string // Redis key prefix for streams
	group      string // consumer group name
	consumer   string // consumer name within the group
	mu         sync.RWMutex
	subs       map[string]*subscription
	middleware []eventbus.Middleware
	metrics    eventbus.MetricsCollector
	closed     atomic.Bool
	started    atomic.Bool
	blockTime  time.Duration
	workers    int
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

type subscription struct {
	id      string
	topic   string
	handler eventbus.Handler
}

// Option configures the Bus.
type Option func(*Bus)

// WithConsumerGroup sets the consumer group and consumer name.
// Required for subscribing (consuming events). Without this, the bus
// can only publish.
func WithConsumerGroup(consumer string) Option {
	return func(b *Bus) {
		b.consumer = consumer
		b.group = "eventbus-" + consumer
	}
}

// WithMiddleware adds middleware applied to all handlers.
func WithMiddleware(mw ...eventbus.Middleware) Option {
	return func(b *Bus) { b.middleware = append(b.middleware, mw...) }
}

// WithBlockTime sets how long XReadGroup blocks waiting for new events.
// Default: 1 second.
func WithBlockTime(d time.Duration) Option {
	return func(b *Bus) { b.blockTime = d }
}

// WithWorkers sets the number of concurrent consumer goroutines.
// Default: 1.
func WithWorkers(n int) Option {
	return func(b *Bus) { b.workers = n }
}

// New creates a new Redis Streams event bus.
//   - client: Redis client
//   - prefix: key prefix for streams (e.g. "myapp:events")
func New(client Client, prefix string, opts ...Option) (*Bus, error) {
	if prefix == "" {
		prefix = "eventbus"
	}
	b := &Bus{
		client:    client,
		prefix:    prefix,
		subs:      make(map[string]*subscription),
		blockTime: time.Second,
		workers:   1,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b, nil
}

// streamKey returns the Redis key for a topic.
func (b *Bus) streamKey(topic string) string {
	return fmt.Sprintf("%s:%s", b.prefix, topic)
}

// Publish serializes the event and adds it to the Redis stream for its topic.
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

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("eventbus: marshal event: %w", err)
	}

	stream := b.streamKey(e.Name)
	err = b.client.XAdd(ctx, &goredis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"event": string(data),
		},
		MaxLen: 10000, // trim to ~10k entries
		Approx: true,
	}).Err()
	if err != nil {
		return fmt.Errorf("eventbus: xadd to %s: %w", stream, err)
	}

	b.metrics.RecordPublish()
	return nil
}

// Subscribe registers a handler for the given topic pattern.
// The bus must have a consumer group set (via WithConsumerGroup).
// Start() must be called to begin consuming.
func (b *Bus) Subscribe(topic string, handler eventbus.Handler) eventbus.Subscription {
	if topic == "" {
		topic = "*"
	}
	if b.group == "" {
		// Use default group based on prefix.
		b.group = "eventbus-default"
		b.consumer = "consumer-1"
	}
	id := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	s := &subscription{id: id, topic: topic, handler: handler}

	b.mu.Lock()
	b.subs[topic] = s // one handler per topic for Redis (simplified)
	b.mu.Unlock()

	b.metrics.RecordSubscribe()
	return eventbus.NewSubscription(id, topic)
}

// Unsubscribe removes a subscription.
func (b *Bus) Unsubscribe(sub eventbus.Subscription) error {
	if sub == nil {
		return fmt.Errorf("eventbus: subscription is nil")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subs[sub.Topic()]; ok {
		delete(b.subs, sub.Topic())
		b.metrics.RecordUnsubscribe()
	}
	return nil
}

// Start begins consuming events from Redis in background goroutines.
// This blocks until Close() is called or the context is cancelled.
// Call this in a goroutine: go bus.Start()
func (b *Bus) Start() {
	if !b.started.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	// Create consumer groups for all subscribed topic streams.
	b.mu.RLock()
	for topic := range b.subs {
		stream := b.streamKey(topic)
		// MKSTREAM ensures the stream exists even if no events have been published.
		_ = b.client.XGroupCreateMkStream(ctx, stream, b.group, "$").Err()
	}
	b.mu.RUnlock()

	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.consume(ctx, i)
	}
}

// consume reads events from Redis streams and dispatches to handlers.
func (b *Bus) consume(ctx context.Context, workerID int) {
	defer b.wg.Done()

	// Build the list of streams to read from based on subscriptions.
	// For wildcard topics, we read from a single combined stream.
	b.mu.RLock()
	topics := make([]string, 0, len(b.subs))
	for topic := range b.subs {
		topics = append(topics, topic)
	}
	b.mu.RUnlock()

	if len(topics) == 0 {
		return
	}

	// Build stream IDs for XReadGroup (start from new messages).
	streams := make([]string, len(topics))
	ids := make([]string, len(topics))
	for i, t := range topics {
		streams[i] = b.streamKey(t)
		ids[i] = ">"
	}

	for {
		if b.closed.Load() {
			return
		}

		// Check context.
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := b.client.XReadGroup(ctx, &goredis.XReadGroupArgs{
			Group:    b.group,
			Consumer: b.consumer,
			Streams:  append(streams, ids...),
			Count:    10,
			Block:    b.blockTime,
		}).Result()
		if err != nil && err != goredis.Nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if resp == nil {
			continue
		}

		for _, stream := range resp {
			for _, msg := range stream.Messages {
				b.handleMessage(ctx, stream.Stream, msg)
			}
		}
	}
}

// handleMessage deserializes and dispatches a Redis stream message.
func (b *Bus) handleMessage(ctx context.Context, stream string, msg goredis.XMessage) {
	data, ok := msg.Values["event"].(string)
	if !ok {
		_ = b.client.XAck(ctx, stream, b.group, msg.ID)
		return
	}

	var e eventbus.Event
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		_ = b.client.XAck(ctx, stream, b.group, msg.ID)
		return
	}

	// Find matching handler.
	b.mu.RLock()
	var handler eventbus.Handler
	for _, s := range b.subs {
		if eventbus.TopicMatches(s.topic, e.Name) {
			handler = s.handler
			break
		}
	}
	b.mu.RUnlock()

	if handler == nil {
		_ = b.client.XAck(ctx, stream, b.group, msg.ID)
		return
	}

	if len(b.middleware) > 0 {
		handler = eventbus.ApplyMiddleware(handler, b.middleware...)
	}

	b.metrics.RecordPending()
	start := time.Now()
	err := handler(ctx, &e)
	b.metrics.RecordDelivered(time.Since(start))

	if err != nil {
		b.metrics.RecordFailed()
		// Don't ACK on error — the message stays in the pending list
		// for retry/manual inspection.
		return
	}

	_ = b.client.XAck(ctx, stream, b.group, msg.ID)
}

// Close shuts down the bus, stopping all consumer goroutines.
func (b *Bus) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
	return nil
}

// Metrics returns a snapshot of bus metrics.
func (b *Bus) Metrics() eventbus.Metrics {
	return b.metrics.Snapshot()
}

// StreamLen returns the length of a stream (number of events).
func (b *Bus) StreamLen(ctx context.Context, topic string) (int64, error) {
	return b.client.XLen(ctx, b.streamKey(topic)).Result()
}

// PurgeStream deletes all events in a stream.
func (b *Bus) PurgeStream(ctx context.Context, topic string) error {
	return b.client.Del(ctx, b.streamKey(topic)).Err()
}
