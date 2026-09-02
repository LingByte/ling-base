# emoji

Convert between emoji shortcodes and Unicode, and detect / strip emoji in
strings. Pure standard library with a built-in shortcode → Unicode map
(120+ common emoji).

## Key functions

- `ShortcodeToUnicode(s string) string` — `:smile:` → `😄`
- `UnicodeToShortcode(s string) string` — `😄` → `:smile:`
- `RemoveEmoji(s string) string` — strip all known emoji
- `ContainsEmoji(s string) bool` — detect emoji presence
- `CountEmoji(s string) int` — count known emoji (multi-rune emoji count once)

Multi-rune emoji that include a variation selector (e.g. `❤️` = U+2764 +
U+FE0F) are matched atomically.

## Quick start

```go
import "github.com/LingByte/ling-base/common/emoji"

emoji.ShortcodeToUnicode(":smile:") // "😄"
emoji.UnicodeToShortcode("😄")      // ":smile:"
emoji.RemoveEmoji("hi 😄!")         // "hi !"
emoji.ContainsEmoji("hi 😄")        // true
emoji.CountEmoji("a😄b😎c")         // 2
```
