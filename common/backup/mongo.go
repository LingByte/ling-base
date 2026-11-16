// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import "fmt"

// MongoConfig configures mongodump / mongorestore with --archive on stdout/stdin.
type MongoConfig struct {
	URI string // e.g. mongodb://user:pass@host:27017
	// Database limits dump/restore to one DB when non-empty.
	Database string
	// DumpBinary defaults to "mongodump".
	DumpBinary string
	// RestoreBinary defaults to "mongorestore".
	RestoreBinary string
	ExtraArgs     []string
}

func (c MongoConfig) dumpBinary() string {
	if c.DumpBinary != "" {
		return c.DumpBinary
	}
	return "mongodump"
}

func (c MongoConfig) restoreBinary() string {
	if c.RestoreBinary != "" {
		return c.RestoreBinary
	}
	return "mongorestore"
}

// MongoDumpArgs returns argv for [MongoDump].
func MongoDumpArgs(c MongoConfig) []string {
	args := []string{"--archive"}
	if c.URI != "" {
		args = append(args, "--uri="+c.URI)
	}
	if c.Database != "" {
		args = append(args, "--db="+c.Database)
	}
	args = append(args, c.ExtraArgs...)
	return args
}

// MongoDump returns a [CommandSource] that runs mongodump --archive.
func MongoDump(c MongoConfig) *CommandSource {
	return &CommandSource{
		Name: c.dumpBinary(),
		Args: MongoDumpArgs(c),
	}
}

// MongoRestoreArgs returns argv for [MongoRestore].
func MongoRestoreArgs(c MongoConfig) []string {
	args := []string{"--archive"}
	if c.URI != "" {
		args = append(args, "--uri="+c.URI)
	}
	if c.Database != "" {
		args = append(args, "--db="+c.Database)
	}
	args = append(args, c.ExtraArgs...)
	return args
}

// MongoRestore returns a [CommandSink] that runs mongorestore --archive.
func MongoRestore(c MongoConfig) *CommandSink {
	return &CommandSink{
		Name: c.restoreBinary(),
		Args: MongoRestoreArgs(c),
	}
}

// ValidateMongoConfig checks required fields.
func ValidateMongoConfig(c MongoConfig) error {
	if c.URI == "" {
		return fmt.Errorf("backup: mongo URI is required")
	}
	return nil
}
