// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package sqlite provides a SQLite-based persistent store for expired
// stats keys. It implements the OnExpire callback interface used by
// memory.WithTTL, so that data is automatically saved to SQLite before
// being evicted from memory.
//
// # Schema
//
//	CREATE TABLE stats_archive (
//	    key       TEXT PRIMARY KEY,  -- e.g. "pv:2026-08-18:/home"
//	    type      TEXT,               -- "counter", "gauge", "set", "hll", "timer"
//	    value     TEXT,               -- JSON-encoded value
//	    date      TEXT,               -- extracted date "YYYY-MM-DD"
//	    archived  TEXT                -- ISO timestamp
//	);
//
// # Usage
//
//	store, _ := sqlite.New("data/stats.db")
//	c := memory.New(
//	    memory.WithReservoirTimer(4096),
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        OnExpire:      store.OnExpire,
//	    }),
//	)
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/common/stats/memory"
	_ "github.com/mattn/go-sqlite3"
)

// Store is a SQLite-backed archive for expired stats keys.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// New creates a new SQLite store. The database file is created if it
// doesn't exist. WAL mode is enabled for better concurrent read/write.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Create schema.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS stats_archive (
			key       TEXT PRIMARY KEY,
			type      TEXT NOT NULL,
			value     TEXT NOT NULL,
			date      TEXT,
			archived  TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_stats_date ON stats_archive(date);
		CREATE INDEX IF NOT EXISTS idx_stats_type ON stats_archive(type);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil
}

// OnExpire is the callback for memory.WithTTL. It saves the expired key's
// value to SQLite before it's removed from memory.
func (s *Store) OnExpire(key string, entry memory.SnapshotEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract date from key.
	date := extractDate(key)

	// Serialize value to JSON.
	valueJSON, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO stats_archive (key, type, value, date, archived) VALUES (?, ?, ?, ?, ?)`,
		key, entry.Type, string(valueJSON), date, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// ArchiveRecord represents a row in the stats_archive table.
type ArchiveRecord struct {
	Key      string
	Type     string
	Value    string
	Date     string
	Archived string
}

// Query retrieves archived records by date range.
// dateFrom and dateTo are inclusive, format "YYYY-MM-DD".
func (s *Store) Query(dateFrom, dateTo string) ([]ArchiveRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, archived FROM stats_archive WHERE date >= ? AND date <= ? ORDER BY date, key`,
		dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ArchiveRecord
	for rows.Next() {
		var r ArchiveRecord
		if err := rows.Scan(&r.Key, &r.Type, &r.Value, &r.Date, &r.Archived); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// QueryByType retrieves archived records by type and date range.
func (s *Store) QueryByType(entryType, dateFrom, dateTo string) ([]ArchiveRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, archived FROM stats_archive WHERE type = ? AND date >= ? AND date <= ? ORDER BY date, key`,
		entryType, dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ArchiveRecord
	for rows.Next() {
		var r ArchiveRecord
		if err := rows.Scan(&r.Key, &r.Type, &r.Value, &r.Date, &r.Archived); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// GetPV returns the total PV for a date (sum across all paths).
func (s *Store) GetPV(date string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var total int64
	rows, err := s.db.Query(
		`SELECT value FROM stats_archive WHERE type = 'counter' AND key LIKE ?`,
		"pv:"+date+"%",
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return 0, err
		}
		var val int64
		if err := json.Unmarshal([]byte(valStr), &val); err == nil {
			total += val
		}
	}
	return total, nil
}

// GetUV returns the estimated UV for a date from the archive.
func (s *Store) GetUV(date string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valStr string
	err := s.db.QueryRow(
		`SELECT value FROM stats_archive WHERE type = 'hll' AND key = ?`,
		"uv:"+date,
	).Scan(&valStr)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var val uint64
	json.Unmarshal([]byte(valStr), &val)
	return val, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// extractDate extracts "YYYY-MM-DD" from a key string.
func extractDate(key string) string {
	for i := 0; i <= len(key)-10; i++ {
		if isDigit(key[i]) && isDigit(key[i+1]) && isDigit(key[i+2]) && isDigit(key[i+3]) &&
			key[i+4] == '-' &&
			isDigit(key[i+5]) && isDigit(key[i+6]) &&
			key[i+7] == '-' &&
			isDigit(key[i+8]) && isDigit(key[i+9]) {
			return key[i : i+10]
		}
	}
	return ""
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// suppress unused import warning
var _ = strings.Contains
