// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package im

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──────────────────────────────────────────────
// NormalizeProvider
// ──────────────────────────────────────────────

func TestNormalizeProvider(t *testing.T) {
	cases := []struct{ in, want string }{
		{"wecom", "wecom"},
		{"WECOM", "wecom"},
		{"WeCom", "wecom"},
		{" feishu ", "feishu"},
		{"FEISHU", "feishu"},
		{"unknown", "unknown"},
		{"  ", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, NormalizeProvider(c.in), "input %q", c.in)
	}
}
