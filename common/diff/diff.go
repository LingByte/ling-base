// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package diff computes line- and word-level diffs between two strings using
// the Myers diff algorithm (via a longest-common-subsequence dynamic program).
//
// # Quick start
//
//	diff.TextDiff("a\nb\nc", "a\nx\nc") // unified diff text
//	diff.LineDiff("a\nb", "a\nc")       // []DiffLine
//	diff.WordDiff("hello world", "hello go")
//	diff.HTMLDiff("a\nb", "a\nc")       // HTML with green/red spans
package diff

import (
	"fmt"
	"strings"
)

// DiffType describes the kind of change for a single diff line/token.
type DiffType int

const (
	// Unchanged indicates the line/token is identical in both inputs.
	Unchanged DiffType = iota
	// Added indicates the line/token exists only in the new input.
	Added
	// Removed indicates the line/token exists only in the old input.
	Removed
)

// DiffLine is a single line/token of a diff result.
type DiffLine struct {
	Type    DiffType
	Content string
}

// LineDiff performs a line-level diff between old and new and returns the
// sequence of DiffLines.
func LineDiff(old, new string) []DiffLine {
	return diffTokens(splitLines(old), splitLines(new))
}

// WordDiff performs a word-level diff between old and new and returns the
// sequence of DiffLines (one per word).
func WordDiff(old, new string) []DiffLine {
	return diffTokens(splitWords(old), splitWords(new))
}

// TextDiff returns a unified-diff-style string for the line-level diff of old
// and new.
func TextDiff(old, new string) string {
	lines := LineDiff(old, new)
	var b strings.Builder
	for _, l := range lines {
		switch l.Type {
		case Added:
			b.WriteString("+ ")
		case Removed:
			b.WriteString("- ")
		default:
			b.WriteString("  ")
		}
		b.WriteString(l.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

// HTMLDiff returns an HTML representation of the line-level diff where added
// lines are wrapped in green <span> elements and removed lines in red ones.
func HTMLDiff(old, new string) string {
	lines := LineDiff(old, new)
	var b strings.Builder
	for _, l := range lines {
		escaped := htmlEscape(l.Content)
		switch l.Type {
		case Added:
			fmt.Fprintf(&b, `<div class="diff-added">%s</div>`, escaped)
		case Removed:
			fmt.Fprintf(&b, `<div class="diff-removed">%s</div>`, escaped)
		default:
			fmt.Fprintf(&b, `<div class="diff-unchanged">%s</div>`, escaped)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// diffTokens computes the LCS-based diff between two token slices and emits a
// merged DiffLine sequence (removed lines precede added lines).
func diffTokens(a, b []string) []DiffLine {
	m, n := len(a), len(b)
	// dp[i][j] = length of LCS of a[i:] and b[j:].
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}

	var out []DiffLine
	i, j := 0, 0
	for i < m && j < n {
		if a[i] == b[j] {
			out = append(out, DiffLine{Type: Unchanged, Content: a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			out = append(out, DiffLine{Type: Removed, Content: a[i]})
			i++
		} else {
			out = append(out, DiffLine{Type: Added, Content: b[j]})
			j++
		}
	}
	for i < m {
		out = append(out, DiffLine{Type: Removed, Content: a[i]})
		i++
	}
	for j < n {
		out = append(out, DiffLine{Type: Added, Content: b[j]})
		j++
	}
	return out
}

// splitLines splits s into lines, preserving the content without trailing
// newlines. A trailing newline produces no extra empty element (matching
// typical line-diff semantics where "a\nb\n" => ["a","b"]).
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	parts := strings.Split(s, "\n")
	// Drop a trailing empty element caused by a final newline.
	if len(parts) > 0 && parts[len(parts)-1] == "" && strings.HasSuffix(s, "\n") {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// splitWords splits s into whitespace-delimited words.
func splitWords(s string) []string {
	return strings.Fields(s)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// htmlEscape escapes the four significant XML/HTML characters.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
