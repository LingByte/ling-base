// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package retry

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errTransient is a retryable error used in tests.
var errTransient = errors.New("transient error")

// errFatal is a non-retryable error used in tests.
var errFatal = errors.New("fatal error")

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}, WithMaxAttempts(3))

	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errTransient
		}
		return nil
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_MaxAttemptsExceeded(t *testing.T) {
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		calls.Add(1)
		return errTransient
	},
		WithMaxAttempts(3),
		WithNoBackoff(),
	)

	require.Error(t, err)
	assert.True(t, IsMaxAttempts(err))
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_NonRetryableErrorStopsImmediately(t *testing.T) {
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		calls.Add(1)
		return errFatal
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
		WithRetryIf(func(err error) bool {
			return errors.Is(err, errTransient)
		}),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errFatal)
	assert.False(t, IsMaxAttempts(err))
	assert.Equal(t, int32(1), calls.Load())
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var calls atomic.Int32
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, func(ctx context.Context) error {
		calls.Add(1)
		return errTransient
	},
		WithMaxAttempts(-1), // unlimited
		WithFixedInterval(100*time.Millisecond),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDo_TimeoutExceeded(t *testing.T) {
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		calls.Add(1)
		return errTransient
	},
		WithMaxAttempts(-1), // unlimited
		WithTimeout(100*time.Millisecond),
		WithFixedInterval(50*time.Millisecond),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.GreaterOrEqual(t, calls.Load(), int32(1))
}

func TestDo_OnRetryCallback(t *testing.T) {
	var retryCalls atomic.Int32
	var successCalls atomic.Int32

	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errTransient
		}
		return nil
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
		WithOnRetry(func(attempt int, err error) {
			retryCalls.Add(1)
		}),
		WithOnSuccess(func(attempts int) {
			successCalls.Add(1)
			assert.Equal(t, 3, attempts)
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), retryCalls.Load()) // 2 retries before success on 3rd
	assert.Equal(t, int32(1), successCalls.Load())
}

func TestDo_OnErrorCallback(t *testing.T) {
	var errorCalls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		return errTransient
	},
		WithMaxAttempts(3),
		WithNoBackoff(),
		WithOnError(func(attempts int, err error) {
			errorCalls.Add(1)
			assert.Equal(t, 3, attempts)
		}),
	)

	require.Error(t, err)
	assert.Equal(t, int32(1), errorCalls.Load())
}

func TestDo_NilOperation(t *testing.T) {
	err := Do(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoOperation)
}

func TestExponentialBackoff_NextDelay(t *testing.T) {
	bo := NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0, false)

	assert.Equal(t, 100*time.Millisecond, bo.NextDelay(0))
	assert.Equal(t, 200*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 400*time.Millisecond, bo.NextDelay(2))
	assert.Equal(t, 800*time.Millisecond, bo.NextDelay(3))
}

func TestExponentialBackoff_MaxDelayCap(t *testing.T) {
	bo := NewExponentialBackoff(100*time.Millisecond, 500*time.Millisecond, 2.0, false)

	// 100, 200, 400, 500 (capped), 500 (capped)
	assert.Equal(t, 100*time.Millisecond, bo.NextDelay(0))
	assert.Equal(t, 200*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 400*time.Millisecond, bo.NextDelay(2))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(3))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(4))
}

func TestExponentialBackoff_Jitter(t *testing.T) {
	bo := NewExponentialBackoff(100*time.Millisecond, 0, 2.0, true)

	// With jitter, the delay should be within ±25% of the base delay.
	d0 := bo.NextDelay(0)
	assert.GreaterOrEqual(t, d0, 75*time.Millisecond)
	assert.LessOrEqual(t, d0, 125*time.Millisecond)
}

func TestFixedInterval_NextDelay(t *testing.T) {
	bo := NewFixedInterval(500 * time.Millisecond)

	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(0))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(1))
	assert.Equal(t, 500*time.Millisecond, bo.NextDelay(100))
}

func TestNoBackoff_NextDelay(t *testing.T) {
	bo := NoBackoff{}
	assert.Equal(t, time.Duration(0), bo.NextDelay(0))
	assert.Equal(t, time.Duration(0), bo.NextDelay(100))
}

func TestDecorate(t *testing.T) {
	var calls atomic.Int32

	retriedFn := Decorate(
		func(ctx context.Context, x int) (int, error) {
			n := calls.Add(1)
			if n < 2 {
				return 0, errTransient
			}
			return x * 2, nil
		},
		WithMaxAttempts(3),
		WithNoBackoff(),
	)

	result, err := retriedFn(context.Background(), 21)
	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDecorate0(t *testing.T) {
	var calls atomic.Int32

	retriedFn := Decorate0(
		func(ctx context.Context) error {
			n := calls.Add(1)
			if n < 2 {
				return errTransient
			}
			return nil
		},
		WithMaxAttempts(3),
		WithNoBackoff(),
	)

	err := retriedFn(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDo_DefaultMaxAttempts(t *testing.T) {
	// Without WithMaxAttempts, default is 3.
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		calls.Add(1)
		return errTransient
	}, WithNoBackoff())

	require.Error(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDo_UnlimitedRetriesUntilSuccess(t *testing.T) {
	// MaxAttempts < 0 means unlimited. Should succeed eventually.
	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := calls.Add(1)
		if n >= 10 {
			return nil
		}
		return errTransient
	},
		WithMaxAttempts(-1),
		WithNoBackoff(),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(10), calls.Load())
}

func TestDo_PreservesLastErrorInTimeoutMessage(t *testing.T) {
	customErr := fmt.Errorf("custom network error")
	err := Do(context.Background(), func(ctx context.Context) error {
		return customErr
	},
		WithMaxAttempts(-1),
		WithTimeout(50*time.Millisecond),
		WithFixedInterval(20*time.Millisecond),
	)

	require.Error(t, err)
	// The error should mention the last operation error.
	assert.Contains(t, err.Error(), "custom network error")
}

func TestExponentialBackoff_Defaults(t *testing.T) {
	// Zero base and factor <= 1 should get sensible defaults.
	bo := NewExponentialBackoff(0, 0, 1.0, false)
	assert.Equal(t, 100*time.Millisecond, bo.Base)
	assert.Equal(t, 2.0, bo.Factor)
}

func TestFixedInterval_Default(t *testing.T) {
	bo := NewFixedInterval(0)
	assert.Equal(t, 100*time.Millisecond, bo.Interval)
}

func TestDo_RetryIfWithErrorsAs(t *testing.T) {
	// Use a proper error type that implements error.
	myErr := errors.New("custom transient")

	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 2 {
			return fmt.Errorf("wrapped: %w", myErr)
		}
		return nil
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
		WithRetryIf(func(err error) bool {
			return errors.Is(err, myErr)
		}),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

// stubCircuitBreaker is a test-only CircuitBreaker implementation.
type stubCircuitBreaker struct {
	open     bool
	executed atomic.Int32
}

func (s *stubCircuitBreaker) Execute(ctx context.Context, op func(context.Context) error) error {
	s.executed.Add(1)
	if s.open {
		return errors.New("circuitbreaker: circuit is open")
	}
	return op(ctx)
}

func TestDo_WithCircuitBreaker_Open(t *testing.T) {
	cb := &stubCircuitBreaker{open: true}

	var opCalls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		opCalls.Add(1)
		return nil
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
		WithCircuitBreaker(cb),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit is open")
	// Operation should never execute when breaker is open.
	assert.Equal(t, int32(0), opCalls.Load())
	// Breaker.Execute should be called once (then loop stops).
	assert.Equal(t, int32(1), cb.executed.Load())
}

func TestDo_WithCircuitBreaker_Closed(t *testing.T) {
	cb := &stubCircuitBreaker{open: false}

	var calls atomic.Int32
	err := Do(context.Background(), func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 2 {
			return errTransient
		}
		return nil
	},
		WithMaxAttempts(5),
		WithNoBackoff(),
		WithCircuitBreaker(cb),
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, int32(2), cb.executed.Load())
}
