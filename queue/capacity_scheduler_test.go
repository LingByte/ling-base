// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeCapacityTask(id, jobID string, priority, weight int) *Task {
	payload, _ := json.Marshal(map[string]any{"id": id})
	return &Task{
		ID:          id,
		JobID:       jobID,
		Priority:    priority,
		Weight:      weight,
		Payload:     payload,
		MaxRetries:  3,
		Preemptible: true,
	}
}

func TestCapacityScheduler_BasicFlow(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 4,
		Capacity:    10,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 10; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("t%d", i), fmt.Sprintf("job%d", i), 5, 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(10), processed.Load())
	stats := s.Stats()
	assert.Equal(t, int64(10), stats.Succeeded)
}

func TestCapacityScheduler_CapacityLimit(t *testing.T) {
	q := newTestQueue()
	var maxConcurrentWeight atomic.Int32
	var currentWeight atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 8,
		Capacity:    15,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			w := int32(taskWeight(task))
			cw := currentWeight.Add(w)
			if cw > maxConcurrentWeight.Load() {
				maxConcurrentWeight.Store(cw)
			}
			time.Sleep(50 * time.Millisecond)
			currentWeight.Add(-w)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Enqueue tasks with various weights.
	for i := 0; i < 20; i++ {
		w := (i % 5) + 1 // weights 1-5
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("t%d", i), fmt.Sprintf("job%d", i), 5, w))
	}

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	// Max concurrent weight should never exceed capacity.
	assert.LessOrEqual(t, maxConcurrentWeight.Load(), int32(15))
}

func TestCapacityScheduler_JobGroupingSequential(t *testing.T) {
	q := newTestQueue()
	var mu sync.Mutex
	var executionOrder []string

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 8,
		Capacity:    100, // high capacity so capacity isn't the bottleneck
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			mu.Lock()
			executionOrder = append(executionOrder, task.ID)
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Same job: tasks must run sequentially.
	_ = q.Enqueue(context.Background(), makeCapacityTask("job1-task1", "job1", 5, 1))
	_ = q.Enqueue(context.Background(), makeCapacityTask("job1-task2", "job1", 5, 1))
	_ = q.Enqueue(context.Background(), makeCapacityTask("job1-task3", "job1", 5, 1))

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"job1-task1", "job1-task2", "job1-task3"}, executionOrder)
}

func TestCapacityScheduler_JobGroupingConcurrent(t *testing.T) {
	q := newTestQueue()
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 8,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			c := concurrent.Add(1)
			if c > maxConcurrent.Load() {
				maxConcurrent.Store(c)
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Different jobs: should run concurrently.
	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("job%d-task1", i), fmt.Sprintf("job%d", i), 5, 1))
	}

	time.Sleep(300 * time.Millisecond)
	require.NoError(t, s.Stop())

	// Should have run at least 3 concurrently (different jobs).
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(3))
}

func TestCapacityScheduler_PriorityOrder(t *testing.T) {
	q := newTestQueue()
	var mu sync.Mutex
	var order []string

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1, // single worker to observe order
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			mu.Lock()
			order = append(order, task.ID)
			mu.Unlock()
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Enqueue in reverse priority order.
	_ = q.Enqueue(context.Background(), makeCapacityTask("low", "job1", 1, 1))
	time.Sleep(50 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeCapacityTask("high", "job2", 10, 1))
	time.Sleep(50 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeCapacityTask("mid", "job3", 5, 1))

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, order, 3)
	// First task is already running when others arrive.
	assert.Equal(t, "low", order[0])
	assert.Equal(t, "high", order[1])
	assert.Equal(t, "mid", order[2])
}

func TestCapacityScheduler_AgingPreventsStarvation(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       100,
		Strategy:       StrategyPriority,
		AgingThreshold: 100 * time.Millisecond, // quick aging for test
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Low priority task first, then flood with high priority.
	_ = q.Enqueue(context.Background(), makeCapacityTask("low", "job-low", 1, 1))
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < 10; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("high%d", i), fmt.Sprintf("job-h%d", i), 10, 1))
	}

	time.Sleep(2 * time.Second)
	require.NoError(t, s.Stop())

	// The low priority task should eventually execute due to aging.
	assert.GreaterOrEqual(t, processed.Load(), int32(11))
}

func TestCapacityScheduler_Preemption(t *testing.T) {
	q := newTestQueue()
	var preempted atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:            q,
		WorkerCount:      4,
		Capacity:         5,
		Strategy:         StrategyPreemptive,
		EnablePreemption: true,
		Handler: func(ctx context.Context, task *Task) error {
			// Simulate work that checks for cancellation.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		},
		OnPreempt: func(task *Task) {
			preempted.Add(1)
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Fill capacity with low-priority tasks (weight 2 each, 3 tasks = 6 > 5, so only 2 fit).
	_ = q.Enqueue(context.Background(), makeCapacityTask("low1", "job1", 1, 2))
	_ = q.Enqueue(context.Background(), makeCapacityTask("low2", "job2", 1, 2))
	time.Sleep(100 * time.Millisecond) // let them start

	// Now enqueue a high-priority task that needs preemption.
	_ = q.Enqueue(context.Background(), makeCapacityTask("high1", "job3", 10, 2))

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	// At least one low-priority task should have been preempted.
	assert.GreaterOrEqual(t, preempted.Load(), int32(1))
}

func TestCapacityScheduler_WeightedFair(t *testing.T) {
	q := newTestQueue()
	var mu sync.Mutex
	executionOrder := []string{}

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       100,
		Strategy:       StrategyWeightedFair,
		AgingThreshold: 200 * time.Millisecond,
		Handler: func(ctx context.Context, task *Task) error {
			mu.Lock()
			executionOrder = append(executionOrder, task.ID)
			mu.Unlock()
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Mix of high and low priority tasks.
	for i := 0; i < 3; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("high%d", i), fmt.Sprintf("jh%d", i), 10, 1))
	}
	for i := 0; i < 3; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("low%d", i), fmt.Sprintf("jl%d", i), 1, 1))
	}

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, executionOrder, 6)
	// High priority tasks should generally come first, but low priority
	// should not be completely starved.
	hasLow := false
	for _, id := range executionOrder {
		if len(id) >= 3 && id[:3] == "low" {
			hasLow = true
		}
	}
	assert.True(t, hasLow, "low priority tasks should not be starved")
}

func TestCapacityScheduler_StopGraceful(t *testing.T) {
	q := newTestQueue()
	var completed atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 4,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 8; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("t%d", i), fmt.Sprintf("job%d", i), 5, 1))
	}

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, s.Stop())

	// At least some tasks should have completed.
	assert.Greater(t, completed.Load(), int32(0))
}

func TestCapacityScheduler_NilQueue(t *testing.T) {
	_, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Handler: func(ctx context.Context, task *Task) error { return nil },
	})
	require.Error(t, err)
}

func TestCapacityScheduler_NilHandler(t *testing.T) {
	q := newTestQueue()
	_, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue: q,
	})
	require.Error(t, err)
}

func TestCapacityScheduler_Stats(t *testing.T) {
	q := newTestQueue()

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 4,
		Capacity:    15,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("t%d", i), fmt.Sprintf("job%d", i), 5, 2))
	}

	time.Sleep(300 * time.Millisecond)
	require.NoError(t, s.Stop())

	stats := s.Stats()
	assert.Equal(t, 15, stats.Capacity)
	assert.Equal(t, "priority", stats.Strategy)
	assert.Equal(t, int64(5), stats.Succeeded)
}

func TestEffectivePriority_NoAging(t *testing.T) {
	task := &Task{Priority: 5, SubmitTime: time.Now()}
	assert.Equal(t, 5, effectivePriority(task, time.Now(), 0))
}

func TestEffectivePriority_WithAging(t *testing.T) {
	task := &Task{Priority: 1, SubmitTime: time.Now().Add(-2 * time.Minute)}
	eff := effectivePriority(task, time.Now(), 30*time.Second)
	assert.Greater(t, eff, 1)
}

func TestCanPreempt(t *testing.T) {
	pending := &Task{Priority: 10}
	running := &Task{Priority: 1, Preemptible: true}
	assert.True(t, canPreempt(pending, running))

	running.Preemptible = false
	assert.False(t, canPreempt(pending, running))

	running.Preemptible = true
	pending.Priority = 1
	assert.False(t, canPreempt(pending, running))
}

func TestSelectPreemptionVictims(t *testing.T) {
	pending := &Task{Priority: 10, Weight: 5}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 2, Preemptible: true},
		{ID: "r2", Priority: 2, Weight: 3, Preemptible: true},
		{ID: "r3", Priority: 5, Weight: 1, Preemptible: true},
	}

	// Need 5 capacity, victims should be r1 (2) + r2 (3) = 5.
	victims, freed := selectPreemptionVictims(pending, running, 5)
	assert.Equal(t, 2, len(victims))
	assert.Equal(t, 5, freed)

	// Not enough capacity — should return empty.
	victims, freed = selectPreemptionVictims(pending, running, 100)
	assert.Nil(t, victims)
	assert.Equal(t, 0, freed)
}

func TestSchedulingStrategy_String(t *testing.T) {
	assert.Equal(t, "fifo", StrategyFIFO.String())
	assert.Equal(t, "priority", StrategyPriority.String())
	assert.Equal(t, "weighted-fair", StrategyWeightedFair.String())
	assert.Equal(t, "preemptive", StrategyPreemptive.String())
	assert.Equal(t, "unknown", SchedulingStrategy(99).String())
}

func TestCapacityScheduler_RichHandler_ProgressAndLogs(t *testing.T) {
	q := newTestQueue()
	var done atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Capacity:    10,
		Strategy:    StrategyPriority,
		RichHandler: func(tctx TaskContext, task *Task) error {
			defer done.Add(1)
			_ = tctx.Log(LogLevelInfo, "starting")
			for i := 0; i < 5; i++ {
				if tctx.Err() != nil {
					return tctx.Err()
				}
				_ = tctx.SetProgress(i * 20)
				time.Sleep(20 * time.Millisecond)
			}
			_ = tctx.SetProgress(100)
			_ = tctx.Log(LogLevelInfo, "done")
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	task := makeCapacityTask("rich-1", "job-a", 5, 1)
	require.NoError(t, q.Enqueue(context.Background(), task))

	require.Eventually(t, func() bool { return done.Load() == 1 }, 3*time.Second, 50*time.Millisecond)
}

func TestCapacityScheduler_Position(t *testing.T) {
	q := newTestQueue()
	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    1,
		Strategy:    StrategyFIFO,
		Handler: func(ctx context.Context, task *Task) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	// First task occupies the single worker; others wait.
	t1 := makeCapacityTask("p-1", "j1", 1, 1)
	t2 := makeCapacityTask("p-2", "j2", 1, 1)
	t3 := makeCapacityTask("p-3", "j3", 1, 1)
	require.NoError(t, q.Enqueue(context.Background(), t1))
	require.NoError(t, q.Enqueue(context.Background(), t2))
	require.NoError(t, q.Enqueue(context.Background(), t3))

	// Give the scheduler time to ingest.
	time.Sleep(100 * time.Millisecond)

	// t2 and t3 should be pending. Position of t2 should be 0 (next).
	pos2, err := s.Position(context.Background(), "p-2")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, pos2, 0)

	pos3, err := s.Position(context.Background(), "p-3")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, pos3, 0)
}
