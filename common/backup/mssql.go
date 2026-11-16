// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import "fmt"

// MSSQLConfig configures sqlcmd for logical export against SQL Server /
// Azure SQL / RDS SQL Server (point Host at the cloud endpoint).
type MSSQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	// Binary defaults to "sqlcmd".
	Binary string
	// Query overrides the default query. Prefer an explicit BACKUP DATABASE
	// … TO URL / disk statement for production Azure/RDS workflows.
	Query     string
	ExtraArgs []string
}

func (c MSSQLConfig) port() int {
	if c.Port <= 0 {
		return 1433
	}
	return c.Port
}

func (c MSSQLConfig) binary() string {
	if c.Binary != "" {
		return c.Binary
	}
	return "sqlcmd"
}

// MSSQLDumpArgs returns argv for [MSSQLDump].
func MSSQLDumpArgs(c MSSQLConfig) []string {
	server := c.Host
	if c.Port > 0 {
		server = fmt.Sprintf("%s,%d", c.Host, c.Port)
	} else if c.port() != 1433 {
		server = fmt.Sprintf("%s,%d", c.Host, c.port())
	}
	args := []string{"-S", server}
	if c.User != "" {
		args = append(args, "-U", c.User)
	}
	if c.Password != "" {
		args = append(args, "-P", c.Password)
	}
	if c.Database != "" {
		args = append(args, "-d", c.Database)
	}
	args = append(args, c.ExtraArgs...)
	q := c.Query
	if q == "" && c.Database != "" {
		q = "SELECT name FROM sys.tables"
	}
	if q != "" {
		args = append(args, "-Q", q, "-y", "0", "-Y", "0")
	}
	return args
}

// MSSQLDump returns a [CommandSource] wrapping sqlcmd.
func MSSQLDump(c MSSQLConfig) *CommandSource {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSource{
		Name: c.binary(),
		Args: MSSQLDumpArgs(c),
	}
}

// ValidateMSSQLConfig checks required fields.
func ValidateMSSQLConfig(c MSSQLConfig) error {
	if c.Host == "" {
		return fmt.Errorf("backup: mssql host is required")
	}
	if c.Database == "" && c.Query == "" {
		return fmt.Errorf("backup: mssql database or query is required")
	}
	return nil
}
