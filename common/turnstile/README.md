# common/turnstile

Cloudflare Turnstile 服务端验证客户端 + 前端 widget 渲染。仅依赖 Go 标准库。

## 功能

- **Token 验证**：调用 Cloudflare siteverify API 验证 Turnstile token
- **请求验证**：从 HTTP 请求中自动提取 token 和客户端 IP（正确处理 IPv6）
- **Widget 渲染**：生成 Turnstile 前端 widget HTML（支持回调函数）
- **可配置**：自定义 HTTP 客户端、siteverify 端点（便于测试）

## 快速开始

```bash
go get github.com/LingByte/ling-base/common/turnstile
```

### 验证 token

```go
import (
    "context"
    "github.com/LingByte/ling-base/common/turnstile"
)

client := turnstile.NewClient("0xsecretKey", "0xsiteKey")

// 直接验证 token
resp, err := client.Verify(context.Background(), "turnstile-token", "1.2.3.4")
if err != nil { /* 网络/API 错误 */ }
if !resp.Success { /* 验证失败，查看 resp.ErrorCodes */ }

// 从 HTTP 请求中验证（自动提取 token 和客户端 IP）
resp, err = client.VerifyRequest(context.Background(), r, "cf-turnstile-response")
```

### 渲染 widget

```go
client := turnstile.NewClient("0xsecretKey", "0xsiteKey")

// 基本 widget
html := client.RenderHTML()

// 带回调函数
html := client.RenderHTMLWithCallback("onTurnstileSuccess")
```

## API

| 方法 | 说明 |
|------|------|
| `NewClient(secret, siteKey)` | 创建客户端 |
| `Verify(ctx, token, remoteIP)` | 验证 token |
| `VerifyRequest(ctx, r, fieldName)` | 从 HTTP 请求提取并验证 |
| `RenderHTML()` | 渲染 widget HTML |
| `RenderHTMLWithCallback(cb)` | 渲染带回调的 widget HTML |
| `IsTokenValid(token)` | 基本格式检查 |

## 测试

```bash
go test ./... -v
```
