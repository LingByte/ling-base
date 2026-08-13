# ling-base

Go 基础工具库。采用**多 module 按需引入**：业务只 import 用到的驱动，不会把无关 SDK 拉进依赖树。

## 模块结构

```
ling-base/
├─ go.mod / go.work          # 仓库锚点 + 本地多 module 开发
├─ cache/                    # module: .../cache  （接口 + 纯标准库实现）
│  ├─ lru / memory / noop / multilevel
│  ├─ bigcache/              # 独立 module，依赖 allegro/bigcache
│  ├─ redis/                 # 独立 module，依赖 go-redis
│  ├─ memcache / freecache / ristretto
├─ lock/                     # module: .../lock   （接口 + memory）
│  ├─ redis / redlock / etcd / zookeeper / consul / mysql / postgres
└─ bloom/                    # module: .../bloom  （接口 + 估算 + 共享哈希）
   ├─ memory / counting / scalable   # 独立 module，纯标准库
   └─ redis / redisbloom      # 独立 module，依赖 go-redis
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

## 开发

```bash
go work sync
go test ./cache/...
go test ./lock/...
go test ./bloom/...
# 或按 module：
(cd cache/bigcache && go test ./...)
(cd bloom/redis && go test ./...)
```
