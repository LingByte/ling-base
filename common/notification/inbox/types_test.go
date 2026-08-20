// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package inbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidFilter(t *testing.T) {
	assert.True(t, IsValidFilter(FilterAll))
	assert.True(t, IsValidFilter(FilterUnread))
	assert.True(t, IsValidFilter(FilterRead))
	assert.False(t, IsValidFilter(""))
	assert.False(t, IsValidFilter("deleted"))
	assert.False(t, IsValidFilter("ALL"))
}
