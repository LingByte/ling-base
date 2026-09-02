# common/dnsutil

高级 DNS 查询库。基于 `miekg/dns`，支持自定义上游服务器、所有常见记录类型、DNSSEC、反向解析、并发多类型查询。

## 功能

- 支持 10+ 记录类型：A / AAAA / CNAME / MX / NS / TXT / SOA / PTR / SRV / CAA
- 可指定任意 DNS 服务器（Google / Cloudflare / 114 / 阿里 / Quad9 / OpenDNS）
- 反向 DNS 查询（IPv4 + IPv6）
- 并发查询所有记录类型（QueryAll）
- DNSSEC 支持
- UDP / TCP 自动切换

## 快速开始

```bash
go get github.com/LingByte/ling-base/common/dnsutil
```

### 基本查询

```go
import "github.com/LingByte/ling-base/common/dnsutil"

// 查询 A 记录
records, err := dnsutil.QueryA("google.com", dnsutil.ServerGoogle)
// → [{Name:"google.com.", Type:"A", TTL:300, Value:"142.250.191.14"}]

// 查询 MX 记录
records, _ = dnsutil.QueryMX("google.com", dnsutil.ServerCloudflare)

// 指定记录类型
records, _ := dnsutil.Query("example.com", "TXT", "1.1.1.1:53")

// 反向 DNS
records, _ := dnsutil.ReverseLookup("8.8.8.8", dnsutil.ServerGoogle)
```

### 并发查询所有类型

```go
result, err := dnsutil.QueryAll("example.com", "8.8.8.8:53")
// → AllRecords{A:[...], AAAA:[...], MX:[...], NS:[...], TXT:[...], ...}
```

### 可配置 Client

```go
client := dnsutil.NewClient(
    dnsutil.WithServer("223.5.5.5:53"),  // 阿里 DNS
    dnsutil.WithTimeout(10 * time.Second),
    dnsutil.WithDNSSEC(),
)
records, _ := client.Query("example.com", "A")
result, _ := client.QueryAll("example.com")
```

## 预置 DNS 服务器

| 常量 | 地址 |
|------|------|
| `ServerGoogle` | 8.8.8.8:53 |
| `ServerCloudflare` | 1.1.1.1:53 |
| `Server114` | 114.114.114.114:53 |
| `ServerAliDNS` | 223.5.5.5:53 |
| `ServerQuad9` | 9.9.9.9:53 |
| `ServerOpenDNS` | 208.67.222.222:53 |
