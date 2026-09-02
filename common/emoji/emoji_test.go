// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package emoji

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmojiMapSize(t *testing.T) {
	// Spec requires at least 50 emoji.
	assert.GreaterOrEqual(t, len(emojiMap), 50)
}

func TestShortcodeToUnicode(t *testing.T) {
	assert.Equal(t, "😄", ShortcodeToUnicode(":smile:"))
	assert.Equal(t, "hi 😄!", ShortcodeToUnicode("hi :smile:!"))
	assert.Equal(t, "😄🔥", ShortcodeToUnicode(":smile::fire:"))
}

func TestShortcodeToUnicode_Unknown(t *testing.T) {
	// Unknown shortcodes are left untouched.
	assert.Equal(t, ":unknown:", ShortcodeToUnicode(":unknown:"))
	assert.Equal(t, "a :nope: b", ShortcodeToUnicode("a :nope: b"))
}

func TestShortcodeToUnicode_NoClosing(t *testing.T) {
	assert.Equal(t, "hello :smile", ShortcodeToUnicode("hello :smile"))
}

func TestUnicodeToShortcode(t *testing.T) {
	assert.Equal(t, ":smile:", UnicodeToShortcode("😄"))
	assert.Equal(t, "hi :smile:!", UnicodeToShortcode("hi 😄!"))
	assert.Equal(t, ":smile::fire:", UnicodeToShortcode("😄🔥"))
}

func TestUnicodeToShortcode_None(t *testing.T) {
	assert.Equal(t, "plain text", UnicodeToShortcode("plain text"))
}

func TestRemoveEmoji(t *testing.T) {
	assert.Equal(t, "hi !", RemoveEmoji("hi 😄!"))
	assert.Equal(t, "abc", RemoveEmoji("a😄b😎c"))
	assert.Equal(t, "no emoji", RemoveEmoji("no emoji"))
}

func TestContainsEmoji(t *testing.T) {
	assert.True(t, ContainsEmoji("hi 😄"))
	assert.True(t, ContainsEmoji("😄"))
	assert.False(t, ContainsEmoji("plain text"))
	assert.False(t, ContainsEmoji(""))
}

func TestCountEmoji(t *testing.T) {
	assert.Equal(t, 0, CountEmoji("plain"))
	assert.Equal(t, 1, CountEmoji("a😄b"))
	assert.Equal(t, 2, CountEmoji("a😄b😎c"))
	assert.Equal(t, 3, CountEmoji("😄🔥⭐"))
}

func TestRoundTrip(t *testing.T) {
	for code, uni := range emojiMap {
		assert.Equal(t, ":"+code+":", UnicodeToShortcode(uni))
		assert.Equal(t, uni, ShortcodeToUnicode(":"+code+":"))
	}
}

func TestShortcodeToUnicode_Empty(t *testing.T) {
	assert.Equal(t, "", ShortcodeToUnicode(""))
}

func TestRemoveEmoji_VariationSelectorEmoji(t *testing.T) {
	// "❤️" is U+2764 + U+FE0F and is a single mapped emoji ("heart"), so both
	// runes are removed together.
	out := RemoveEmoji("a❤️b")
	assert.Equal(t, "ab", out)
}

func TestCountEmoji_MultiRune(t *testing.T) {
	// Multi-rune emoji (with variation selector) count as one.
	assert.Equal(t, 1, CountEmoji("❤️"))
	assert.Equal(t, 2, CountEmoji("❤️🔥"))
}
