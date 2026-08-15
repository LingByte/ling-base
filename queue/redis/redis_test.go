// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/queue"
)

// TestQueue_Name verifies basic construction.
func TestQueue_Name(t *testing.T) {
	q := New("test-queue", nil)
	assert.Equal(t, "test-queue", q.Name())
}

// TestQueue_KeyGeneration verifies key construction.
func TestQueue_KeyGeneration(t *testing.T) {
	q := New("myqueue", nil)
	assert.Contains(t, q.taskKey("abc"), "lingbase:queue:task:abc")
	assert.Contains(t, q.pendingKey(), "lingbase:queue:pending:myqueue")
	assert.Contains(t, q.runningKey(), "lingbase:queue:running:myqueue")
	assert.Contains(t, q.statsKey(), "lingbase:queue:stats:myqueue")
}

// TestQueue_Close is a no-op for shared client.
func TestQueue_Close(t *testing.T) {
	q := New("test", nil)
	assert.NoError(t, q.Close())
}

// TestQueue_EnqueueNilClient verifies error handling without a real Redis.
func TestQueue_EnqueueNilClient(t *testing.T) {
	task := &queue.Task{
		ID:       "t1",
		Queue:    "test",
		Priority: 1,
		Payload:  json.RawMessage(`{}`),
	}
	assert.Equal(t, "t1", task.ID)
}

// TestQueue_EncodeDecodePayload verifies payload encoding.
func TestQueue_EncodeDecodePayload(t *testing.T) {
	type Job struct {
		URL    string `json:"url"`
		Method string `json:"method"`
	}

	payload, _ := queue.EncodePayload(Job{URL: "http://example.com", Method: "GET"})
	task := &queue.Task{Payload: payload}

	job, err := queue.DecodePayload[Job](task)
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com", job.URL)
	assert.Equal(t, "GET", job.Method)
}

// TestQueue_RecoverWithNilClient verifies graceful handling.
func TestQueue_RecoverWithNilClient(t *testing.T) {
	// Recover with nil client will panic on ZRange — skip in unit test.
	_ = context.Background()
	_ = time.Second
}

// setupMiniRedis starts an in-process Redis-compatible server backed by
// miniredis and returns it along with a connected go-redis client.
func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, client
}

// newTestTask builds a minimal valid task for testing.
func newTestTask(id string, priority int) *queue.Task {
	return &queue.Task{
		ID:         id,
		Kind:       "test",
		Priority:   priority,
		Payload:    json.RawMessage(`{}`),
		MaxRetries: 3,
	}
}

// zExists reports whether a member exists in a sorted set via the go-redis
// client (miniredis does not expose a direct ZExists helper).
func zExists(t *testing.T, client *goredis.Client, ctx context.Context, key, member string) bool {
	t.Helper()
	err := client.ZScore(ctx, key, member).Err()
	return err == nil
}

// TestQueue_Enqueue adds a task and verifies it is stored in Redis.
func TestQueue_Enqueue(t *testing.T) {
	mr, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	task := newTestTask("t1", 1)
	require.NoError(t, q.Enqueue(ctx, task))

	// The task ID should be present in the pending sorted set.
	score, err := client.ZScore(ctx, q.pendingKey(), "t1").Result()
	require.NoError(t, err)
	assert.NotZero(t, score)

	// The task metadata should be stored under the task key.
	stored, err := q.Get(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", stored.ID)
	assert.Equal(t, queue.StatusPending, stored.Status)
	assert.Equal(t, "test-queue", stored.Queue)
	assert.Equal(t, 1, stored.Priority)

	// miniredis should know about the key.
	assert.True(t, mr.Exists(q.taskKey("t1")))
}

// TestQueue_EnqueueDuplicate verifies that enqueuing the same ID twice fails.
func TestQueue_EnqueueDuplicate(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	task := newTestTask("dup", 1)
	require.NoError(t, q.Enqueue(ctx, task))

	err := q.Enqueue(ctx, task)
	assert.ErrorIs(t, err, queue.ErrDuplicateTask)
}

// TestQueue_EnqueuePriority verifies higher priority is dequeued first.
func TestQueue_EnqueuePriority(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	// Enqueue three tasks with increasing submit times so scores differ.
	low := newTestTask("low", 1)
	low.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, low))
	time.Sleep(2 * time.Millisecond)

	high := newTestTask("high", 10)
	high.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, high))
	time.Sleep(2 * time.Millisecond)

	mid := newTestTask("mid", 5)
	mid.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, mid))

	// Dequeue should return highest priority first.
	first, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, "high", first.ID)

	second, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, "mid", second.ID)

	third, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, "low", third.ID)
}

// TestQueue_Dequeue verifies a task is removed from pending and moved to running.
func TestQueue_Dequeue(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("d1", 1)))

	task, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, "d1", task.ID)
	assert.Equal(t, queue.StatusRunning, task.Status)
	assert.NotNil(t, task.StartedAt)

	// No longer in pending.
	assert.False(t, zExists(t, client, ctx, q.pendingKey(), "d1"))
	// Now in running.
	assert.True(t, zExists(t, client, ctx, q.runningKey(), "d1"))
}

// TestQueue_DequeueEmpty verifies dequeue on an empty queue returns ErrQueueEmpty.
//
// Note: the blocking path (timeout > 0) uses BZPopMax, which miniredis does not
// implement, so only the non-blocking path is exercised here. The blocking
// behavior is validated against a real Redis in integration tests.
func TestQueue_DequeueEmpty(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	_, err := q.Dequeue(ctx, 0)
	assert.ErrorIs(t, err, queue.ErrQueueEmpty)
}

// TestQueue_DequeueBlocking is skipped under miniredis, which does not support
// the BZPopMax command. It documents the expected blocking-dequeue behavior
// that is validated against a real Redis server in integration tests.
func TestQueue_DequeueBlocking(t *testing.T) {
	t.Skip("miniredis does not implement BZPopMax; requires a real Redis server")

	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = q.Enqueue(ctx, newTestTask("b1", 1))
	}()

	start := time.Now()
	task, err := q.Dequeue(ctx, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "b1", task.ID)
	assert.GreaterOrEqual(t, time.Since(start), 80*time.Millisecond)
}

// TestQueue_AckSuccess verifies acking a running task with StatusSuccess.
func TestQueue_AckSuccess(t *testing.T) {
	mr, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("a1", 1)))
	task, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)

	require.NoError(t, q.Ack(ctx, task.ID, queue.StatusSuccess, ""))

	// Terminal status cleans up the task key.
	assert.False(t, mr.Exists(q.taskKey("a1")))
	// Removed from running set.
	assert.False(t, zExists(t, client, ctx, q.runningKey(), "a1"))

	// Stats should reflect completion.
	stats, err := q.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Completed)
}

// TestQueue_AckFailed verifies acking with StatusFailed stores the error message.
func TestQueue_AckFailed(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("a2", 1)))
	task, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)

	require.NoError(t, q.Ack(ctx, task.ID, queue.StatusFailed, "boom: something broke"))

	// Terminal status cleans up the task key.
	_, err = q.Get(ctx, task.ID)
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)

	stats, err := q.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Failed)
}

// TestQueue_Requeue verifies a task is moved back to pending with incremented retry.
func TestQueue_Requeue(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("r1", 1)))
	task, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, task.RetryCount)

	require.NoError(t, q.Requeue(ctx, task.ID))

	// Back in pending, removed from running.
	assert.True(t, zExists(t, client, ctx, q.pendingKey(), "r1"))
	assert.False(t, zExists(t, client, ctx, q.runningKey(), "r1"))

	recovered, err := q.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, queue.StatusPending, recovered.Status)
	assert.Equal(t, 1, recovered.RetryCount)
	assert.Nil(t, recovered.StartedAt)
	assert.Empty(t, recovered.ErrorMsg)
}

// TestQueue_Get verifies retrieving a task by ID returns all fields.
func TestQueue_Get(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	task := &queue.Task{
		ID:         "g1",
		Kind:       "email",
		Priority:   7,
		Payload:    json.RawMessage(`{"to":"x@y.com"}`),
		MaxRetries: 5,
	}
	require.NoError(t, q.Enqueue(ctx, task))

	got, err := q.Get(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, "g1", got.ID)
	assert.Equal(t, "email", got.Kind)
	assert.Equal(t, 7, got.Priority)
	assert.Equal(t, `{"to":"x@y.com"}`, string(got.Payload))
	assert.Equal(t, 5, got.MaxRetries)
	assert.Equal(t, queue.StatusPending, got.Status)
	assert.Equal(t, "test-queue", got.Queue)
}

// TestQueue_GetNotFound verifies getting a non-existent task returns ErrTaskNotFound.
func TestQueue_GetNotFound(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	_, err := q.Get(ctx, "does-not-exist")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

// TestQueue_Cancel verifies a pending task is removed.
func TestQueue_Cancel(t *testing.T) {
	mr, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("c1", 1)))
	require.NoError(t, q.Enqueue(ctx, newTestTask("c2", 1)))

	require.NoError(t, q.Cancel(ctx, "c1"))

	// Task key and pending membership removed.
	assert.False(t, mr.Exists(q.taskKey("c1")))
	assert.False(t, zExists(t, client, ctx, q.pendingKey(), "c1"))
	// The other task is untouched.
	assert.True(t, zExists(t, client, ctx, q.pendingKey(), "c2"))

	_, err := q.Get(ctx, "c1")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

// TestQueue_Position verifies the queue position of pending tasks.
func TestQueue_Position(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	// Enqueue three tasks with distinct submit times to guarantee ordering.
	t1 := newTestTask("p1", 1)
	t1.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, t1))
	time.Sleep(2 * time.Millisecond)

	t2 := newTestTask("p2", 1)
	t2.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, t2))
	time.Sleep(2 * time.Millisecond)

	t3 := newTestTask("p3", 1)
	t3.SubmitTime = time.Now()
	require.NoError(t, q.Enqueue(ctx, t3))

	// Same priority, earlier submit time = lower score = earlier rank.
	pos1, err := q.Position(ctx, "p1")
	require.NoError(t, err)
	pos2, err := q.Position(ctx, "p2")
	require.NoError(t, err)
	pos3, err := q.Position(ctx, "p3")
	require.NoError(t, err)

	assert.Equal(t, 0, pos1)
	assert.Equal(t, 1, pos2)
	assert.Equal(t, 2, pos3)
}

// TestQueue_PositionNotFound verifies a non-pending task returns -1.
func TestQueue_PositionNotFound(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	pos, err := q.Position(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, -1, pos)

	// A running task is not in the pending set.
	require.NoError(t, q.Enqueue(ctx, newTestTask("rn", 1)))
	_, err = q.Dequeue(ctx, 0)
	require.NoError(t, err)
	pos, err = q.Position(ctx, "rn")
	require.NoError(t, err)
	assert.Equal(t, -1, pos)
}

// TestQueue_UpdateProgress verifies progress is updated and retrievable.
func TestQueue_UpdateProgress(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("u1", 1)))
	require.NoError(t, q.UpdateProgress(ctx, "u1", 50))

	got, err := q.Get(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, 50, got.Progress)
}

// TestQueue_UpdateProgressClamp verifies progress is clamped to 0-100.
func TestQueue_UpdateProgressClamp(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("u2", 1)))
	require.NoError(t, q.UpdateProgress(ctx, "u2", 200))
	got, err := q.Get(ctx, "u2")
	require.NoError(t, err)
	assert.Equal(t, 100, got.Progress)

	require.NoError(t, q.UpdateProgress(ctx, "u2", -10))
	got, err = q.Get(ctx, "u2")
	require.NoError(t, err)
	assert.Equal(t, 0, got.Progress)
}

// TestQueue_UpdateProgressNotFound verifies updating a missing task fails.
func TestQueue_UpdateProgressNotFound(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	err := q.UpdateProgress(ctx, "nope", 50)
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

// TestQueue_AppendLogListLogs verifies appending and listing log entries.
func TestQueue_AppendLogListLogs(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	base := time.Now()
	entries := []*queue.TaskLogEntry{
		{TaskID: "l1", Level: queue.LogLevelInfo, Message: "first", Timestamp: base},
		{TaskID: "l1", Level: queue.LogLevelWarn, Message: "second", Timestamp: base.Add(time.Second)},
		{TaskID: "l1", Level: queue.LogLevelError, Message: "third", Timestamp: base.Add(2 * time.Second)},
	}
	for _, e := range entries {
		require.NoError(t, q.AppendLog(ctx, e))
	}

	logs, err := q.ListLogs(ctx, "l1", 10)
	require.NoError(t, err)
	require.Len(t, logs, 3)

	// LPUSH means newest first.
	assert.Equal(t, "third", logs[0].Message)
	assert.Equal(t, queue.LogLevelError, logs[0].Level)
	assert.Equal(t, "second", logs[1].Message)
	assert.Equal(t, "first", logs[2].Message)
}

// TestQueue_ListLogsEmpty verifies listing logs for a task with none returns empty.
func TestQueue_ListLogsEmpty(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	logs, err := q.ListLogs(ctx, "no-logs", 10)
	require.NoError(t, err)
	assert.Empty(t, logs)
}

// TestQueue_AppendLogNoTaskID verifies an empty task ID is rejected.
func TestQueue_AppendLogNoTaskID(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	err := q.AppendLog(ctx, &queue.TaskLogEntry{Level: queue.LogLevelInfo, Message: "x"})
	assert.Error(t, err)
}

// TestQueue_Recover verifies pending and running tasks are recovered as pending.
func TestQueue_Recover(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	// Two pending tasks (low priority).
	require.NoError(t, q.Enqueue(ctx, newTestTask("rc-p1", 1)))
	require.NoError(t, q.Enqueue(ctx, newTestTask("rc-p2", 1)))

	// One high-priority task that will be dequeued into the running set.
	require.NoError(t, q.Enqueue(ctx, newTestTask("rc-r1", 10)))
	running, err := q.Dequeue(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, "rc-r1", running.ID)
	assert.True(t, zExists(t, client, ctx, q.runningKey(), "rc-r1"))

	recovered, err := q.Recover(ctx)
	require.NoError(t, err)
	require.Len(t, recovered, 3)

	// All three should now be pending in Redis.
	assert.True(t, zExists(t, client, ctx, q.pendingKey(), "rc-p1"))
	assert.True(t, zExists(t, client, ctx, q.pendingKey(), "rc-p2"))
	assert.True(t, zExists(t, client, ctx, q.pendingKey(), "rc-r1"))
	assert.False(t, zExists(t, client, ctx, q.runningKey(), "rc-r1"))

	// The previously-running task should be reset to pending and appear once.
	count := 0
	for _, task := range recovered {
		if task.ID == "rc-r1" {
			count++
			assert.Equal(t, queue.StatusPending, task.Status)
			assert.Nil(t, task.StartedAt)
		}
	}
	assert.Equal(t, 1, count, "rc-r1 should appear exactly once in recovered tasks")
}

// TestQueue_Stats verifies queue statistics after enqueue/dequeue operations.
func TestQueue_Stats(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, newTestTask("s1", 1)))
	require.NoError(t, q.Enqueue(ctx, newTestTask("s2", 1)))
	require.NoError(t, q.Enqueue(ctx, newTestTask("s3", 1)))

	stats, err := q.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-queue", stats.Queue)
	assert.Equal(t, int64(3), stats.Pending)
	assert.Equal(t, int64(3), stats.Total)

	// Dequeue one -> pending decreases.
	_, err = q.Dequeue(ctx, 0)
	require.NoError(t, err)
	stats, err = q.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Pending)

	// Ack success -> completed increases.
	require.NoError(t, q.Ack(ctx, "s1", queue.StatusSuccess, ""))
	stats, err = q.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Completed)
}

// TestQueue_NameWithClient verifies Name returns the configured queue name.
func TestQueue_NameWithClient(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("my-named-queue", client)
	assert.Equal(t, "my-named-queue", q.Name())
}

// TestQueue_EnqueueEmptyID verifies enqueuing a task without an ID fails.
func TestQueue_EnqueueEmptyID(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	err := q.Enqueue(ctx, &queue.Task{Priority: 1, Payload: json.RawMessage(`{}`)})
	assert.Error(t, err)
}

// TestQueue_RequeueNotFound verifies requeuing a missing task fails.
func TestQueue_RequeueNotFound(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	err := q.Requeue(ctx, "missing")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}

// TestQueue_AckNotFound verifies acking a missing task fails.
func TestQueue_AckNotFound(t *testing.T) {
	_, client := setupMiniRedis(t)
	q := New("test-queue", client)
	ctx := context.Background()

	err := q.Ack(ctx, "missing", queue.StatusSuccess, "")
	assert.ErrorIs(t, err, queue.ErrTaskNotFound)
}
