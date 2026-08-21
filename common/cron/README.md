# cron

Cron expression parsing and next-fire-time calculation with standard 5-field syntax, optional seconds, and descriptors.

## Supported syntax

- Standard 5-field: `minute hour day-of-month month day-of-week`
- Optional 6-field: `second minute hour day-of-month month day-of-week`
- `*`, literal values, comma lists (`1,3,5`), ranges (`1-5`), steps (`*/2`, `1-10/3`)
- `5L` (last weekday of month), `15W` (nearest weekday)
- Descriptors: `@hourly`, `@daily`, `@every 10s`, etc.

## Key types

- `Expression` -- a parsed cron expression

## Key functions

- `Parse(expr)` -- parse a cron expression string
- `MustParse(expr)` -- parse, panicking on error
- `Expression.Next(t)` -- compute the next fire time after `t`
- `Expression.HasSeconds()` / `Expression.IsEvery()` / `Expression.EveryDuration()`

## Quick start

```go
import (
    "time"
    "github.com/LingByte/ling-base/common/cron"
)

expr, err := cron.Parse("*/5 * * * *")
if err != nil {
    log.Fatal(err)
}
next := expr.Next(time.Now())
fmt.Println("next fire:", next)

// 6-field with seconds
expr2, _ := cron.Parse("0 */30 * * * *")

// Descriptor
expr3, _ := cron.Parse("@every 10s")
```

## License

MIT
