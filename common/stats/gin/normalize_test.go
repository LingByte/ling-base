// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Static paths — unchanged
		{"/", "/"},
		{"", ""},
		{"/health", "/health"},
		{"/api/v1/users", "/api/v1/users"},

		// Numeric IDs → :id
		{"/users/123", "/users/:id"},
		{"/users/99999", "/users/:id"},
		{"/api/v1/posts/789", "/api/v1/posts/:id"},
		{"/api/v1/posts/789/edit", "/api/v1/posts/:id/edit"},

		// UUIDs → :id
		{"/files/550e8400-e29b-41d4-a716-446655440000", "/files/:id"},
		{"/items/6ba7b810-9dad-11d1-80b4-00c04fd430c8", "/items/:id"},

		// Long alphanumeric IDs → :id
		{"/posts/a1b2c3d4e5f6g7h8i9j0", "/posts/:id"},
		{"/links/abc123def456ghi789jkl012mno345pqr890stu123vwx456", "/links/:id"},

		// File hashes → name.ext
		{"/static/css/main.a1b2c3d4.css", "/static/css/main.css"},
		{"/static/js/app.e5f6g7h8.js", "/static/js/app.js"},

		// Short alphanumeric — unchanged
		{"/tags/go", "/tags/go"},
		{"/category/news", "/category/news"},

		// Query string stripped
		{"/search?q=hello&page=1", "/search"},

		// Mixed
		{"/api/v1/users/123/posts/456", "/api/v1/users/:id/posts/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePath(tt.input)
			assert.Equal(t, tt.expected, got, "NormalizePath(%q)", tt.input)
		})
	}
}

func TestNormalizePathKeyExplosion(t *testing.T) {
	// Simulate 10000 different user IDs accessing the same route pattern.
	// All should normalize to the same key.
	uniqueNormalized := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		paths := []string{
			"/users/" + itoa(i),
			"/api/v1/posts/" + itoa(i) + "/comments",
			"/files/" + uuid(i),
		}
		for _, p := range paths {
			normalized := NormalizePath(p)
			uniqueNormalized[normalized] = true
		}
	}

	// Should produce only 3 unique normalized paths, not 30000.
	assert.Equal(t, 3, len(uniqueNormalized),
		"expected 3 unique normalized paths, got %d — key explosion!", len(uniqueNormalized))

	// Verify the expected patterns.
	assert.True(t, uniqueNormalized["/users/:id"])
	assert.True(t, uniqueNormalized["/api/v1/posts/:id/comments"])
	assert.True(t, uniqueNormalized["/files/:id"])
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func uuid(seed int) string {
	// Generate a deterministic UUID-like string from seed.
	hex := "0123456789abcdef"
	var buf [36]byte
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			buf[i] = '-'
		} else {
			buf[i] = hex[(seed+i)%16]
		}
	}
	return string(buf[:])
}
