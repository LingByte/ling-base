# pool

Production-grade object pools, connection pools, and goroutine worker pools.

## Components

### ObjectPool — Generic Object Pool

A generic, thread-safe object pool with configurable max open objects.

```go
pool := pool.NewObjectPool(func() (*sql.Conn, error) {
    return db.Conn(ctx)
}, 10, func(c *sql.Conn) { c.Close() })

conn, err := pool.Get()
defer pool.Put(conn)

stats := pool.Stats() // OpenCount, IdleCount, InUse
```

### ConnPool — Connection Pool with Health Check + Idle Eviction

Extends the object pool with:
- **Health check** on Get — unhealthy connections are destroyed and replaced
- **Max lifetime** — connections older than `MaxLifetime` are recycled
- **Idle eviction** — a background janitor evicts connections idle longer than `MaxIdleTime`

```go
pool := pool.NewConnPool(factory, destroyer, pool.ConnConfig{
    MaxOpen:       20,
    MaxIdle:       10,
    MaxIdleTime:   5 * time.Minute,
    MaxLifetime:   30 * time.Minute,
    HealthCheck:   func(c *sql.Conn) error { return c.PingContext(ctx) },
    HealthPeriod:  1 * time.Minute,
})
```

### WorkerPool — Goroutine Worker Pool

A fixed-size goroutine pool with a buffered task queue and graceful shutdown.

```go
wp := pool.NewWorkerPool(8, 1000) // 8 workers, 1000 task buffer
wp.Start()

err := wp.Submit(func() {
    // do work
})

wp.Stop() // graceful — waits for all tasks to complete
```

## License

MIT
