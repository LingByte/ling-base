// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/LingByte/ling-base/common/stats/file"
	"github.com/LingByte/ling-base/common/stats/memory"
)

func TestFilePersistence(t *testing.T) {
	path := "/tmp/test_stats.gob"
	defer os.Remove(path)

	// Create and record data.
	c, err := file.New(path)
	if err != nil {
		t.Fatal(err)
	}

	c.Counter("pv:2026-08-18:/home").IncrBy(5)
	c.HLL("uv:2026-08-18").Add("user1")
	c.HLL("uv:2026-08-18").Add("user2")
	c.Set("daily:2026-08-18").Add("user1")
	c.Timer("rt:2026-08-18").Record(100)

	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	// Reload and verify.
	c2, err := file.New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()

	assert := func(name string, got, want any) {
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}

	assert("counter", c2.Counter("pv:2026-08-18:/home").Get(), int64(5))
	assert("hll", c2.HLL("uv:2026-08-18").Estimate(), uint64(2))
	assert("set", c2.Set("daily:2026-08-18").Count(), 1)
	assert("timer count", c2.Timer("rt:2026-08-18").Count(), int64(1))
}

func TestMemorySnapshot(t *testing.T) {
	c := memory.New()
	c.Counter("test").IncrBy(42)
	c.Gauge("g").Set(99)

	snap := c.Snapshot()
	if snap.Counters["test"] != 42 {
		t.Errorf("counter: got %d, want 42", snap.Counters["test"])
	}
	if snap.Gauges["g"] != 99 {
		t.Errorf("gauge: got %d, want 99", snap.Gauges["g"])
	}
}
