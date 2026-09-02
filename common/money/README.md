# money

Precise monetary value handling using fixed-point integer arithmetic.

## Features

- Fixed-point `int64` storage in the smallest currency unit (no float drift)
- Construction from smallest unit, decimal float, or string
- Arithmetic: Add, Sub, Mul, Div, DivMod, Neg, Abs
- Comparison: Equal, LessThan, GreaterThan, Compare, IsZero/Positive/Negative
- Proportional allocation with exact remainder distribution
- Configurable rounding modes (half-up, half-down, half-even, down, up)
- Built-in ISO 4217 currency precision & symbol tables

## Key functions

- `New(amount, currency)`, `FromDecimal(value, currency)`, `Parse(s, currency)`
- `Round(value, currency, mode)`
- `(m) Add/Sub/Mul/Div/DivMod/Neg/Abs`
- `(m) Equal/LessThan/GreaterThan/Compare/IsZero/IsPositive/IsNegative`
- `(m) Allocate(parts)`, `(m) Decimal/String/Format`
- `CurrencyPrecision(currency)`, `CurrencySymbol(currency)`

## Quick start

```go
import "github.com/LingByte/ling-base/common/money"

m := money.New(199, "USD")          // $1.99
m.Decimal()                         // 1.99
m.String()                          // "USD 1.99"

a := money.FromDecimal(10.50, "USD")
b := money.New(250, "USD")          // $2.50
sum, _ := a.Add(b)                  // $13.00

parsed, _ := money.Parse("12.34", "USD")

alloc, _ := money.New(100, "USD").Allocate([]int{1, 1, 1}) // 34 + 33 + 33 = 100
r := money.Round(2.345, "USD", money.RoundHalfUp)           // $2.35
```

## License

MIT
