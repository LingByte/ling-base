# constant

Global constants and enums for the relay system: channel types, API types, relay modes, endpoint types, context keys, and task/Midjourney action mappings.

## Key Files

| File | Contents |
|------|----------|
| `channel.go` | `ChannelType*` constants, `ChannelBaseURLs`, `ChannelTypeNames`, `GetChannelTypeName` |
| `api_type.go` | `APIType*` constants used for adaptor dispatch |
| `endpoint_type.go` | `EndpointType*` aliases (delegated to `relaykit/types`) |
| `context_key.go` | `ContextKey` string type and all context key constants |
| `task.go` | `TaskPlatform`, `TaskAction*`, `SunoAction*` constants |
| `midjourney.go` | `MjAction*` constants and `MidjourneyModel2Action` map |
| `finish_reason.go` | `FinishReason*` aliases (delegated to `relaykit/types`) |
| `env.go` | Global environment variables (timeouts, limits, feature flags) |
| `cache_key.go` | Cache key format strings |
| `multi_key_mode.go` | `MultiKeyMode` (random / polling) |
| `setup.go` | `Setup` flag |
| `azure.go` | Azure-specific timestamp constant |

## Usage

```go
import "github.com/LingByte/ling-base/relay/constant"

name := constant.GetChannelTypeName(constant.ChannelTypeOpenAI) // "OpenAI"
```
