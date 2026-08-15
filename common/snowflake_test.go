// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnowflake_NextID(t *testing.T) {
	s, err := NewSnowflake()
	require.NoError(t, err)

	ids := make(map[int64]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := s.NextID()
		if id == 0 {
			t.Fatal("NextID returned 0")
		}
		if ids[id] {
			t.Fatalf("duplicate id %d at iteration %d", id, i)
		}
		ids[id] = true
	}
}

func TestNextSnowflakeUint(t *testing.T) {
	id := NextSnowflakeUint()
	if id == 0 {
		t.Fatal("NextSnowflakeUint returned 0")
	}
	assert.LessOrEqual(t, id, uint(0x7FFFFFFFFFFFFFFF))
}

func TestClampSnowflakeUint(t *testing.T) {
	id := uint(0xFFFFFFFFFFFFFFFF)
	clamped := ClampSnowflakeUint(id)
	assert.Equal(t, uint(0x7FFFFFFFFFFFFFFF), clamped)
}

func TestNewSnowflake_InvalidMachineID(t *testing.T) {
	t.Setenv("MACHINE_ID", "99999")
	_, err := NewSnowflake()
	assert.Error(t, err)
}
