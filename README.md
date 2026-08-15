# ling-base

Go 基础工具库。采用**多 module 按需引入**：业务只 import 用到的驱动，不会把无关 SDK 拉进依赖树。

## 模块结构

```
ling-base/
├─ go.mod / go.work          # 仓库锚点 + 本地多 module 开发
├─ logger/                   # 结构化日志（zap + lumberjack）
├─ constants/                # 全局常量
│
├─ cache/                    # module: .../cache  （接口 + 纯标准库实现）
│  ├─ lru / memory / noop / multilevel
│  ├─ bigcache/              # 独立 module，依赖 allegro/bigcache
│  ├─ redis/                 # 独立 module，依赖 go-redis
│  ├─ memcache / freecache / ristretto
│
├─ lock/                     # module: .../lock   （接口 + memory）
│  ├─ redis / redlock / etcd / zookeeper / consul / mysql / postgres
│
├─ bloom/                    # module: .../bloom  （接口 + 估算 + 共享哈希）
│  ├─ memory / counting / scalable   # 独立 module，纯标准库
│  └─ redis / redisbloom      # 独立 module，依赖 go-redis
│
├─ captcha/                  # module: .../captcha （验证码：滑块/点选/拼图/算术/旋转）
│
├─ censor/                   # module: .../censor  （内容审核接口）
│  ├─ aliyun/                # 独立 module，依赖阿里云 SDK
│  ├─ qcloud/                # 独立 module，依赖腾讯云 SDK
│  └─ qiniu/                 # 独立 module，依赖七牛 SDK
│
├─ stores/                   # module: .../stores  （对象存储接口）
│  ├─ local/                 # 本地文件系统（零云 SDK）
│  ├─ s3/ oss/ cos/ minio/   # 独立 module，各引各的 SDK
│  └─ kodo/ tos/ obs/ ks3/
│
└─ i18n/                     # module: .../i18n   （国际化：翻译 + 格式化 + locale 检测）
   ├─ gin/                   # 独立 module，Gin 中间件
   └─ mymemory/              # 独立 module，MyMemory 机器翻译
```

本地开发使用已提交的 `go.work`；发布后消费者**不需要** `go.work`，直接 `go get` 子模块即可。

## 按需安装示例

```bash
# 只要 LRU（零第三方依赖）
go get github.com/LingByte/ling-base/cache

# 只要 BigCache 驱动（只会拉 bigcache + cache 抽象）
go get github.com/LingByte/ling-base/cache/bigcache

# 只要 Redis 分布式锁
go get github.com/LingByte/ling-base/lock/redis

# 只要 S3 对象存储
go get github.com/LingByte/ling-base/stores/s3

# 只要 i18n 核心（零外部依赖）
go get github.com/LingByte/ling-base/i18n

# 要 Gin i18n 中间件
go get github.com/LingByte/ling-base/i18n/gin
```

```go
import (
    "github.com/LingByte/ling-base/cache"
    "github.com/LingByte/ling-base/cache/bigcache" // 不会间接引入 redis/etcd/...
)
```

## 文档

- [cache/README.md](cache/README.md)
- [lock/README.md](lock/README.md)
- [bloom/README.md](bloom/README.md)
- [captcha/README.md](captcha/README.md)
- [censor/README.md](censor/README.md)
- [stores/README.md](stores/README.md)
- [i18n/README.md](i18n/README.md)

## 开发

```bash
go work sync

# 测试所有模块（根模块 + 子模块各自独立测试）
go test ./...
for mod in $(find . -name go.mod -not -path ./go.mod | xargs -I{} dirname {}); do
  (cd "$mod" && go test ./...)
done

# 格式化
gofmt -w .

# vet
go vet ./...
```
