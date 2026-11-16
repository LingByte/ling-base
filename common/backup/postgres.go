// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"fmt"
	"strconv"
)

// PostgresConfig configures pg_dump / psql for logical backup/restore.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string // required for dump/load of a single DB
	// DumpBinary defaults to "pg_dump".
	DumpBinary string
	// ClientBinary defaults to "psql".
	ClientBinary string
	ExtraArgs    []string
}

func (c PostgresConfig) port() int {
	if c.Port <= 0 {
		return 5432
	}
	return c.Port
}

func (c PostgresConfig) dumpBinary() string {
	if c.DumpBinary != "" {
		return c.DumpBinary
	}
	return "pg_dump"
}

func (c PostgresConfig) clientBinary() string {
	if c.ClientBinary != "" {
		return c.ClientBinary
	}
	return "psql"
}

func (c PostgresConfig) baseArgs() []string {
	args := []string{
		"-h", c.Host,
		"-p", strconv.Itoa(c.port()),
	}
	if c.User != "" {
		args = append(args, "-U", c.User)
	}
	return args
}

func (c PostgresConfig) env() []string {
	if c.Password == "" {
		return nil
	}
	return []string{"PGPASSWORD=" + c.Password}
}

// PostgresDumpArgs returns argv for [PostgresDump].
func PostgresDumpArgs(c PostgresConfig) []string {
	args := c.baseArgs()
	args = append(args, "--no-owner", "--no-acl")
	args = append(args, c.ExtraArgs...)
	if c.Database != "" {
		args = append(args, "-d", c.Database)
	}
	return args
}

// PostgresDump returns a [CommandSource] running pg_dump (plain SQL to stdout).
func PostgresDump(c PostgresConfig) *CommandSource {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSource{
		Name: c.dumpBinary(),
		Args: PostgresDumpArgs(c),
		Env:  c.env(),
	}
}

// PostgresLoadArgs returns argv for [PostgresLoad].
func PostgresLoadArgs(c PostgresConfig) []string {
	args := c.baseArgs()
	args = append(args, c.ExtraArgs...)
	if c.Database != "" {
		args = append(args, "-d", c.Database)
	}
	return args
}

// PostgresLoad returns a [CommandSink] that pipes SQL into psql.
func PostgresLoad(c PostgresConfig) *CommandSink {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSink{
		Name: c.clientBinary(),
		Args: PostgresLoadArgs(c),
		Env:  c.env(),
	}
}

// ValidatePostgresConfig checks required fields.
func ValidatePostgresConfig(c PostgresConfig) error {
	if c.Database == "" {
		return fmt.Errorf("backup: postgres database is required")
	}
	return nil
}
