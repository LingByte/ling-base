// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package stats

// ArchiveRecord represents a single archived metrics record.
// This is the unit of data that gets persisted when a key expires
// from the in-memory store.
type ArchiveRecord struct {
	Key      string `json:"key"`      // e.g. "pv:2026-08-18:/home"
	Type     string `json:"type"`     // "counter", "gauge", "set", "hll", "timer"
	Value    any    `json:"value"`    // type-specific value (int64, uint64, TimerSummary, etc.)
	Date     string `json:"date"`     // extracted "YYYY-MM-DD"
	Archived string `json:"archived"` // ISO timestamp of when it was archived
}

// TimerSummary is a summary of a Timer at archive time.
type TimerSummary struct {
	Count int64   `json:"count"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
}

// ArchiveStore is the abstraction for long-term metrics persistence.
// Implementations include SQLite, MySQL, PostgreSQL, etc.
//
// The flow is:
//  1. In-memory collector holds hot data (recent N days).
//  2. When a key expires (TTL), Save is called with the key's final value.
//  3. Historical data can be queried via Query/QueryByType.
//
// All methods must be goroutine-safe.
type ArchiveStore interface {
	// Save persists a single expired key's value.
	// If the key already exists in the archive, it should be upserted
	// (replaced with the new value).
	Save(record ArchiveRecord) error

	// Query retrieves archived records by date range (inclusive).
	// dateFrom and dateTo are "YYYY-MM-DD" format.
	// Returns records ordered by date, then key.
	Query(dateFrom, dateTo string) ([]ArchiveRecord, error)

	// QueryByType retrieves archived records by type and date range.
	QueryByType(entryType, dateFrom, dateTo string) ([]ArchiveRecord, error)

	// Close releases resources held by the store.
	Close() error
}
