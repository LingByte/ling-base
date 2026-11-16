# uaparse

Multi-engine User-Agent facade for analytics and request classification.

Does **not** hand-roll matching rules. Wraps mature parsers and merges results:

| Mode | Library | Best for |
|------|---------|----------|
| `ModeStandard` (default) | all four, uap-go as identity | 埋点 / 用户分析 |
| `ModeUAP` | [ua-parser/uap-go](https://github.com/ua-parser/uap-go) | 与前端 `ua-parser-js` 对齐 |
| `ModeFast` | [medama-io/go-useragent](https://github.com/medama-io/go-useragent) | 高 QPS 日志解析 |
| `ModeLite` | [mssola/user_agent](https://github.com/mssola/user_agent) | 快速判断移动端 / 浏览器 / Bot |
| `ModeWoothee` | [woothee/woothee-go](https://github.com/woothee/woothee-go) | 稳定分类、亚洲移动端 |

`device-detector` 未作为硬依赖（更重更慢）；多数产品场景用 `ModeStandard` 即可。

## Quick start

```go
import "github.com/LingByte/ling-base/common/uaparse"

r := uaparse.Parse(ua)
// r.Browser.Name / Version
// r.OS.Name / Version
// r.Device.Type / Brand / Model
// r.Engine, r.Category, r.Vendor
// r.Bot() / r.Mobile() / r.Tablet() / r.Desktop()

fast := uaparse.For(uaparse.ModeFast)
_ = fast.Parse(ua)
```

## ModeStandard merge policy

1. **Identity** (browser / OS / version / brand / model) → uap-go  
2. **Rendering engine** → mssola (+ heuristics)  
3. **Category / vendor / crawler** → woothee  
4. **Device class** (desktop/mobile/tablet/tv/bot/console) → medama + uap hints  

Parsers are process-wide cached via `For(mode)` / `Default()`.

## License

MIT
