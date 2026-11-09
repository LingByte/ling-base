# lingcli

LingByte 项目脚手架 — 一键生成完整 Go 项目骨架。

## 安装

```bash
go install github.com/LingByte/ling-base/lingcli@latest
```

或从源码构建：

```bash
git clone https://github.com/LingByte/ling-base.git
cd ling-base/lingcli
go build -o /usr/local/bin/lingcli .
```

## 用法

```bash
# 交互模式（推荐）
lingcli create myapp

# 在当前目录初始化
lingcli create .

# 非交互模式
lingcli create myapp \
    --template web-api \
    --module github.com/me/myapp \
    --modules apidocs,limiter,circuitbreaker,middleware,jwt \
    --port 8080 \
    --author "Your Name"

# 列出可用模板
lingcli list

# 版本号
lingcli version
```

## 命令

| 命令 | 说明 |
|------|------|
| `lingcli create <项目名>` | 创建新项目 |
| `lingcli create .` | 在当前目录初始化 |
| `lingcli list` | 列出可用项目模板 |
| `lingcli version` | 打印版本号 |
| `lingcli help` | 显示帮助 |

## create 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--template` | 项目模板（web-api/grpc-service/cli-tool/library/worker） | 交互选择 |
| `--module` | Go module 路径 | 交互输入 |
| `--author` | 作者名称 | 空 |
| `--port` | 服务端口（web-api/grpc-service） | 8080 / 50051 |
| `--modules` | 要集成的 ling-base 模块（逗号分隔） | 空 |
| `--docker` | 生成 Docker 部署文件 | true |
| `--no-docker` | 不生成 Docker 部署文件 | - |
| `--git` | 初始化 git 仓库 | true |
| `--no-git` | 不初始化 git 仓库 | - |

## 可用模板

| 模板 | 说明 |
|------|------|
| `web-api` | HTTP REST API 服务（Gin + GORM + Bootstrap） |
| `grpc-service` | gRPC 服务 |
| `cli-tool` | 命令行工具 |
| `library` | 可复用 Go 库 |
| `worker` | 后台任务 / 消费者服务 |

## 可集成的 ling-base 模块

### web-api 内置模块（自动包含，不可选）

| 模块 ID | 说明 | 默认后端 |
|---------|------|----------|
| `response` | 统一 JSON 响应封装 + 错误码 | — |
| `validate` | 结构体标签驱动数据校验 | — |
| `stores` | 对象存储（文件上传/下载/删除 API） | local（本地文件系统） |
| `cache` | 缓存抽象 | memory（进程内） |
| `lock` | 分布式锁 | memory（进程内） |
| `retry` | 重试策略（指数退避 + 抖动） | — |
| `limiter` | 令牌桶限流 | middleware |
| `circuitbreaker` | 熔断器（滑动窗口 + 半开探测） | middleware |
| `fallback` | 降级（熔断/限流时返回降级响应） | pkg/fallback |

### 可选模块（交互式询问）

| 模块 ID | 说明 |
|---------|------|
| `apidocs` | API 文档 UI + OpenAPI 3.1 spec |
| `limiter` | 令牌桶限流 |
| `circuitbreaker` | 熔断 + 超时中间件 |
| `middleware` | HTTP 中间件合集 |
| `jwt` | JWT 鉴权 + 登录/刷新路由 |
| `scheduler` | 分布式定时任务 |
| `eventbus` | 本地事件总线 |
| `stats` | 统计采集 |
| `notification` | 通知调度 |
| `mq` | 消息队列 |
| `search` | 全文搜索 |
| `bloom` | 布隆过滤器 |
| `captcha` | 验证码 |
| `opentelemetry` | OpenTelemetry 链路追踪 |
| `i18n` | 国际化 |

## 生成后的项目结构（web-api）

```
myapp/
├── cmd/server/main.go              # 入口
├── internal/
│   ├── configs/config.go           # 配置定义 + InitDB
│   ├── handlers/
│   │   ├── handler.go              # HTTP 处理器（系统端点 + 用户 CRUD）
│   │   ├── storage.go              # 文件上传/下载/删除（调用 pkg/storage 单例）
│   │   ├── urls.go                 # 路由注册
│   │   └── auth.go                 # JWT 鉴权（选 jwt 时）
│   ├── middlewares/middleware.go   # 中间件
│   ├── models/user.go              # 数据模型
│   └── types/types.go              # 通用 DTO
├── pkg/                            # 可被外部引用的通用封装（包级单例）
│   ├── storage/storage.go          # 对象存储（ling-base/common/stores，默认 local）
│   ├── cache/cache.go              # 缓存（ling-base/common/cache，默认 memory）
│   ├── lock/lock.go                # 分布式锁（ling-base/common/lock，默认 memory）
│   ├── retry/retry.go              # 重试策略（ling-base/common/retry）
│   └── fallback/fallback.go        # 降级（熔断/限流时返回降级响应）
├── configs/                        # YAML 配置（含 storage/cache/lock/retry 段）
├── uploads/                        # 本地文件存储根目录（local 后端）
├── .github/workflows/ci.yml        # CI
├── docker/                         # Dockerfile + docker-compose.yml
├── Makefile                        # 构建命令
└── README.md
```

## 生成后操作

```bash
cd myapp
go mod tidy
make run          # 本地运行
make test         # 测试
make test-race    # 竞态检测
make lint         # golangci-lint
make docker-up    # Docker Compose 启动
```

## 模块路径与发布维护

- 对象存储使用 `github.com/LingByte/ling-base/common/stores` 及其后端子模块。
- 语音使用 `github.com/LingByte/ling-base/providers/voice/...`；上述新 module 路径首发均为 `v0.1.0`，旧目录的 tag 不适用于新路径。
- `lib` 模式按生成的版本表添加依赖。新路径 tag 尚未发布时，不能只靠替换仓库根模块解决；请等待库发布，或使用 `full` 模式。
- `full` 模式的本地复制和发布二进制使用相同目录布局；发布二进制还会携带外部依赖版本，不依赖用户机器上的 ling-base 源码目录。
- `lingcli version` 使用发布构建注入的版本，或 `go install` 的模块版本；普通本地开发构建显示 `dev`。

在仓库根目录维护 CLI：

```bash
make sync-cli-versions   # 根据实际模块目录与正式 tag 生成版本表
make check-cli-versions  # 检查生成文件是否同步，不修改文件
make build-cli          # 嵌入源码及依赖版本，构建 bin/lingcli
```

请先发布库 tag，再同步并提交 `lingcli/module_versions.go`，最后发布 lingcli。完整步骤见 [发布指南](../CONTRIBUTING.md#8-发布版本)。`release-all` 不会替你发布 lingcli。

## License

MIT
