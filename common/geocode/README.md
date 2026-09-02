# common/geocode

经纬度定位与逆地理编码库。使用免费、无需 API Key 的服务。

## 功能

- **正向地理编码** (Forward Geocoding)：地址 → 经纬度
- **逆向地理编码** (Reverse Geocoding)：经纬度 → 地址
- **距离计算** (Haversine)：两点间球面距离
- **半径判断**：判断两点是否在指定半径内

## 支持的 Provider

| Provider | 免费 | API Key | 速率限制 | 正向 | 逆向 | 覆盖 |
|----------|------|---------|----------|------|------|------|
| **Nominatim (OSM)** | ✅ | 不需要 | 1 req/sec | ✅ | ✅ | 全球 |
| **BigDataCloud** | ✅ | 不需要 | 无（客户端） | ❌ | ✅ | 全球 |

## 快速开始

```bash
go get github.com/LingByte/ling-base/common/geocode
```

### 逆向地理编码

```go
package main

import (
    "fmt"
    "github.com/LingByte/ling-base/common/geocode"
)

func main() {
    // 默认使用 Nominatim
    result, err := geocode.Reverse(39.9042, 116.4074) // 天安门
    if err != nil {
        panic(err)
    }
    fmt.Printf("地址: %s\n", result.DisplayName)
    fmt.Printf("国家: %s (%s)\n", result.Country, result.CountryCode)
    fmt.Printf("城市: %s\n", result.Locality)

    // 使用 BigDataCloud（无速率限制）
    client := geocode.NewClient(geocode.WithProvider(geocode.ProviderBigDataCloud))
    result, err = client.ReverseBigDataCloud(39.9042, 116.4074)
}
```

### 正向地理编码

```go
// 自由文本查询
result, err := geocode.Forward("Eiffel Tower, Paris")
fmt.Printf("坐标: %.4f, %.4f\n", result.Lat, result.Lon)

// 结构化查询
result, err = geocode.ForwardStructured(&geocode.GeocodeQuery{
    City:    "London",
    Country: "UK",
})
```

### 距离计算

```go
// 北京 → 上海 距离 (km)
dist := geocode.HaversineDistance(39.9042, 116.4074, 31.2304, 121.4737)
fmt.Printf("距离: %.0f km\n", dist) // ~1067 km

// 判断是否在半径内
inRadius := geocode.IsInRadius(lat1, lon1, lat2, lon2, 5.0) // 5km
```

## 注意事项

- **Nominatim 速率限制**：1 req/sec，生产环境请加缓存
- **User-Agent**：Nominatim 要求设置 User-Agent，默认已设置
- **BigDataCloud**：客户端场景免费，服务端批量使用建议加 key
