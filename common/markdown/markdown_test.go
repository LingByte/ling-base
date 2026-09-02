// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHTML_Basic(t *testing.T) {
	out, err := ToHTML("# Hello\n\nSome **bold** text")
	require.NoError(t, err)
	assert.Contains(t, out, "<h1")
	assert.Contains(t, out, "Hello")
	assert.Contains(t, out, "<strong>bold</strong>")
}

func TestToHTML_Empty(t *testing.T) {
	out, err := ToHTML("")
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestToHTMLByte(t *testing.T) {
	out, err := ToHTMLByte("# Title")
	require.NoError(t, err)
	assert.NotEmpty(t, out)
	assert.Contains(t, string(out), "<h1")
}

func TestMustToHTML(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = MustToHTML("# ok")
	})
}

func TestToHTML_Table(t *testing.T) {
	md := `| A | B |
|---|---|
| 1 | 2 |
`
	out, err := ToHTML(md)
	require.NoError(t, err)
	assert.Contains(t, out, "<table")
	assert.Contains(t, out, "<th>A</th>")
	assert.Contains(t, out, "<td>1</td>")
}

func TestToHTML_CodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hi\")\n```\n"
	out, err := ToHTML(md)
	require.NoError(t, err)
	assert.Contains(t, out, "<code")
	assert.Contains(t, out, "Println")
}

func TestToHTML_Strikethrough(t *testing.T) {
	out, err := ToHTML("~~deleted~~")
	require.NoError(t, err)
	assert.Contains(t, out, "<del>deleted</del>")
}

func TestToHTML_TaskList(t *testing.T) {
	md := "- [x] done\n- [ ] todo\n"
	out, err := ToHTML(md)
	require.NoError(t, err)
	assert.Contains(t, out, "checkbox")
}

func TestToHTML_AutoLink(t *testing.T) {
	out, err := ToHTML("https://example.com")
	require.NoError(t, err)
	assert.Contains(t, out, "https://example.com")
	assert.Contains(t, out, "<a")
}

func TestNewRenderer_WithEmoji(t *testing.T) {
	r := NewRenderer(WithEmoji())
	out, err := r.Render("Hello :smile:")
	require.NoError(t, err)
	// emoji shortcode should be replaced with an actual emoji or entity.
	assert.NotContains(t, out, ":smile:")
}

func TestNewRenderer_WithHighlighting(t *testing.T) {
	r := NewRenderer(WithHighlighting())
	out, err := r.Render("```go\nfunc main() {}\n```\n")
	require.NoError(t, err)
	assert.Contains(t, out, "<code")
}

func TestNewRenderer_WithUnsafe(t *testing.T) {
	r := NewRenderer(WithUnsafe())
	out, err := r.Render("<div class=\"raw\">raw html</div>")
	require.NoError(t, err)
	assert.Contains(t, out, "raw html")
}

func TestNewRenderer_WithoutUnsafe_StripsRawHTML(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render("<div class=\"raw\">raw html</div>")
	require.NoError(t, err)
	// By default goldmark escapes raw HTML.
	assert.NotContains(t, out, "<div class=\"raw\">")
}

func TestNewRenderer_WithTOC(t *testing.T) {
	r := NewRenderer(WithTOC())
	md := "# Title\n\n## Sub A\n\n## Sub B\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	assert.Contains(t, out, `<nav class="toc">`)
	assert.Contains(t, out, `href="#title"`)
	assert.Contains(t, out, `href="#sub-a"`)
	assert.Contains(t, out, `href="#sub-b"`)
	// Headings should have ids.
	assert.Contains(t, out, `id="title"`)
}

func TestNewRenderer_WithTOC_NoHeadings(t *testing.T) {
	r := NewRenderer(WithTOC())
	out, err := r.Render("just text")
	require.NoError(t, err)
	assert.NotContains(t, out, `<nav class="toc">`)
}

func TestRender_Empty(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render("")
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestNewRenderer_DefaultGFM(t *testing.T) {
	r := NewRenderer()
	// Tables are part of GFM and should work by default.
	out, err := r.Render("| a |\n|---|\n| 1 |\n")
	require.NoError(t, err)
	assert.Contains(t, out, "<table")
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "hello-world", slugify("Hello World!"))
	assert.Equal(t, "go-is-awesome", slugify("Go is Awesome"))
	assert.Equal(t, "", slugify("!!!"))
	assert.Equal(t, "a-b-c", slugify("a b c"))
}

func TestToHTML_Headings(t *testing.T) {
	md := "# H1\n\n## H2\n\n### H3\n"
	out, err := ToHTML(md)
	require.NoError(t, err)
	assert.Contains(t, out, "<h1>H1</h1>")
	assert.Contains(t, out, "<h2>H2</h2>")
	assert.Contains(t, out, "<h3>H3</h3>")
}

func TestToHTML_NestedList(t *testing.T) {
	md := "- item 1\n  - nested\n- item 2\n"
	out, err := ToHTML(md)
	require.NoError(t, err)
	assert.Contains(t, out, "<ul>")
	assert.Contains(t, out, "nested")
}

func TestToHTML_Blockquote(t *testing.T) {
	out, err := ToHTML("> quote")
	require.NoError(t, err)
	assert.Contains(t, out, "<blockquote>")
	assert.Contains(t, out, "quote")
}

func TestToHTML_Link(t *testing.T) {
	out, err := ToHTML("[example](https://example.com)")
	require.NoError(t, err)
	assert.Contains(t, out, `<a href="https://example.com"`)
	assert.Contains(t, out, "example")
}

func TestToHTML_ErrorSafe(t *testing.T) {
	// Malformed input should not panic and should return something.
	out, err := ToHTML(strings.Repeat("`", 3))
	require.NoError(t, err)
	assert.NotNil(t, out)
}
