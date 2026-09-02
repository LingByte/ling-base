# abtest

Deterministic A/B test (experiment) bucketing using consistent hashing.

A user is assigned to a variant of an experiment using a consistent hash of
`(userID, experimentName)` so the same user always lands in the same variant.
Variant selection is weighted.

## Key types & functions

- `type Variant struct { Name string; Weight int }`
- `type Experiment struct { Name string; Variants []Variant }`
- `type Assigner struct { ... }`
- `NewAssigner() *Assigner`
- `(*Assigner) AddExperiment(exp *Experiment)`
- `(*Assigner) Assign(userID string) map[string]string` — all experiments
- `(*Assigner) AssignOne(expName, userID string) (string, error)` — single experiment
- `HashPercentage(userID, experimentName string) float64` — deterministic [0,1) value (FNV-1a)

## Quick start

```go
import "github.com/LingByte/ling-base/common/abtest"

a := abtest.NewAssigner()
a.AddExperiment(&abtest.Experiment{
    Name: "button-color",
    Variants: []abtest.Variant{
        {Name: "red", Weight: 1},
        {Name: "blue", Weight: 1},
    },
})
a.Assign("user-123")                 // map[string]string{"button-color": "blue"}
a.AssignOne("button-color", "user-123") // "blue"
```
