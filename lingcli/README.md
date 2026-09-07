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
│   ├── storage/storage.go          # 对象存储（ling-base/stores，默认 local）
│   ├── cache/cache.go              # 缓存（ling-base/common/cache，默认 memory）
│   ├── lock/lock.go                # 分布式锁（ling-base/common/lock，默认 memory）
│   └── retry/retry.go              # 重试策略（ling-base/common/retry）
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

## License

MIT
