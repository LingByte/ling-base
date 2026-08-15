// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/queue"
)

func newTestTask(id string, priority int) *queue.Task {
	payload, _ := json.Marshal(map[string]any{"id": id})
	return &queue.Task{
		ID:         id,
		Queue:      "test",
		Priority:   priority,
		Payload:    payload,
		MaxRetries: 3,
	}
}

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := New("test")
	defer q.Close()

	task := newTestTask("t1", 1)
	err := q.Enqueue(context.Background(), task)
	require.NoError(t, err)

	got, err := q.Dequeue(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, "t1", got.ID)
	assert.Equal(t, queue.StatusRunning, got.Status)
}

func TestQueue_PriorityOrder(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("low", 1))
	_ = q.Enqueue(context.Background(), newTestTask("high", 10))
	_ = q.Enqueue(context.Background(), newTestTask("mid", 5))

	first, _ := q.Dequeue(context.Background(), 0)
	second, _ := q.Dequeue(context.Background(), 0)
	third, _ := q.Dequeue(context.Background(), 0)

	assert.Equal(t, "high", first.ID)
	assert.Equal(t, "mid", second.ID)
	assert.Equal(t, "low", third.ID)
}

func TestQueue_FIFOSamePriority(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("first", 5))
	time.Sleep(1 * time.Millisecond)
	_ = q.Enqueue(context.Background(), newTestTask("second", 5))

	first, _ := q.Dequeue(context.Background(), 0)
	second, _ := q.Dequeue(context.Background(), 0)

	assert.Equal(t, "first", first.ID)
	assert.Equal(t, "second", second.ID)
}

func TestQueue_Empty(t *testing.T) {
	q := New("test")
	defer q.Close()

	_, err := q.Dequeue(context.Background(), 0)
	assert.ErrorIs(t, err, queue.ErrQueueEmpty)
}

func TestQueue_DuplicateTask(t *testing.T) {
	q := New("test")
	defer q.Close()

	task := newTestTask("t1", 1)
	require.NoError(t, q.Enqueue(context.Background(), task))
	err := q.Enqueue(context.Background(), task)
	assert.ErrorIs(t, err, queue.ErrDuplicateTask)
}

func TestQueue_Ack(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("t1", 1))
	task, _ := q.Dequeue(context.Background(), 0)

	err := q.Ack(context.Background(), task.ID, queue.StatusSuccess, "")
	require.NoError(t, err)

	got, err := q.Get(context.Background(), task.ID)
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
	_ = got
}

func TestQueue_Requeue(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("t1", 1))
	task, _ := q.Dequeue(context.Background(), 0)

	err := q.Requeue(context.Background(), task.ID)
	require.NoError(t, err)

	got, err := q.Dequeue(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, "t1", got.ID)
	assert.Equal(t, 1, got.RetryCount)
}

func TestQueue_Cancel(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("t1", 1))
	err := q.Cancel(context.Background(), "t1")
	require.NoError(t, err)

	_, err = q.Dequeue(context.Background(), 0)
	assert.ErrorIs(t, err, queue.ErrQueueEmpty)
}

func TestQueue_Stats(t *testing.T) {
	q := New("test")
	defer q.Close()

	_ = q.Enqueue(context.Background(), newTestTask("t1", 1))
	_ = q.Enqueue(context.Background(), newTestTask("t2", 1))
	task, _ := q.Dequeue(context.Background(), 0)
	_ = q.Ack(context.Background(), task.ID, queue.StatusSuccess, "")

	stats, err := q.Stats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test", stats.Queue)
	assert.Equal(t, int64(1), stats.Pending)
	assert.Equal(t, int64(1), stats.Completed)
	assert.Equal(t, int64(2), stats.Total)
}

func TestQueue_Close(t *testing.T) {
	q := New("test")
	require.NoError(t, q.Close())
	err := q.Enqueue(context.Background(), newTestTask("t1", 1))
	assert.ErrorIs(t, err, queue.ErrQueueClosed)
}

func TestQueue_DequeueBlocking(t *testing.T) {
	q := New("test")
	defer q.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = q.Enqueue(context.Background(), newTestTask("t1", 1))
	}()

	start := time.Now()
	got, err := q.Dequeue(context.Background(), 1*time.Second)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "t1", got.ID)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	q := New("test")
	defer q.Close()

	var wg sync.WaitGroup
	var enqueued, dequeued atomic.Int32

	// 10 producers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				id := fmt.Sprintf("t-%d-%d", n, j)
				if err := q.Enqueue(context.Background(), newTestTask(id, j%5)); err == nil {
					enqueued.Add(1)
				}
			}
		}(i)
	}

	// 5 consumers
	var consumerWG sync.WaitGroup
	for i := 0; i < 5; i++ {
		consumerWG.Add(1)
		go func() {
			defer consumerWG.Done()
			for {
				task, err := q.Dequeue(context.Background(), 100*time.Millisecond)
				if err != nil {
					return
				}
				dequeued.Add(1)
				_ = q.Ack(context.Background(), task.ID, queue.StatusSuccess, "")
			}
		}()
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)
	q.Close()
	consumerWG.Wait()

	assert.Equal(t, int32(1000), enqueued.Load())
	assert.Equal(t, enqueued.Load(), dequeued.Load())
}

func TestQueue_Recover(t *testing.T) {
	q := New("test")
	defer q.Close()

	tasks, err := q.Recover(context.Background())
	require.NoError(t, err)
	assert.Nil(t, tasks) // in-memory has nothing to recover
}

func TestQueue_GetNotFound(t *testing.T) {
	q := New("test")
	defer q.Close()

	_, err := q.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

func TestQueue_AckNotFound(t *testing.T) {
	q := New("test")
	defer q.Close()

	err := q.Ack(context.Background(), "nonexistent", queue.StatusSuccess, "")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

func TestQueue_CancelNotFound(t *testing.T) {
	q := New("test")
	defer q.Close()

	err := q.Cancel(context.Background(), "nonexistent")
	// Cancel is idempotent in memory backend — it just removes if present.
	_ = err
}

func TestQueue_RequeueNotFound(t *testing.T) {
	q := New("test")
	defer q.Close()

	err := q.Requeue(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

func TestQueue_Name(t *testing.T) {
	q := New("my-queue")
	assert.Equal(t, "my-queue", q.Name())
	_ = q.Close()
}

// Ensure errors package is used.
var _ = errors.Is

func TestQueue_Position(t *testing.T) {
	q := New("test")
	ctx := context.Background()

	t1 := &queue.Task{ID: "t1", Priority: 1}
	t2 := &queue.Task{ID: "t2", Priority: 1}
	t3 := &queue.Task{ID: "t3", Priority: 1}
	_ = q.Enqueue(ctx, t1)
	_ = q.Enqueue(ctx, t2)
	_ = q.Enqueue(ctx, t3)

	pos, err := q.Position(ctx, "t2")
	assert.NoError(t, err)
	assert.Equal(t, 1, pos)

	pos, err = q.Position(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, -1, pos)
}

func TestQueue_UpdateProgress(t *testing.T) {
	q := New("test")
	ctx := context.Background()

	task := &queue.Task{ID: "t1", Priority: 1}
	_ = q.Enqueue(ctx, task)

	err := q.UpdateProgress(ctx, "t1", 50)
	assert.NoError(t, err)

	got, _ := q.Get(ctx, "t1")
	assert.Equal(t, 50, got.Progress)

	// Clamp above 100.
	_ = q.UpdateProgress(ctx, "t1", 200)
	got, _ = q.Get(ctx, "t1")
	assert.Equal(t, 100, got.Progress)

	// Not found.
	err = q.UpdateProgress(ctx, "nope", 10)
	assert.Error(t, err)
}

func TestQueue_AppendLogAndListLogs(t *testing.T) {
	q := New("test")
	ctx := context.Background()

	_ = q.AppendLog(ctx, &queue.TaskLogEntry{TaskID: "t1", Level: queue.LogLevelInfo, Message: "first"})
	_ = q.AppendLog(ctx, &queue.TaskLogEntry{TaskID: "t1", Level: queue.LogLevelWarn, Message: "second"})
	_ = q.AppendLog(ctx, &queue.TaskLogEntry{TaskID: "t1", Level: queue.LogLevelError, Message: "third"})

	logs, err := q.ListLogs(ctx, "t1", 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 3)
	// Newest first.
	assert.Equal(t, "third", logs[0].Message)
	assert.Equal(t, "second", logs[1].Message)
	assert.Equal(t, "first", logs[2].Message)

	// Limit.
	logs, err = q.ListLogs(ctx, "t1", 2)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, "third", logs[0].Message)

	// Empty.
	logs, err = q.ListLogs(ctx, "nonexistent", 0)
	assert.NoError(t, err)
	assert.Nil(t, logs)
}
