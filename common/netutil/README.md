# netutil

网络工具合集：端口/IP、HTTP Client、TCP/UDP，以及 DNS 查询与 curl 调试能力（原独立 `dnsutil` / `curlutil` 已并入本包）。

## Features

- Port availability: check, find, and allocate free ports (TCP/UDP)
- IP address helpers: local, outbound, public, private/loopback detection
- HTTP client with configurable retry, timeout, and circuit breaker
- TCP/UDP utilities: dial, listen, ping, forward
- DNS queries: custom upstream, common record types, reverse lookup, concurrent QueryAll
- Curl debug: parse curl command strings and execute with redirect/TLS/binary inspection

## Key types

- `HTTPClientConfig` / `HTTPClient` — HTTP client with retry and circuit breaker
- `DNSClient` / `DNSRecord` / `AllRecords` — advanced DNS querying
- `CurlRequest` / `CurlResponse` — curl-style debug HTTP request/response

## Quick start

```go
import (
    "time"
    "github.com/LingByte/ling-base/common/netutil"
)

ok := netutil.IsPortAvailable("tcp", 8080)
ip := netutil.LocalIP()

client := netutil.NewHTTPClient(netutil.HTTPClientConfig{
    Timeout:       10 * time.Second,
    MaxRetries:    3,
    RetryInterval: time.Second,
})
resp, err := client.Get("https://example.com")

// DNS
records, _ := netutil.QueryA("google.com", netutil.ServerGoogle)
all, _ := netutil.QueryAll("example.com", "8.8.8.8:53")

// curl debug
req, _ := netutil.ParseCurlCommand(`curl -X POST https://httpbin.org/post -d '{"a":1}' -L`)
debug, _ := netutil.ExecuteCurl(req)
```
