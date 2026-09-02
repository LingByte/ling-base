# common/consistent

一致性哈希（consistent hashing）实现，纯标准库，零第三方依赖。支持虚拟节点、自定义哈希函数、副本放置（GetN）和快照。

## 特性

- **虚拟节点**：每个物理节点映射多个虚拟节点到环上（默认 50），改善 key 分布均匀性。
- **最小迁移**：增删节点时只有相邻区间的 key 会重新映射，其余 key 保持不变。
- **自定义哈希**：默认 `crc32.ChecksumIEEE`，可通过 `WithHash` 替换。
- **副本放置**：`GetN(key, n)` 返回 key 在环上顺时针方向的 n 个不同物理节点。
- **快照**：`Snapshot()` 返回独立副本，用于一致性点读。
- **线程安全**：所有方法用 `sync.RWMutex` 保护。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/LingByte/ling-base/common/consistent"
)

func main() {
    ring := consistent.New(
        consistent.WithReplicas(150),
    )
    ring.Add("cache1", "cache2", "cache3")

    // 单节点路由
    node, ok := ring.Get("user:42")
    fmt.Println(node, ok) // 同一 key 始终路由到同一 node

    // 副本放置：取 3 个不同节点
    replicas, _ := ring.GetN("user:42", 3)
    fmt.Println(replicas)

    // 动态扩容：只影响相邻区间
    ring.Add("cache4")

    // 分布统计
    fmt.Println(ring.Distribution()) // map[cache1:150 cache2:150 ...]
}
```

## 一致性原理

一致性哈希将节点和 key 都映射到同一个哈希环（0 ~ 2^32-1）。`Get(key)` 从 key 的哈希位置顺时针找到第一个虚拟节点，返回其所属的物理节点。

**为什么虚拟节点？** 当物理节点很少时，环上的区间划分不均。为每个物理节点创建多个虚拟节点（如 50~150 个）可使区间分布近似均匀，避免数据倾斜。

**最小迁移特性**：新增节点 N 时，只有原本落在 N 的区间内的 key 会迁移到 N，其余 key 不变。删除节点同理。这使得节点拓扑变化时缓存失效最小化。

## API

| 方法/类型 | 说明 |
|---|---|
| `New(opts...) *Ring` | 创建空哈希环 |
| `WithHash(fn)` | 自定义哈希函数（默认 crc32 IEEE） |
| `WithReplicas(n)` | 虚拟节点数（默认 50，必须 >0） |
| `(*Ring) Add(nodes...)` | 添加物理节点 |
| `(*Ring) Remove(nodes...)` | 移除物理节点 |
| `(*Ring) Get(key) (string, bool)` | 路由 key 到节点（空环返回 false） |
| `(*Ring) GetN(key, n) ([]string, error)` | 取 n 个不同节点（副本放置） |
| `(*Ring) Nodes() []string` | 所有物理节点（排序） |
| `(*Ring) Size() int` | 物理节点数 |
| `(*Ring) Contains(node) bool` | 是否包含节点 |
| `(*Ring) Distribution() map[string]int` | 每个节点的虚拟节点数 |
| `(*Ring) Snapshot() *Ring` | 独立快照 |
| `DefaultHash` | 默认哈希函数（crc32.ChecksumIEEE） |
| `ErrEmptyRing` | 空环错误 |

## 开发

```bash
GOWORK=off go mod tidy
GOWORK=off go build ./...
GOWORK=off go test ./... -short -cover
```
