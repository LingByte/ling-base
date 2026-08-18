// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package memory

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/common/stats"
)

// ArchiveAdapter wraps a stats.ArchiveStore and provides an OnExpire
// callback function compatible with TTLConfig.OnExpire.
//
// This bridges the in-memory TTL expiration to any ArchiveStore
// implementation (SQLite, MySQL, PostgreSQL, etc.).
//
// Usage:
//
//	store, _ := sqlite.New("data/stats.db")     // or mysql.New(dsn)
//	c := memory.New(
//	    memory.WithTTL(memory.TTLConfig{
//	        RetentionDays: 7,
//	        OnExpire:      memory.ArchiveAdapter(store),
//	    }),
//	)
func ArchiveAdapter(store stats.ArchiveStore) func(key string, entry SnapshotEntry) error {
	return func(key string, entry SnapshotEntry) error {
		record := stats.ArchiveRecord{
			Key:      key,
			Type:     entry.Type,
			Value:    entry.Value,
			Date:     extractDateFromKey(key),
			Archived: time.Now().Format(time.RFC3339),
		}
		return store.Save(record)
	}
}

// extractDateFromKey extracts "YYYY-MM-DD" from a key string.
func extractDateFromKey(key string) string {
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

// suppress unused import
var _ = fmt.Sprintf
