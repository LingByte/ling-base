// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package sanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClean_RemovesScriptTag(t *testing.T) {
	in := `<p>hello</p><script>alert('xss')</script>`
	out := Clean(in)
	assert.NotContains(t, out, "script")
	assert.NotContains(t, out, "alert")
	assert.Contains(t, out, "hello")
}

func TestClean_RemovesOnAttributes(t *testing.T) {
	in := `<p onclick="evil()">hi</p>`
	out := Clean(in)
	assert.NotContains(t, out, "onclick")
	assert.NotContains(t, out, "evil")
}

func TestClean_KeepsSafeTags(t *testing.T) {
	in := `<a href="https://example.com">link</a><b>bold</b>`
	out := Clean(in)
	assert.Contains(t, out, "link")
	assert.Contains(t, out, "bold")
}

func TestClean_Empty(t *testing.T) {
	assert.Equal(t, "", Clean(""))
}

func TestCleanStrict_PlainTextOnly(t *testing.T) {
	in := `<b>hi</b><script>x</script>`
	out := CleanStrict(in)
	assert.NotContains(t, out, "<")
	assert.NotContains(t, out, ">")
	assert.Contains(t, out, "hi")
}

func TestCleanStrict_KeepsNoTags(t *testing.T) {
	assert.NotContains(t, CleanStrict(`<a href="x">y</a>`), "<a")
}

func TestCleanUGC_KeepsSafeTags(t *testing.T) {
	in := `<p>para</p><b>bold</b><i>italic</i><code>code</code><pre>pre</pre>`
	out := CleanUGC(in)
	assert.Contains(t, out, "<p>para</p>")
	assert.Contains(t, out, "<b>bold</b>")
	assert.Contains(t, out, "<i>italic</i>")
	assert.Contains(t, out, "<code>code</code>")
	assert.Contains(t, out, "<pre>pre</pre>")
}

func TestCleanUGC_RemovesScriptStyleIframe(t *testing.T) {
	cases := []string{
		`<script>alert(1)</script>`,
		`<style>body{}</style>`,
		`<iframe src="evil"></iframe>`,
	}
	for _, in := range cases {
		out := CleanUGC(in)
		assert.NotContains(t, out, "script", in)
		assert.NotContains(t, out, "style", in)
		assert.NotContains(t, out, "iframe", in)
	}
}

func TestCleanUGC_RemovesOnAttributes(t *testing.T) {
	out := CleanUGC(`<div onclick="x">y</div>`)
	assert.NotContains(t, out, "onclick")
}

func TestCleanUGC_RemovesJavascriptURL(t *testing.T) {
	out := CleanUGC(`<a href="javascript:alert(1)">x</a>`)
	assert.NotContains(t, out, "javascript")
}

func TestEscapeHTML(t *testing.T) {
	assert.Equal(t, "", EscapeHTML(""))
	out := EscapeHTML(`<a href="x">&"y"'</a>`)
	assert.Contains(t, out, "&lt;")
	assert.Contains(t, out, "&gt;")
	assert.Contains(t, out, "&amp;")
	assert.NotContains(t, out, "<a")
}

func TestStripTags_KeepsText(t *testing.T) {
	assert.Equal(t, "hello world", StripTags(`<b>hello</b> <i>world</i>`))
}

func TestStripTags_DropsScriptContent(t *testing.T) {
	out := StripTags(`<p>ok</p><script>alert(1)</script>`)
	assert.Contains(t, out, "ok")
	assert.NotContains(t, out, "alert")
}

func TestStripTags_Empty(t *testing.T) {
	assert.Equal(t, "", StripTags(""))
}

func TestSanitizeURL_SafeSchemes(t *testing.T) {
	safe := []string{
		"https://example.com/path?q=1",
		"http://example.com",
		"/relative/path",
		"relative.html",
		"mailto:a@b.com",
		"tel:+1234",
	}
	for _, u := range safe {
		assert.True(t, IsSafeURL(u), u)
		assert.NotEmpty(t, SanitizeURL(u), u)
	}
}

func TestSanitizeURL_UnsafeSchemes(t *testing.T) {
	unsafe := []string{
		"javascript:alert(1)",
		"JaVaScRiPt:alert(1)",
		"data:text/html,<script>",
		"vbscript:msgbox",
		"file:///etc/passwd",
		"about:blank",
	}
	for _, u := range unsafe {
		assert.False(t, IsSafeURL(u), u)
		assert.Equal(t, "", SanitizeURL(u), u)
	}
}

func TestSanitizeURL_Empty(t *testing.T) {
	assert.False(t, IsSafeURL(""))
	assert.Equal(t, "", SanitizeURL(""))
}

func TestCleanMarkdown_RemovesHTML(t *testing.T) {
	in := "# Title\n\nSome **bold** text <script>x</script>\n\n[link](https://x.com)"
	out := CleanMarkdown(in)
	assert.NotContains(t, out, "script")
	// Markdown syntax is preserved by the policy (it only touches HTML).
	assert.Contains(t, out, "Title")
}

func TestUGCPolicy_NotNil(t *testing.T) {
	p := UGCPolicy()
	require.NotNil(t, p)
	// Should strip script.
	out := p.Sanitize(`<script>x</script><b>y</b>`)
	assert.NotContains(t, out, "script")
	assert.Contains(t, out, "y")
}

func TestXSSAttackVectors(t *testing.T) {
	vectors := []string{
		`<script>alert('xss')</script>`,
		`<img src="x" onerror="alert(1)">`,
		`<svg onload="alert(1)">`,
		`<a href="javascript:alert(1)">x</a>`,
		`<iframe src="javascript:alert(1)"></iframe>`,
		`"><script>alert(1)</script>`,
		`<body onload="alert(1)">`,
	}
	for _, v := range vectors {
		out := Clean(v)
		low := strings.ToLower(out)
		assert.NotContains(t, low, "alert", v)
		assert.NotContains(t, low, "onerror", v)
		assert.NotContains(t, low, "onload", v)
		assert.NotContains(t, low, "javascript:", v)
	}
}
