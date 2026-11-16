// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import "fmt"

// SQLiteConfig configures sqlite3 CLI logical dump/restore.
type SQLiteConfig struct {
	// Path is the database file path.
	Path string
	// Binary defaults to "sqlite3".
	Binary    string
	ExtraArgs []string
}

func (c SQLiteConfig) binary() string {
	if c.Binary != "" {
		return c.Binary
	}
	return "sqlite3"
}

// SQLiteDumpArgs returns argv for [SQLiteDump] (without the SQL command).
func SQLiteDumpArgs(c SQLiteConfig) []string {
	args := append([]string{}, c.ExtraArgs...)
	args = append(args, c.Path, ".dump")
	return args
}

// SQLiteDump returns a [CommandSource] that runs `sqlite3 db .dump`.
func SQLiteDump(c SQLiteConfig) *CommandSource {
	return &CommandSource{
		Name: c.binary(),
		Args: SQLiteDumpArgs(c),
	}
}

// SQLiteLoad returns a [CommandSink] that pipes SQL into `sqlite3 db`.
func SQLiteLoad(c SQLiteConfig) *CommandSink {
	args := append([]string{}, c.ExtraArgs...)
	args = append(args, c.Path)
	return &CommandSink{
		Name: c.binary(),
		Args: args,
	}
}

// ValidateSQLiteConfig checks required fields.
func ValidateSQLiteConfig(c SQLiteConfig) error {
	if c.Path == "" {
		return fmt.Errorf("backup: sqlite path is required")
	}
	return nil
}
