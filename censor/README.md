# censor

Unified content moderation across Aliyun, Tencent Cloud, and Qiniu for text, image, audio, and video.

## Structure

```
censor/
├── censor.go          # Core: shared types (CensorResult, JobSnapshot), constants, interfaces
├── text.go            # TextCensor interface
├── image.go           # ImageCensor interface
├── audio.go           # AudioCensor interface
├── video.go           # VideoCensor interface
├── qiniu/             # Independent module — only depends on Qiniu SDK
│   ├── qiniu.go       # Config struct
│   ├── text.go        # TextCensor impl + text_test.go
│   ├── image.go       # ImageCensor impl + image_test.go
│   ├── audio.go       # AudioCensor impl + audio_test.go
│   └── video.go       # VideoCensor impl + video_test.go
├── aliyun/            # Independent module — only depends on Aliyun Green SDK
│   ├── aliyun.go      # Config struct
│   └── ...            # 4 impls + 4 tests
└── qcloud/            # Independent module — only depends on Tencent Cloud SDK
    ├── qcloud.go      # Config struct
    └── ...            # 4 impls + 4 tests
```

## Design: Per-Provider Modules

Each provider is a **separate Go module** with its own `go.mod`. This means
applications only import the cloud SDK they actually use — no transitive
dependencies from providers you don't need.

```bash
# Only need Qiniu? Import just the Qiniu module:
go get github.com/LingByte/ling-base/censor/qiniu

# Only need Aliyun?
go get github.com/LingByte/ling-base/censor/aliyun

# Only need Tencent Cloud?
go get github.com/LingByte/ling-base/censor/qcloud
```

The core `censor` package has **zero cloud SDK dependencies** — it only
defines shared types and interfaces.

## Quick Start

### Text Moderation (Synchronous)

```go
import (
    "context"
    "fmt"

    "github.com/LingByte/ling-base/censor"
    qiniu "github.com/LingByte/ling-base/censor/qiniu"
)

func main() {
    c, err := qiniu.NewTextCensor(qiniu.Config{
        AccessKey: "your-qiniu-access-key",
        SecretKey: "your-qiniu-secret-key",
    })
    if err != nil {
        panic(err)
    }

    result, err := c.CensorText(context.Background(), "some text to check")
    if err != nil {
        panic(err)
    }
    fmt.Printf("suggestion=%s label=%s score=%.2f msg=%s\n",
        result.Suggestion, result.Label, result.Score, result.Msg)
}
```

### Image Moderation (Synchronous)

```go
import "github.com/LingByte/ling-base/censor/qcloud"

c, _ := qcloud.NewImageCensor(qcloud.Config{
    SecretID:  "your-secret-id",
    SecretKey: "your-secret-key",
    Region:    "ap-guangzhou",
})
result, err := c.CensorImage(ctx, "https://example.com/image.png")
```

### Audio/Video Moderation (Asynchronous)

Audio and video moderation are async: submit a task, then poll for results.

```go
import (
    "time"
    "github.com/LingByte/ling-base/censor"
    "github.com/LingByte/ling-base/censor/aliyun"
)

c, _ := aliyun.NewAudioCensor(aliyun.Config{
    AccessKeyID:     "your-aliyun-access-key-id",
    AccessKeySecret: "your-aliyun-access-key-secret",
})

// Step 1: Submit
taskID, err := c.SubmitCensorAudio(ctx, "https://example.com/audio.mp3")
if err != nil {
    panic(err)
}

// Step 2: Poll (retry with backoff in production)
for {
    snap, err := c.GetCensorResult(ctx, taskID)
    if err != nil {
        panic(err)
    }
    if snap.Status == censor.JobFinished {
        fmt.Printf("suggestion=%s label=%s\n", snap.Suggestion, snap.Label)
        break
    }
    if snap.Status == censor.JobFailed {
        panic(snap.Error)
    }
    time.Sleep(5 * time.Second)
}
```

## Provider Configs

Each provider has its own config struct with only the fields it needs:

### Qiniu (`qiniu.Config`)
| Field      | Description                        | Default             |
|------------|------------------------------------|---------------------|
| AccessKey  | Qiniu AccessKey                    | (required)          |
| SecretKey  | Qiniu SecretKey                    | (required)          |
| Host       | API host                           | `ai.qiniuapi.com`   |
| Client     | Custom `*http.Client`              | `http.DefaultClient`|

### Aliyun (`aliyun.Config`)
| Field            | Description              | Default                              |
|------------------|--------------------------|--------------------------------------|
| AccessKeyID      | Aliyun AccessKeyID       | (required)                           |
| AccessKeySecret  | Aliyun AccessKeySecret   | (required)                           |
| Endpoint         | Green CIP endpoint       | `green-cip.cn-shanghai.aliyuncs.com` |

### QCloud (`qcloud.Config`)
| Field      | Description              | Default         |
|------------|--------------------------|-----------------|
| SecretID   | Tencent Cloud SecretID   | (required)      |
| SecretKey  | Tencent Cloud SecretKey  | (required)      |
| Region     | Tencent Cloud region     | `ap-guangzhou`  |
| BizType    | Business type            | `default`       |

## Unified Types

All providers normalize their responses into the same types defined in the
core `censor` package:

- `censor.CensorResult` — for synchronous checks (text, image)
  - `Suggestion`: `pass` | `review` | `block`
  - `Label`: `normal` | `spam` | `ad` | `politics` | `terrorism` | `abuse` | `porn` | ...
  - `Score`: 0.0 to 1.0
  - `Details`: provider-specific extra info
  - `Msg`: human-readable English message

- `censor.JobSnapshot` — for async jobs (audio, video)
  - `Status`: `WAITING` | `DOING` | `FINISHED` | `FAILED`
  - `Suggestion` / `Label` / `Score`: populated when `FINISHED`
  - `Error`: populated when `FAILED`
  - `Raw`: original provider response for advanced inspection

### Context Support

All API methods accept `context.Context` for cancellation and timeouts.

## Testing

```bash
# Core package (no cloud SDK needed)
cd censor && go test -cover

# Per-provider modules (constructor validation + input checks)
cd censor/qiniu  && go test -cover
cd censor/aliyun && go test -cover
cd censor/qcloud && go test -cover
```

Coverage notes:
- `censor` (core): 100%
- Provider modules: constructor validation and empty-input paths are covered.
  Actual API call paths require real cloud credentials and are not tested
  in unit tests.
