// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"fmt"
	"strconv"
)

// MySQLConfig configures mysqldump / mysql CLI for logical backup/restore.
// Works for self-hosted and cloud MySQL (RDS, PolarDB, Cloud SQL, …) whenever
// the host is reachable with the given credentials.
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string // empty = all databases (dump uses --all-databases)

	// DumpBinary overrides the dump executable (default "mysqldump").
	DumpBinary string
	// ClientBinary overrides the restore client (default "mysql").
	ClientBinary string
	// ExtraArgs are appended after the standard connection flags.
	ExtraArgs []string
}

func (c MySQLConfig) port() int {
	if c.Port <= 0 {
		return 3306
	}
	return c.Port
}

func (c MySQLConfig) dumpBinary() string {
	if c.DumpBinary != "" {
		return c.DumpBinary
	}
	return "mysqldump"
}

func (c MySQLConfig) clientBinary() string {
	if c.ClientBinary != "" {
		return c.ClientBinary
	}
	return "mysql"
}

func (c MySQLConfig) baseArgs() []string {
	args := []string{
		"-h", c.Host,
		"-P", strconv.Itoa(c.port()),
	}
	if c.User != "" {
		args = append(args, "-u", c.User)
	}
	return args
}

func (c MySQLConfig) env() []string {
	if c.Password == "" {
		return nil
	}
	return []string{"MYSQL_PWD=" + c.Password}
}

// MySQLDumpArgs returns the argv that [MySQLDump] would use (for tests).
func MySQLDumpArgs(c MySQLConfig) []string {
	args := append(c.baseArgs(),
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
	)
	args = append(args, c.ExtraArgs...)
	if c.Database == "" {
		args = append(args, "--all-databases")
	} else {
		args = append(args, c.Database)
	}
	return args
}

// MySQLDump returns a [CommandSource] that runs mysqldump.
func MySQLDump(c MySQLConfig) *CommandSource {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSource{
		Name: c.dumpBinary(),
		Args: MySQLDumpArgs(c),
		Env:  c.env(),
	}
}

// MySQLLoadArgs returns the argv that [MySQLLoad] would use (for tests).
func MySQLLoadArgs(c MySQLConfig) []string {
	args := c.baseArgs()
	args = append(args, c.ExtraArgs...)
	if c.Database != "" {
		args = append(args, c.Database)
	}
	return args
}

// MySQLLoad returns a [CommandSink] that pipes SQL into the mysql client.
func MySQLLoad(c MySQLConfig) *CommandSink {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSink{
		Name: c.clientBinary(),
		Args: MySQLLoadArgs(c),
		Env:  c.env(),
	}
}

// ValidateMySQLConfig checks required fields for dump/load helpers.
func ValidateMySQLConfig(c MySQLConfig) error {
	if c.User == "" {
		return fmt.Errorf("backup: mysql user is required")
	}
	return nil
}
