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

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

func TestSlugWithSeparator_EmptySeparator(t *testing.T) {
	// Empty separator collapses all tokens together (no separator inserted).
	assert.Equal(t, "helloworld", SlugWithSeparator("Hello World!", ""))
	assert.Equal(t, "abc", SlugWithSeparator("a b c", ""))
}

func TestSlugWithSeparator_DashSeparator(t *testing.T) {
	assert.Equal(t, "hello-world", SlugWithSeparator("Hello World!", "-"))
}

func TestSlugWithSeparator_CollapsesDuplicates(t *testing.T) {
	// Multiple separators in input collapse to a single one.
	assert.Equal(t, "a-b", SlugWithSeparator("a---b", "-"))
}

func TestSlugWithSeparator_TildeSeparator(t *testing.T) {
	// '~' is treated as a separator character and replaced with the chosen sep.
	assert.Equal(t, "a~b", SlugWithSeparator("a~b", "~"))
}

func TestSlugWithSeparator_SeparatorCharInInput(t *testing.T) {
	// When the separator char also appears in input it is normalized.
	assert.Equal(t, "a_b", SlugWithSeparator("a-b", "_"))
}

func TestSlug_DigitsLeading(t *testing.T) {
	assert.Equal(t, "123abc", Slug("123abc"))
	assert.Equal(t, "123-abc", Slug("123 abc"))
}

func TestSlug_SpecialUnicode(t *testing.T) {
	// Emojis and other non-alphanumeric unicode act as separators.
	got := Slug("hello 😎 world")
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "world")
	assert.NotContains(t, got, "😎")
}

func TestSlug_MixedPunctuation(t *testing.T) {
	assert.Equal(t, "a-b-c", Slug("a/b/c"))
	assert.Equal(t, "a-b-c", Slug("a+b+c"))
}

func TestSlugLower_Mixed(t *testing.T) {
	assert.Equal(t, "hello-123", SlugLower("Hello 123"))
}

func TestSlugUnique_OnlySpecialChars(t *testing.T) {
	// Base slug is empty => returns just the random suffix.
	got := SlugUnique("!@#$")
	assert.Len(t, got, 8)
}

func TestSlugUnique_DifferentInputs(t *testing.T) {
	a := SlugUnique("title")
	b := SlugUnique("other")
	// Both have suffixes but different bases.
	assert.True(t, strings.HasPrefix(a, "title-"))
	assert.True(t, strings.HasPrefix(b, "other-"))
}

func TestTruncateSlug_ZeroAndNegative(t *testing.T) {
	assert.Equal(t, "abc", TruncateSlug("abc", 0))
	assert.Equal(t, "abc", TruncateSlug("abc", -1))
}

func TestTruncateSlug_ExactLength(t *testing.T) {
	assert.Equal(t, "one-two", TruncateSlug("one-two", 7))
}

func TestTruncateSlug_CutOnSeparatorBoundary(t *testing.T) {
	// maxLen lands exactly on a separator: s[maxLen] == '-' so the token is kept.
	assert.Equal(t, "one-two", TruncateSlug("one-two-three", 7))
}

func TestTruncateSlug_SeparatorAtEnd(t *testing.T) {
	// len(s) == maxLen: returned unchanged (no truncation occurs).
	assert.Equal(t, "one-two-", TruncateSlug("one-two-", 8))
}

func TestTruncateSlug_PartialTokenKeptWhenBoundary(t *testing.T) {
	// When the char after maxLen is a separator, the partial token is complete.
	assert.Equal(t, "one-two", TruncateSlug("one-two-three", 8))
}

func TestTruncateSlug_PartialTokenDropped(t *testing.T) {
	// maxLen=6 cuts "one-tw" mid-token; the partial "tw" is dropped => "one".
	assert.Equal(t, "one", TruncateSlug("one-two-three", 6))
}

func TestTruncateSlug_EmptyInput(t *testing.T) {
	assert.Equal(t, "", TruncateSlug("", 5))
}

func TestRandomHex_Length(t *testing.T) {
	// 4 bytes => 8 hex chars.
	assert.Len(t, randomHex(4), 8)
	assert.Len(t, randomHex(0), 0)
}

func TestRandomHex_NonDeterministic(t *testing.T) {
	a := randomHex(8)
	b := randomHex(8)
	assert.NotEqual(t, a, b)
}

func TestIsAlnum(t *testing.T) {
	assert.True(t, isAlnum('a'))
	assert.True(t, isAlnum('Z'))
	assert.True(t, isAlnum('0'))
	assert.True(t, isAlnum('9'))
	assert.False(t, isAlnum('-'))
	assert.False(t, isAlnum(' '))
	assert.False(t, isAlnum('你'))
}

func TestIsSeparator(t *testing.T) {
	for _, r := range []rune{'-', '_', '.', '~', '/', '+'} {
		assert.True(t, isSeparator(r), string(r))
	}
	for _, r := range []rune{'a', ' ', '!', '@'} {
		assert.False(t, isSeparator(r), string(r))
	}
}
