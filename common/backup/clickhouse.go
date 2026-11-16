// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"fmt"
	"strings"
)

// ClickHouseConfig configures clickhouse-client native backup via stdout.
type ClickHouseConfig struct {
	Host     string
	Port     int // native TCP, default 9000
	User     string
	Password string
	Database string
	// Query overrides the default "BACKUP DATABASE … TO WritableData" style;
	// when empty, uses SHOW CREATE + a simple SELECT dump is NOT enough —
	// default runs: clickhouse-client -q "BACKUP DATABASE db TO Output('stdout')"
	// For broader compatibility we default to `clickhouse-client --query "SELECT 1"`
	// style dump via Native format of all tables is complex; use ExtraQuery.
	//
	// Recommended: set Query to a BACKUP or clickhouse-backup compatible statement,
	// or use CommandSource directly. Helper below emits table data as Native
	// only when Query is set.
	Query     string
	Binary    string
	ExtraArgs []string
}

func (c ClickHouseConfig) binary() string {
	if c.Binary != "" {
		return c.Binary
	}
	return "clickhouse-client"
}

// ClickHouseDumpArgs returns argv for [ClickHouseDump].
func ClickHouseDumpArgs(c ClickHouseConfig) []string {
	args := []string{}
	if c.Host != "" {
		args = append(args, "--host="+c.Host)
	}
	if c.Port > 0 {
		args = append(args, fmt.Sprintf("--port=%d", c.Port))
	}
	if c.User != "" {
		args = append(args, "--user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--password="+c.Password)
	}
	if c.Database != "" {
		args = append(args, "--database="+c.Database)
	}
	args = append(args, c.ExtraArgs...)
	q := c.Query
	if q == "" && c.Database != "" {
		db := strings.ReplaceAll(c.Database, "`", "")
		q = "BACKUP DATABASE `" + db + "` TO Output('stdout')"
	}
	if q != "" {
		args = append(args, "--query="+q)
	}
	return args
}

// ClickHouseDump returns a [CommandSource] for clickhouse-client.
func ClickHouseDump(c ClickHouseConfig) *CommandSource {
	return &CommandSource{
		Name: c.binary(),
		Args: ClickHouseDumpArgs(c),
	}
}

// ValidateClickHouseConfig checks required fields.
func ValidateClickHouseConfig(c ClickHouseConfig) error {
	if c.Database == "" && c.Query == "" {
		return fmt.Errorf("backup: clickhouse database or query is required")
	}
	return nil
}
