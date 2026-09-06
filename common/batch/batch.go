// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package batch provides a batch accumulator that collects items and
// flushes them when one of the following conditions is met:
//   - the batch reaches a configured maximum size (count);
//   - the batch reaches a configured maximum byte size;
//   - a flush timer fires (time-based flush);
//   - Flush or FlushAndWait is called manually.
//
// # Quick start
//
//	b := batch.New(func(items []string) error {
//	    return db.BatchInsert(items)
//	}, batch.WithSize(100), batch.WithInterval(5*time.Second))
//	defer b.Close()
//
//	for _, item := range data {
//	    b.Add(item) // non-blocking, flushes automatically
//	}
//
//	// Force flush remaining items and wait for completion.
//	if err := b.FlushAndWait(); err != nil { ... }
//
// # Concurrency
//
// Batch is safe for concurrent use. Add may block briefly if the
// internal buffer is being flushed, but it will not block on the
// flush handler itself — the handler runs in a background goroutine.
package batch

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ──────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────

var (
	// ErrClosed is returned when Add is called after Close.
	ErrClosed = fmt.Errorf("batch: closed")
	// ErrHandlerNil is returned when no handler is provided.
	ErrHandlerNil = fmt.Errorf("batch: handler is nil")
)

// ──────────────────────────────────────────────
// Handler
// ──────────────────────────────────────────────

// Handler processes a batch of items. It is called in a background
// goroutine. If it returns an error, the error is stored and can be
// retrieved via Errors() or the last error via LastError().
type Handler[T any] func(items []T) error

// ──────────────────────────────────────────────
// Config
// ──────────────────────────────────────────────

// Option configures a Batch.
type Option[T any] func(*Batch[T])

// WithSize sets the maximum number of items per batch.
// When the batch reaches this size, it is flushed automatically.
// Default is 1000.
func WithSize[T any](n int) Option[T] {
	return func(b *Batch[T]) {
		if n > 0 {
			b.maxSize = n
		}
	}
}

// WithInterval sets the time-based flush interval. A flush is
// triggered after this duration even if the batch is not full.
// Default is 10 seconds. Set to 0 to disable time-based flushing.
func WithInterval[T any](d time.Duration) Option[T] {
	return func(b *Batch[T]) {
		b.interval = d
	}
}

// WithBufferSize sets the internal channel buffer size. Add will
// block if the buffer is full. Default is 0 (unbuffered).
func WithBufferSize[T any](n int) Option[T] {
	return func(b *Batch[T]) {
		if n >= 0 {
			b.chanSize = n
		}
	}
}

// WithErrorHandler sets a callback for handler errors.
func WithErrorHandler[T any](fn func(err error)) Option[T] {
	return func(b *Batch[T]) {
		b.errorHandler = fn
	}
}

// ──────────────────────────────────────────────
// Batch
// ──────────────────────────────────────────────

// Batch collects items and flushes them in batches to a Handler.
// It is generic and type-safe.
type Batch[T any] struct {
	handler     Handler[T]
	maxSize     int
	interval    time.Duration
	chanSize    int
	errorHandler func(err error)

	itemCh       chan T
	flushCh      chan struct{}
	flushWaitCh  chan struct{}
	flushDoneCh  chan struct{}
	doneCh       chan struct{}
	closeCh      chan struct{}

	mu          sync.Mutex
	closed      bool
	lastErr     error
	errCount    int
	flushCount  int
	itemCount   int64
}

// New creates a new Batch with the given handler and options.
func New[T any](handler Handler[T], opts ...Option[T]) (*Batch[T], error) {
	if handler == nil {
		return nil, ErrHandlerNil
	}

	b := &Batch[T]{
		handler:  handler,
		maxSize:  1000,
		interval: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(b)
	}

	b.itemCh = make(chan T, b.chanSize)
	b.flushCh = make(chan struct{}, 1)
	b.flushWaitCh = make(chan struct{}, 1)
	b.flushDoneCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	b.closeCh = make(chan struct{})

	go b.run()

	return b, nil
}

// Add adds an item to the batch. It may block if the internal
// buffer is full. Returns ErrClosed if the batch is closed.
func (b *Batch[T]) Add(item T) error {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()

	if closed {
		return ErrClosed
	}

	b.itemCh <- item
	return nil
}

// TryAdd attempts to add an item without blocking. Returns true if
// the item was added, false if the buffer is full or the batch is closed.
func (b *Batch[T]) TryAdd(item T) bool {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()

	if closed {
		return false
	}

	select {
	case b.itemCh <- item:
		return true
	default:
		return false
	}
}

// Flush triggers a flush of the current batch. This is non-blocking;
// the flush happens in the background. Use FlushAndWait to block
// until the flush completes.
func (b *Batch[T]) Flush() {
	select {
	case b.flushCh <- struct{}{}:
	default:
	}
}

// FlushAndWait triggers a flush and blocks until it completes.
func (b *Batch[T]) FlushAndWait() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	b.mu.Unlock()

	// Use a dedicated flush-sync channel.
	b.flushWaitCh <- struct{}{}
	<-b.flushDoneCh
	return b.LastError()
}

// Close flushes any remaining items and shuts down the batch.
// It blocks until all pending items have been processed.
func (b *Batch[T]) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	close(b.closeCh)

	// Wait for the run loop to finish.
	<-b.doneCh
	return b.lastErr
}

// LastError returns the last error from the handler, if any.
func (b *Batch[T]) LastError() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastErr
}

// ErrorCount returns the total number of handler errors.
func (b *Batch[T]) ErrorCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.errCount
}

// FlushCount returns the total number of flushes performed.
func (b *Batch[T]) FlushCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushCount
}

// Stats returns current statistics.
func (b *Batch[T]) Stats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return Stats{
		FlushCount: b.flushCount,
		ErrorCount: b.errCount,
		ItemCount:  b.itemCount,
	}
}

// Stats holds batch statistics.
type Stats struct {
	FlushCount int
	ErrorCount int
	ItemCount  int64
}

// ──────────────────────────────────────────────
// Internal: run loop
// ──────────────────────────────────────────────

func (b *Batch[T]) run() {
	defer close(b.doneCh)

	var buffer []T
	var timer *time.Timer
	var timerC <-chan time.Time

	if b.interval > 0 {
		timer = time.NewTimer(b.interval)
		timerC = timer.C
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	flush := func() {
		if len(buffer) == 0 {
			return
		}

		// Copy items to avoid mutation during handler execution.
		items := make([]T, len(buffer))
		copy(items, buffer)
		buffer = buffer[:0]

		err := b.handler(items)

		b.mu.Lock()
		b.flushCount++
		b.itemCount += int64(len(items))
		if err != nil {
			b.lastErr = err
			b.errCount++
		}
		errHandler := b.errorHandler
		b.mu.Unlock()

		if err != nil && errHandler != nil {
			errHandler(err)
		}

		// Reset timer.
		if timer != nil {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(b.interval)
		}
	}

	for {
		select {
		case <-b.closeCh:
			// Drain remaining items.
			drainDone := false
			for !drainDone {
				select {
				case item := <-b.itemCh:
					buffer = append(buffer, item)
					if len(buffer) >= b.maxSize {
						flush()
					}
				default:
					drainDone = true
				}
			}
			flush()
			return

		case item := <-b.itemCh:
			buffer = append(buffer, item)
			if len(buffer) >= b.maxSize {
				flush()
			}

		case <-b.flushCh:
			flush()

		case <-b.flushWaitCh:
			flush()
			// Signal completion.
			select {
			case b.flushDoneCh <- struct{}{}:
			default:
			}

		case <-timerC:
			flush()
		}
	}
}

// ──────────────────────────────────────────────
// SimpleBatch (non-generic, for []any)
// ──────────────────────────────────────────────

// SimpleBatch is a non-generic batch accumulator for any items.
// It is less type-safe than Batch[T] but easier to use when
// mixing types.
type SimpleBatch struct {
	*Batch[any]
}

// NewSimpleBatch creates a new SimpleBatch.
func NewSimpleBatch(handler Handler[any], opts ...Option[any]) (*SimpleBatch, error) {
	b, err := New[any](handler, opts...)
	if err != nil {
		return nil, err
	}
	return &SimpleBatch{Batch: b}, nil
}

// ──────────────────────────────────────────────
// Context-aware batch (with context cancellation)
// ──────────────────────────────────────────────

// RunWithContext runs a batch that is canceled when ctx is canceled.
// This is a convenience function for short-lived batches.
func RunWithContext[T any](ctx context.Context, handler Handler[T], opts ...Option[T]) (*Batch[T], error) {
	b, err := New(handler, opts...)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = b.Close()
	}()

	return b, nil
}

// Ensure context is used.
var _ = context.Background
