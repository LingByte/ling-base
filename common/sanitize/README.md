# sanitize

HTML/XSS filtering helpers built on top of [bluemonday](https://github.com/microcosm-cc/bluemonday). Provides several canned policies ranging from a plain-text strict mode to a permissive UGC mode that keeps a safe subset of HTML tags, plus utilities for escaping, stripping tags and sanitizing URLs.

## Key functions

- `Clean(s string) string` — basic HTML cleanup (keeps safe formatting tags, removes script/style/iframe, `on*` attrs and `javascript:`/`data:` URLs)
- `CleanStrict(s string) string` — strict mode: plain text only, all HTML tags removed
- `CleanUGC(s string) string` — UGC mode: allows `b/i/em/strong/a/p/br/ul/ol/li/code/pre/...`, removes `script/style/iframe`, `on*` attrs and dangerous URLs
- `EscapeHTML(s string) string` — HTML entity escaping
- `StripTags(s string) string` — remove all HTML tags but keep text content (drops `<script>`/`<style>`/`<iframe>` bodies)
- `SanitizeURL(s string) string` — drop `javascript:`/`data:`/`vbscript:`/`file:`/`about:` URLs
- `IsSafeURL(s string) bool` — check whether a URL uses a safe scheme
- `CleanMarkdown(s string) string` — sanitize HTML embedded in Markdown while preserving Markdown syntax
- `UGCPolicy() *bluemonday.Policy` — the reusable UGC policy

## Quick start

```go
import "github.com/LingByte/ling-base/common/sanitize"

safe := sanitize.Clean(`<a href="https://x.com">ok</a><script>x</script>`)
// <a href="https://x.com" rel="nofollow">ok</a>

text := sanitize.CleanStrict(`<b>hi</b>`) // "hi"

body := sanitize.CleanUGC(userContent)

if !sanitize.IsSafeURL(u) {
    return errors.New("unsafe url")
}
u = sanitize.SanitizeURL(u)
```

## Dependencies

- `github.com/microcosm-cc/bluemonday` v1.0.27
