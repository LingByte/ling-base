# ling-base

Go 基础工具库。采用**多 module 按需引入**：业务只 import 用到的驱动，不会把无关 SDK 拉进依赖树。

## 模块结构

```
ling-base/
├─ go.mod / go.work          # 仓库锚点 + 本地多 module 开发（175 个模块）
├─ ARCHITECTURE.md           # 架构文档（relay/voice/common 详解）
│
├─ common/                   # 通用工具库（50+ 独立 module）
│  ├─ logger/                # 结构化日志（zap + lumberjack）
│  ├─ constants/             # 全局常量
│  ├─ config/                # 配置加载（YAML + 环境变量）
│  ├─ crypto/                # 加密/解密工具
│  ├─ hash/                  # 哈希工具
│  ├─ convert/               # 类型转换
│  ├─ validate/              # 数据校验
│  ├─ random/                # 随机数/字符串
│  ├─ idgen/                 # ID 生成（snowflake/uuid/...）
│  ├─ timeutil/              # 时间工具
│  ├─ mathutil/              # 数学工具
│  ├─ stats/                 # 统计工具
│  ├─ pinyin/                # 拼音转换
│  ├─ qrcode/                # 二维码生成
│  ├─ barcode/               # 条形码生成
│  ├─ imageutil/             # 图片处理
│  ├─ videoutil/             # 视频处理
│  ├─ audioutil/             # 音频处理
│  ├─ netutil/               # 网络工具
│  ├─ response/              # 统一 HTTP 响应封装
│  ├─ cron/                  # 定时任务
│  ├─ scheduler/             # 任务调度
│  ├─ migration/             # 数据迁移
│  ├─ nltime/                # 自然语言时间解析
│  ├─ compress/              # 压缩工具（zstd/snappy/lz4）
│  │
│  ├─ totp/                  # TOTP 两步验证 + QR 码
│  │  └─ qr/                 # QR 码生成子模块
│  ├─ passkey/               # Passkey/WebAuthn 无密码认证
│  ├─ password/              # 密码哈希（argon2/bcrypt/scrypt）
│  ├─ jwtutil/               # JWT 工具
│  ├─ captcha/               # 验证码（滑块/点选/拼图/算术/旋转）
│  │
│  ├─ cache/                 # 缓存接口 + 纯标准库实现
│  │  ├─ lru / memory / noop / multilevel
│  │  ├─ bigcache/           # 独立 module，依赖 allegro/bigcache
│  │  ├─ redis/              # 独立 module，依赖 go-redis
│  │  └─ memcache / freecache / ristretto
│  │
│  ├─ lock/                  # 分布式锁接口 + memory 实现
│  │  └─ redis / redlock / etcd / zookeeper / consul / mysql / postgres
│  │
│  ├─ bloom/                 # 布隆过滤器接口 + 估算 + 共享哈希
│  │  ├─ memory / counting / scalable   # 纯标准库
│  │  └─ redis / redisbloom              # 依赖 go-redis
│  │
│  ├─ limiter/               # 限流器接口
│  │  ├─ tokenbucket/        # 令牌桶（mutex/blocking/atomic 三种变体）
│  │  ├─ count / memory / redis
│  │  └─ etcd / zookeeper / consul
│  │
│  ├─ circuitbreaker/        # 熔断器
│  ├─ retry/                 # 重试策略（指数退避/固定间隔 + 熔断器集成）
│  ├─ pool/                  # 连接池
│  ├─ queue/                 # 任务队列（内存/Redis + 容量调度）
│  ├─ eventbus/              # 本地事件总线
│  │
│  ├─ geoip/                 # IP 地理位置查询（国内 pconline + 国际 ip-api）
│  ├─ phone/                 # 手机号归属地查询（内置离线号段库）
│  ├─ i18n/                  # 国际化（翻译 + 格式化 + locale 检测）
│  │  ├─ gin/                # Gin 中间件
│  │  └─ mymemory/           # MyMemory 机器翻译
│  │
│  ├─ opentelemetry/         # OpenTelemetry SDK 封装
│  ├─ tracing/               # 链路追踪工具
│  ├─ metrics/               # Prometheus 指标
│  ├─ system/                # 系统信息（磁盘缓存/pprof/健康检查）
│  └─ parser/                # 文档解析（PDF/DOCX/XLSX/HTML/EPUB/...）
│
├─ relay/                    # AI provider 中继库（生产级，40+ channel 适配器）
│  ├─ client.go              # 统一 Client API（Chat/Stream/Embed/Image/Audio/Responses）
│  ├─ channel/               # Provider 适配器
│  │  ├─ openai/             # OpenAI（基础适配器，其他兼容 provider 继承）
│  │  ├─ claude/             # Anthropic Claude
│  │  ├─ gemini/             # Google Gemini
│  │  ├─ aws/                # AWS Bedrock
│  │  ├─ vertex/             # Google Vertex AI
│  │  ├─ ali/ baidu/ tencent/ xunfei/ zhipu/ moonshot/ deepseek/
│  │  ├─ cohere/ mistral/ perplexity/ xai/ coze/ dify/
│  │  ├─ cloudflare/ replicate/ minimax/ volcengine/ siliconflow/
│  │  └─ ...                 # 共 39 个 channel provider
│  ├─ task/                  # 异步任务 provider（视频/音乐生成）
│  │  └─ ali/ doubao/ gemini/ kling/ sora/ suno/ vidu/ ...
│  ├─ relaykit/              # 协议层 DTO 和格式转换
│  │  ├─ dto/                # 请求/响应数据结构
│  │  ├─ types/              # 类型定义和错误码
│  │  └─ relayconvert/       # 格式转换（OpenAI ↔ Claude ↔ Gemini）
│  ├─ common/                # relay 内部共享类型
│  ├─ relaymode/             # 中继模式常量
│  ├─ meter/                 # 用量计量
│  ├─ realtime/              # WebSocket 实时 API
│  ├─ constant/              # provider endpoint 常量
│  └─ helper/                # 辅助函数
│
├─ voice/                    # 语音处理库
│  ├─ recognizer/            # ASR 语音识别（12 个 provider）
│  │  ├─ aliyun/ baidu/ qcloud/ volcengine/ volcengine_llm
│  │  ├─ aws/ google/ deepgram/ gladia/ funasr/ whisper
│  │  └─ voiceapi/ local
│  ├─ synthesizer/           # TTS 语音合成（16 个 provider）
│  │  ├─ aliyun/ baidu/ qcloud/ volcengine/ xunfei
│  │  ├─ aws/ azure/ google/ openai
│  │  ├─ elevenlabs/ fishaudio/ fishspeech/ coqui/ minimax/ qiniu
│  │  └─ local/              # 本地 PCM 合成
│  └─ realtime/              # 实时多模态语音对话（8 个 provider）
│     ├─ openai/             # OpenAI Realtime API
│     ├─ gemini/             # Google Gemini Live API
│     ├─ aliyunomni/         # Qwen-Omni / DashScope
│     ├─ volcdialogue/       # 豆包实时对话
│     ├─ iflytek/ minimax/ stepfun/ tencentsts
│
├─ stores/                   # 对象存储接口（9 个 provider）
│  ├─ local/                 # 本地文件系统（零云 SDK）
│  ├─ s3/ oss/ cos/ minio/   # 独立 module，各引各的 SDK
│  └─ kodo/ tos/ obs/ ks3/
│
├─ mq/                       # 消息队列接口（5 个 broker）
│  ├─ factory/               # 工厂注册
│  └─ kafka / rabbitmq / activemq / rocketmq / redisstream
│
├─ censor/                   # 内容审核接口
│  ├─ aliyun/                # 阿里云 SDK
│  ├─ qcloud/                # 腾讯云 SDK
│  └─ qiniu/                 # 七牛 SDK
│
├─ ocr/                      # OCR 光学字符识别（6 个 provider）
│  └─ aws/ google/ aliyun/ baidu/ tencent/ qcloud
│
├─ search/                   # 全文搜索接口
│  ├─ bleve/                 # 本地 Bleve 索引
│  └─ elasticsearch/         # Elasticsearch 8.x
│
├─ notification/             # 通知调度（邮件/短信/IM/Webhook）
│  └─ email / sms / im / inbox / webhook
│
├─ middleware/               # HTTP 中间件（限流/熔断/CORS/API 版本）
│
├─ bootstrap/                # 应用启动框架（生命周期管理 + banner）
├─ version/                  # 版本信息
├─ lingcli/                  # 项目脚手架 CLI
├─ example/                  # 示例应用
└─ apidocs/                  # API 文档
```

本地开发使用已提交的 `go.work`；发布后消费者**不需要** `go.work`，直接 `go get` 子模块即可。

## 按需安装示例

```bash
# 只要 LRU 缓存（零第三方依赖）
go get github.com/LingByte/ling-base/common/cache

# 只要 BigCache 驱动（只会拉 bigcache + cache 抽象）
go get github.com/LingByte/ling-base/common/cache/bigcache

# 只要 Redis 分布式锁
go get github.com/LingByte/ling-base/common/lock/redis

# 只要 S3 对象存储
go get github.com/LingByte/ling-base/stores/s3

# 只要 Elasticsearch 搜索后端
go get github.com/LingByte/ling-base/common/search/elasticsearch

# 只要 i18n 核心（零外部依赖）
go get github.com/LingByte/ling-base/common/i18n

# 要 Gin i18n 中间件
go get github.com/LingByte/ling-base/common/i18n/gin

# 语音合成 — 只要阿里云 TTS
go get github.com/LingByte/ling-base/voice/synthesizer/aliyun

# 语音识别 — 只要 Whisper ASR
go get github.com/LingByte/ling-base/voice/recognizer/whisper

# 实时多模态语音对话 — 只要 OpenAI Realtime
go get github.com/LingByte/ling-base/voice/realtime/openai

# AI provider 中继 — 只要 OpenAI 适配器
go get github.com/LingByte/ling-base/relay

# TOTP 两步验证
go get github.com/LingByte/ling-base/common/totp

# Passkey 无密码认证
go get github.com/LingByte/ling-base/common/passkey

# IP 地理位置查询（国内/国际自动切换，零第三方依赖）
go get github.com/LingByte/ling-base/common/geoip

# 手机号归属地查询（内置离线号段库，无需联网）
go get github.com/LingByte/ling-base/common/phone
```

```go
import (
    "github.com/LingByte/ling-base/common/cache"
    "github.com/LingByte/ling-base/common/cache/bigcache" // 不会间接引入 redis/etcd/...
)
```

### realtime 快速上手

```go
import (
    base "github.com/LingByte/ling-base/voice/realtime"
    _ "github.com/LingByte/ling-base/voice/realtime/openai" // 注册 provider
)

agent, err := base.NewAgentFromCredential(
    map[string]any{
        "provider": "openai_realtime",
        "apiKey":   "sk-...",
    },
    base.Options{
        SystemPrompt: "你是一个友好的助手",
        Voice:        "alloy",
        OnEvent: func(ev base.Event) {
            switch ev.Type {
            case base.EventAssistantAudio:
                // 播放 ev.AudioPC (PCM16LE 24kHz)
            case base.EventAssistantText:
                fmt.Print(ev.Text)
            case base.EventUserTranscript:
                log.Printf("用户说: %s", ev.Text)
            case base.EventError:
                log.Printf("错误: %v (fatal=%v)", ev.Err, ev.Fatal)
            }
        },
    },
)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := agent.Start(ctx); err != nil {
    log.Fatal(err)
}
defer agent.Close()

// 推送 PCM16LE 16kHz 音频
for {
    agent.PushAudio(pcmChunk)
}

// 手动结束输入（server VAD 关闭时）
agent.CommitInputAudio()

// 打断当前回复（barge-in）
agent.Cancel()

// 运行时更新系统指令
agent.UpdateInstructions("请用更简短的回答")
```

### geoip 快速上手

通过 IP 查询地理位置，自动根据 IP 段选择国内（pconline）或国际（ip-api）接口：

```go
import "github.com/LingByte/ling-base/common/geoip"

// 自动选择国内/国际接口
country, city, location, err := geoip.GetIPLocation("8.8.8.8")
// → "United States", "Mountain View", "Mountain View, United States", nil

// 强制国内接口（国内 IP 更准确）
country, city, location, _ := geoip.GetIPLocationCN("112.0.0.1")
// → "中国", "南京", "江苏 南京", nil

// 强制国际接口
country, city, location, _ := geoip.GetIPLocationGlobal("8.8.8.8")

// 只取展示字符串
addr := geoip.GetRealAddressByIP("8.8.8.8")
// → "Mountain View, United States"

// 内网 IP 自动识别
addr = geoip.GetRealAddressByIP("192.168.1.1")
// → "内网IP"
```

### phone 快速上手

通过手机号查询归属地，使用内置离线号段库（`phone.dat`，约 4.5MB），无需联网：

```go
import "github.com/LingByte/ling-base/common/phone"

// 查询归属地（返回格式化字符串）
loc := phone.LookupPhoneLocation("19511899044")
// → "四川成都(中国移动)"

// 查询归属地各字段
province, city, cardType := phone.LookupPhoneLocationParts("19208101234")
// → "四川", "成都", "中国广电"

// 底层 API：返回完整记录
rec, err := phone.Find("1952947")
// → PhoneRecord{Province:"广西", City:"玉林", ZipCode:"537000", AreaZone:"0775", CardType:"中国移动"}

// 号码归一化（去除非数字字符）
digits := phone.NormalizePhoneDigits("+86 138-0013-8000")
// → "8613800138000"
```

## lingcli 脚手架

`lingcli` 是 ling-base 自带的项目脚手架工具，类似 `create-vue` / `create-react-app`，
一键生成完整的 Go 项目骨架（目录结构 + Docker + Makefile + CI + 测试 + README），
并支持按需集成 ling-base 模块。

### 安装

```bash
go install github.com/LingByte/ling-base/lingcli@latest
```

或从源码构建：

```bash
git clone https://github.com/LingByte/ling-base.git
cd ling-base/lingcli
go build -o /usr/local/bin/lingcli .
```

### 快速开始

```bash
# 交互模式（推荐，会引导你逐步选择模板、模块路径、端口、ling-base 模块等）
lingcli create myapp

# 在当前目录初始化
lingcli create .

# 非交互模式：一步到位
lingcli create myapp \
    --template web-api \
    --module github.com/me/myapp \
    --modules apidocs,limiter,circuitbreaker,middleware,jwt \
    --port 8080 \
    --author "Your Name"
```

### 可用模板

| 模板 | 说明 |
|------|------|
| `web-api` | HTTP REST API 服务（Gin + GORM + Bootstrap + 可选 JWT/APIDocs/限流/熔断） |
| `grpc-service` | gRPC 服务 |
| `cli-tool` | 命令行工具 |
| `library` | 可复用 Go 库 |
| `worker` | 后台任务 / 消费者服务 |

```bash
lingcli list   # 查看所有模板
```

### 可集成的 ling-base 模块

生成 `web-api` 项目时可按需勾选以下模块，脚手架会自动生成对应的集成代码、
配置项、中间件和路由（未选的模块不会引入任何依赖）：

| 模块 ID | 说明 |
|---------|------|
| `apidocs` | API 文档 UI（Scalar 主题）+ OpenAPI 3.1 spec 自动生成 |
| `limiter` | 令牌桶限流（ling-base/common/limiter/tokenbucket） |
| `circuitbreaker` | 熔断 + 超时中间件（ling-base/middleware） |
| `middleware` | HTTP 中间件合集（RequestID / Logging / Recover / CORS） |
| `jwt` | JWT 鉴权（ling-base/common/jwtutil）+ 登录/刷新路由 |
| `cache` | 缓存抽象（memory / redis / bigcache ...） |
| `lock` | 分布式锁（memory / redis / etcd / zookeeper ...） |
| `retry` | 重试策略（指数退避 / 固定间隔） |
| `scheduler` | 分布式定时任务（分布式锁 + 任务分发） |
| `eventbus` | 本地事件总线 |
| `stats` | 统计采集（PV/UV/QPS/延迟 ...，memory / redis 实现） |
| `notification` | 通知调度（邮件 / 短信 / IM / Webhook） |
| `mq` | 消息队列（Kafka / RabbitMQ / Redis Stream ...） |
| `stores` | 对象存储（S3 / OSS / COS / MinIO / 本地 ...） |
| `search` | 全文搜索（Bleve / Elasticsearch） |
| `bloom` | 布隆过滤器（memory / redis / counting / scalable） |
| `captcha` | 验证码（滑块 / 点选 / 拼图 / 算术 / 旋转） |
| `opentelemetry` | OpenTelemetry 链路追踪 |
| `i18n` | 国际化（翻译 + 格式化 + locale 检测） |

### 生成后的项目结构（web-api 示例）

```
myapp/
├── cmd/server/main.go              # 入口（ldflags 注入版本信息）
├── internal/
│   ├── app/app.go                  # 启动 + 路由注册 + 生命周期
│   ├── auth/auth.go                # JWT 鉴权（选 jwt 时生成）
│   ├── config/config.go            # 配置（YAML + 环境变量覆盖）
│   ├── handler/handler.go          # HTTP 处理器（DTO 校验）
│   ├── middleware/middleware.go    # 中间件（限流/熔断/CORS...）
│   ├── model/user.go               # 数据模型
│   ├── repository/user_repository  # DAO 层
│   └── service/user_service.go     # 业务逻辑层
├── pkg/response/response.go        # 统一响应封装
├── configs/
│   ├── config.yaml                 # 开发环境
│   └── config.prod.yaml            # 生产环境
├── .github/workflows/ci.yml        # GitHub Actions CI
├── Dockerfile                      # 多阶段构建 + HEALTHCHECK
├── docker-compose.yml              # App + MySQL + Redis
├── Makefile                        # build/test/lint/coverage/benchmark
├── go.mod
└── README.md
```

### 生成后操作

```bash
cd myapp

# 安装依赖
go mod tidy

# 本地运行
make run

# 测试
make test           # 单元测试
make test-race      # 竞态检测
make test-cover     # 覆盖率

# 代码质量
make vet            # go vet
make lint           # golangci-lint
make fmt            # 格式化

# Docker 部署
make docker-build
make docker-up      # 启动 App + MySQL + Redis

# 验证服务
curl http://localhost:8080/health
curl http://localhost:8080/live    # K8s liveness
curl http://localhost:8080/ready   # K8s readiness
curl http://localhost:8080/docs    # API 文档 UI（选 apidocs 时）
```

### 环境变量覆盖

生成的项目支持 `APP_` 前缀环境变量覆盖 YAML 配置（优先级最高）：

```bash
APP_SERVER_PORT=9090 APP_DATABASE_DRIVER=mysql APP_DATABASE_DSN="user:pass@tcp(host:3306)/db" \
    ./myapp
```

| 环境变量 | 说明 |
|----------|------|
| `APP_APP_NAME` | 应用名称 |
| `APP_APP_ENVIRONMENT` | 运行环境（dev/test/staging/prod） |
| `APP_SERVER_PORT` | 服务端口 |
| `APP_DATABASE_DRIVER` | 数据库驱动（mysql/postgres/sqlite） |
| `APP_DATABASE_DSN` | 数据库连接串 |
| `APP_REDIS_ADDR` | Redis 地址 |
| `APP_JWT_SECRET` | JWT 密钥 |
| `APP_JWT_ENABLED` | 启用 JWT（true/1） |
| `APP_DOCS_ENABLED` | 启用 API 文档（true/1） |
| `APP_RATELIMIT_ENABLED` | 启用限流（true/1） |

更多细节见 [lingcli/README.md](lingcli/README.md)。

## 文档

- [cache/README.md](cache/README.md)
- [lock/README.md](lock/README.md)
- [bloom/README.md](bloom/README.md)
- [captcha/README.md](captcha/README.md)
- [censor/README.md](censor/README.md)
- [stores/README.md](stores/README.md)
- [i18n/README.md](i18n/README.md)
- [mq/README.md](mq/README.md)
- [queue/README.md](queue/README.md)
- [limiter/README.md](limiter/README.md)
- [retry/README.md](retry/README.md)
- [sandbox/README.md](sandbox/README.md)
- [circuitbreaker/README.md](circuitbreaker/README.md)
- [pool/README.md](pool/README.md)

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
