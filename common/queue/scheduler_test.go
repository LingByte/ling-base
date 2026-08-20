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

// testQueue is a simple in-memory queue for scheduler tests.
type testQueue struct {
	mu     sync.Mutex
	tasks  []*Task
	byID   map[string]*Task
	stats  QueueStats
	closed bool
}

func newTestQueue() *testQueue {
	return &testQueue{byID: make(map[string]*Task)}
}

func (q *testQueue) Enqueue(ctx context.Context, task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.byID[task.ID]; ok {
		return ErrDuplicateTask
	}
	task.Status = StatusPending
	if task.SubmitTime.IsZero() {
		task.SubmitTime = time.Now()
	}
	q.byID[task.ID] = task
	q.tasks = append(q.tasks, task)
	q.stats.Pending++
	q.stats.Total++
	return nil
}

func (q *testQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.tasks) == 0 {
		return nil, ErrQueueEmpty
	}
	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	task.Status = StatusRunning
	now := time.Now()
	task.StartedAt = &now
	q.stats.Pending--
	return task, nil
}

func (q *testQueue) Ack(ctx context.Context, taskID string, status TaskStatus, errMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.byID[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = status
	task.ErrorMsg = errMsg
	now := time.Now()
	task.FinishedAt = &now
	if status == StatusSuccess {
		q.stats.Completed++
	} else if status == StatusFailed {
		q.stats.Failed++
	}
	if status.IsTerminal() {
		delete(q.byID, taskID)
	}
	return nil
}

func (q *testQueue) Requeue(ctx context.Context, taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.byID[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	task.Status = StatusPending
	task.RetryCount++
	q.tasks = append(q.tasks, task)
	q.stats.Pending++
	return nil
}

func (q *testQueue) Get(ctx context.Context, taskID string) (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.byID[taskID]
	if !ok {
		return nil, ErrTaskNotFound
	}
	cp := *task
	return &cp, nil
}

func (q *testQueue) Cancel(ctx context.Context, taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.byID, taskID)
	for i, t := range q.tasks {
		if t.ID == taskID {
			q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)
			break
		}
	}
	return nil
}

func (q *testQueue) Position(ctx context.Context, taskID string) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, t := range q.tasks {
		if t.ID == taskID {
			return i, nil
		}
	}
	return -1, nil
}

func (q *testQueue) UpdateProgress(ctx context.Context, taskID string, progress int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	task, ok := q.byID[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	task.Progress = progress
	return nil
}

func (q *testQueue) AppendLog(ctx context.Context, entry *TaskLogEntry) error {
	return nil
}

func (q *testQueue) ListLogs(ctx context.Context, taskID string, limit int) ([]*TaskLogEntry, error) {
	return nil, nil
}

func (q *testQueue) Recover(ctx context.Context) ([]*Task, error) {
	return nil, nil
}

func (q *testQueue) Stats(ctx context.Context) (QueueStats, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.stats, nil
}

func (q *testQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}

func (q *testQueue) Name() string { return "test" }

func makeTask(id string, priority int) *Task {
	payload, _ := json.Marshal(map[string]any{"id": id})
	return &Task{
		ID:         id,
		Queue:      "test",
		Priority:   priority,
		Payload:    payload,
		MaxRetries: 3,
	}
}

func TestScheduler_BasicFlow(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())

	for i := 0; i < 10; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("t%d", i), 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(10), processed.Load())
	stats := s.Stats()
	assert.Equal(t, int64(10), stats.Succeeded)
}

func TestScheduler_PriorityOrder(t *testing.T) {
	q := newTestQueue()
	var order []string
	var mu sync.Mutex

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 1, // single worker to preserve order
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

	_ = q.Enqueue(context.Background(), makeTask("low", 1))
	time.Sleep(50 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeTask("high", 10))
	time.Sleep(50 * time.Millisecond)
	_ = q.Enqueue(context.Background(), makeTask("mid", 5))

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, order, 3)
	// With 1 worker, first task is already being processed when high arrives.
	// So order is: low (already dequeued), then high, then mid.
	assert.Equal(t, "low", order[0])
	assert.Equal(t, "high", order[1])
	assert.Equal(t, "mid", order[2])
}

func TestScheduler_Retry(t *testing.T) {
	q := newTestQueue()
	var attempts atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Handler: func(ctx context.Context, task *Task) error {
			n := attempts.Add(1)
			if n < 3 {
				return fmt.Errorf("transient error")
			}
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())
	_ = q.Enqueue(context.Background(), makeTask("retry-task", 1))

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(3), attempts.Load())
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Retried)
	assert.Equal(t, int64(1), stats.Succeeded)
}

func TestScheduler_MaxRetriesExceeded(t *testing.T) {
	q := newTestQueue()
	var attempts atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 1,
		Handler: func(ctx context.Context, task *Task) error {
			attempts.Add(1)
			return fmt.Errorf("permanent error")
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	task := makeTask("fail-task", 1)
	task.MaxRetries = 2

	require.NoError(t, s.Start())
	_ = q.Enqueue(context.Background(), task)

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(3), attempts.Load()) // 1 initial + 2 retries
	stats := s.Stats()
	assert.Equal(t, int64(2), stats.Retried)
	assert.Equal(t, int64(1), stats.Failed)
}

func TestScheduler_Callbacks(t *testing.T) {
	q := newTestQueue()
	var starts, completes atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Handler: func(ctx context.Context, task *Task) error {
			return nil
		},
		OnTaskStart: func(task *Task) {
			starts.Add(1)
		},
		OnTaskComplete: func(task *Task, err error) {
			completes.Add(1)
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())
	for i := 0; i < 5; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("t%d", i), 1))
	}
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(5), starts.Load())
	assert.Equal(t, int32(5), completes.Load())
}

func TestScheduler_StopGraceful(t *testing.T) {
	q := newTestQueue()
	var completed atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 2,
		Handler: func(ctx context.Context, task *Task) error {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())
	for i := 0; i < 4; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("t%d", i), 1))
	}

	time.Sleep(100 * time.Millisecond) // let tasks start
	require.NoError(t, s.Stop())       // should wait for completion

	assert.Equal(t, int32(4), completed.Load())
}

func TestScheduler_NilQueue(t *testing.T) {
	_, err := NewScheduler(SchedulerConfig{
		Handler: func(ctx context.Context, task *Task) error { return nil },
	})
	require.Error(t, err)
}

func TestScheduler_NilHandler(t *testing.T) {
	q := newTestQueue()
	_, err := NewScheduler(SchedulerConfig{
		Queue: q,
	})
	require.Error(t, err)
}

func TestScheduler_ConcurrentSubmit(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:       q,
		WorkerCount: 4,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("t%d", n), n%10))
		}(i)
	}
	wg.Wait()

	time.Sleep(1 * time.Second)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(100), processed.Load())
}

func TestScheduler_CPUAdaptiveMode(t *testing.T) {
	q := newTestQueue()
	var processed atomic.Int32

	s, err := NewScheduler(SchedulerConfig{
		Queue:            q,
		WorkerCount:      2,
		Mode:             ModeCPUAdaptive,
		MinWorkers:       1,
		MaxWorkers:       8,
		CPUCheckInterval: 100 * time.Millisecond,
		Handler: func(ctx context.Context, task *Task) error {
			processed.Add(1)
			return nil
		},
		DequeueTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, s.Start())
	for i := 0; i < 10; i++ {
		_ = q.Enqueue(context.Background(), makeTask(fmt.Sprintf("t%d", i), 1))
	}

	time.Sleep(500 * time.Millisecond)
	require.NoError(t, s.Stop())

	assert.Equal(t, int32(10), processed.Load())
}
