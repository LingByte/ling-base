// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package tokenbucket implements a token-bucket rate limiter for QPS
// control. Tokens are refilled at a fixed rate (rate per second) up to
// a maximum burst size.
//
// Two variants are provided:
//
//   - New(rate, burst)         — non-blocking, returns ErrLimitExceeded
//   - NewBlocking(rate, burst) — blocks until a token is available or ctx is done
package tokenbucket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/ling-base/common/limiter"
)

// non-blocking token bucket using lazy refill.
type bucket struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64 // max tokens
	tokens   float64 // current tokens
	lastTime time.Time
}

// New creates a non-blocking token-bucket rate limiter.
//   - rate:  sustained tokens per second (QPS)
//   - burst: maximum burst size (initial token count)
//
// Acquire returns limiter.ErrLimitExceeded immediately when no token is
// available.
func New(rate, burst int) limiter.Limiter {
	if burst <= 0 {
		burst = rate
	}
	return &bucket{
		rate:     float64(rate),
		burst:    float64(burst),
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

func (b *bucket) Running() int {
	return -1 // not meaningful for rate limiters
}

func (b *bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.lastTime = now
}

func (b *bucket) Acquire(ctx context.Context, key []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	if b.tokens >= 1 {
		b.tokens--
		return nil
	}
	return limiter.ErrLimitExceeded
}

func (b *bucket) Release(key []byte) {
	// Token bucket doesn't use Release; tokens are auto-refilled.
}

// ---------------------------------------------------------------
// Blocking variant
// ---------------------------------------------------------------

type blockingBucket struct {
	rate   float64
	burst  float64
	tokens float64
	mu     sync.Mutex
	last   time.Time
}

// NewBlocking creates a blocking token-bucket rate limiter.
// Acquire blocks until a token is available or ctx is cancelled.
func NewBlocking(rate, burst int) limiter.Limiter {
	if burst <= 0 {
		burst = rate
	}
	return &blockingBucket{
		rate:   float64(rate),
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (b *blockingBucket) Running() int { return -1 }

func (b *blockingBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
}

func (b *blockingBucket) Acquire(ctx context.Context, key []byte) error {
	for {
		b.mu.Lock()
		b.refill()
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		// Calculate wait time for next token.
		needed := 1 - b.tokens
		waitDur := time.Duration(needed / b.rate * float64(time.Second))
		b.mu.Unlock()

		if waitDur <= 0 {
			waitDur = time.Millisecond
		}

		select {
		case <-time.After(waitDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b *blockingBucket) Release(key []byte) {}

// ---------------------------------------------------------------
// Atomic high-throughput variant (lock-free Acquire for low contention)
// ---------------------------------------------------------------

type atomicBucket struct {
	tokens   int64 // scaled by 1000 for atomic ops
	rate     int64 // tokens per second * 1000
	burst    int64
	lastNano int64
}

// NewAtomic creates a high-throughput non-blocking token-bucket using
// atomic operations. It is designed for scenarios with very high QPS
// where lock contention on the mutex variant becomes a bottleneck.
func NewAtomic(rate, burst int) limiter.Limiter {
	if burst <= 0 {
		burst = rate
	}
	return &atomicBucket{
		tokens:   int64(burst) * 1000,
		rate:     int64(rate) * 1000,
		burst:    int64(burst) * 1000,
		lastNano: time.Now().UnixNano(),
	}
}

func (b *atomicBucket) Running() int { return -1 }

func (b *atomicBucket) Acquire(ctx context.Context, key []byte) error {
	for {
		now := time.Now().UnixNano()
		last := atomic.LoadInt64(&b.lastNano)
		elapsed := now - last
		if elapsed > 0 && atomic.CompareAndSwapInt64(&b.lastNano, last, now) {
			refill := elapsed * b.rate / int64(time.Second)
			t := atomic.LoadInt64(&b.tokens) + refill
			if t > b.burst {
				t = b.burst
			}
			atomic.StoreInt64(&b.tokens, t)
		}

		for {
			t := atomic.LoadInt64(&b.tokens)
			if t < 1000 {
				return limiter.ErrLimitExceeded
			}
			if atomic.CompareAndSwapInt64(&b.tokens, t, t-1000) {
				return nil
			}
		}
	}
}

func (b *atomicBucket) Release(key []byte) {}
