# types

Generic data structures used across the relay system.

## Key Types

- **`PriceData`** -- holds pricing ratios and quota information for a model (model price, completion ratio, cache ratio, image/audio ratios, group ratio, custom "other" ratios). Supports applying and removing other-ratio multipliers on float and `decimal.Decimal` values.
- **`GroupRatioInfo`** -- group-level pricing ratio with optional special ratio.
- **`RWMap[K, V]`** -- generic concurrent read/write map with `sync.RWMutex`. Supports JSON marshal/unmarshal, `Get`, `Set`, `AddAll`, `Clear`, `ReadAll`, `Len`, and `LoadFromJsonString`.
- **`Set[T]`** -- generic set with `Add`, `Remove`, `Contains`, `Len`, and `Items`.

## Usage

```go
m := types.NewRWMap[string, int]()
m.Set("a", 1)
val, ok := m.Get("a")

s := types.NewSet[string]()
s.Add("x")
s.Contains("x") // true
```
