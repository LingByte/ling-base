# markdown

Renders Markdown to HTML using [goldmark](https://github.com/yuin/goldmark). Provides a configurable `Renderer` with functional options for GitHub Flavored Markdown (enabled by default), emoji shortcodes, syntax highlighting, raw HTML and table-of-contents generation.

## Key functions

- `ToHTML(md string) (string, error)` — Markdown → HTML (GFM enabled by default)
- `ToHTMLByte(md string) ([]byte, error)` — same, returning bytes
- `MustToHTML(md string) string` — panic on error
- `NewRenderer(opts ...Option) *Renderer` — configurable renderer
- `(*Renderer) Render(md string) (string, error)` — render with the configured options

## Options

- `WithGFM()` — GitHub Flavored Markdown (tables, strikethrough, task lists, auto-links); enabled by default
- `WithEmoji()` — emoji shortcodes (`:smile:` → 😄)
- `WithHighlighting()` — Chroma-based syntax highlighting for fenced code blocks
- `WithUnsafe()` — allow raw HTML to pass through
- `WithTOC()` — generate heading IDs and prepend a `<nav class="toc">` table of contents

## Quick start

```go
import "github.com/LingByte/ling-base/common/markdown"

html, err := markdown.ToHTML("# Hello\n\nSome **bold** text")

r := markdown.NewRenderer(
    markdown.WithGFM(),
    markdown.WithEmoji(),
    markdown.WithHighlighting(),
    markdown.WithTOC(),
)
out, err := r.Render(md)
```

## Dependencies

- `github.com/yuin/goldmark` v1.7.8
- `github.com/yuin/goldmark-emoji` v1.0.6
- `github.com/yuin/goldmark-highlighting/v2` v2.0.0-20230729083705-37449abec8cc
