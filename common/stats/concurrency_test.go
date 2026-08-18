// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stats_test

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	redisstats "github.com/LingByte/ling-base/common/stats/redis"
	"github.com/redis/go-redis/v9"
)

// ──────────────────────────────────────────────
// 并发写测试：不同 goroutine 数
// ──────────────────────────────────────────────

func benchCounterConcurrentMemory(b *testing.B, parallelism int) {
	c := memory.New()
	ctr := c.Counter("bench:cc")
	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctr.Incr()
		}
	})
}

func benchCounterConcurrentRedis(b *testing.B, parallelism int) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", PoolSize: parallelism * 2})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	ctr := c.Counter("cc")
	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctr.Incr()
		}
	})
}

func BenchmarkCounter_Concurrent_Memory_1(b *testing.B)  { benchCounterConcurrentMemory(b, 1) }
func BenchmarkCounter_Concurrent_Memory_4(b *testing.B)  { benchCounterConcurrentMemory(b, 4) }
func BenchmarkCounter_Concurrent_Memory_16(b *testing.B) { benchCounterConcurrentMemory(b, 16) }
func BenchmarkCounter_Concurrent_Memory_64(b *testing.B) { benchCounterConcurrentMemory(b, 64) }

func BenchmarkCounter_Concurrent_Redis_1(b *testing.B)  { benchCounterConcurrentRedis(b, 1) }
func BenchmarkCounter_Concurrent_Redis_4(b *testing.B)  { benchCounterConcurrentRedis(b, 4) }
func BenchmarkCounter_Concurrent_Redis_16(b *testing.B) { benchCounterConcurrentRedis(b, 16) }
func BenchmarkCounter_Concurrent_Redis_64(b *testing.B) { benchCounterConcurrentRedis(b, 64) }

// ──────────────────────────────────────────────
// HLL 并发写测试
// ──────────────────────────────────────────────

func benchHLLConcurrentMemory(b *testing.B, parallelism int) {
	c := memory.New()
	h := c.HLL("bench:hllc")
	var counter int64
	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1)
			h.Add(fmt.Sprintf("user-%d", i))
		}
	})
}

func benchHLLConcurrentRedis(b *testing.B, parallelism int) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", PoolSize: parallelism * 2})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	h := c.HLL("hllc")
	var counter int64
	b.SetParallelism(parallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1)
			h.Add(fmt.Sprintf("user-%d", i))
		}
	})
}

func BenchmarkHLL_Concurrent_Memory_1(b *testing.B)  { benchHLLConcurrentMemory(b, 1) }
func BenchmarkHLL_Concurrent_Memory_4(b *testing.B)  { benchHLLConcurrentMemory(b, 4) }
func BenchmarkHLL_Concurrent_Memory_16(b *testing.B) { benchHLLConcurrentMemory(b, 16) }
func BenchmarkHLL_Concurrent_Memory_64(b *testing.B) { benchHLLConcurrentMemory(b, 64) }

func BenchmarkHLL_Concurrent_Redis_1(b *testing.B)  { benchHLLConcurrentRedis(b, 1) }
func BenchmarkHLL_Concurrent_Redis_4(b *testing.B)  { benchHLLConcurrentRedis(b, 4) }
func BenchmarkHLL_Concurrent_Redis_16(b *testing.B) { benchHLLConcurrentRedis(b, 16) }
func BenchmarkHLL_Concurrent_Redis_64(b *testing.B) { benchHLLConcurrentRedis(b, 64) }

// ──────────────────────────────────────────────
// HLL 规模测试：不同数据量下的 Estimate 效率
// ──────────────────────────────────────────────

func benchHLLEstimateMemory(b *testing.B, size int) {
	c := memory.New()
	h := c.HLL("bench:hll_size")
	for i := 0; i < size; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Estimate()
	}
}

func benchHLLEstimateRedis(b *testing.B, size int) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	h := c.HLL("hll_size")
	for i := 0; i < size; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Estimate()
	}
}

func BenchmarkHLLEstimate_Memory_1K(b *testing.B)   { benchHLLEstimateMemory(b, 1000) }
func BenchmarkHLLEstimate_Memory_10K(b *testing.B)  { benchHLLEstimateMemory(b, 10000) }
func BenchmarkHLLEstimate_Memory_100K(b *testing.B) { benchHLLEstimateMemory(b, 100000) }
func BenchmarkHLLEstimate_Memory_1M(b *testing.B)   { benchHLLEstimateMemory(b, 1000000) }

func BenchmarkHLLEstimate_Redis_1K(b *testing.B)   { benchHLLEstimateRedis(b, 1000) }
func BenchmarkHLLEstimate_Redis_10K(b *testing.B)  { benchHLLEstimateRedis(b, 10000) }
func BenchmarkHLLEstimate_Redis_100K(b *testing.B) { benchHLLEstimateRedis(b, 100000) }
func BenchmarkHLLEstimate_Redis_1M(b *testing.B)   { benchHLLEstimateRedis(b, 1000000) }

// ──────────────────────────────────────────────
// Timer Percentile 规模测试
// ──────────────────────────────────────────────

func benchTimerPctMemory(b *testing.B, size int) {
	c := memory.New()
	t := c.Timer("bench:tp")
	for i := 0; i < size; i++ {
		t.Record(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Percentile(95)
	}
}

func benchTimerPctRedis(b *testing.B, size int) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	t := c.Timer("tp")
	for i := 0; i < size; i++ {
		t.Record(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Percentile(95)
	}
}

func BenchmarkTimerPct_Memory_1K(b *testing.B)  { benchTimerPctMemory(b, 1000) }
func BenchmarkTimerPct_Memory_10K(b *testing.B) { benchTimerPctMemory(b, 10000) }
func BenchmarkTimerPct_Memory_50K(b *testing.B) { benchTimerPctMemory(b, 50000) }

func BenchmarkTimerPct_Redis_1K(b *testing.B)  { benchTimerPctRedis(b, 1000) }
func BenchmarkTimerPct_Redis_10K(b *testing.B) { benchTimerPctRedis(b, 10000) }
func BenchmarkTimerPct_Redis_50K(b *testing.B) { benchTimerPctRedis(b, 50000) }

// ──────────────────────────────────────────────
// 混合读写测试（模拟真实场景：80%写 + 20%读）
// ──────────────────────────────────────────────

func BenchmarkMixed_Memory(b *testing.B) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"
	var counter int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1)
			if i%5 == 0 {
				// 20% 读
				_ = wm.GetUV(date)
				_ = wm.GetPV(date, "/home")
			} else {
				// 80% 写
				wm.RecordPV(date, fmt.Sprintf("/page/%d", i%10))
				wm.RecordUV(date, fmt.Sprintf("user-%d", i))
			}
		}
	})
}

func BenchmarkMixed_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", PoolSize: 50})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	defer c.Close()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"
	var counter int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddInt64(&counter, 1)
			if i%5 == 0 {
				_ = wm.GetUV(date)
				_ = wm.GetPV(date, "/home")
			} else {
				wm.RecordPV(date, fmt.Sprintf("/page/%d", i%10))
				wm.RecordUV(date, fmt.Sprintf("user-%d", i))
			}
		}
	})
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
