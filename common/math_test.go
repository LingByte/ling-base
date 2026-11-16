// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampMinMaxRoundGCD(t *testing.T) {
	assert.Equal(t, 10, Clamp(15, 0, 10))
	assert.Equal(t, 0, Clamp(-1, 0, 10))
	assert.Equal(t, 3, Min(3, 5))
	assert.Equal(t, 5, Max(3, 5))
	assert.InDelta(t, 3.14, RoundFloat(3.14159, 2), 1e-9)
	assert.Equal(t, 6, GCD(12, 18))
}
