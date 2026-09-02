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

func TestMatchEmojiAt2_OutOfBounds(t *testing.T) {
	// Index at or beyond the rune slice length should return false.
	runes := []rune("😄")
	_, _, ok := matchEmojiAt2(runes, len(runes))
	assert.False(t, ok)
	// Far past the end.
	_, _, ok = matchEmojiAt2(runes, len(runes)+5)
	assert.False(t, ok)
}

func TestMatchEmojiAt2_FirstRuneNotInMap(t *testing.T) {
	// A plain ASCII rune is not the start of any emoji.
	runes := []rune("abc")
	_, _, ok := matchEmojiAt2(runes, 0)
	assert.False(t, ok)
}

func TestMatchEmojiAt2_PartialEmojiNoMatch(t *testing.T) {
	// "❤" (U+2764 without the variation selector) is the first rune of the
	// "heart" emoji ("❤️") so it is in firstRuneIndex, but the bare rune does
	// not match any full emoji — exercising the final return-false branch.
	runes := []rune("❤")
	_, _, ok := matchEmojiAt2(runes, 0)
	assert.False(t, ok)
}

func TestShortcodeToUnicode_MultipleAndMixed(t *testing.T) {
	// Multiple consecutive emoji with surrounding text.
	assert.Equal(t, "a😄b🔥c⭐d", ShortcodeToUnicode("a:smile:b:fire:c:star:d"))
}

func TestShortcodeToUnicode_NoClosingColon(t *testing.T) {
	// A shortcode without a closing colon should be left as-is.
	assert.Equal(t, "hello :smile world", ShortcodeToUnicode("hello :smile world"))
	assert.Equal(t, ":smile", ShortcodeToUnicode(":smile"))
}

func TestUnicodeToShortcode_MixedText(t *testing.T) {
	// Mixed emoji and plain text.
	assert.Equal(t, "hello :smile: world :fire:!", UnicodeToShortcode("hello 😄 world 🔥!"))
}

func TestRemoveEmoji_Empty(t *testing.T) {
	assert.Equal(t, "", RemoveEmoji(""))
}

func TestRemoveEmoji_OnlyEmoji(t *testing.T) {
	assert.Equal(t, "", RemoveEmoji("😄🔥😎"))
}

func TestContainsEmoji_MultipleEmoji(t *testing.T) {
	assert.True(t, ContainsEmoji("😄🔥😎"))
	assert.True(t, ContainsEmoji("text 😄 more 🔥"))
}

func TestCountEmoji_Empty(t *testing.T) {
	assert.Equal(t, 0, CountEmoji(""))
}

func TestCountEmoji_ConsecutiveEmoji(t *testing.T) {
	assert.Equal(t, 3, CountEmoji("😄🔥😄"))
}
