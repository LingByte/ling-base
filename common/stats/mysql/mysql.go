// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package mysql provides a MySQL-based implementation of stats.ArchiveStore.
// Expired metrics keys are persisted to MySQL for long-term historical queries.
//
// # Schema
//
//	CREATE TABLE stats_archive (
//	    key       VARCHAR(255) PRIMARY KEY,
//	    type      VARCHAR(32)  NOT NULL,
//	    value     JSON         NOT NULL,
//	    date      VARCHAR(10),
//	    archived  VARCHAR(40)  NOT NULL,
//	    INDEX idx_stats_date (date),
//	    INDEX idx_stats_type (type)
//	);
//
// # Usage
//
//	store, _ := mysql.New("user:pass@tcp(127.0.0.1:3306)/stats?charset=utf8mb4&parseTime=true")
//	defer store.Close()
//	c := memory.New(
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        OnExpire:      memory.ArchiveAdapter(store),
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

// Store implements stats.ArchiveStore backed by MySQL.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// Compile-time interface check.
var _ stats.ArchiveStore = (*Store)(nil)

// New creates a new MySQL archive store.
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
			archived  VARCHAR(40)  NOT NULL,
			INDEX idx_stats_date (date),
			INDEX idx_stats_type (type)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	return err
}

// Save implements stats.ArchiveStore.
// Uses INSERT ... ON DUPLICATE KEY UPDATE for upsert.
func (s *Store) Save(record stats.ArchiveRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valueJSON, err := json.Marshal(record.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO stats_archive (key, type, value, date, archived) VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE type=VALUES(type), value=VALUES(value), date=VALUES(date), archived=VALUES(archived)`,
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

func scanRecords(rows *sql.Rows) ([]stats.ArchiveRecord, error) {
	defer rows.Close()

	var records []stats.ArchiveRecord
	for rows.Next() {
		var r stats.ArchiveRecord
		var valueStr string
		if err := rows.Scan(&r.Key, &r.Type, &valueStr, &r.Date, &r.Archived); err != nil {
			return nil, err
		}
		r.Value = json.RawMessage(valueStr)
		records = append(records, r)
	}
	return records, nil
}
