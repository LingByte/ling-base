# money

Precise monetary value handling using fixed-point `int64` arithmetic (smallest currency unit).

## Features

- Fixed-point storage (no float drift in arithmetic)
- Strict `Parse` + optional `ParseTrunc`
- `Round` / `FromDecimal` return errors (no silent zero on bad floats)
- Overflow-checked Add / Sub / Mul / Allocate
- Same-currency `LessThan` / `GreaterThan` / `CompareAmount` (cross-currency = error)
- `Compare` kept for sortable collections (lexicographic on currency when mismatched)
- Custom currencies via `RegisterCurrency`
- FX: `Convert` / `RateTable` for display; `ConvertRatio(num,den)` for settlement-safe rational rates

## Quick start

```go
import "github.com/LingByte/ling-base/common/money"

m := money.New(199, "USD")
parsed, err := money.Parse("12.34", "USD") // excess decimals => error
trunc, err := money.ParseTrunc("12.3499", "USD")

a, _ := money.FromDecimal(10.50, "USD")
sum, _ := a.Add(money.New(250, "USD"))

alloc, _ := money.New(100, "USD").Allocate([]int{1, 1, 1}) // 34+33+33

_ = money.RegisterCurrency(money.Currency{Code: "POINTS", Precision: 0, Symbol: "P"})

rates := money.NewRateTable()
_ = rates.Set("USD", "CNY", 7.2)
cny, _ := rates.Convert(money.New(100, "USD"), "CNY", money.RoundHalfUp)

// settlement-safe: 1 USD = 7.2 CNY
cny2, _ := money.New(100, "USD").ConvertRatio("CNY", 72, 10, money.RoundHalfUp)
```

## Notes

- Prefer `Parse` for external input; `FromDecimal` / `Round` are for controlled floats.
- Do not use `LessThan`/`GreaterThan` across currencies — they return `ErrCurrencyMismatch`.
- `Allocate` uses integer largest-remainder (no float fraction ranking).
- `Neg` / `Abs` / `Div(-1)` return `ErrOverflow` on `MinInt64`.
- Builtin ISO currencies cannot be overridden via `RegisterCurrency`.

## License

MIT
