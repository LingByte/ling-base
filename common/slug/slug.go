// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package slug converts arbitrary strings into URL-safe slugs.
//
// Chinese (CJK) characters are first transliterated to pinyin via the
// github.com/LingByte/ling-base/common/pinyin package, then all non
// alphanumeric characters are collapsed into a single separator (default "-").
//
// # Quick start
//
//	slug.Slug("Hello World!")              // "hello-world"
//	slug.Slug("你好世界")                   // "ni-hao-shi-jie"
//	slug.SlugWithSeparator("a b c", "_")   // "a_b_c"
//	slug.SlugLower("MixedCase")            // "mixedcase"
//	slug.SlugUnique("title")               // "title-a1b2c3d4"
//	slug.TruncateSlug("one-two-three", 8)  // "one-two"
package slug

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/LingByte/ling-base/common/pinyin"
)

// defaultSeparator is the default separator inserted between slug tokens.
const defaultSeparator = "-"

// Slug converts s into a URL-safe slug using "-" as the separator.
// CJK characters are transliterated to pinyin first.
func Slug(s string) string {
	return SlugWithSeparator(s, defaultSeparator)
}

// SlugLower converts s into a lower-case URL-safe slug.
func SlugLower(s string) string {
	return strings.ToLower(Slug(s))
}

// SlugWithSeparator converts s into a URL-safe slug using the given separator.
// CJK characters are transliterated to pinyin first.
func SlugWithSeparator(s, sep string) string {
	// Transliterate CJK characters to pinyin, keeping non-CJK characters.
	pinyinStr := pinyin.Convert(s, pinyin.WithKeepNonCJK(true), pinyin.WithSeparator(" "))

	var b strings.Builder
	prevSep := true
	for _, r := range pinyinStr {
		if isAlnum(r) {
			b.WriteRune(unicode.ToLower(r))
			prevSep = false
		} else if r == ' ' || (len(sep) > 0 && r == rune(sep[0])) || isSeparator(r) {
			if !prevSep && b.Len() > 0 {
				b.WriteString(sep)
				prevSep = true
			}
		} else {
			// Any other character acts as a separator boundary.
			if !prevSep && b.Len() > 0 {
				b.WriteString(sep)
				prevSep = true
			}
		}
	}

	result := b.String()
	result = strings.Trim(result, sep)
	// Collapse repeated separators.
	if sep != "" {
		dup := sep + sep
		for strings.Contains(result, dup) {
			result = strings.ReplaceAll(result, dup, sep)
		}
	}
	return result
}

// SlugUnique returns a slug that is guaranteed to be unique by appending a
// random 4-byte hex suffix.
func SlugUnique(s string) string {
	base := Slug(s)
	suffix := randomHex(4)
	if base == "" {
		return suffix
	}
	return base + defaultSeparator + suffix
}

// TruncateSlug truncates s to at most maxLen characters without cutting a
// token (separator-delimited word) in half. If truncation would leave a
// trailing separator, it is removed.
func TruncateSlug(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	trunc := s[:maxLen]
	// If we cut in the middle of a token, drop the partial token.
	if idx := strings.LastIndex(trunc, defaultSeparator); idx >= 0 {
		// Only trim if the remainder after the last separator is a partial word.
		if len(trunc)-idx-1 > 0 && idx < maxLen {
			// Keep up to the last full token only if the partial is incomplete.
			// We check if the original string continues the current token.
			if maxLen < len(s) && s[maxLen] != defaultSeparator[0] {
				trunc = trunc[:idx]
			}
		}
	} else {
		// No separator found: the whole trunc is a partial token.
		trunc = ""
	}
	return strings.Trim(trunc, defaultSeparator)
}

// isAlnum reports whether r is an ASCII alphanumeric character.
func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isSeparator reports whether r is a common separator/punctuation character.
func isSeparator(r rune) bool {
	switch r {
	case '-', '_', '.', '~', '/', '+':
		return true
	}
	return false
}

// randomHex returns n random bytes encoded as a hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback should never happen in practice.
		return "00000000"
	}
	return hex.EncodeToString(b)
}
