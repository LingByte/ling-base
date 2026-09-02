# common/trie

基于 rune 的前缀树（Trie）实现，纯标准库，零第三方依赖。

## 特性

- **rune 感知**：以 rune 为基本单元，正确处理 Unicode（中文、emoji 等）键。
- **键值存储**：每个键可关联任意 `any` 值，`Insert` 覆盖已有键。
- **前缀匹配**：`HasPrefix` / `PrefixMatch` / `PrefixMatchWithValues`。
- **最长前缀匹配**：`LongestPrefix` 返回输入字符串的最长已注册前缀，适用于路由匹配。
- **自动补全**：`Autocomplete(prefix, limit)` 返回前缀匹配的键列表。
- **有序遍历**：`Walk` 按字典序遍历所有键值对，可提前终止。
- **线程安全版本**：`SafeTrie` 用 `sync.RWMutex` 保护所有操作。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/LingByte/ling-base/common/trie"
)

func main() {
    t := trie.New()
    t.InsertMany(map[string]any{
        "/api":         1,
        "/api/v1":      2,
        "/api/v1/users": 3,
    })

    // 最长前缀匹配（路由）
    prefix, ok := t.LongestPrefix("/api/v1/users/123")
    fmt.Println(prefix, ok) // /api/v1/users true

    // 自动补全
    fmt.Println(t.Autocomplete("/api", 10))

    // 前缀匹配
    fmt.Println(t.PrefixMatch("/api/v1"))

    // 遍历
    t.Walk(func(key string, value any) bool {
        fmt.Printf("%s = %v\n", key, value)
        return true
    })
}
```

## 线程安全

```go
s := trie.NewSafe()
s.Insert("key", 1)
v, ok := s.Search("key") // 读操作加读锁
```

## API

| 方法 | 说明 |
|---|---|
| `New() *Trie` | 创建空 trie |
| `Insert(key, value)` | 插入/覆盖键值对（空键忽略） |
| `InsertMany(items)` | 批量插入 |
| `Search(key) (any, bool)` | 精确查找 |
| `HasPrefix(prefix) bool` | 是否存在以 prefix 开头的键 |
| `PrefixMatch(prefix) []string` | 所有以 prefix 开头的键（排序） |
| `PrefixMatchWithValues(prefix) map[string]any` | 键值对 |
| `Delete(key) bool` | 删除键（自动剪枝） |
| `Size() int` | 键数量 |
| `Clear()` | 清空 |
| `Keys() []string` | 所有键（排序） |
| `LongestPrefix(s) (string, bool)` | s 的最长已注册前缀 |
| `Autocomplete(prefix, limit) []string` | 自动补全（limit<=0 返回全部） |
| `Walk(fn)` | 遍历，fn 返回 false 停止 |

`SafeTrie` 提供上述所有方法的并发安全版本。

## 开发

```bash
GOWORK=off go mod tidy
GOWORK=off go build ./...
GOWORK=off go test ./... -short -cover
```
