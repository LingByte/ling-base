// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package diff

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineDiff_Same(t *testing.T) {
	lines := LineDiff("a\nb\nc", "a\nb\nc")
	require.Len(t, lines, 3)
	for _, l := range lines {
		assert.Equal(t, Unchanged, l.Type)
	}
}

func TestLineDiff_Added(t *testing.T) {
	lines := LineDiff("a\nb", "a\nb\nc")
	require.Len(t, lines, 3)
	assert.Equal(t, Unchanged, lines[0].Type)
	assert.Equal(t, "a", lines[0].Content)
	assert.Equal(t, Unchanged, lines[1].Type)
	assert.Equal(t, Added, lines[2].Type)
	assert.Equal(t, "c", lines[2].Content)
}

func TestLineDiff_Removed(t *testing.T) {
	lines := LineDiff("a\nb\nc", "a\nc")
	require.Len(t, lines, 3)
	assert.Equal(t, Unchanged, lines[0].Type)
	assert.Equal(t, "a", lines[0].Content)
	assert.Equal(t, Removed, lines[1].Type)
	assert.Equal(t, "b", lines[1].Content)
	assert.Equal(t, Unchanged, lines[2].Type)
	assert.Equal(t, "c", lines[2].Content)
}

func TestLineDiff_Modified(t *testing.T) {
	lines := LineDiff("a\nb\nc", "a\nx\nc")
	require.Len(t, lines, 4)
	assert.Equal(t, Unchanged, lines[0].Type)
	assert.Equal(t, Removed, lines[1].Type)
	assert.Equal(t, "b", lines[1].Content)
	assert.Equal(t, Added, lines[2].Type)
	assert.Equal(t, "x", lines[2].Content)
	assert.Equal(t, Unchanged, lines[3].Type)
}

func TestLineDiff_Empty(t *testing.T) {
	lines := LineDiff("", "")
	assert.Empty(t, lines)

	lines = LineDiff("", "a\nb")
	require.Len(t, lines, 2)
	assert.Equal(t, Added, lines[0].Type)
	assert.Equal(t, "a", lines[0].Content)
	assert.Equal(t, Added, lines[1].Type)

	lines = LineDiff("a\nb", "")
	require.Len(t, lines, 2)
	assert.Equal(t, Removed, lines[0].Type)
	assert.Equal(t, Removed, lines[1].Type)
}

func TestTextDiff(t *testing.T) {
	out := TextDiff("a\nb", "a\nc")
	assert.Contains(t, out, "  a\n")
	assert.Contains(t, out, "- b\n")
	assert.Contains(t, out, "+ c\n")
}

func TestWordDiff(t *testing.T) {
	lines := WordDiff("hello world", "hello go")
	require.Len(t, lines, 3)
	assert.Equal(t, Unchanged, lines[0].Type)
	assert.Equal(t, "hello", lines[0].Content)
	assert.Equal(t, Removed, lines[1].Type)
	assert.Equal(t, "world", lines[1].Content)
	assert.Equal(t, Added, lines[2].Type)
	assert.Equal(t, "go", lines[2].Content)
}

func TestHTMLDiff(t *testing.T) {
	out := HTMLDiff("a\nb", "a\nc")
	assert.Contains(t, out, `class="diff-unchanged"`)
	assert.Contains(t, out, `class="diff-removed"`)
	assert.Contains(t, out, `class="diff-added"`)
	// Verify escaping.
	out2 := HTMLDiff("<b>", "&amp;")
	assert.True(t, strings.Contains(out2, "&lt;b&gt;"))
}

func TestHTMLDiff_EscapeAmp(t *testing.T) {
	out := HTMLDiff("a&b", "a&b")
	assert.Contains(t, out, "a&amp;b")
}
