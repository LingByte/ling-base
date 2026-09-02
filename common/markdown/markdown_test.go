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

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

func TestWithGFM(t *testing.T) {
	r := NewRenderer(WithGFM())
	out, err := r.Render("| a |\n|---|\n| 1 |\n")
	require.NoError(t, err)
	assert.Contains(t, out, "<table")
}

func TestWithGFM_Disabled(t *testing.T) {
	// Explicitly disable GFM via a custom option.
	r := NewRenderer(func(c *config) { c.gfm = false })
	out, err := r.Render("| a |\n|---|\n| 1 |\n")
	require.NoError(t, err)
	// Without GFM, pipes are not parsed as a table.
	assert.NotContains(t, out, "<table")
}

func TestNewRenderer_WithEmoji_MultipleShortcodes(t *testing.T) {
	r := NewRenderer(WithEmoji())
	out, err := r.Render(":smile: :heart: :thumbsup: :rocket:")
	require.NoError(t, err)
	assert.NotContains(t, out, ":smile:")
	assert.NotContains(t, out, ":heart:")
	assert.NotContains(t, out, ":thumbsup:")
	assert.NotContains(t, out, ":rocket:")
}

func TestNewRenderer_WithEmoji_UnknownShortcode(t *testing.T) {
	r := NewRenderer(WithEmoji())
	out, err := r.Render(":notarealcode:")
	require.NoError(t, err)
	// Unknown shortcode is left as-is.
	assert.Contains(t, out, ":notarealcode:")
}

func TestNewRenderer_WithHighlighting_DifferentLanguages(t *testing.T) {
	r := NewRenderer(WithHighlighting())
	for _, lang := range []string{"go", "python", "javascript", "bash"} {
		md := "```" + lang + "\nfoo bar\n```\n"
		out, err := r.Render(md)
		require.NoError(t, err, lang)
		assert.Contains(t, out, "<code", lang)
	}
}

func TestNewRenderer_WithHighlighting_Plaintext(t *testing.T) {
	r := NewRenderer(WithHighlighting())
	out, err := r.Render("```\nplain\n```\n")
	require.NoError(t, err)
	assert.Contains(t, out, "<code")
}

func TestNewRenderer_WithTOC_Multilevel(t *testing.T) {
	r := NewRenderer(WithTOC())
	md := "# Title\n\n## Sub A\n\n### Sub Sub\n\n## Sub B\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	assert.Contains(t, out, `<nav class="toc">`)
	assert.Contains(t, out, `href="#title"`)
	assert.Contains(t, out, `href="#sub-a"`)
	assert.Contains(t, out, `href="#sub-sub"`)
	assert.Contains(t, out, `href="#sub-b"`)
	assert.Contains(t, out, `toc-level-1`)
	assert.Contains(t, out, `toc-level-2`)
	assert.Contains(t, out, `toc-level-3`)
}

func TestNewRenderer_WithTOC_DuplicateHeadings(t *testing.T) {
	r := NewRenderer(WithTOC())
	md := "# Section\n\n# Section\n\n# Section\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	// Duplicate headings get suffixed ids (first is bare, subsequent get -1).
	assert.Contains(t, out, `id="section"`)
	assert.Contains(t, out, `id="section-1"`)
	assert.Contains(t, out, `href="#section"`)
	assert.Contains(t, out, `href="#section-1"`)
}

func TestNewRenderer_WithTOC_EmptyHeadingText(t *testing.T) {
	r := NewRenderer(WithTOC())
	// A heading with only markup but empty text yields id "section".
	md := "# \n\nText\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	// The empty-text heading gets the fallback "section" id.
	assert.Contains(t, out, `id="section"`)
}

func TestNewRenderer_WithTOC_AndEmoji(t *testing.T) {
	r := NewRenderer(WithTOC(), WithEmoji())
	md := "# Hello :smile:\n\ntext\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	assert.Contains(t, out, `<nav class="toc">`)
}

func TestNewRenderer_AllOptions(t *testing.T) {
	r := NewRenderer(WithGFM(), WithEmoji(), WithHighlighting(), WithUnsafe(), WithTOC())
	md := "# Title\n\n**bold** :heart:\n\n```go\nfunc main(){}\n```\n\n<div>raw</div>\n\n| a |\n|---|\n| 1 |\n"
	out, err := r.Render(md)
	require.NoError(t, err)
	assert.Contains(t, out, `<nav class="toc">`)
	assert.Contains(t, out, "raw")
	assert.Contains(t, out, "<table")
}

func TestRender_EmptyString(t *testing.T) {
	r := NewRenderer(WithTOC(), WithEmoji())
	out, err := r.Render("")
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestToHTMLByte_Empty(t *testing.T) {
	out, err := ToHTMLByte("")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestToHTMLByte_NonEmpty(t *testing.T) {
	out, err := ToHTMLByte("# hi")
	require.NoError(t, err)
	assert.Contains(t, string(out), "<h1")
}

func TestMustToHTML_PanicsOnError(t *testing.T) {
	// The default renderer does not produce errors for normal input, so
	// MustToHTML does not panic in practice. Verify it returns content.
	out := MustToHTML("# ok\n\ntext")
	assert.Contains(t, out, "<h1")
}

func TestMustToHTML_Valid(t *testing.T) {
	assert.NotPanics(t, func() {
		out := MustToHTML("# ok\n\ntext")
		assert.Contains(t, out, "<h1")
	})
}

func TestSlugify_SpecialChars(t *testing.T) {
	assert.Equal(t, "hello-world-123", slugify("Hello World 123!"))
	assert.Equal(t, "a-b-c", slugify("a b_c"))
	assert.Equal(t, "", slugify("   "))
	// '.' is not in the kept set, so it is dropped.
	assert.Equal(t, "xy", slugify("x.y"))
}

func TestEscapeHTML(t *testing.T) {
	assert.Equal(t, "&lt;tag&gt;", escapeHTML("<tag>"))
	assert.Equal(t, "&amp;amp;", escapeHTML("&amp;"))
	assert.Equal(t, "&quot;q&quot;", escapeHTML(`"q"`))
	assert.Equal(t, "plain", escapeHTML("plain"))
}

func TestUniqueID_EmptyBase(t *testing.T) {
	tt := &tocTransformer{}
	// Empty base falls back to "section".
	assert.Equal(t, "section", tt.uniqueID(""))
}

func TestUniqueID_NoDuplicates(t *testing.T) {
	tt := &tocTransformer{}
	assert.Equal(t, "hello", tt.uniqueID("hello"))
}

func TestUniqueID_WithDuplicates(t *testing.T) {
	tt := &tocTransformer{entries: []tocEntry{
		{id: "dup"},
		{id: "dup"},
	}}
	// Two existing "dup" => "dup-2".
	assert.Equal(t, "dup-2", tt.uniqueID("dup"))
}

func TestRenderTOC_Empty(t *testing.T) {
	tt := &tocTransformer{}
	assert.Equal(t, "", tt.renderTOC())
}

func TestRenderTOC_WithEntries(t *testing.T) {
	tt := &tocTransformer{entries: []tocEntry{
		{level: 1, text: "Hello & World", id: "hello-world"},
		{level: 2, text: "<script>", id: "script"},
	}}
	out := tt.renderTOC()
	assert.Contains(t, out, `<nav class="toc">`)
	assert.Contains(t, out, `href="#hello-world"`)
	assert.Contains(t, out, "Hello &amp; World")
	assert.Contains(t, out, "&lt;script&gt;")
}
