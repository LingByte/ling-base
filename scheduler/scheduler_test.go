// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/lock"
	"github.com/LingByte/ling-base/lock/memory"
)

// ──────────────────────────────────────────────
// Config / construction
// ──────────────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.IsType(t, NoLockFactory{}, cfg.LockFactory)
	assert.Equal(t, 30*time.Second, cfg.LockTTL)
}

func TestNew_Defaults(t *testing.T) {
	s := New(Config{})
	assert.NotNil(t, s)
	assert.Equal(t, 30*time.Second, s.cfg.LockTTL)
	assert.Equal(t, 10*time.Second, s.cfg.LockRefreshInterval)
	assert.NotNil(t, s.cfg.Timezone)
}

func TestNew_WithLockFactory(t *testing.T) {
	mgr := memory.NewManager()
	s := New(Config{
		LockFactory: LockFactoryFunc(func(name string) (lock.Locker, error) {
			return mgr.NewMutex("test:"+name, lock.WithTTL(10*time.Second))
		}),
	})
	assert.NotNil(t, s)
}

// ──────────────────────────────────────────────
// Schedule parsing
// ──────────────────────────────────────────────

func TestParseSchedule_Cron(t *testing.T) {
	expr, interval, err := parseSchedule("*/5 * * * *")
	require.NoError(t, err)
	require.NotNil(t, expr)
	assert.Equal(t, time.Duration(0), interval)
}

func TestParseSchedule_Every(t *testing.T) {
	expr, interval, err := parseSchedule("@every 5s")
	require.NoError(t, err)
	assert.Nil(t, expr)
	assert.Equal(t, 5*time.Second, interval)
}

func TestParseSchedule_InvalidCron(t *testing.T) {
	_, _, err := parseSchedule("invalid")
	require.Error(t, err)
}

func TestParseSchedule_InvalidEvery(t *testing.T) {
	_, _, err := parseSchedule("@every abc")
	require.Error(t, err)
}

func TestParseSchedule_NegativeInterval(t *testing.T) {
	_, _, err := parseSchedule("@every -5s")
	require.Error(t, err)
}

// ──────────────────────────────────────────────
// Add / Remove
// ──────────────────────────────────────────────

func TestAdd(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("test", "@every 1s", func(ctx context.Context) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, 1, s.JobCount())
}

func TestAdd_DuplicateName(t *testing.T) {
	s := New(DefaultConfig())
	_ = s.Add("test", "@every 1s", func(ctx context.Context) error { return nil })
	err := s.Add("test", "@every 2s", func(ctx context.Context) error { return nil })
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrJobExists)
}

func TestAdd_EmptyName(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("", "@every 1s", func(ctx context.Context) error { return nil })
	require.Error(t, err)
}

func TestAdd_EmptySchedule(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("test", "", func(ctx context.Context) error { return nil })
	require.Error(t, err)
}

func TestAdd_NilFunc(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("test", "@every 1s", nil)
	require.Error(t, err)
}

func TestAdd_InvalidSchedule(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("test", "not-a-cron-expr", func(ctx context.Context) error { return nil })
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSchedule)
}

func TestAddWithOptions(t *testing.T) {
	s := New(DefaultConfig())
	err := s.AddWithOptions("test", "@every 1s",
		func(ctx context.Context) error { return nil },
		Options{
			Description: "test job",
			Timeout:     5 * time.Second,
			Singleton:   true,
			Tags:        []string{"test"},
		},
	)
	require.NoError(t, err)

	job, err := s.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "test job", job.Description)
	assert.Equal(t, 5*time.Second, job.Timeout)
	assert.True(t, job.Singleton)
	assert.Contains(t, job.Tags, "test")
}

func TestRemove(t *testing.T) {
	s := New(DefaultConfig())
	_ = s.Add("test", "@every 1s", func(ctx context.Context) error { return nil })

	err := s.Remove("test")
	require.NoError(t, err)
	assert.Equal(t, 0, s.JobCount())
}

func TestRemove_NotFound(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Remove("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestGet(t *testing.T) {
	s := New(DefaultConfig())
	_ = s.Add("test", "@every 1s", func(ctx context.Context) error { return nil })

	job, err := s.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "test", job.Name)
}

func TestGet_NotFound(t *testing.T) {
	s := New(DefaultConfig())
	_, err := s.Get("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestJobs(t *testing.T) {
	s := New(DefaultConfig())
	_ = s.Add("job1", "@every 1s", func(ctx context.Context) error { return nil })
	_ = s.Add("job2", "@every 2s", func(ctx context.Context) error { return nil })

	jobs := s.Jobs()
	assert.Len(t, jobs, 2)
}

// ──────────────────────────────────────────────
// Execution (single node, NoLockFactory)
// ──────────────────────────────────────────────

func TestStartStop(t *testing.T) {
	s := New(DefaultConfig())
	s.Start()
	s.Stop()
}

func TestStartStop_Idempotent(t *testing.T) {
	s := New(DefaultConfig())
	s.Start()
	s.Start() // second call should be no-op
	s.Stop()
	s.Stop() // second call should be no-op
}

func TestJobExecution_Interval(t *testing.T) {
	s := New(DefaultConfig())
	var count int32
	err := s.Add("counter", "@every 50ms", func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	require.NoError(t, err)

	s.Start()
	time.Sleep(250 * time.Millisecond)
	s.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(3))
}

func TestJobExecution_Cron(t *testing.T) {
	s := New(DefaultConfig())
	var count int32
	// Every second.
	err := s.Add("cron-counter", "* * * * * *", func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	require.NoError(t, err)

	s.Start()
	time.Sleep(2500 * time.Millisecond)
	s.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(2))
}

func TestJobExecution_Error(t *testing.T) {
	s := New(DefaultConfig())
	var events []JobEvent
	s.cfg.EventListener = EventListenerFunc(func(e JobEvent) {
		events = append(events, e)
	})

	err := s.Add("error-job", "@every 50ms", func(ctx context.Context) error {
		return errors.New("job failed")
	})
	require.NoError(t, err)

	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	// Should have at least one failed event.
	var hasFailed bool
	for _, e := range events {
		if e.Type == EventJobFailed {
			hasFailed = true
			assert.Error(t, e.Error)
		}
	}
	assert.True(t, hasFailed)

	// Check job status.
	job, _ := s.Get("error-job")
	status := job.Status()
	assert.Positive(t, status.ErrorCount)
	assert.Error(t, status.LastError)
}

func TestJobExecution_Success(t *testing.T) {
	s := New(DefaultConfig())
	var events []JobEvent
	var mu sync.Mutex
	s.cfg.EventListener = EventListenerFunc(func(e JobEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	err := s.Add("success-job", "@every 50ms", func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)

	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	var hasSucceeded bool
	for _, e := range events {
		if e.Type == EventJobSucceeded {
			hasSucceeded = true
		}
	}
	assert.True(t, hasSucceeded)
}

func TestJobStatus(t *testing.T) {
	s := New(DefaultConfig())
	err := s.Add("status-job", "@every 50ms", func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)

	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	job, _ := s.Get("status-job")
	status := job.Status()
	assert.Positive(t, status.RunCount)
	assert.False(t, status.LastRun.IsZero())
	assert.False(t, status.LastEnd.IsZero())
	assert.False(t, status.NextRun.IsZero())
}

func TestJobTimeout(t *testing.T) {
	s := New(DefaultConfig())
	var executed bool
	var mu sync.Mutex

	err := s.AddWithOptions("timeout-job", "@every 50ms",
		func(ctx context.Context) error {
			mu.Lock()
			executed = true
			mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				return nil
			}
		},
		Options{Timeout: 100 * time.Millisecond},
	)
	require.NoError(t, err)

	s.Start()
	time.Sleep(300 * time.Millisecond)
	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, executed, "job should have been executed")
}

// ──────────────────────────────────────────────
// Singleton mode
// ──────────────────────────────────────────────

func TestSingleton_SkipsIfRunning(t *testing.T) {
	s := New(DefaultConfig())
	var started, completed int32

	err := s.AddWithOptions("singleton-job", "@every 50ms",
		func(ctx context.Context) error {
			atomic.AddInt32(&started, 1)
			time.Sleep(200 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		},
		Options{Singleton: true},
	)
	require.NoError(t, err)

	s.Start()
	time.Sleep(300 * time.Millisecond)
	s.Stop()

	// Should have started fewer times than the number of ticks.
	startedVal := atomic.LoadInt32(&started)
	completedVal := atomic.LoadInt32(&completed)
	assert.GreaterOrEqual(t, startedVal, int32(1))
	assert.Equal(t, startedVal, completedVal)
	// In 300ms with 50ms interval, we'd have ~6 ticks, but singleton
	// should have blocked most of them.
	assert.LessOrEqual(t, startedVal, int32(3))
}

// ──────────────────────────────────────────────
// Distributed lock (memory backend)
// ──────────────────────────────────────────────

func TestDistributedLock_OnlyOneNodeRuns(t *testing.T) {
	mgr := memory.NewManager()

	// Create two schedulers sharing the same lock backend (via same manager).
	// They use the same lock key, so only one should run the job at a time.
	factory := func(name string) (lock.Locker, error) {
		return mgr.NewMutex("sched:"+name, lock.WithTTL(5*time.Second))
	}

	s1 := New(Config{LockFactory: LockFactoryFunc(factory), LockTTL: 5 * time.Second})
	s2 := New(Config{LockFactory: LockFactoryFunc(factory), LockTTL: 5 * time.Second})

	var count int32
	jobFunc := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		// Hold the lock for a bit so the other scheduler's TryLock fails.
		time.Sleep(30 * time.Millisecond)
		return nil
	}

	_ = s1.Add("distributed", "@every 100ms", jobFunc)
	_ = s2.Add("distributed", "@every 100ms", jobFunc)

	s1.Start()
	s2.Start()
	time.Sleep(500 * time.Millisecond)
	s1.Stop()
	s2.Stop()

	// Total count should be roughly the number of ticks, not double.
	// (Each tick, only one of the two schedulers runs the job.)
	total := atomic.LoadInt32(&count)
	assert.GreaterOrEqual(t, total, int32(3))
	assert.LessOrEqual(t, total, int32(7))
}

func TestDistributedLock_SkippedEvent(t *testing.T) {
	mgr := memory.NewManager()
	factory := func(name string) (lock.Locker, error) {
		return mgr.NewMutex("sched:"+name, lock.WithTTL(5*time.Second))
	}

	var mu sync.Mutex
	var skippedCount int
	s1 := New(Config{LockFactory: LockFactoryFunc(factory), LockTTL: 5 * time.Second})
	s2 := New(Config{LockFactory: LockFactoryFunc(factory), LockTTL: 5 * time.Second})

	s2.cfg.EventListener = EventListenerFunc(func(e JobEvent) {
		if e.Type == EventJobSkipped {
			mu.Lock()
			skippedCount++
			mu.Unlock()
		}
	})

	_ = s1.Add("lock-test", "@every 50ms", func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond) // hold lock for a while
		return nil
	})
	_ = s2.Add("lock-test", "@every 50ms", func(ctx context.Context) error {
		return nil
	})

	s1.Start()
	s2.Start()
	time.Sleep(300 * time.Millisecond)
	s1.Stop()
	s2.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, skippedCount, 1, "s2 should have skipped at least once")
}

// ──────────────────────────────────────────────
// Event listener
// ──────────────────────────────────────────────

func TestEventListener(t *testing.T) {
	s := New(DefaultConfig())
	var mu sync.Mutex
	var events []JobEvent
	s.cfg.EventListener = EventListenerFunc(func(e JobEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	_ = s.Add("event-job", "@every 50ms", func(ctx context.Context) error {
		return nil
	})

	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	// Should have: Added, Started, Succeeded events.
	var hasAdded, hasStarted, hasSucceeded bool
	for _, e := range events {
		switch e.Type {
		case EventJobAdded:
			hasAdded = true
		case EventJobStarted:
			hasStarted = true
		case EventJobSucceeded:
			hasSucceeded = true
		}
	}
	assert.True(t, hasAdded)
	assert.True(t, hasStarted)
	assert.True(t, hasSucceeded)
}

// ──────────────────────────────────────────────
// LockFactoryFunc
// ──────────────────────────────────────────────

func TestLockFactoryFunc(t *testing.T) {
	f := LockFactoryFunc(func(name string) (lock.Locker, error) {
		return nil, nil
	})
	l, err := f.NewLock("test")
	require.NoError(t, err)
	assert.Nil(t, l)
}

func TestNoLockFactory(t *testing.T) {
	f := NoLockFactory{}
	l, err := f.NewLock("test")
	require.NoError(t, err)
	assert.Nil(t, l)
}

// ──────────────────────────────────────────────
// Add after Start
// ──────────────────────────────────────────────

func TestAddAfterStart(t *testing.T) {
	s := New(DefaultConfig())
	s.Start()
	defer s.Stop()

	var count int32
	err := s.Add("late-job", "@every 50ms", func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&count), int32(2))
}

// ──────────────────────────────────────────────
// Remove while running
// ──────────────────────────────────────────────

func TestRemoveWhileRunning(t *testing.T) {
	s := New(DefaultConfig())
	_ = s.Add("removable", "@every 50ms", func(ctx context.Context) error {
		return nil
	})
	s.Start()
	time.Sleep(100 * time.Millisecond)

	err := s.Remove("removable")
	require.NoError(t, err)
	assert.Equal(t, 0, s.JobCount())

	s.Stop()
}
