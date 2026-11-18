# common/media

媒体处理合集（原独立 `imageutil` / `audioutil` / `videoutil`）。

| 子模块 | Module 路径 | Go package |
|--------|-------------|------------|
| image | `github.com/LingByte/ling-base/common/media/image` | `imageutil` |
| audio | `github.com/LingByte/ling-base/common/media/audio` | `audioutil` |
| video | `github.com/LingByte/ling-base/common/media/video` | `videoutil` |

包名刻意保留 `*util`，避免与标准库 `image` 等冲突；import path 使用 `media/image` 等新前缀。

```bash
go get github.com/LingByte/ling-base/common/media/image
go get github.com/LingByte/ling-base/common/media/audio
go get github.com/LingByte/ling-base/common/media/video
```
