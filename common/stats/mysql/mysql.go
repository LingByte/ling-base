// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package mysql provides a MySQL-based persistence helper for expired
// stats keys. The Save method is a stats.ExpireFunc — pass it directly
// to memory.WithTTL without any adapter.
//
// # Schema
//
//	CREATE TABLE stats_archive (
//	    key       VARCHAR(255) PRIMARY KEY,
//	    type      VARCHAR(32)  NOT NULL,
//	    value     JSON         NOT NULL,
//	    date      VARCHAR(10),
//	    expired   VARCHAR(40)  NOT NULL,
//	    INDEX idx_stats_date (date),
//	    INDEX idx_stats_type (type)
//	);
//
// # Usage
//
//	store, _ := mysql.New("user:pass@tcp(127.0.0.1:3306)/stats?charset=utf8mb4")
//	defer store.Close()
//	c := memory.New(
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        OnExpire:      store.Save,   // ← directly, no adapter
//	    }),
//	)
package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LingByte/ling-base/common/stats"
	_ "github.com/go-sql-driver/mysql"
)

// Store is a MySQL-backed persistence helper for expired stats keys.
// Save implements stats.ExpireFunc, so it can be passed directly to
// memory.TTLConfig.OnExpire.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// New creates a new MySQL store.
// dsn example: "user:pass@tcp(127.0.0.1:3306)/stats?charset=utf8mb4&parseTime=true"
func New(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// NewFromDB creates a store from an existing *sql.DB.
func NewFromDB(db *sql.DB) (*Store, error) {
	if err := createSchema(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS stats_archive (
			key       VARCHAR(255) PRIMARY KEY,
			type      VARCHAR(32)  NOT NULL,
			value     JSON         NOT NULL,
			date      VARCHAR(10),
			expired   VARCHAR(40)  NOT NULL,
			INDEX idx_stats_date (date),
			INDEX idx_stats_type (type)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	return err
}

// Save persists an expired key to MySQL. It implements stats.ExpireFunc,
// so it can be passed directly as memory.TTLConfig.OnExpire.
// Uses INSERT ... ON DUPLICATE KEY UPDATE for upsert.
func (s *Store) Save(ek stats.ExpiredKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valueJSON, err := json.Marshal(ek.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO stats_archive (key, type, value, date, expired) VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE type=VALUES(type), value=VALUES(value), date=VALUES(date), expired=VALUES(expired)`,
		ek.Key, ek.Type, string(valueJSON), ek.Date, ek.ExpiredAt,
	)
	return err
}

// Query retrieves archived records by date range (inclusive).
func (s *Store) Query(dateFrom, dateTo string) ([]ArchiveRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, expired FROM stats_archive WHERE date >= ? AND date <= ? ORDER BY date, key`,
		dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

// QueryByType retrieves archived records by type and date range.
func (s *Store) QueryByType(typ, dateFrom, dateTo string) ([]ArchiveRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT key, type, value, date, expired FROM stats_archive WHERE type = ? AND date >= ? AND date <= ? ORDER BY date, key`,
		typ, dateFrom, dateTo,
	)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ArchiveRow is a row in the stats_archive table.
type ArchiveRow struct {
	Key     string
	Type    string
	Value   json.RawMessage
	Date    string
	Expired string
}

func scanRows(rows *sql.Rows) ([]ArchiveRow, error) {
	defer rows.Close()

	var out []ArchiveRow
	for rows.Next() {
		var r ArchiveRow
		var valStr string
		if err := rows.Scan(&r.Key, &r.Type, &valStr, &r.Date, &r.Expired); err != nil {
			return nil, err
		}
		r.Value = json.RawMessage(valStr)
		out = append(out, r)
	}
	return out, nil
}
