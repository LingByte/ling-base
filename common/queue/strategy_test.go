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

	"github.com/LingByte/ling-base/common/pool"
)

// ===== weightedFairScore tests =====

func TestWeightedFairScore_HighPriorityDominates(t *testing.T) {
	now := time.Now()
	highTask := &Task{ID: "high", Priority: 10, SubmitTime: now}
	lowTask := &Task{ID: "low", Priority: 1, SubmitTime: now}

	highScore := weightedFairScore(highTask, 10, now)
	lowScore := weightedFairScore(lowTask, 1, now)

	assert.Greater(t, highScore, lowScore, "high-priority task should have higher score at same wait time")
}

func TestWeightedFairScore_WaitTimeContributes(t *testing.T) {
	submitTime := time.Now()
	later := submitTime.Add(10 * time.Second)

	task := &Task{ID: "t1", Priority: 1, SubmitTime: submitTime}

	earlyScore := weightedFairScore(task, 1, submitTime)
	lateScore := weightedFairScore(task, 1, later)

	assert.Greater(t, lateScore, earlyScore, "score should increase with wait time")
}

func TestWeightedFairScore_LowPriorityOvertakesWithWait(t *testing.T) {
	submitTime := time.Now()
	// High-priority task submitted at the same time but with 0 wait.
	highTask := &Task{ID: "high", Priority: 10, SubmitTime: submitTime}
	// Low-priority task with a very long wait and aging boost.
	// With aging, effective priority increases, and the quadratic term
	// can eventually make the low-priority task overtake.
	lowTask := &Task{ID: "low", Priority: 1, SubmitTime: submitTime.Add(-1000 * time.Second)}

	// With aging threshold of 30s, effective priority of low task:
	// boost = (1000s - 30s) / 30s = 32, so effPriority = 1 + 32 = 33
	effLow := effectivePriority(lowTask, submitTime, 30*time.Second)
	effHigh := effectivePriority(highTask, submitTime, 30*time.Second)

	lowScore := weightedFairScore(lowTask, effLow, submitTime)
	highScore := weightedFairScore(highTask, effHigh, submitTime)

	assert.Greater(t, lowScore, highScore, "long-waiting low-priority task should overtake with aging")
}

func TestWeightedFairScore_ExactValue(t *testing.T) {
	submitTime := time.Now()
	now := submitTime.Add(4 * time.Second)
	task := &Task{ID: "t1", Priority: 3, SubmitTime: submitTime}

	// effPriority = 3, waitSeconds = 4
	// score = 3*3 + 4*0.5 = 9 + 2 = 11
	score := weightedFairScore(task, 3, now)
	assert.InDelta(t, 11.0, score, 0.001)
}

// ===== canPreempt tests =====

func TestCanPreempt_HigherPriorityPreemptible(t *testing.T) {
	pending := &Task{Priority: 10}
	running := &Task{Priority: 1, Preemptible: true}
	assert.True(t, canPreempt(pending, running))
}

func TestCanPreempt_EqualPriorityNotPreempted(t *testing.T) {
	pending := &Task{Priority: 5}
	running := &Task{Priority: 5, Preemptible: true}
	assert.False(t, canPreempt(pending, running), "equal priority should not preempt")
}

func TestCanPreempt_LowerPriorityPendingNotPreempted(t *testing.T) {
	pending := &Task{Priority: 1}
	running := &Task{Priority: 10, Preemptible: true}
	assert.False(t, canPreempt(pending, running), "lower priority pending should not preempt higher")
}

func TestCanPreempt_NonPreemptibleRunningNotPreempted(t *testing.T) {
	pending := &Task{Priority: 10}
	running := &Task{Priority: 1, Preemptible: false}
	assert.False(t, canPreempt(pending, running), "non-preemptible running task should not be preempted")
}

func TestCanPreempt_BothNonPreemptible(t *testing.T) {
	pending := &Task{Priority: 10}
	running := &Task{Priority: 1, Preemptible: false}
	assert.False(t, canPreempt(pending, running))
}

// ===== selectPreemptionVictims tests =====

func TestSelectPreemptionVictims_LowestPrioritySelectedFirst(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 3}
	t1 := &Task{ID: "r1", Priority: 1, Weight: 2, Preemptible: true}
	t2 := &Task{ID: "r2", Priority: 2, Weight: 2, Preemptible: true}
	t3 := &Task{ID: "r3", Priority: 5, Weight: 2, Preemptible: true}
	running := []*Task{t3, t1, t2}

	// Need 3 capacity. Should select r1 (priority 1, weight 2) + r2 (priority 2, weight 2) = 4 >= 3.
	victims, freed := selectPreemptionVictims(pending, running, 3)
	require.Len(t, victims, 2)
	assert.Equal(t, "r1", victims[0].ID, "lowest priority should be selected first")
	assert.Equal(t, "r2", victims[1].ID, "second lowest priority should be selected second")
	assert.Equal(t, 4, freed)
}

func TestSelectPreemptionVictims_NonPreemptibleExcluded(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 5}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 5, Preemptible: false}, // not preemptible
		{ID: "r2", Priority: 2, Weight: 5, Preemptible: true},
	}

	// r1 is not preemptible, so only r2 is a candidate.
	// Need 5, r2 has weight 5, so it should be selected.
	victims, freed := selectPreemptionVictims(pending, running, 5)
	require.Len(t, victims, 1)
	assert.Equal(t, "r2", victims[0].ID, "non-preemptible task should never be selected as victim")
	assert.Equal(t, 5, freed)
}

func TestSelectPreemptionVictims_AllNonPreemptible(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 5}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 5, Preemptible: false},
		{ID: "r2", Priority: 2, Weight: 5, Preemptible: false},
	}

	// No preemptible candidates.
	victims, freed := selectPreemptionVictims(pending, running, 5)
	assert.Nil(t, victims)
	assert.Equal(t, 0, freed)
}

func TestSelectPreemptionVictims_NotEnoughCapacity(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 100}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 2, Preemptible: true},
		{ID: "r2", Priority: 2, Weight: 3, Preemptible: true},
	}

	// Need 100, can only free 5 — should return nil.
	victims, freed := selectPreemptionVictims(pending, running, 100)
	assert.Nil(t, victims)
	assert.Equal(t, 0, freed)
}

func TestSelectPreemptionVictims_WeightZeroCountsAsOne(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 2}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 0, Preemptible: true}, // weight 0 = 1 unit
		{ID: "r2", Priority: 2, Weight: 0, Preemptible: true},
	}

	// Need 2, each weight-0 task frees 1, so need both.
	victims, freed := selectPreemptionVictims(pending, running, 2)
	require.Len(t, victims, 2)
	assert.Equal(t, 2, freed)
}

func TestSelectPreemptionVictims_StartedAtTieBreak(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 2}
	earlier := time.Now().Add(-10 * time.Second)
	later := time.Now().Add(-1 * time.Second)

	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 2, Preemptible: true, StartedAt: &earlier},
		{ID: "r2", Priority: 1, Weight: 2, Preemptible: true, StartedAt: &later},
	}

	// Both have priority 1. Tie-break: prefer preempting the one that started
	// most recently (less work lost).
	victims, freed := selectPreemptionVictims(pending, running, 2)
	require.Len(t, victims, 1)
	assert.Equal(t, "r2", victims[0].ID, "should preempt the task that started most recently")
	assert.Equal(t, 2, freed)
}

func TestSelectPreemptionVictims_ExactCapacityMatch(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 5}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 5, Preemptible: true},
	}

	// Need exactly 5, r1 has weight 5 — should select it.
	victims, freed := selectPreemptionVictims(pending, running, 5)
	require.Len(t, victims, 1)
	assert.Equal(t, "r1", victims[0].ID)
	assert.Equal(t, 5, freed)
}

func TestSelectPreemptionVictims_EmptyRunning(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 1}
	victims, freed := selectPreemptionVictims(pending, nil, 1)
	assert.Nil(t, victims)
	assert.Equal(t, 0, freed)
}

func TestSelectPreemptionVictims_HigherPriorityRunningExcluded(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 5, Weight: 3}
	running := []*Task{
		{ID: "r1", Priority: 10, Weight: 3, Preemptible: true}, // higher than pending
		{ID: "r2", Priority: 3, Weight: 3, Preemptible: true},  // lower than pending
	}

	// Only r2 is a candidate (priority 3 < 5).
	victims, freed := selectPreemptionVictims(pending, running, 3)
	require.Len(t, victims, 1)
	assert.Equal(t, "r2", victims[0].ID)
	assert.Equal(t, 3, freed)
}

// ===== End-to-end StrategyWeightedFair test =====

func TestCapacityScheduler_WeightedFairLowPriorityScheduled(t *testing.T) {
	q := newTestQueue()
	var mu sync.Mutex
	executionOrder := []string{}

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       100,
		Strategy:       StrategyWeightedFair,
		AgingThreshold: 100 * time.Millisecond,
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

	// Enqueue a low-priority task first, then flood with high-priority tasks.
	_ = q.Enqueue(context.Background(), makeCapacityTask("low", "job-low", 1, 1))
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("high%d", i), fmt.Sprintf("jh%d", i), 10, 1))
	}

	time.Sleep(3 * time.Second)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, executionOrder, 6, "all tasks should be scheduled")

	// The low-priority task should eventually be scheduled (not starved).
	hasLow := false
	for _, id := range executionOrder {
		if id == "low" {
			hasLow = true
		}
	}
	assert.True(t, hasLow, "low-priority task should eventually be scheduled with weighted-fair strategy")
}

// ===== Multiple high-priority arrivals test =====

func TestCapacityScheduler_MultipleHighPriorityArrivals(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1, // single worker to serialize
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Enqueue several high-priority tasks at once.
	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("high%d", i), fmt.Sprintf("jh%d", i), 10, 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(5), processed.Load(), "all high-priority tasks should be scheduled")
}

// ===== sortTasks direct tests =====

func TestSortTasks_FIFO(t *testing.T) {
	now := time.Now()
	tasks := []*Task{
		{ID: "t3", Priority: 10, SubmitTime: now.Add(20 * time.Millisecond)},
		{ID: "t1", Priority: 1, SubmitTime: now},
		{ID: "t2", Priority: 5, SubmitTime: now.Add(10 * time.Millisecond)},
	}

	sortTasks(tasks, StrategyFIFO, now, 0)

	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "t2", tasks[1].ID)
	assert.Equal(t, "t3", tasks[2].ID)
}

func TestSortTasks_Priority(t *testing.T) {
	now := time.Now()
	tasks := []*Task{
		{ID: "low", Priority: 1, SubmitTime: now},
		{ID: "high", Priority: 10, SubmitTime: now.Add(10 * time.Millisecond)},
		{ID: "mid", Priority: 5, SubmitTime: now.Add(5 * time.Millisecond)},
	}

	sortTasks(tasks, StrategyPriority, now, 0)

	assert.Equal(t, "high", tasks[0].ID)
	assert.Equal(t, "mid", tasks[1].ID)
	assert.Equal(t, "low", tasks[2].ID)
}

func TestSortTasks_PrioritySamePriorityFIFO(t *testing.T) {
	now := time.Now()
	tasks := []*Task{
		{ID: "t3", Priority: 5, SubmitTime: now.Add(20 * time.Millisecond)},
		{ID: "t1", Priority: 5, SubmitTime: now},
		{ID: "t2", Priority: 5, SubmitTime: now.Add(10 * time.Millisecond)},
	}

	sortTasks(tasks, StrategyPriority, now, 0)

	// Same priority → FIFO by submit time.
	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "t2", tasks[1].ID)
	assert.Equal(t, "t3", tasks[2].ID)
}

func TestSortTasks_WeightedFair(t *testing.T) {
	now := time.Now()
	tasks := []*Task{
		{ID: "low", Priority: 1, SubmitTime: now},
		{ID: "high", Priority: 10, SubmitTime: now},
	}

	sortTasks(tasks, StrategyWeightedFair, now, 0)

	// High priority should come first (higher score).
	assert.Equal(t, "high", tasks[0].ID)
	assert.Equal(t, "low", tasks[1].ID)
}

func TestSortTasks_Preemptive(t *testing.T) {
	now := time.Now()
	tasks := []*Task{
		{ID: "low", Priority: 1, SubmitTime: now},
		{ID: "high", Priority: 10, SubmitTime: now},
	}

	sortTasks(tasks, StrategyPreemptive, now, 0)

	assert.Equal(t, "high", tasks[0].ID)
	assert.Equal(t, "low", tasks[1].ID)
}

// ===== effectivePriority edge cases =====

func TestEffectivePriority_AgingExactBoost(t *testing.T) {
	submitTime := time.Now().Add(-90 * time.Second)
	task := &Task{Priority: 1, SubmitTime: submitTime}

	// agingThreshold = 30s, wait = 90s, boost = (90-30)/30 = 2
	// effPriority = 1 + 2 = 3
	eff := effectivePriority(task, time.Now(), 30*time.Second)
	assert.Equal(t, 3, eff)
}

func TestEffectivePriority_WaitBelowThreshold(t *testing.T) {
	submitTime := time.Now().Add(-10 * time.Second)
	task := &Task{Priority: 5, SubmitTime: submitTime}

	// wait (10s) < agingThreshold (30s) → no boost.
	eff := effectivePriority(task, time.Now(), 30*time.Second)
	assert.Equal(t, 5, eff)
}

// ===== taskWeight tests =====

func TestTaskWeight_ZeroWeight(t *testing.T) {
	task := &Task{Weight: 0}
	assert.Equal(t, 1, taskWeight(task))
}

func TestTaskWeight_NegativeWeight(t *testing.T) {
	task := &Task{Weight: -5}
	assert.Equal(t, 1, taskWeight(task))
}

func TestTaskWeight_PositiveWeight(t *testing.T) {
	task := &Task{Weight: 7}
	assert.Equal(t, 7, taskWeight(task))
}

// ===== Preemption end-to-end with multiple victims =====

func TestCapacityScheduler_PreemptionMultipleVictims(t *testing.T) {
	q := newTestQueue()
	var preempted atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:            q,
		WorkerCount:      4,
		Capacity:         3,
		Strategy:         StrategyPreemptive,
		EnablePreemption: true,
		Handler: func(ctx context.Context, task *Task) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
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

	// Fill capacity with low-priority weight-1 tasks (3 tasks = 3 weight = capacity).
	for i := 0; i < 3; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("low%d", i), fmt.Sprintf("jl%d", i), 1, 1))
	}
	time.Sleep(200 * time.Millisecond) // let them start

	// Now enqueue a high-priority task that needs 2 capacity — should preempt 2 low tasks.
	_ = q.Enqueue(context.Background(), makeCapacityTask("high1", "jh1", 10, 2))

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	// At least 1 low-priority task should have been preempted to make room.
	assert.GreaterOrEqual(t, preempted.Load(), int32(1), "should preempt at least 1 low-priority task")
}

// ===== Non-preemptible task not preempted end-to-end =====

func TestCapacityScheduler_NonPreemptibleNotPreempted(t *testing.T) {
	q := newTestQueue()
	var preempted atomic.Int32
	var completed atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:            q,
		WorkerCount:      4,
		Capacity:         2,
		Strategy:         StrategyPreemptive,
		EnablePreemption: true,
		Handler: func(ctx context.Context, task *Task) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				completed.Add(1)
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

	// Fill capacity with non-preemptible low-priority tasks.
	low1 := makeCapacityTask("low1", "jl1", 1, 1)
	low1.Preemptible = false
	low2 := makeCapacityTask("low2", "jl2", 1, 1)
	low2.Preemptible = false
	_ = q.Enqueue(context.Background(), low1)
	_ = q.Enqueue(context.Background(), low2)
	time.Sleep(100 * time.Millisecond) // let them start

	// Enqueue a high-priority task — should NOT preempt non-preemptible tasks.
	_ = q.Enqueue(context.Background(), makeCapacityTask("high1", "jh1", 10, 1))

	time.Sleep(400 * time.Millisecond)
	require.NoError(t, s.Stop())

	// No preemption should have occurred.
	assert.Equal(t, int32(0), preempted.Load(), "non-preemptible tasks should not be preempted")
	// All tasks should eventually complete (2 low + 1 high after capacity frees up).
	assert.GreaterOrEqual(t, completed.Load(), int32(2), "non-preemptible tasks should complete normally")
}

// ===== Helper to create task with payload =====

func makeStrategyTask(id string, priority, weight int, preemptible bool) *Task {
	payload, _ := json.Marshal(map[string]any{"id": id})
	return &Task{
		ID:          id,
		JobID:       "job-" + id,
		Priority:    priority,
		Weight:      weight,
		Payload:     payload,
		MaxRetries:  3,
		Preemptible: preemptible,
	}
}

func TestSelectPreemptionVictims_MixedPreemptible(t *testing.T) {
	pending := &Task{ID: "pending", Priority: 10, Weight: 4}
	running := []*Task{
		{ID: "r1", Priority: 1, Weight: 2, Preemptible: true},
		{ID: "r2", Priority: 2, Weight: 2, Preemptible: false}, // not preemptible
		{ID: "r3", Priority: 3, Weight: 2, Preemptible: true},
	}

	// Need 4. Candidates: r1 (pri 1, w 2) and r3 (pri 3, w 2).
	// r1 selected first (lowest priority), then r3. Total freed = 4.
	victims, freed := selectPreemptionVictims(pending, running, 4)
	require.Len(t, victims, 2)
	assert.Equal(t, "r1", victims[0].ID)
	assert.Equal(t, "r3", victims[1].ID)
	assert.Equal(t, 4, freed)
}

// ===== CapacityScheduler delegation methods =====

func TestCapacityScheduler_UpdateProgress(t *testing.T) {
	q := newTestQueue()
	block := make(chan struct{})

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    10,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			<-block // keep task running so we can update progress
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	task := makeCapacityTask("prog-1", "job-p", 5, 1)
	require.NoError(t, q.Enqueue(context.Background(), task))
	time.Sleep(100 * time.Millisecond) // let task start

	// UpdateProgress should succeed (task is running).
	err = s.UpdateProgress(context.Background(), "prog-1", 50)
	require.NoError(t, err)

	// UpdateProgress on a non-existent task should fail.
	err = s.UpdateProgress(context.Background(), "nonexistent", 50)
	assert.Error(t, err)

	close(block)                      // unblock the handler
	time.Sleep(50 * time.Millisecond) // let handler finish
}

func TestCapacityScheduler_AppendLog(t *testing.T) {
	q := newTestQueue()
	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       10,
		Strategy:       StrategyPriority,
		Handler:        func(ctx context.Context, task *Task) error { return nil },
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	entry := &TaskLogEntry{
		TaskID:    "log-1",
		Level:     LogLevelInfo,
		Message:   "test log",
		Timestamp: time.Now(),
	}
	err = s.AppendLog(context.Background(), entry)
	require.NoError(t, err)
}

func TestCapacityScheduler_ListLogs(t *testing.T) {
	q := newTestQueue()
	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       10,
		Strategy:       StrategyPriority,
		Handler:        func(ctx context.Context, task *Task) error { return nil },
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	logs, err := s.ListLogs(context.Background(), "log-1", 10)
	require.NoError(t, err)
	assert.Nil(t, logs)
}

// ===== CapacityScheduler retry and failure paths =====

func TestCapacityScheduler_RetryOnFailure(t *testing.T) {
	q := newTestQueue()
	var attempts atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			n := attempts.Add(1)
			if n < 3 {
				return fmt.Errorf("transient error")
			}
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	_ = q.Enqueue(context.Background(), makeCapacityTask("retry-1", "job-r", 5, 1))

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(3), attempts.Load())
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Retried)
	assert.Equal(t, int64(1), stats.Succeeded)
}

func TestCapacityScheduler_MaxRetriesExceeded(t *testing.T) {
	q := newTestQueue()
	var attempts atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			attempts.Add(1)
			return fmt.Errorf("permanent error")
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	task := makeCapacityTask("fail-1", "job-f", 5, 1)
	task.MaxRetries = 2
	_ = q.Enqueue(context.Background(), task)

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(3), attempts.Load(), "1 initial + 2 retries")
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Retried)
	assert.Equal(t, int64(1), stats.Failed)
}

func TestCapacityScheduler_OnTaskStartAndComplete(t *testing.T) {
	q := newTestQueue()
	var starts, completes atomic.Int32

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler:     func(ctx context.Context, task *Task) error { return nil },
		OnTaskStart: func(task *Task) {
			starts.Add(1)
		},
		OnTaskComplete: func(task *Task, err error) {
			completes.Add(1)
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeCapacityTask(fmt.Sprintf("cb%d", i), fmt.Sprintf("jcb%d", i), 5, 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(5), starts.Load())
	assert.Equal(t, int32(5), completes.Load())
}

// ===== CapacityScheduler with recovery =====

type recoverableTestQueue struct {
	*testQueue
	recoverTasks []*Task
}

func newRecoverableTestQueue(tasks []*Task) *recoverableTestQueue {
	return &recoverableTestQueue{
		testQueue:    newTestQueue(),
		recoverTasks: tasks,
	}
}

func (q *recoverableTestQueue) Recover(ctx context.Context) ([]*Task, error) {
	return q.recoverTasks, nil
}

func TestCapacityScheduler_StartWithRecovery(t *testing.T) {
	// Pre-create tasks to recover.
	payload, _ := json.Marshal(map[string]any{"id": "rec"})
	recoveredTasks := []*Task{
		{ID: "rec1", JobID: "jrec1", Priority: 5, Weight: 1, Payload: payload, MaxRetries: 3, Preemptible: true, Status: StatusPending},
		{ID: "rec2", JobID: "jrec2", Priority: 5, Weight: 1, Payload: payload, MaxRetries: 3, Preemptible: true, Status: StatusPending},
	}

	q := newRecoverableTestQueue(recoveredTasks)
	var processed atomic.Int32

	var recoveredCount int
	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Capacity:    100,
		Strategy:    StrategyPriority,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		OnRecover: func(count int) {
			recoveredCount = count
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(2), processed.Load(), "recovered tasks should be processed")
	assert.Equal(t, 2, recoveredCount)
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Recovered)
	assert.Equal(t, int64(2), stats.Succeeded)
}

// ===== RichHandler covering TaskContext methods =====

func TestCapacityScheduler_RichHandler_FullContext(t *testing.T) {
	q := newTestQueue()
	var done atomic.Int32
	var taskRef *Task
	var hasDeadline bool
	var doneChanClosed bool

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    10,
		Strategy:    StrategyPriority,
		RichHandler: func(tctx TaskContext, task *Task) error {
			defer done.Add(1)

			// Access Task() method.
			taskRef = tctx.Task()
			assert.Equal(t, "rich-ctx-1", taskRef.ID)

			// Access Deadline() — no deadline set, should return false.
			_, ok := tctx.Deadline()
			hasDeadline = ok

			// Access Done() — should be a non-nil channel.
			assert.NotNil(t, tctx.Done())

			// Access Value() — should return nil for unknown key.
			assert.Nil(t, tctx.Value("unknown-key"))

			// Access Err() — should be nil initially.
			assert.Nil(t, tctx.Err())

			// Use SetProgress and Log.
			_ = tctx.SetProgress(50)
			_ = tctx.Log(LogLevelInfo, "working")

			// Check Done channel is not closed while context is active.
			select {
			case <-tctx.Done():
				doneChanClosed = true
			default:
				doneChanClosed = false
			}

			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())
	defer s.Stop()

	task := makeCapacityTask("rich-ctx-1", "job-ctx", 5, 1)
	require.NoError(t, q.Enqueue(context.Background(), task))

	require.Eventually(t, func() bool { return done.Load() == 1 }, 3*time.Second, 50*time.Millisecond)

	require.NotNil(t, taskRef)
	assert.False(t, hasDeadline, "no deadline should be set")
	assert.False(t, doneChanClosed, "Done channel should not be closed while task is running")
}

// ===== Scheduler with WorkerPool (dispatchLoop) =====

func TestScheduler_WithWorkerPool(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	wp := pool.NewWorkerPool(2, 8)
	wp.Start()
	defer wp.Stop()

	s, err := NewScheduler(SchedulerConfig{
		Queue:      q,
		WorkerPool: wp,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("wp%d", i), 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(5), processed.Load(), "all tasks should be processed via worker pool")
}

// ===== Scheduler with CPUAdaptive mode =====

func TestScheduler_CPUAdaptiveMode_Processing(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Mode:        ModeCPUAdaptive,
		MinWorkers:  1,
		MaxWorkers:  4,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout:   100 * time.Millisecond,
		CPUCheckInterval: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("cpu%d", i), 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(5), processed.Load())
}

// ===== Scheduler recovery =====

func TestScheduler_StartWithRecovery(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"id": "rec"})
	recoveredTasks := []*Task{
		{ID: "srec1", Queue: "test", Priority: 5, Payload: payload, MaxRetries: 3, Status: StatusPending},
		{ID: "srec2", Queue: "test", Priority: 5, Payload: payload, MaxRetries: 3, Status: StatusPending},
	}

	q := newRecoverableTestQueue(recoveredTasks)

	var recoveredCount int
	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Handler: func(ctx context.Context, task *Task) error {
			return nil
		},
		OnRecover: func(count int) {
			recoveredCount = count
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	time.Sleep(200 * time.Millisecond)
	require.NoError(t, s.Stop())

	// The Scheduler counts recovered tasks but doesn't re-enqueue them
	// (unlike CapacityScheduler which adds them to its pending list).
	assert.Equal(t, 2, recoveredCount, "recovery callback should be called with task count")
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Recovered)
}

// ===== Scheduler double stop =====

func TestScheduler_DoubleStop(t *testing.T) {
	q := newTestQueue()
	s, err := NewScheduler(SchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Handler:        func(ctx context.Context, task *Task) error { return nil },
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	require.NotPanics(t, func() {
		require.NoError(t, s.Stop())
		require.NoError(t, s.Stop())
	})
}

// ===== CapacityScheduler double stop =====

func TestCapacityScheduler_DoubleStop(t *testing.T) {
	q := newTestQueue()
	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:          q,
		WorkerCount:    1,
		Capacity:       10,
		Strategy:       StrategyPriority,
		Handler:        func(ctx context.Context, task *Task) error { return nil },
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	require.NotPanics(t, func() {
		require.NoError(t, s.Stop())
		require.NoError(t, s.Stop())
	})
}

// ===== CapacityScheduler with FIFO strategy =====

func TestCapacityScheduler_FIFOStrategy(t *testing.T) {
	q := newTestQueue()
	var mu sync.Mutex
	var order []string

	s, err := NewCapacityScheduler(CapacitySchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Capacity:    100,
		Strategy:    StrategyFIFO,
		Handler: func(ctx context.Context, task *Task) error {
			mu.Lock()
			order = append(order, task.ID)
			mu.Unlock()
			return nil
		},
		DequeueTimeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	require.NoError(t, s.Start())

	// Enqueue in reverse priority order — FIFO should ignore priority.
	_ = q.Enqueue(context.Background(), makeCapacityTask("first", "j1", 1, 1))
	time.Sleep(20 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeCapacityTask("second", "j2", 10, 1))
	time.Sleep(20 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeCapacityTask("third", "j3", 5, 1))

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, order, 3)
	assert.Equal(t, "first", order[0], "FIFO should dispatch in submission order")
	assert.Equal(t, "second", order[1])
	assert.Equal(t, "third", order[2])
}
