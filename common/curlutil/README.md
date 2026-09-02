# common/curlutil

curl 命令解析 + 调试型 HTTP 客户端。专为构建 HTTP 调试工具设计。

## 功能

- **curl 命令解析**：将 curl 命令字符串解析为结构化请求
- **调试型 HTTP 客户端**：重定向链追踪、TLS 版本检测、二进制识别
- **十六进制预览**：二进制响应的 hex+ASCII 预览
- **简单 API**：Get / Post / Head 快捷函数

## 快速开始

```bash
go get github.com/LingByte/ling-base/common/curlutil
```

### 解析 curl 命令

```go
import "github.com/LingByte/ling-base/common/curlutil"

req, err := curlutil.ParseCurlCommand(`curl -X POST https://httpbin.org/post -H "Content-Type: application/json" -d '{"key":"value"}' -L -k`)
// → &Request{URL:"https://httpbin.org/post", Method:"POST", Headers:{"Content-Type":"application/json"}, Body:`{"key":"value"}`, FollowRedirect:true, VerifySSL:false}
```

### 执行请求（带调试信息）

```go
resp, err := curlutil.Execute(req)
fmt.Printf("Status: %d\n", resp.StatusCode)
fmt.Printf("Time: %dms\n", resp.ResponseTime)
fmt.Printf("TLS: %s\n", resp.RequestInfo.TLSVersion)
fmt.Printf("Redirects: %v\n", resp.RedirectChain)
fmt.Printf("Binary: %v\n", resp.IsBinary)
```

### 快捷函数

```go
resp, _ := curlutil.Get("https://example.com")
resp, _ := curlutil.Post("https://httpbin.org/post", `{"key":"value"}`)
resp, _ := curlutil.Head("https://example.com")
```

### 支持的 curl 参数

| 参数 | 说明 |
|------|------|
| `-X` / `--request` | HTTP 方法 |
| `-H` / `--header` | 请求头 |
| `-d` / `--data` / `--data-raw` | 请求体（自动转 POST） |
| `-k` / `--insecure` | 跳过 SSL 验证 |
| `-L` / `--location` | 跟随重定向 |
| `-I` / `--head` | HEAD 请求 |
| `-A` / `--user-agent` | User-Agent |
| `-e` / `--referer` | Referer |
| `-u` / `--user` | Basic 认证 |
| `--compressed` | Accept-Encoding |
| `--max-time` | 超时（秒） |

### Response 结构

```go
type Response struct {
    StatusCode    int               // HTTP 状态码
    Headers       map[string]string // 响应头
    Body          string            // 响应体（文本）
    BodyPreview   string            // 预览（截断或 hex）
    IsBinary      bool              // 是否二进制
    ResponseTime  int64             // 响应时间（ms）
    RedirectChain []string          // 重定向链
    RequestInfo   RequestInfo       // 请求元信息
}

type RequestInfo struct {
    FinalURL       string           // 最终 URL
    RemoteAddr     string           // 远程地址
    Protocol       string           // HTTP 协议版本
    TLSVersion     string           // TLS 版本
    RequestHeaders map[string]string // 实际发送的请求头
}
```
