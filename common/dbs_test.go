// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"os"
	"testing"
)

func TestConfigureConnectionPool(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}

	ConfigureConnectionPool(db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_WithCustomWriter(t *testing.T) {
	customWriter, err := os.CreateTemp("", "test_db_log_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(customWriter.Name())
	defer customWriter.Close()

	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(customWriter, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase with custom writer error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_WithNilWriter(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase with nil writer error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_InvalidDSN(t *testing.T) {
	_, err := InitDatabase(nil, "sqlite", "invalid://dsn")
	if err == nil {
		t.Fatalf("InitDatabase expected error for invalid DSN")
	}
}

func TestMakeMigrates(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	type TestModel struct {
		ID   uint `gorm:"primarykey"`
		Name string
	}

	err = MakeMigrates(db, []any{&TestModel{}})
	if err != nil {
		t.Fatalf("MakeMigrates error: %v", err)
	}

	var count int64
	db.Model(&TestModel{}).Count(&count)
}

func TestMakeMigrates_EmptyInstances(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = MakeMigrates(db, []any{})
	if err != nil {
		t.Fatalf("MakeMigrates with empty instances error: %v", err)
	}
}

func TestMakeMigrates_InvalidModel(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = MakeMigrates(db, []any{"not a struct"})
	if err == nil {
		t.Fatalf("MakeMigrates expected error for invalid model")
	}
}

func TestInitDatabase_DefaultDriverAndDSN(t *testing.T) {
	originalDriver := os.Getenv("DB_DRIVER")
	originalDSN := os.Getenv("DSN")
	defer func() {
		if originalDriver != "" {
			os.Setenv("DB_DRIVER", originalDriver)
		} else {
			os.Unsetenv("DB_DRIVER")
		}
		if originalDSN != "" {
			os.Setenv("DSN", originalDSN)
		} else {
			os.Unsetenv("DSN")
		}
	}()

	os.Unsetenv("DB_DRIVER")
	os.Unsetenv("DSN")

	_, err := InitDatabase(nil, "", "")
	_ = err
}

func TestConfigureConnectionPool_InvalidDB(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	ConfigureConnectionPool(db)
}
