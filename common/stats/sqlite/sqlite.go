// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package sqlite provides a SQLite-based implementation of stats.ArchiveStore.
// Expired metrics keys are persisted to SQLite for long-term historical queries.
//
// # Schema
//
//	CREATE TABLE stats_archive (
//	    key       TEXT PRIMARY KEY,
//	    type      TEXT NOT NULL,
//	    value     TEXT NOT NULL,   -- JSON-encoded
//	    date      TEXT,
//	    archived  TEXT NOT NULL
//	);
//
// # Usage
//
//	store, _ := sqlite.New("data/stats.db")
//	defer store.Close()
//	c := memory.New(
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        OnExpire:      memory.ArchiveAdapter(store),
//	    }),
//	)
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	_ "github.com/mattn/go-sqlite3"
)

// Store implements stats.ArchiveStore backed by SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// Compile-time interface check.
var _ stats.ArchiveStore = (*Store)(nil)

// New creates a new SQLite archive store.
// WAL mode is enabled for better concurrent read/write performance.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// NewFromDB creates a store from an existing *sql.DB (for testing or shared connections).
func NewFromDB(db *sql.DB) (*Store, error) {
	if err := createSchema(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
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
	return err
}

// Save implements stats.ArchiveStore.
func (s *Store) Save(record stats.ArchiveRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valueJSON, err := json.Marshal(record.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO stats_archive (key, type, value, date, archived) VALUES (?, ?, ?, ?, ?)`,
		record.Key, record.Type, string(valueJSON), record.Date, record.Archived,
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// Query implements stats.ArchiveStore.
func (s *Store) Query(dateFrom, dateTo string) ([]stats.ArchiveRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, archived FROM stats_archive WHERE date >= ? AND date <= ? ORDER BY date, key`,
		dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	return scanRecords(rows)
}

// QueryByType implements stats.ArchiveStore.
func (s *Store) QueryByType(entryType, dateFrom, dateTo string) ([]stats.ArchiveRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, archived FROM stats_archive WHERE type = ? AND date >= ? AND date <= ? ORDER BY date, key`,
		entryType, dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	return scanRecords(rows)
}

// Close implements stats.ArchiveStore.
func (s *Store) Close() error {
	return s.db.Close()
}

// ──────────────────────────────────────────────
// Convenience query helpers (SQLite-specific, not part of the interface)
// ──────────────────────────────────────────────

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

// ──────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────

func scanRecords(rows *sql.Rows) ([]stats.ArchiveRecord, error) {
	defer rows.Close()

	var records []stats.ArchiveRecord
	for rows.Next() {
		var r stats.ArchiveRecord
		var valueStr string
		if err := rows.Scan(&r.Key, &r.Type, &valueStr, &r.Date, &r.Archived); err != nil {
			return nil, err
		}
		// Value is stored as JSON; keep it as raw for the caller to decode.
		r.Value = json.RawMessage(valueStr)
		records = append(records, r)
	}
	return records, nil
}

// suppress unused import
var _ = time.Now
