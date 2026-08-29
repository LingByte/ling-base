# bitmap

精确位图（Bitmap / Bitset）抽象与多种后端实现。采用与 `cache` / `bloom` / `lock` 一致的**多 module 按需引入**：业务只 import 用到的驱动。

## 什么是 Bitmap

Bitmap 用一位表示一个 offset 是否置位，语义是**精确**的：

- `Get(offset) == true` → 该位一定为 1
- `Get(offset) == false` → 该位一定为 0

这与布隆过滤器不同：Bloom 可能假阳性，Bitmap 不会。典型场景：签到、在线状态、权限位、标签过滤、精确去重集合。

## 模块结构

```
ling-base/
├─ bitmap/              # module: .../bitmap  （接口 + 错误，纯标准库）
│  └─ bitmap.go         # Bitmap / Batcher / Snapshotter
├─ bitmap/memory/       # 稠密 []byte，可增长或固定容量
├─ bitmap/roaring/      # Roaring 压缩位图（稀疏大 offset）
└─ bitmap/redis/        # Redis SETBIT/GETBIT/BITCOUNT（分布式）
```

## 公共接口

```go
type Bitmap interface {
    Set(ctx context.Context, offset uint64) error
    Get(ctx context.Context, offset uint64) (bool, error)
    Clear(ctx context.Context, offset uint64) error
    Count(ctx context.Context) (uint64, error)
    Reset(ctx context.Context) error
    Close() error
}

type Batcher interface {
    SetBatch(ctx context.Context, offsets []uint64) error
    GetBatch(ctx context.Context, offsets []uint64) ([]bool, error)
    ClearBatch(ctx context.Context, offsets []uint64) error
}

// 进程重启后本地恢复用（memory / roaring）
type Snapshotter interface {
    WriteTo(w io.Writer) (int64, error)
    ReadFrom(r io.Reader) (int64, error)
}
```

## 后端对比

| 后端 | 结构 | 分布式 | 重启 | 依赖 | 适用 |
|---|---|---|---|---|---|
| memory | 稠密 `[]byte` | 否 | 默认丢失；可用 `WriteTo`/`ReadFrom` 快照 | 无 | 小宇宙、稠密（签到天、flag） |
| roaring | Roaring 压缩 | 否 | 默认丢失；Roaring 可移植序列化 | roaring/v2 | 稀疏大 ID 集合 |
| redis | Redis 位串 | 是 | 跟 Redis RDB/AOF；应用重启不丢 | go-redis | 多实例共享 |

> Redis 是稠密分配：对极大稀疏 offset 会按 `offset/8` 占内存。稀疏场景优先本地 roaring，或拆 key。

## 系统重启怎么办

| 策略 | 做法 |
|---|---|
| 可丢 | 纯 memory/roaring，重启后空集合或从业务表回填 |
| 跟存储 | 用 redis；依赖 Redis 持久化配置 |
| 本地可恢复 | memory/roaring 实现 `Snapshotter`：定期 `WriteTo(file)`，启动 `ReadFrom` |

## 快速开始

### memory（稠密本地）

```go
import "github.com/LingByte/ling-base/common/bitmap/memory"

bm, _ := memory.New(memory.WithFixed(366)) // 一年签到
_ = bm.Set(ctx, 42)
ok, _ := bm.Get(ctx, 42) // true
```

可增长（默认）：

```go
bm, _ := memory.New()
_ = bm.Set(ctx, 10000) // 自动扩容
```

快照（过重启）：

```go
f, _ := os.Create("bitmap.snap")
_, _ = bm.WriteTo(f)
_ = f.Close()

bm2, _ := memory.New()
f2, _ := os.Open("bitmap.snap")
_, _ = bm2.ReadFrom(f2)
```

### roaring（稀疏本地）

```go
import bmroaring "github.com/LingByte/ling-base/common/bitmap/roaring"

bm := bmroaring.New()
_ = bm.Set(ctx, 1_000_000)
n, _ := bm.Count(ctx)

a := bmroaring.New()
b := bmroaring.New()
_ = a.SetBatch(ctx, []uint64{1, 2, 3})
_ = b.SetBatch(ctx, []uint64{2, 3, 4})
_ = a.AndInPlace(b) // a ∩ b
```

### redis（分布式）

```go
import (
    goredis "github.com/redis/go-redis/v9"
    bmredis "github.com/LingByte/ling-base/common/bitmap/redis"
)

bm, _ := bmredis.New(&goredis.Options{Addr: "127.0.0.1:6379"},
    bmredis.WithKey("user:online"),
    bmredis.WithTTL(24*time.Hour), // 可选
)
_ = bm.Set(ctx, userID)
_ = bm.SetBatch(ctx, []uint64{1, 2, 3})
```

Redis `BITOP`（按 key）：

```go
_ = bm.BitOpOr(ctx, "dest", "a", "b")
```

## 安装

```bash
go get github.com/LingByte/ling-base/common/bitmap
go get github.com/LingByte/ling-base/common/bitmap/memory
go get github.com/LingByte/ling-base/common/bitmap/roaring
go get github.com/LingByte/ling-base/common/bitmap/redis
```

## 测试与基准

```bash
# 单测 + 压力（精确性 / 并发）
go test ./common/bitmap/memory ./common/bitmap/roaring ./common/bitmap/redis

# 性能
go test ./common/bitmap/memory ./common/bitmap/roaring ./common/bitmap/redis \
  -bench=. -benchmem -benchtime=300ms
```
