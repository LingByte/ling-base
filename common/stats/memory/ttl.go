// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"strings"
	"sync"
	"time"
)

// TTLConfig configures time-to-live based automatic cleanup for the
// in-memory collector. Keys containing a date prefix (e.g. "pv:2026-08-18:...")
// are eligible for expiration.
//
// When a key expires:
//  1. The onExpire callback is invoked with the key and its current value,
//     allowing the caller to persist the data to a database before removal.
//  2. The key is removed from all in-memory maps.
//
// This ensures bounded memory usage for long-running services: only the
// most recent `retentionDays` of data stays in memory, while older data
// is flushed to external storage (SQLite, MySQL, etc.).
type TTLConfig struct {
	// RetentionDays is the number of days of data to keep in memory.
	// Keys with dates older than this are expired.
	// Default: 7
	RetentionDays int

	// CheckInterval is how often the cleanup goroutine runs.
	// Default: 1 hour
	CheckInterval time.Duration

	// OnExpire is called for each expired key before removal.
	// It receives the key and a SnapshotEntry containing all primitive values.
	// If OnExpire returns an error, the key is NOT removed (will retry next cycle).
	// If OnExpire is nil, keys are removed without callback.
	OnExpire func(key string, entry SnapshotEntry) error

	// KeyDateExtractor extracts a date from a key string.
	// If it returns ok=false, the key is never expired.
	// Default: extracts "YYYY-MM-DD" from any position in the key.
	KeyDateExtractor func(key string) (date string, ok bool)
}

// SnapshotEntry represents the value of a single key at expiration time.
type SnapshotEntry struct {
	Type  string // "counter", "gauge", "set", "hll", "timer"
	Value any    // type-specific value
}

// DefaultKeyDateExtractor extracts a "YYYY-MM-DD" date from a key.
// It searches for a 10-character substring matching the date pattern.
// Returns ok=false if no date is found.
func DefaultKeyDateExtractor(key string) (string, bool) {
	// Look for "YYYY-MM-DD" pattern anywhere in the key.
	// Format: 4 digits + "-" + 2 digits + "-" + 2 digits
	for i := 0; i <= len(key)-10; i++ {
		if isDigit(key[i]) && isDigit(key[i+1]) && isDigit(key[i+2]) && isDigit(key[i+3]) &&
			key[i+4] == '-' &&
			isDigit(key[i+5]) && isDigit(key[i+6]) &&
			key[i+7] == '-' &&
			isDigit(key[i+8]) && isDigit(key[i+9]) {
			return key[i : i+10], true
		}
	}
	return "", false
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// ttlManager runs a background goroutine that periodically checks for
// expired keys and invokes the OnExpire callback before removing them.
type ttlManager struct {
	config  TTLConfig
	collector *Collector
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func newTTLManager(c *Collector, config TTLConfig) *ttlManager {
	if config.RetentionDays <= 0 {
		config.RetentionDays = 7
	}
	if config.CheckInterval <= 0 {
		config.CheckInterval = time.Hour
	}
	if config.KeyDateExtractor == nil {
		config.KeyDateExtractor = DefaultKeyDateExtractor
	}
	return &ttlManager{
		config:    config,
		collector: c,
		stopCh:    make(chan struct{}),
	}
}

func (m *ttlManager) start() {
	m.wg.Add(1)
	go m.loop()
}

func (m *ttlManager) stop() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *ttlManager) loop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	// Run once at start (after a short delay to allow initialization).
	initialTimer := time.NewTimer(5 * time.Second)
	defer initialTimer.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-initialTimer.C:
			m.cleanup()
		case <-ticker.C:
			m.cleanup()
		}
	}
}

func (m *ttlManager) cleanup() {
	cutoff := time.Now().AddDate(0, 0, -m.config.RetentionDays)
	cutoffStr := cutoff.Format("2006-01-02")

	m.collector.mu.Lock()
	defer m.collector.mu.Unlock()

	// Collect expired keys from all maps.
	var expired []string
	var entries []SnapshotEntry

	for key, ctr := range m.collector.counters {
		date, ok := m.config.KeyDateExtractor(key)
		if !ok || date >= cutoffStr {
			continue
		}
		if m.config.OnExpire != nil {
			if err := m.config.OnExpire(key, SnapshotEntry{Type: "counter", Value: ctr.Get()}); err != nil {
				continue // skip removal on error
			}
		}
		expired = append(expired, key)
		entries = append(entries, SnapshotEntry{})
	}
	for _, key := range expired {
		delete(m.collector.counters, key)
	}

	// Gauges
	expired = expired[:0]
	for key, g := range m.collector.gauges {
		date, ok := m.config.KeyDateExtractor(key)
		if !ok || date >= cutoffStr {
			continue
		}
		if m.config.OnExpire != nil {
			if err := m.config.OnExpire(key, SnapshotEntry{Type: "gauge", Value: g.Get()}); err != nil {
				continue
			}
		}
		expired = append(expired, key)
	}
	for _, key := range expired {
		delete(m.collector.gauges, key)
	}

	// Sets
	expired = expired[:0]
	for key, s := range m.collector.sets {
		date, ok := m.config.KeyDateExtractor(key)
		if !ok || date >= cutoffStr {
			continue
		}
		if m.config.OnExpire != nil {
			if err := m.config.OnExpire(key, SnapshotEntry{Type: "set", Value: s.Count()}); err != nil {
				continue
			}
		}
		expired = append(expired, key)
	}
	for _, key := range expired {
		delete(m.collector.sets, key)
	}

	// HLLs
	expired = expired[:0]
	for key, h := range m.collector.hlls {
		date, ok := m.config.KeyDateExtractor(key)
		if !ok || date >= cutoffStr {
			continue
		}
		if m.config.OnExpire != nil {
			if err := m.config.OnExpire(key, SnapshotEntry{Type: "hll", Value: h.Estimate()}); err != nil {
				continue
			}
		}
		expired = append(expired, key)
	}
	for _, key := range expired {
		delete(m.collector.hlls, key)
	}

	// Timers
	expired = expired[:0]
	for key, t := range m.collector.timers {
		date, ok := m.config.KeyDateExtractor(key)
		if !ok || date >= cutoffStr {
			continue
		}
		if m.config.OnExpire != nil {
			entry := SnapshotEntry{Type: "timer", Value: TimerSnapshot{
				Count:      t.Count(),
				Mean:       t.Mean(),
				P50:        t.Percentile(50),
				P95:        t.Percentile(95),
				P99:        t.Percentile(99),
			}}
			if err := m.config.OnExpire(key, SnapshotEntry{Type: "timer", Value: entry.Value}); err != nil {
				continue
			}
		}
		expired = append(expired, key)
	}
	for _, key := range expired {
		delete(m.collector.timers, key)
	}
}

// TimerSnapshot is a summary of a Timer at expiration time.
type TimerSnapshot struct {
	Count int64
	Mean  float64
	P50   float64
	P95   float64
	P99   float64
}

// WithTTL enables automatic TTL-based cleanup of old keys.
//
// Keys containing a date prefix (e.g. "pv:2026-08-18:/home") older than
// RetentionDays are automatically expired. Before removal, OnExpire is
// called so the caller can persist data to a database.
//
// Example:
//
//	c := memory.New(
//	    memory.WithReservoirTimer(4096),
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        CheckInterval: time.Hour,
//	        OnExpire: func(key string, entry memory.SnapshotEntry) error {
//	            return dbStore.Save(key, entry)
//	        },
//	    }),
//	)
func WithTTL(config TTLConfig) Option {
	return func(c *Collector) {
		c.ttlConfig = &config
	}
}

// extractDate is a helper that uses the configured KeyDateExtractor.
func (c *Collector) extractDate(key string) (string, bool) {
	if c.ttlConfig == nil || c.ttlConfig.KeyDateExtractor == nil {
		return DefaultKeyDateExtractor(key)
	}
	return c.ttlConfig.KeyDateExtractor(key)
}

// KeyCount returns the total number of keys across all primitive types.
// Useful for monitoring memory growth.
func (c *Collector) KeyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.counters) + len(c.gauges) + len(c.sets) + len(c.hlls) + len(c.timers)
}

// CleanupNow triggers an immediate TTL cleanup cycle (blocking).
// Returns the number of keys removed.
func (c *Collector) CleanupNow() int {
	if c.ttlManager == nil {
		return 0
	}
	before := c.KeyCount()
	c.ttlManager.cleanup()
	after := c.KeyCount()
	return before - after
}

// suppress unused import warning
var _ = strings.Contains
