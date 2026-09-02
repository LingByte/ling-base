# slug

Converts arbitrary strings into URL-safe slugs. Chinese (CJK) characters are
first transliterated to pinyin via
[`common/pinyin`](../pinyin), then all non-alphanumeric characters are
collapsed into a single separator (default `-`).

## Key functions

- `Slug(s string) string` — URL-safe slug using `-` as separator
- `SlugWithSeparator(s, sep string) string` — custom separator
- `SlugLower(s string) string` — lower-case slug
- `SlugUnique(s string) string` — slug with a random 4-byte hex suffix for uniqueness
- `TruncateSlug(s string, maxLen int) string` — truncate without cutting a word

## Quick start

```go
import "github.com/LingByte/ling-base/common/slug"

slug.Slug("Hello World!")            // "hello-world"
slug.Slug("你好世界")                 // "ni-hao-shi-jie"
slug.SlugWithSeparator("a b c", "_") // "a_b_c"
slug.SlugUnique("title")             // "title-a1b2c3d4"
slug.TruncateSlug("one-two-three", 8) // "one-two"
```
