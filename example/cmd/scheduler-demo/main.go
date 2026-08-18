// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command scheduler-demo demonstrates the ling-base scheduler package.
//
// It shows:
//   - Single-node scheduling (NoLockFactory)
//   - Distributed scheduling with in-memory locks (simulating 2 nodes)
//   - Cron expression and @every interval scheduling
//   - Job status tracking
//   - Event listener for monitoring
//   - Graceful shutdown
//
// Usage:
//
//	go run ./cmd/scheduler-demo
package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/ling-base/common/lock"
	"github.com/LingByte/ling-base/common/lock/memory"
	"github.com/LingByte/ling-base/scheduler"
)

func main() {
	fmt.Println("=== Scheduler Demo ===")

	// ── Part 1: Single-node scheduling ──
	fmt.Println("\n--- Part 1: Single-node scheduling ---")

	var events []scheduler.JobEvent
	var mu sync.Mutex

	s1 := scheduler.New(scheduler.Config{
		LockFactory: scheduler.NoLockFactory{},
		EventListener: scheduler.EventListenerFunc(func(e scheduler.JobEvent) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)
		}),
	})

	var heartbeatCount int32
	_ = s1.Add("heartbeat", "@every 500ms", func(ctx context.Context) error {
		c := atomic.AddInt32(&heartbeatCount, 1)
		fmt.Printf("  [heartbeat] tick #%d at %s\n", c, time.Now().Format("15:04:05.000"))
		return nil
	})

	var cleanupCount int32
	_ = s1.AddWithOptions("cleanup", "*/2 * * * * *", // every 2 seconds
		func(ctx context.Context) error {
			c := atomic.AddInt32(&cleanupCount, 1)
			fmt.Printf("  [cleanup]   run #%d at %s\n", c, time.Now().Format("15:04:05.000"))
			return nil
		},
		scheduler.Options{
			Description: "Periodic cleanup task",
			Tags:        []string{"maintenance"},
		},
	)

	s1.Start()
	fmt.Println("  Scheduler started, running for 5 seconds...")
	time.Sleep(5 * time.Second)
	s1.Stop()

	fmt.Printf("\n  Results: heartbeat ran %d times, cleanup ran %d times\n",
		atomic.LoadInt32(&heartbeatCount), atomic.LoadInt32(&cleanupCount))

	mu.Lock()
	fmt.Printf("  Events captured: %d\n", len(events))
	mu.Unlock()

	// ── Part 2: Distributed scheduling (2 simulated nodes) ──
	fmt.Println("\n--- Part 2: Distributed scheduling (2 nodes) ---")

	mgr := memory.NewManager()
	factory := func(name string) (lock.Locker, error) {
		return mgr.NewMutex("scheduler:"+name, lock.WithTTL(10*time.Second))
	}

	var node1Count, node2Count int32

	node1 := scheduler.New(scheduler.Config{
		LockFactory:         scheduler.LockFactoryFunc(factory),
		LockTTL:             10 * time.Second,
		LockRefreshInterval: 3 * time.Second,
	})
	node2 := scheduler.New(scheduler.Config{
		LockFactory:         scheduler.LockFactoryFunc(factory),
		LockTTL:             10 * time.Second,
		LockRefreshInterval: 3 * time.Second,
	})

	// Both nodes register the same job. Only one should execute it per tick.
	jobFunc := func(nodeName string, counter *int32) scheduler.JobFunc {
		return func(ctx context.Context) error {
			c := atomic.AddInt32(counter, 1)
			fmt.Printf("  [%s] distributed job run #%d at %s\n",
				nodeName, c, time.Now().Format("15:04:05.000"))
			time.Sleep(200 * time.Millisecond) // hold lock briefly
			return nil
		}
	}

	_ = node1.Add("distributed-task", "@every 1s", jobFunc("node1", &node1Count))
	_ = node2.Add("distributed-task", "@every 1s", jobFunc("node2", &node2Count))

	node1.Start()
	node2.Start()
	fmt.Println("  Both nodes started, running for 5 seconds...")
	time.Sleep(5 * time.Second)
	node1.Stop()
	node2.Stop()

	n1 := atomic.LoadInt32(&node1Count)
	n2 := atomic.LoadInt32(&node2Count)
	total := n1 + n2
	fmt.Printf("\n  Results: node1=%d, node2=%d, total=%d\n", n1, n2, total)
	fmt.Printf("  (Total should be ~5, not ~10, because distributed lock ensures\n")
	fmt.Printf("   only one node runs the job per tick)\n")

	// ── Part 3: Singleton mode ──
	fmt.Println("\n--- Part 3: Singleton mode ---")

	s3 := scheduler.New(scheduler.DefaultConfig())
	var singletonCount int32
	var singletonSkipped int32

	_ = s3.AddWithOptions("singleton-task", "@every 500ms",
		func(ctx context.Context) error {
			c := atomic.AddInt32(&singletonCount, 1)
			fmt.Printf("  [singleton] run #%d at %s (sleeping 1s)\n",
				c, time.Now().Format("15:04:05.000"))
			time.Sleep(1 * time.Second)
			return nil
		},
		scheduler.Options{Singleton: true},
	)

	s3.Start()
	fmt.Println("  Singleton scheduler started, running for 3 seconds...")
	time.Sleep(3 * time.Second)
	s3.Stop()

	fmt.Printf("  Results: ran %d times (should be ~2-3, not ~6, because\n",
		atomic.LoadInt32(&singletonCount))
	fmt.Printf("  singleton mode skips if previous run is still active)\n")
	_ = singletonSkipped

	fmt.Println("\n=== Demo complete ===")
}
