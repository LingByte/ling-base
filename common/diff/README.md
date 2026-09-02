# diff

Line- and word-level text diffing using the Myers diff algorithm (via a
longest-common-subsequence dynamic program). Pure standard library.

## Key types & functions

- `type DiffType int` — `Unchanged`, `Added`, `Removed`
- `type DiffLine struct { Type DiffType; Content string }`
- `TextDiff(old, new string) string` — unified-diff-style text
- `LineDiff(old, new string) []DiffLine` — line-level diff
- `WordDiff(old, new string) []DiffLine` — word-level diff
- `HTMLDiff(old, new string) string` — HTML with `diff-added`/`diff-removed`/`diff-unchanged` divs

## Quick start

```go
import "github.com/LingByte/ling-base/common/diff"

diff.TextDiff("a\nb\nc", "a\nx\nc")
diff.LineDiff("a\nb", "a\nc")
diff.WordDiff("hello world", "hello go")
diff.HTMLDiff("a\nb", "a\nc")
```
