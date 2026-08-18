// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package stats_test contains benchmarks comparing memory and Redis backends.
package stats_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
	redisstats "github.com/LingByte/ling-base/common/stats/redis"
	"github.com/redis/go-redis/v9"
)

// ──────────────────────────────────────────────
// Counter benchmarks
// ──────────────────────────────────────────────

func BenchmarkCounter_Memory(b *testing.B) {
	c := memory.New()
	ctr := c.Counter("bench:counter")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctr.Incr()
		}
	})
}

func BenchmarkCounter_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	ctr := c.Counter("counter")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctr.Incr()
		}
	})
}

// ──────────────────────────────────────────────
// HLL (UV) benchmarks
// ──────────────────────────────────────────────

func BenchmarkHLL_Memory(b *testing.B) {
	c := memory.New()
	h := c.HLL("bench:hll")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
}

func BenchmarkHLL_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	h := c.HLL("hll")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
}

func BenchmarkHLLEstimate_Memory(b *testing.B) {
	c := memory.New()
	h := c.HLL("bench:hll_est")
	for i := 0; i < 100000; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Estimate()
	}
}

func BenchmarkHLLEstimate_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	h := c.HLL("hll_est")
	for i := 0; i < 100000; i++ {
		h.Add(fmt.Sprintf("user-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Estimate()
	}
}

// ──────────────────────────────────────────────
// Set benchmarks
// ──────────────────────────────────────────────

func BenchmarkSet_Memory(b *testing.B) {
	c := memory.New()
	s := c.Set("bench:set")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(fmt.Sprintf("user-%d", i))
	}
}

func BenchmarkSet_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	s := c.Set("set")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(fmt.Sprintf("user-%d", i))
	}
}

// ──────────────────────────────────────────────
// Timer benchmarks
// ──────────────────────────────────────────────

func BenchmarkTimer_Memory(b *testing.B) {
	c := memory.New()
	t := c.Timer("bench:timer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.Record(int64(i))
	}
}

func BenchmarkTimer_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	t := c.Timer("timer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.Record(int64(i))
	}
}

func BenchmarkTimerPercentile_Memory(b *testing.B) {
	c := memory.New()
	t := c.Timer("bench:timer_pct")
	for i := 0; i < 10000; i++ {
		t.Record(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Percentile(95)
	}
}

func BenchmarkTimerPercentile_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	t := c.Timer("timer_pct")
	for i := 0; i < 10000; i++ {
		t.Record(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.Percentile(95)
	}
}

// ──────────────────────────────────────────────
// Full WebsiteMetrics workflow benchmark
// ──────────────────────────────────────────────

func BenchmarkWebsiteMetrics_Memory(b *testing.B) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		path := fmt.Sprintf("/page/%d", i%10)
		wm.RecordPV(date, path)
		wm.RecordUV(date, userID)
		wm.RecordIP(date, fmt.Sprintf("10.0.%d.%d", i/256, i%256))
		wm.RecordVV(date)
		wm.RecordRequest(date)
	}
}

func BenchmarkWebsiteMetrics_Redis(b *testing.B) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer client.Close()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(context.Background())

	c := redisstats.New(client, redisstats.WithKeyPrefix("bench:"))
	defer c.Close()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		path := fmt.Sprintf("/page/%d", i%10)
		wm.RecordPV(date, path)
		wm.RecordUV(date, userID)
		wm.RecordIP(date, fmt.Sprintf("10.0.%d.%d", i/256, i%256))
		wm.RecordVV(date)
		wm.RecordRequest(date)
	}
}

// ──────────────────────────────────────────────
// Parallel WebsiteMetrics benchmark
// ──────────────────────────────────────────────

func BenchmarkWebsiteMetrics_Memory_Parallel(b *testing.B) {
	c := memory.New()
	wm := stats.NewWebsiteMetrics(c)
	date := "2026-08-18"
	b.ResetTimer()
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter
			counter++
			wm.RecordPV(date, fmt.Sprintf("/page/%d", i%10))
			wm.RecordUV(date, fmt.Sprintf("user-%d", i))
		}
	})
}

func BenchmarkWebsiteMetrics_Redis_Parallel(b *testing.B) {
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
	b.ResetTimer()
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter
			counter++
			wm.RecordPV(date, fmt.Sprintf("/page/%d", i%10))
			wm.RecordUV(date, fmt.Sprintf("user-%d", i))
		}
	})
}
