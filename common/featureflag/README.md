# featureflag

A thread-safe feature-flag manager supporting full on/off toggles,
percentage-based gradual rollouts (deterministic via FNV hashing), and
per-user whitelists/blacklists.

## Evaluation order

`IsEnabled(name, userID)` evaluates rules in order and short-circuits:

1. Flag disabled → `false`
2. User in blacklist → `false`
3. User in whitelist → `true`
4. Percentage rollout: enabled when `hash(name+userID) % 100 < Percentage`

The FNV-1a hash guarantees the **same user always gets the same result**
for a given flag, which is essential for consistent UX during rollouts.

## Quick start

```go
import "github.com/LingByte/ling-base/common/featureflag"

m := featureflag.NewManager()

// Full rollout
m.Enable("new-ui")

// 30% gradual rollout
m.SetPercentage("new-api", 30)

// Always-on for specific users regardless of percentage
m.AddWhitelist("new-api", "beta-tester-1")

// Block a specific user
m.SetFlag(&featureflag.Flag{
    Name:      "new-api",
    Enabled:   true,
    Percentage: 100,
    Blacklist: []string{"abuser"},
})

if m.IsEnabled("new-api", "beta-tester-1") {
    // serve new API
}
```

## API

| Method | Description |
| --- | --- |
| `NewManager()` | Create an empty manager |
| `SetFlag(*Flag)` | Insert/update a flag |
| `IsEnabled(name, userID)` | Evaluate a flag for a user |
| `Enable(name)` | Full on (100%) |
| `Disable(name)` | Full off |
| `SetPercentage(name, pct)` | Set rollout %, clamped to [0,100] |
| `AddWhitelist(name, userID)` | Always-on for a user |
| `RemoveFlag(name)` | Delete a flag |
| `GetFlag(name)` | Get a copy of a flag |
| `AllFlags()` | Get copies of all flags, sorted by name |

All operations are safe for concurrent use (`sync.RWMutex`).
