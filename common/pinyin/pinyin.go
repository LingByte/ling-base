// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package pinyin converts Chinese characters to pinyin readings.
//
// It is a pure-Go port of the getpinyin.c library, covering the entire
// CJK Unified Ideographs basic block (U+4E00–U+9FA5, 20,812 characters).
// Multi-tone characters (多音字) are supported: all readings are stored
// and can be retrieved individually or flattened.
//
// Basic usage:
//
//	// Simple conversion (first reading per character)
//	fmt.Println(pinyin.Convert("你好世界"))
//	// Output: ni hao shi jie
//
//	// Keep non-Chinese characters
//	fmt.Println(pinyin.Convert("Hello世界", pinyin.WithKeepNonCJK(true)))
//	// Output: Hello shi jie
//
//	// Get all readings for multi-tone characters
//	for _, readings := range pinyin.ConvertAll("重庆") {
//	    fmt.Println(readings)
//	}
//	// [chong]
//	// [qing]
//
//	// Single character lookup
//	fmt.Println(pinyin.Pinyin('乐'))  // "le luo yao yue"
//	fmt.Println(pinyin.PinyinFirst('乐'))  // "le"
package pinyin

import (
	"strings"
	"unicode"
)

// Pinyin returns all pinyin readings for a single Chinese character,
// separated by spaces. Returns empty string for non-CJK characters.
func Pinyin(r rune) string {
	return pinyinMap[r]
}

// PinyinFirst returns the first (primary) pinyin reading for a character.
// Returns empty string for non-CJK characters.
func PinyinFirst(r rune) string {
	s := pinyinMap[r]
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, ' '); idx >= 0 {
		return s[:idx]
	}
	return s
}

// PinyinAll returns all pinyin readings as a slice for a character.
// Returns nil for non-CJK characters.
func PinyinAll(r rune) []string {
	s := pinyinMap[r]
	if s == "" {
		return nil
	}
	parts := strings.Fields(s)
	return parts
}

// IsCJK reports whether r is in the CJK Unified Ideographs basic block
// (U+4E00–U+9FA5).
func IsCJK(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FA5
}

// HasPinyin reports whether the character has a known pinyin reading.
func HasPinyin(r rune) bool {
	_, ok := pinyinMap[r]
	return ok
}

// Convert converts a Chinese string to pinyin. Each Chinese character is
// replaced by its first pinyin reading; non-Chinese characters are dropped
// unless WithKeepNonCJK(true) is set. Consecutive non-CJK characters are
// kept as a single group. Readings are separated by a space
// (configurable via WithSeparator).
func Convert(s string, opts ...Option) string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var result []string
	var nonCJKBuf strings.Builder

	flushNonCJK := func() {
		if nonCJKBuf.Len() > 0 {
			result = append(result, nonCJKBuf.String())
			nonCJKBuf.Reset()
		}
	}

	for _, r := range s {
		if py := pinyinMap[r]; py != "" {
			flushNonCJK()
			reading := py
			if !cfg.allTones {
				reading = PinyinFirst(r)
			}
			if cfg.uppercase {
				reading = strings.ToUpper(reading)
			} else if cfg.titleCase {
				reading = toTitleCase(reading)
			}
			result = append(result, reading)
		} else if cfg.keepNonCJK {
			nonCJKBuf.WriteRune(r)
		}
	}
	flushNonCJK()

	return strings.Join(result, cfg.separator)
}

// ConvertAll converts a Chinese string to pinyin, returning all possible
// readings for each character. Non-Chinese characters produce a single-element
// slice containing the character itself (if WithKeepNonCJK is true) or are
// skipped.
func ConvertAll(s string, opts ...Option) [][]string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var result [][]string
	for _, r := range s {
		if py := pinyinMap[r]; py != "" {
			readings := strings.Fields(py)
			if cfg.uppercase {
				for i, rd := range readings {
					readings[i] = strings.ToUpper(rd)
				}
			} else if cfg.titleCase {
				for i, rd := range readings {
					readings[i] = toTitleCase(rd)
				}
			}
			result = append(result, readings)
		} else if cfg.keepNonCJK {
			result = append(result, []string{string(r)})
		}
	}
	return result
}

// ConvertSlice converts a Chinese string to a slice of pinyin readings,
// one element per character (using the first reading for multi-tone chars).
// Non-Chinese characters are skipped unless WithKeepNonCJK is true, in which
// case consecutive non-CJK characters are grouped into a single element.
func ConvertSlice(s string, opts ...Option) []string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var result []string
	var nonCJKBuf strings.Builder

	flushNonCJK := func() {
		if nonCJKBuf.Len() > 0 {
			result = append(result, nonCJKBuf.String())
			nonCJKBuf.Reset()
		}
	}

	for _, r := range s {
		if py := pinyinMap[r]; py != "" {
			flushNonCJK()
			reading := PinyinFirst(r)
			if cfg.uppercase {
				reading = strings.ToUpper(reading)
			} else if cfg.titleCase {
				reading = toTitleCase(reading)
			}
			result = append(result, reading)
		} else if cfg.keepNonCJK {
			nonCJKBuf.WriteRune(r)
		}
	}
	flushNonCJK()
	return result
}

// Count returns the number of characters in the data set.
func Count() int {
	return len(pinyinMap)
}

// toTitleCase capitalizes the first letter, leaves the rest unchanged.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
