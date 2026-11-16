// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package backup

import (
	"fmt"
	"strconv"
)

// RedisConfig configures redis-cli RDB dump via `--rdb -` (stdout).
// Works against self-hosted Redis and cloud Redis when the endpoint allows
// the SYNC/PSYNC family used by --rdb (ACL permitting).
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int // ignored by --rdb (full instance dump); kept for docs
	// Binary defaults to "redis-cli".
	Binary    string
	ExtraArgs []string
}

func (c RedisConfig) port() int {
	if c.Port <= 0 {
		return 6379
	}
	return c.Port
}

func (c RedisConfig) binary() string {
	if c.Binary != "" {
		return c.Binary
	}
	return "redis-cli"
}

// RedisDumpArgs returns argv for [RedisDump].
func RedisDumpArgs(c RedisConfig) []string {
	args := []string{"-h", c.Host, "-p", strconv.Itoa(c.port())}
	if c.Password != "" {
		args = append(args, "-a", c.Password)
	}
	args = append(args, c.ExtraArgs...)
	args = append(args, "--rdb", "-")
	return args
}

// RedisDump returns a [CommandSource] that streams an RDB snapshot to stdout.
func RedisDump(c RedisConfig) *CommandSource {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	return &CommandSource{
		Name: c.binary(),
		Args: RedisDumpArgs(c),
	}
}

// ValidateRedisConfig checks required fields.
func ValidateRedisConfig(c RedisConfig) error {
	if c.Host == "" {
		return fmt.Errorf("backup: redis host is required")
	}
	return nil
}
