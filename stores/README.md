# stores

Unified object storage abstraction across 9 backends with zero cloud SDK
dependencies in the core module.

## Structure

```
stores/
├── stores.go          # Core: Store interface, StoreError, sentinel errors
├── private.go         # PrivateURLSigner, DirectUploadPresigner, helpers
├── local/             # Local filesystem (no cloud SDK)
├── s3/                # AWS S3 / S3-compatible
├── oss/               # Alibaba Cloud OSS
├── cos/               # Tencent Cloud COS
├── minio/             # MinIO
├── kodo/              # Qiniu Kodo
├── tos/               # Volcengine TOS
├── obs/               # Huawei Cloud OBS (S3-compatible)
└── ks3/               # Kingsoft Cloud KS3
```

## Design: Per-Provider Modules

Each provider is a **separate Go module** with its own `go.mod`. Applications
only import the cloud SDK they actually use — no transitive dependencies from
providers you don't need.

```bash
# Only need S3?
go get github.com/LingByte/ling-base/stores/s3

# Only need Alibaba Cloud OSS?
go get github.com/LingByte/ling-base/stores/oss

# Only need local filesystem (no cloud SDK at all)?
go get github.com/LingByte/ling-base/stores/local
```

The core `stores` package has **zero cloud SDK dependencies** — it only
defines the `Store` interface, error types, and signing/presign helpers.

## Quick Start

### Local Filesystem

```go
import "github.com/LingByte/ling-base/stores/local"

s := local.New(local.Config{
    Root:       "/var/uploads",
    NewDirPerm: 0755,
})

// Write
s.Write("photos/cat.jpg", file)

// Read
r, size, err := s.Read("photos/cat.jpg")
defer r.Close()

// Delete
s.Delete("photos/cat.jpg")

// Exists
ok, _ := s.Exists("photos/cat.jpg")
```

### Amazon S3

```go
import "github.com/LingByte/ling-base/stores/s3"

s := s3.New(s3.Config{
    Region:          "us-east-1",
    AccessKeyID:     "your-access-key-id",
    AccessKeySecret: "your-secret-access-key",
    BucketName:      "my-bucket",
})

s.Write("file.txt", reader)
r, size, _ := s.Read("file.txt")
```

### Alibaba Cloud OSS

```go
import "github.com/LingByte/ling-base/stores/oss"

s := oss.New(oss.Config{
    AccessKeyID:     "your-access-key-id",
    AccessKeySecret: "your-access-key-secret",
    Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
    BucketName:      "my-bucket",
})
```

### Tencent Cloud COS

```go
import "github.com/LingByte/ling-base/stores/cos"

s := cos.New(cos.Config{
    SecretID:   "your-secret-id",
    SecretKey:  "your-secret-key",
    Region:     "ap-guangzhou",
    BucketName: "my-bucket-1250000000",
})
```

### MinIO

```go
import "github.com/LingByte/ling-base/stores/minio"

s := minio.New(minio.Config{
    Endpoint:  "minio.local:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Bucket:    "my-bucket",
    UseSSL:    false,
})
```

### Qiniu Kodo

```go
import "github.com/LingByte/ling-base/stores/kodo"

s := kodo.New(kodo.Config{
    AccessKey:  "your-access-key",
    SecretKey:  "your-secret-key",
    BucketName: "my-bucket",
    Domain:     "https://cdn.example.com",
    Private:    true,
})
```

### Volcengine TOS

```go
import "github.com/LingByte/ling-base/stores/tos"

s := tos.New(tos.Config{
    Endpoint:        "https://tos-cn-beijing.volces.com",
    Region:          "cn-beijing",
    AccessKeyID:     "your-access-key-id",
    AccessKeySecret: "your-access-key-secret",
    BucketName:      "my-bucket",
})
```

### Huawei Cloud OBS

```go
import "github.com/LingByte/ling-base/stores/obs"

s := obs.New(obs.Config{
    Endpoint:        "https://obs.cn-north-4.myhuaweicloud.com",
    Region:          "cn-north-4",
    AccessKeyID:     "your-access-key-id",
    AccessKeySecret: "your-access-key-secret",
    BucketName:      "my-bucket",
})
```

### Kingsoft Cloud KS3

```go
import "github.com/LingByte/ling-base/stores/ks3"

s := ks3.New(ks3.Config{
    Endpoint:        "https://ks3-cn-beijing.ksyuncs.com",
    Region:          "BEIJING",
    AccessKeyID:     "your-access-key-id",
    AccessKeySecret: "your-access-key-secret",
    BucketName:      "my-bucket",
})
```

## Store Interface

Every backend implements the same 5-method interface:

```go
type Store interface {
    Read(key string) (io.ReadCloser, int64, error)
    Write(key string, r io.Reader) error
    Delete(key string) error
    Exists(key string) (bool, error)
    PublicURL(key string) string
}
```

## Private URLs & Direct Upload

Backends that support private buckets implement `PrivateURLSigner`:

```go
if signer, ok := s.(stores.PrivateURLSigner); ok {
    url, _ := signer.SignedURL("file.txt", time.Hour)
}
```

Or use the helper:

```go
url, err := stores.SignedURL(s, "file.txt", time.Hour)
```

Backends that support client direct upload implement `DirectUploadPresigner`:

```go
du, err := stores.PresignUpload(s, "file.txt", "image/jpeg", time.Hour)
// du.Method, du.URL, du.Headers, du.Form, du.FileField
```

## Configuration

All configuration is **explicit** — no environment variables are read inside
the library. Each provider has its own `Config` struct with only the fields it
needs.

## Testing

```bash
# Core package (no cloud SDK needed)
cd stores && go test -cover

# Per-provider modules
cd stores/local && go test -cover
cd stores/s3    && go test -cover
cd stores/oss   && go test -cover
# ... etc
```

Coverage notes:
- `stores` (core): ~90%
- `stores/local`: ~81% (full filesystem CRUD tested)
- Cloud providers: constructor + PublicURL paths tested locally; actual
  cloud API calls require real credentials and are not unit-tested.
