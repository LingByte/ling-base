# signature

Request signing and verification utilities using only the Go standard library.

## Key functions

- `SignHMACSHA256(data string, key []byte) string` / `VerifyHMACSHA256(...)` — HMAC-SHA256 (hex)
- `SignHMACSHA512(data string, key []byte) string` / `VerifyHMACSHA512(...)` — HMAC-SHA512 (hex)
- `SignMD5(data string) string` — MD5 (hex, for legacy systems)
- `SignRequest(method, url string, params url.Values, body []byte, key []byte) string` — standard request signing (`method\nurl\nsorted params\nbody\ntimestamp`)
- `VerifyRequest(method, url string, params url.Values, body []byte, key []byte, sig string, maxAge time.Duration) bool` — verify signature and freshness (reads `timestamp` from params)
- `SignSortedParams(params url.Values, key string) string` — WeChat/Alipay-style sorted-params MD5 signing (upper-case hex)

## Quick start

```go
import "github.com/LingByte/ling-base/common/signature"

sig := signature.SignHMACSHA256("data", []byte("secret"))
signature.VerifyHMACSHA256("data", []byte("secret"), sig) // true

signature.SignMD5("hello") // "5d41402abc4b2a76b9719d911017c592"

params := url.Values{}
params.Set("foo", "bar")
params.Set("abc", "123")
signature.SignSortedParams(params, "key")
```
