// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package slug

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlug_Chinese(t *testing.T) {
	got := Slug("你好世界")
	assert.Equal(t, "ni-hao-shi-jie", got)
}

func TestSlug_English(t *testing.T) {
	assert.Equal(t, "hello-world", Slug("Hello World!"))
	assert.Equal(t, "hello-world", Slug("Hello, World!"))
	assert.Equal(t, "foo-bar-baz", Slug("foo bar baz"))
}

func TestSlug_Mixed(t *testing.T) {
	got := Slug("Hello世界")
	assert.Equal(t, "hello-shi-jie", got)
}

func TestSlug_Empty(t *testing.T) {
	assert.Equal(t, "", Slug(""))
	assert.Equal(t, "", Slug("   "))
	assert.Equal(t, "", Slug("!@#$%"))
}

func TestSlug_SpecialChars(t *testing.T) {
	assert.Equal(t, "a-b-c", Slug("a.b.c"))
	assert.Equal(t, "a-b-c", Slug("a_b_c"))
	assert.Equal(t, "foo-123-bar", Slug("foo 123 bar"))
	assert.Equal(t, "foo123bar", Slug("foo123bar"))
}

func TestSlugWithSeparator(t *testing.T) {
	assert.Equal(t, "hello_world", SlugWithSeparator("Hello World!", "_"))
	assert.Equal(t, "ni_hao", SlugWithSeparator("你好", "_"))
	assert.Equal(t, "a.b", SlugWithSeparator("a b", "."))
}

func TestSlugLower(t *testing.T) {
	assert.Equal(t, "mixedcase", SlugLower("MixedCase"))
	assert.Equal(t, "hello-world", SlugLower("Hello World!"))
}

func TestSlugUnique(t *testing.T) {
	got := SlugUnique("title")
	require.NotEmpty(t, got)
	assert.True(t, strings.HasPrefix(got, "title-"))
	// Suffix is 8 hex chars.
	assert.Len(t, got, len("title-")+8)

	// Two calls produce different results.
	got2 := SlugUnique("title")
	assert.NotEqual(t, got, got2)
}

func TestSlugUnique_Empty(t *testing.T) {
	got := SlugUnique("")
	assert.Len(t, got, 8)
}

func TestTruncateSlug(t *testing.T) {
	assert.Equal(t, "one-two", TruncateSlug("one-two-three", 8))
	assert.Equal(t, "one-two-three", TruncateSlug("one-two-three", 20))
	assert.Equal(t, "one-two-three", TruncateSlug("one-two-three", 13))
	assert.Equal(t, "", TruncateSlug("one-two-three", 2))
}

func TestTruncateSlug_NoSeparator(t *testing.T) {
	// Truncating in the middle of a single long token yields empty.
	assert.Equal(t, "", TruncateSlug("verylongword", 5))
	assert.Equal(t, "verylongword", TruncateSlug("verylongword", 50))
}
