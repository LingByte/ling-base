# queue/redis

Redis-backed distributed task queue for the `queue` package. Uses Redis
sorted sets for priority scheduling and hashes for task metadata.

## Features

- **Priority scheduling** — higher priority tasks are dequeued first
  (within the same priority, earlier submit time wins)
- **Durable storage** — tasks survive process restarts
- **Distributed** — multiple consumer nodes can share the same queue
- **Crash recovery** — `Recover()` returns all pending + running tasks
- **Duplicate prevention** — atomic `SETNX` rejects duplicate task IDs
- **Queue position** — `Position()` returns a task's rank via `ZRevRank`
- **Progress tracking** — `UpdateProgress()` updates task progress (0-100)
- **Execution logging** — `AppendLog()` / `ListLogs()` via Redis lists

## Usage

```go
import (
    goredis "github.com/redis/go-redis/v9"
    redisqueue "github.com/LingByte/ling-base/queue/redis"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
q := redisqueue.New("my-tasks", client)

// Enqueue
task := &queue.Task{
    ID:       "task-1",
    Priority: 10,
    Payload:  payload,
}
err := q.Enqueue(ctx, task)

// Dequeue (highest priority first)
task, err := q.Dequeue(ctx, 5*time.Second)

// Check queue position
pos, _ := q.Position(ctx, "task-1")  // 0 = next to dispatch

// Update progress
q.UpdateProgress(ctx, "task-1", 50)

// Execution logs
q.AppendLog(ctx, &queue.TaskLogEntry{
    TaskID: "task-1", Level: "info", Message: "starting",
})
logs, _ := q.ListLogs(ctx, "task-1", 50)

// Crash recovery
tasks, _ := q.Recover(ctx)  // running tasks reset to pending
```

## Redis Key Layout

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `lingbase:queue:task:{id}` | String | Task JSON metadata |
| `lingbase:queue:pending:{name}` | Sorted Set | Pending queue (score = priority) |
| `lingbase:queue:running:{name}` | Sorted Set | Running tasks |
| `lingbase:queue:stats:{name}` | Hash | Queue statistics |
| `lingbase:queue:log:{id}` | List | Execution logs (LPUSH, newest first) |

## Scoring

The sorted set score is `priority * 1e10 - submitTime.UnixNano()`, so:

- Higher priority = higher score = dequeued first (`ZPopMax`)
- Within the same priority, earlier submit time = higher score = FIFO

## Testing

Tests use [`miniredis`](https://github.com/alicebob/miniredis) — an
in-memory Redis-compatible server. No external Redis required.

The blocking dequeue path (`BZPopMax` with `timeout > 0`) is not covered
by miniredis tests because miniredis does not implement blocking commands.
Use integration tests with a real Redis server for that path.

## License

MIT
