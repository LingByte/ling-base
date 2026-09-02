# common/antispam

反垃圾（anti-spam）工具包，纯标准库实现，零第三方依赖。

## 功能

- **KeywordFilter**：基于 trie 的敏感词过滤，支持添加/批量添加/移除/匹配/替换/计数/清空，rune 感知（支持中文）。
- **RateLimiter**：基于内存 map + 滑动时间窗口的频率限制器，支持 Allow/Count/Reset/Cleanup。
- **ContentScorer**：内容评分器，基于重复字符、连续标点、URL 数量、大写比例、特殊字符比例等启发式信号计算 0-100 垃圾评分，`score >= 60` 判定为垃圾。
- **Checker**：综合反垃圾检查器，聚合关键词过滤、频率限制、长度校验、禁止正则、内容评分，通过 Options 模式配置。

## 使用示例

```go
package main

import (
    "fmt"
    "regexp"
    "time"
    "github.com/LingByte/ling-base/common/antispam"
)

func main() {
    checker := antispam.NewChecker(
        antispam.WithKeywords([]string{"赌博", "色情", "spam"}),
        antispam.WithRateLimit(time.Minute, 5),
        antispam.WithMinLength(2),
        antispam.WithMaxLength(1000),
        antispam.WithBannedPatterns([]*regexp.Regexp{
            regexp.MustCompile(`(?i)buy\s+now`),
        }),
    )

    res := checker.Check("这是赌博内容 buy now!!!", "user123")
    fmt.Println(res.Passed)        // false
    fmt.Println(res.Score)         // 垃圾评分
    fmt.Println(res.Reasons)       // 失败原因
    fmt.Println(res.MatchedKeywords) // ["赌博"]
}
```

## KeywordFilter

```go
kf := antispam.NewKeywordFilter([]string{"spam", "scam"})
kf.Add("fraud")
kf.Contains("this is spam")     // true
kf.Match("spam and scam")       // ["spam", "scam"]
kf.Replace("spam text", "***")  // "*** text"
kf.Remove("spam")
kf.Count()                      // 2
kf.Clear()
```

## RateLimiter

```go
rl := antispam.NewRateLimiter(time.Minute, 10)
rl.Allow("user1")  // true
rl.Count("user1")  // 1
rl.Reset("user1")
rl.Cleanup()       // 清理过期记录
```

## ContentScorer

评分规则（每项封顶后累加，总分封顶 100）：

| 信号 | 最大分 | 说明 |
|---|---|---|
| 重复字符 | 30 | 连续 ≥4 个相同字符占比 |
| 连续标点 | 25 | 连续标点 ≥3 个 |
| URL 数量 | 30 | 每个 URL +15 |
| 大写比例 | 15 | 字母中大写占比 >50% |
| 特殊字符 | 20 | 非字母/数字/空格占比 |

`IsSpam(score)` 当 `score >= 60` 返回 true。

## API

### Checker

| 方法/类型 | 说明 |
|---|---|
| `NewChecker(opts...) *Checker` | 创建检查器 |
| `WithKeywords(kws)` | 敏感词列表 |
| `WithRateLimit(window, max)` | 频率限制 |
| `WithMinLength(min)` | 最小长度（rune） |
| `WithMaxLength(max)` | 最大长度（rune，<=0 不限） |
| `WithBannedPatterns(ps)` | 禁止的正则模式 |
| `(*Checker) Check(content, userID) *Result` | 检查内容 |
| `Result{Passed, Score, Reasons, MatchedKeywords}` | 检查结果 |

### KeywordFilter

| 方法 | 说明 |
|---|---|
| `NewKeywordFilter(kws) *KeywordFilter` | 创建过滤器 |
| `Add(kw)` / `AddMany(kws)` | 添加 |
| `Remove(kw)` | 移除（自动剪枝） |
| `Match(text) []string` | 返回匹配关键词（去重） |
| `Contains(text) bool` | 是否包含 |
| `Replace(text, repl) string` | 替换 |
| `Count() int` / `Clear()` | 数量 / 清空 |

### RateLimiter

| 方法 | 说明 |
|---|---|
| `NewRateLimiter(window, max) *RateLimiter` | 创建限流器 |
| `Allow(key) bool` | 是否允许（记录请求） |
| `Count(key) int` | 当前窗口计数 |
| `Reset(key)` / `Cleanup()` | 重置 / 清理过期 |

### ContentScorer

| 方法 | 说明 |
|---|---|
| `NewContentScorer() *ContentScorer` | 创建评分器 |
| `Score(content) int` | 返回 0-100 评分 |
| `IsSpam(score) bool` | score >= 60 为垃圾 |

## 开发

```bash
GOWORK=off go mod tidy
GOWORK=off go build ./...
GOWORK=off go test ./... -short -cover
```
