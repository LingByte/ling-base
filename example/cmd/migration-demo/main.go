// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Command migration-demo demonstrates the ling-base migration framework
// integrated with bootstrap.
//
// It shows:
//   - Versioned SQL migrations with embedded .sql files
//   - Up / Down / Steps / Status operations
//   - Integration with bootstrap's init phase
//   - AutoMigrate as an alternative for dev/test
//
// Usage:
//
//	go run ./cmd/migration-demo
package main

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/LingByte/ling-base/bootstrap"
	"github.com/LingByte/ling-base/common/migration"
	gormmigrator "github.com/LingByte/ling-base/common/migration/gormmigrator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Product is a GORM model for AutoMigrate demo.
type Product struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"not null"`
	Price float64
}

func main() {
	fmt.Println("=== Migration Demo ===")

	// ── Part 1: Versioned SQL migrations ──
	fmt.Println("\n--- Part 1: Versioned SQL migrations ---")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("Error opening DB: %v\n", err)
		return
	}

	// Create migration source from embedded SQL files.
	src := migration.NewEmbedSource(migrationFS, "migrations")

	// Create the migrator.
	migrator, err := gormmigrator.New(db, src)
	if err != nil {
		fmt.Printf("Error creating migrator: %v\n", err)
		return
	}

	// Show initial status.
	status, _ := migrator.Status(context.Background())
	fmt.Println("\n  Initial status:")
	for _, s := range status {
		fmt.Printf("    v%d %s — applied: %v\n", s.Version, s.Description, s.Applied)
	}

	// Apply all migrations.
	fmt.Println("\n  Running Up()...")
	if err := migrator.Up(context.Background()); err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	// Show status after Up.
	status, _ = migrator.Status(context.Background())
	fmt.Println("\n  Status after Up():")
	for _, s := range status {
		appliedAt := "—"
		if s.AppliedAt != "" {
			appliedAt = s.AppliedAt
		}
		fmt.Printf("    v%d %s — applied: %v, at: %s\n", s.Version, s.Description, s.Applied, appliedAt)
	}

	ver, _, _ := migrator.Version(context.Background())
	fmt.Printf("\n  Current version: %d\n", ver)

	// Verify tables exist.
	fmt.Printf("  Table 'products' exists: %v\n", db.Migrator().HasTable("products"))
	fmt.Printf("  Table 'orders' exists: %v\n", db.Migrator().HasTable("orders"))

	// Rollback one step.
	fmt.Println("\n  Running Steps(-1)...")
	_ = migrator.Steps(context.Background(), -1)
	ver, _, _ = migrator.Version(context.Background())
	fmt.Printf("  Current version after rollback: %d\n", ver)
	fmt.Printf("  Table 'orders' exists: %v\n", db.Migrator().HasTable("orders"))

	// Re-apply.
	fmt.Println("\n  Running Steps(1)...")
	_ = migrator.Steps(context.Background(), 1)
	ver, _, _ = migrator.Version(context.Background())
	fmt.Printf("  Current version after re-apply: %d\n", ver)

	// Full rollback.
	fmt.Println("\n  Running Down()...")
	_ = migrator.Down(context.Background())
	ver, applied, _ := migrator.Version(context.Background())
	fmt.Printf("  Current version after Down(): %d (applied: %v)\n", ver, applied)

	// ── Part 2: Bootstrap integration ──
	fmt.Println("\n--- Part 2: Bootstrap integration ---")

	// Re-create migrator for bootstrap demo.
	db2, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	migrator2, _ := gormmigrator.New(db2, src)

	app := bootstrap.New("migration-demo",
		bootstrap.WithProfile(bootstrap.ProfileProd),
		bootstrap.WithMigration(migrator2),
		bootstrap.WithShutdownTimeout(1*time.Second),
	)

	fmt.Println("\n  Starting app with WithMigration()...")
	errCh := app.RunAsync()
	if err := <-errCh; err != nil {
		fmt.Printf("  Startup error: %v\n", err)
		return
	}

	ver, _, _ = migrator2.Version(context.Background())
	fmt.Printf("  Migrations applied during init: version %d\n", ver)
	fmt.Printf("  Table 'products' exists: %v\n", db2.Migrator().HasTable("products"))

	app.Stop()

	// ── Part 3: AutoMigrate (dev/test style) ──
	fmt.Println("\n--- Part 3: AutoMigrate (dev/test style) ---")

	db3, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	app2 := bootstrap.New("migration-demo-dev",
		bootstrap.WithProfile(bootstrap.ProfileDev),
		bootstrap.WithAutoMigrate(db3, &Product{}),
		bootstrap.WithShutdownTimeout(1*time.Second),
	)

	fmt.Println("\n  Starting app with WithAutoMigrate()...")
	errCh2 := app2.RunAsync()
	if err := <-errCh2; err != nil {
		fmt.Printf("  Startup error: %v\n", err)
		return
	}

	fmt.Printf("  Table 'products' exists: %v\n", db3.Migrator().HasTable("products"))

	app2.Stop()

	fmt.Println("\n=== Demo complete ===")
}
