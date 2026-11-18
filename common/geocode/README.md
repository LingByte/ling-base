# common/geocode

经纬度定位与逆地理编码。使用免费、无需 API Key 的服务。

## 功能

- **正向地理编码**：地址 → 经纬度（Nominatim）
- **逆向地理编码**：经纬度 → 地址（Nominatim / BigDataCloud）
- **Context / 限流 / Accept-Language**
- **Haversine 距离**与半径判断

## Provider

| Provider | 免费 | API Key | 速率限制 | 正向 | 逆向 |
|----------|------|---------|----------|------|------|
| Nominatim (OSM) | ✅ | 不需要 | 1 req/sec | ✅ | ✅ |
| BigDataCloud | ✅ | 不需要 | 无（客户端向） | ❌ | ✅ |

## 快速开始

```bash
go get github.com/LingByte/ling-base/common/geocode
```

```go
ctx := context.Background()
client := geocode.NewClient(
    geocode.WithLimiter(geocode.NewIntervalLimiter(time.Second)),
    geocode.WithLanguage("zh"),
    geocode.WithFallback(true),
)

addr, err := client.Reverse(ctx, 39.9042, 116.4074)
result, err := client.Forward(ctx, "Eiffel Tower, Paris")

// 包级便捷函数会复用内置 defaultClient（已带 1s 间隔限流）
addr, err = geocode.Reverse(39.9042, 116.4074)
```

## 错误类型

- `geocode.ErrNotFound`
- `geocode.ErrRateLimited`
- `geocode.ErrEmptyQuery`

## 注意

- Nominatim 要求设置 User-Agent（默认已设置）并遵守 1 req/sec；生产请 `WithLimiter`
- BigDataCloud 服务端批量建议自备 key / 缓存
