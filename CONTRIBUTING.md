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

模块 tag 前缀必须与当前目录和 `go.mod` 的 module 路径一致。迁移后的 **52 个模块均从 `v0.1.0` 首发**，不继承、移动或覆盖旧路径的 tag：

- `common/stores/v0.1.0`、`common/stores/local/v0.1.0` 等；
- `providers/voice/recognizer/v0.1.0`、`providers/voice/synthesizer/v0.1.0`、`providers/voice/realtime/v0.1.0` 及各 provider；
- `providers/voice/vad/v0.1.0`、`providers/voice/voiceprint/v0.1.0`、`providers/voice/voiceclone/v0.1.0`。

**先发布库，最后发布 lingcli**。`release-all` 只处理库模块，不包含 lingcli。真实创建 tag 前必须提交全部改动；预览不要求工作区干净，也不会写入 tag。

```bash
# 1. 预览新 tag（不会创建或推送任何 tag）
make release-patch PKG=common/stores DRY_RUN=1
make release-patch PKG=providers/voice/recognizer DRY_RUN=1
make release-all DRY_RUN=1

# 2. 提交改动后，为变更的库创建本地 tag；检查清单后推送库 tag
make release-all
make push-tag TAG=common/stores/v0.1.0  # 单个推送示例
# 首次迁移需要发布全部 52 个新路径 tag；确认本地 tag 清单后也可 make push-tags

# 3. 库 tag 已发布后，同步 lingcli 内置版本表并提交生成文件
make sync-cli-versions
make check-cli-versions
git add lingcli/module_versions.go
git commit -m "chore(lingcli): sync published module versions"

# 4. 预览并发布 CLI；当前 lingcli/v0.32.5 的下一 minor 为 lingcli/v0.33.0
make release-minor PKG=lingcli DRY_RUN=1
make release-minor PKG=lingcli
make push-tag TAG=lingcli/v0.33.0
```

`lingcli/module_versions.go` 由 `go run ./scripts/moduleversions` 自动生成：扫描当前实际存在的模块，只匹配各模块自己的正式 tag；忽略旧目录、子模块 tag 和预发布 tag。尚未首发的新路径在开发版目录中暂按 `v0.1.0` 记录；发布 lingcli 时会检查所有库是否已有正式 tag，缺失则拒绝发布。发布机器应先获取最新 tag，例如 `git fetch --tags`，再同步版本表。

发布工作流使用 `**/v*` 匹配多层模块路径。CLI 发布构建会递归嵌入库源码及外部依赖版本，并将 CLI tag 版本注入二进制；`go install .../lingcli@版本` 安装的程序则从 Go 构建信息读取自身版本。

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
