# 贡献指南

感谢你对 ling-base 的兴趣！本文档介绍如何为项目添加新模块或改进现有模块。

## 项目结构

ling-base 是一个**多 module Go 库**，每个子目录是一个独立的 Go module：

```
ling-base/
├─ go.mod / go.work          # 仓库锚点 + 本地多 module 开发
├─ common/                   # 通用工具模块
│  ├─ stats/                 # 统计指标库（独立 module）
│  │  ├─ memory/             # memory 后端（独立 module）
│  │  ├─ redis/              # redis 后端（独立 module）
│  │  └─ gin/                # gin 中间件（独立 module）
│  └─ jwtutil/               # JWT 工具（独立 module）
├─ cache/                    # 缓存抽象（独立 module）
│  └─ redis/                 # redis 后端（独立 module）
├─ lock/                     # 分布式锁（独立 module）
│  └─ redis/                 # redis 后端（独立 module）
└─ ...
```

**核心原则**：业务只 import 用到的驱动，不会把无关 SDK 拉进依赖树。

## 开发环境

### 前置要求

- Go 1.26+
- Git
- Make（可选，用于快捷命令）

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/LingByte/ling-base.git
cd ling-base

# 同步 workspace
go work sync

# 构建所有模块
make build

# 测试所有模块
make test

# 格式化 + vet + 构建 + 测试（CI 等价）
make check
```

### 本地依赖服务

需要 Redis / PostgreSQL / MySQL 进行集成测试时：

```bash
# 启动所有依赖服务
docker compose up -d redis postgres mysql

# 在容器中运行完整测试
docker compose run test

# 清理
docker compose down
```

## 添加新模块

### 1. 创建目录和 go.mod

```bash
mkdir -p mymodule
cd mymodule
go mod init github.com/LingByte/ling-base/mymodule
```

### 2. 注册到 go.work

在仓库根目录的 `go.work` 的 `use` 块中添加：

```
use (
    ...
    ./mymodule
)
```

### 3. 编写代码

每个模块应包含：

- `*.go` — 实现代码
- `*_test.go` — 单元测试
- `go.mod` — 模块定义（依赖其他本仓库模块时用 `replace` 指向本地路径）

如果模块依赖本仓库的其他模块，在 `go.mod` 中使用 replace：

```
require github.com/LingByte/ling-base/common v0.1.0

replace github.com/LingByte/ling-base/common => ../common
```

### 4. 编写测试

```go
package mymodule

import "testing"

func TestMyFunc(t *testing.T) {
    got := MyFunc()
    if got != expected {
        t.Errorf("MyFunc() = %v, want %v", got, expected)
    }
}
```

运行测试：

```bash
make test-pkg PKG=mymodule
```

### 5. 代码质量检查

```bash
# 格式化
make fmt

# 检查格式
make fmt-check

# go vet
make vet

# golangci-lint（需安装）
make lint

# 漏洞扫描
make vuln

# 完整检查
make check
```

### 6. 添加到 CHANGELOG

在 `CHANGELOG.md` 的 `[Unreleased]` 部分添加变更记录。

### 7. 提交代码

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
feat(mymodule): 添加 XXX 功能
fix(mymodule): 修复 XXX bug
perf(mymodule): 优化 XXX 性能
docs(mymodule): 更新 XXX 文档
refactor(mymodule): 重构 XXX
test(mymodule): 添加 XXX 测试
chore: 更新构建配置
```

### 8. 发布版本

```bash
# 单个模块发版
make release-patch PKG=mymodule      # v0.1.0 → v0.1.1
make release-minor PKG=mymodule      # v0.1.0 → v0.2.0
make release-major PKG=mymodule      # v0.1.0 → v1.0.0

# 推送 tag
make push-tags
```

## 代码规范

### Go 代码

- 遵循 `gofmt -s` 格式
- 遵循 `golangci-lint` 规则（见 `.golangci.yml`）
- 导出函数必须有文档注释（以函数名开头）
- 错误必须处理，不能忽略 `_ = err`
- 接口优先于具体类型
- context 作为第一个参数传递

### 模块设计

- **接口在父模块，实现在子模块**（如 `cache` 定义接口，`cache/redis` 实现接口）
- **每个实现是独立 module**，避免拉入无关依赖
- **不跨模块共享第三方依赖**，各引各的
- **测试覆盖核心逻辑**，集成测试可选

### Commit 规范

使用 Conventional Commits：

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `perf` | 性能优化 |
| `refactor` | 重构（不改行为） |
| `docs` | 文档 |
| `test` | 测试 |
| `chore` | 构建/工具/配置 |

## CI/CD

所有 push 和 PR 会自动触发 CI（`.github/workflows/ci.yml`）：

1. **fmt + vet** — 格式和静态检查
2. **build + test** — 矩阵构建和测试所有模块
3. **coverage** — 覆盖率报告
4. **lint** — golangci-lint
5. **vuln** — govulncheck 漏洞扫描

打 tag 时自动触发 Release（`.github/workflows/release.yml`）：

1. 验证对应模块能编译 + 测试通过
2. 创建 GitHub Release

## 联系

- Issue: https://github.com/LingByte/ling-base/issues
- Email: 19511899044@163.com
