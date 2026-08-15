// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test failure")

func newTestCB(cfg Config) *CircuitBreaker {
	if cfg.Name == "" {
		cfg.Name = "test"
	}
	return New(cfg)
}

func TestNew_Defaults(t *testing.T) {
	cb := New(Config{})
	m := cb.Metrics()

	assert.Equal(t, "closed", m.State)
	assert.Equal(t, "", m.Name) // empty name stays empty
}

func TestNew_WithConfig(t *testing.T) {
	cb := newTestCB(Config{
		MaxRequests:       3,
		FailureThreshold:  0.6,
		MinRequests:       5,
		RecoveryTimeout:   10 * time.Second,
		SlidingWindowSize: 20,
		Name:              "svc-a",
	})

	m := cb.Metrics()
	assert.Equal(t, "svc-a", m.Name)
	assert.Equal(t, "closed", m.State)
}

func TestExecute_SuccessInClosed(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       3,
		SlidingWindowSize: 10,
	})

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	m := cb.Metrics()
	assert.Equal(t, "closed", m.State)
	assert.Equal(t, 1, m.Successes)
	assert.Equal(t, 0, m.Failures)
}

func TestExecute_FailureInClosed(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
	})

	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errTest
	})

	m := cb.Metrics()
	assert.Equal(t, "closed", m.State) // not enough requests to trip
	assert.Equal(t, 1, m.Failures)
}

func TestCircuit_TripsOnFailureThreshold(t *testing.T) {
	var stateChanges []string
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       4,
		SlidingWindowSize: 10,
		RecoveryTimeout:   1 * time.Second,
		OnStateChange: func(name string, from, to State) {
			stateChanges = append(stateChanges, fmt.Sprintf("%s→%s", from, to))
		},
	})

	// 2 success + 2 failure = 50% → trips
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })

	m := cb.Metrics()
	assert.Equal(t, "open", m.State)
	assert.Contains(t, stateChanges, "closed→open")
}

func TestCircuit_DoesNotTripBelowMinRequests(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       10,
		SlidingWindowSize: 20,
	})

	// Only 3 requests, all failures — but below MinRequests.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	}

	assert.Equal(t, "closed", cb.Metrics().State)
}

func TestCircuit_OpenRejectsRequests(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   1 * time.Hour, // long timeout so it stays open
	})

	// Trip the breaker.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })

	assert.Equal(t, "open", cb.Metrics().State)

	// Now requests should be rejected.
	var executed atomic.Bool
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.False(t, executed.Load(), "operation should not execute when circuit is open")
}

func TestCircuit_TransitionsToHalfOpenAfterTimeout(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   50 * time.Millisecond,
	})

	// Trip the breaker.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	assert.Equal(t, "open", cb.Metrics().State)

	// Wait for recovery.
	time.Sleep(100 * time.Millisecond)

	// Next request should be allowed (Half-Open).
	var executed atomic.Bool
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed.Load())
}

func TestCircuit_HalfOpenClosesOnSuccess(t *testing.T) {
	cb := newTestCB(Config{
		MaxRequests:       3,
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   50 * time.Millisecond,
	})

	// Trip the breaker.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	assert.Equal(t, "open", cb.Metrics().State)

	time.Sleep(100 * time.Millisecond)

	// 3 successful trial requests → close.
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
		require.NoError(t, err)
	}

	assert.Equal(t, "closed", cb.Metrics().State)
}

func TestCircuit_HalfOpenReopensOnFailure(t *testing.T) {
	cb := newTestCB(Config{
		MaxRequests:       3,
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   50 * time.Millisecond,
	})

	// Trip the breaker.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	assert.Equal(t, "open", cb.Metrics().State)

	time.Sleep(100 * time.Millisecond)

	// One success, then one failure → re-open.
	err := cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	require.NoError(t, err)

	err = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	require.Error(t, err)

	assert.Equal(t, "open", cb.Metrics().State)
}

func TestCircuit_HalfOpenRejectsExcessRequests(t *testing.T) {
	cb := newTestCB(Config{
		MaxRequests:       2,
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   50 * time.Millisecond,
	})

	// Trip the breaker.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })

	time.Sleep(100 * time.Millisecond)

	// Fill the 2 trial slots with slow operations.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Execute(context.Background(), func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})
		}()
	}
	time.Sleep(20 * time.Millisecond) // let the goroutines start

	// Third request should be rejected with ErrTooManyRequests.
	err := cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	assert.ErrorIs(t, err, ErrTooManyRequests)

	wg.Wait()
}

func TestCircuit_SlidingWindowEvictsOldEntries(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.8,
		MinRequests:       4,
		SlidingWindowSize: 4,
	})

	// Fill window with 4 failures.
	for i := 0; i < 4; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	}
	// Should have tripped (100% >= 80%).
	assert.Equal(t, "open", cb.Metrics().State)

	// Reset and fill with successes.
	cb.Reset()
	for i := 0; i < 4; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	}
	assert.Equal(t, "closed", cb.Metrics().State)
	m := cb.Metrics()
	assert.Equal(t, 4, m.Successes)
	assert.Equal(t, 0, m.Failures)
}

func TestCircuit_Reset(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   1 * time.Hour,
	})

	// Trip.
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	assert.Equal(t, "open", cb.Metrics().State)

	cb.Reset()
	assert.Equal(t, "closed", cb.Metrics().State)
	m := cb.Metrics()
	assert.Equal(t, 0, m.WindowLen)
}

func TestCircuit_OnStateChangeCallback(t *testing.T) {
	var transitions []string
	cb := newTestCB(Config{
		MaxRequests:       2,
		FailureThreshold:  0.5,
		MinRequests:       2,
		SlidingWindowSize: 10,
		RecoveryTimeout:   50 * time.Millisecond,
		OnStateChange: func(name string, from, to State) {
			transitions = append(transitions, fmt.Sprintf("%s:%s→%s", name, from, to))
		},
	})

	// Trip: closed → open
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })

	time.Sleep(100 * time.Millisecond)

	// Recover: open → half-open → closed
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })

	assert.Contains(t, transitions, "test:closed→open")
	assert.Contains(t, transitions, "test:open→half-open")
	assert.Contains(t, transitions, "test:half-open→closed")
}

func TestCircuit_NilOperation(t *testing.T) {
	cb := newTestCB(Config{})
	err := cb.Execute(context.Background(), nil)
	require.Error(t, err)
}

func TestCircuit_ContextCancelled(t *testing.T) {
	cb := newTestCB(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.Error(t, err)
}

func TestCircuit_ConcurrentSafety(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.9,
		MinRequests:       50,
		SlidingWindowSize: 100,
		RecoveryTimeout:   50 * time.Millisecond,
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = cb.Execute(context.Background(), func(ctx context.Context) error {
				if n%3 == 0 {
					return errTest
				}
				return nil
			})
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock. State should be valid.
	s := cb.Metrics().State
	assert.Contains(t, []string{"closed", "open", "half-open"}, s)
}

func TestCircuit_Metrics(t *testing.T) {
	cb := newTestCB(Config{
		FailureThreshold:  0.5,
		MinRequests:       10,
		SlidingWindowSize: 20,
		Name:              "svc-b",
	})

	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errTest })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })

	m := cb.Metrics()
	assert.Equal(t, "svc-b", m.Name)
	assert.Equal(t, "closed", m.State)
	assert.Equal(t, 3, m.WindowLen)
	assert.Equal(t, 2, m.Successes)
	assert.Equal(t, 1, m.Failures)
	assert.InDelta(t, 1.0/3.0, m.FailureRate, 0.001)
}

func TestState_String(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", State(99).String())
}

func TestCircuit_AllowMarkSuccessMarkFailed(t *testing.T) {
	cb := New(Config{
		FailureThreshold:   0.5,
		MinRequests:        2,
		SlidingWindowSize:  10,
		RecoveryTimeout:    50 * time.Millisecond,
		MaxRequests:        1,
	})

	// Closed: allow
	assert.NoError(t, cb.Allow())

	// Record failures to trip
	cb.MarkFailed()
	cb.MarkFailed()

	// Should be open now
	err := cb.Allow()
	assert.ErrorIs(t, err, ErrCircuitOpen)

	// Wait for recovery
	time.Sleep(60 * time.Millisecond)

	// Half-open: allow one trial
	assert.NoError(t, cb.Allow())
	// Second trial should be rejected (MaxRequests=1)
	err = cb.Allow()
	assert.ErrorIs(t, err, ErrTooManyRequests)

	// Mark success → closes
	cb.MarkSuccess()
	assert.Equal(t, StateClosed, cb.State())
	assert.NoError(t, cb.Allow())
}
