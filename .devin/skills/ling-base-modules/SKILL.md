# ling-base 模块库

> 本 skill 介绍 `github.com/LingByte/ling-base` 多模块 Go 工具库的全部子模块，
> 供 AI 在生成代码或引入依赖时参考，确保 import 路径和用法正确。

## 仓库结构

- 仓库路径: `github.com/LingByte/ling-base`
- 多模块 Go workspace（`go.work`），每个子目录是独立 Go module
- import 路径格式: `github.com/LingByte/ling-base/<子目录>`
- 所有模块遵循 MIT 许可证

## 核心基础设施

### bootstrap
- **import**: `github.com/LingByte/ling-base/bootstrap`
- 应用启动框架：Banner、生命周期管理、优雅关闭、事件钩子、数据库迁移
- 用法: `app := bootstrap.New("myapp", bootstrap.WithProfile("dev"), bootstrap.WithShutdownTimeout(15*time.Second))`

### logger
- **import**: `github.com/LingByte/ling-base/logger`
- zap + lumberjack 日志：按天/按大小轮转、敏感字段脱敏、时区管理
- 子模块: `logger/gin`（Gin 中间件）
- 用法: `logger.Init(&logger.LogConfig{Level: "info", Filename: "logs/app.log", Daily: true}, "dev")`

### constants
- **import**: `github.com/LingByte/ling-base/constants`
- 全局常量：时区名（TimezoneShanghai）、默认端口等

### version
- **import**: `github.com/LingByte/ling-base/version`
- 版本信息（Version / BuildTime / GitCommit，由 ldflags 注入）

## Web/API 层

### apidocs
- **import**: `github.com/LingByte/ling-base/apidocs`
- Huma + OpenAPI 3.1 API 文档库，一行挂载到 Gin
- 支持 4 种 UI: Scalar（默认）、Swagger UI、Redoc、Stoplight Elements
- 支持 CDN 公网 / 自托管 / 自定义 URL 三种资源模式
- 支持 OAuth2、OpenID Connect、全局安全声明
- 支持 topbar 自定义、环境标识、暗黑模式、自定义 CSS/JS/Logo
- 支持 EnabledFunc 按环境开关文档
- 用法: `api := apidocs.Mount(r, apidocs.Options{Title: "My API", Version: "1.0.0"})`

### middleware
- **import**: `github.com/LingByte/ling-base/middleware`
- Gin 中间件：超时（TimeoutMiddleware）、熔断（CircuitBreakerMiddleware）、组合（CombinedTimeoutCircuitMiddleware）
- 恢复、CORS、请求 ID、日志等

### common/response
- **import**: `github.com/LingByte/ling-base/common/response`
- 统一 API 响应封装
- 子模块: `common/response/gin`（Gin 专用）

### i18n
- **import**: `github.com/LingByte/ling-base/i18n`
- 国际化/多语言翻译
- 子模块: `i18n/gin`（Gin 中间件）、`i18n/mymemory`（MyMemory 翻译 API）

## 限流 / 熔断 / 重试

### common/limiter
- **import**: `github.com/LingByte/ling-base/common/limiter`
- 统一限流接口，多种实现:
  - `common/limiter/tokenbucket` — 令牌桶（QPS + burst）
  - `common/limiter/count` — 全局并发数上限
  - `common/limiter/keycount` — 按 key 并发数上限
  - `common/limiter/keysize` — 按 key 累计字节上限
  - `common/limiter/redis` — Redis 分布式限流
  - `common/limiter/etcd` — Etcd 分布式限流
  - `common/limiter/memcached` — Memcached 限流
  - `common/limiter/mongodb` — MongoDB 限流
  - `common/limiter/null` — 无操作（禁用限流）
- 用法: `l := tokenbucket.New(100, 200); err := l.Acquire(ctx, nil)`

### common/circuitbreaker
- **import**: `github.com/LingByte/ling-base/common/circuitbreaker`
- 线程安全熔断器：Closed / Open / Half-Open 状态机
- 滑动窗口失败率统计，可配置阈值
- 用法: `cb := circuitbreaker.New(circuitbreaker.Config{FailureThreshold: 0.5, MinRequests: 10, RecoveryTimeout: 30*time.Second}); err := cb.Execute(ctx, func(ctx context.Context) error { ... })`

### common/retry
- **import**: `github.com/LingByte/ling-base/common/retry`
- 重试策略：固定间隔、指数退避
- 可与熔断器组合
- 用法: `err := retry.Do(ctx, func(ctx context.Context) error { ... }, retry.WithMaxAttempts(3), retry.WithExponentialBackoff(100*time.Millisecond, 10*time.Second))`

## 锁 / 分布式协调

### common/lock
- **import**: `github.com/LingByte/ling-base/common/lock`
- 统一分布式锁接口，多种实现:
  - `common/lock/redis` — Redis 分布式锁
  - `common/lock/redlock` — Redlock 算法（多 Redis 节点）
  - `common/lock/mysql` — MySQL GET_LOCK
  - `common/lock/postgres` — PostgreSQL advisory lock
  - `common/lock/etcd` — Etcd 分布式锁
  - `common/lock/consul` — Consul 分布式锁
  - `common/lock/zookeeper` — Zookeeper 分布式锁

## 缓存

### cache
- **import**: `github.com/LingByte/ling-base/cache`
- 统一缓存接口，多种实现:
  - `cache/redis` — Redis 缓存
  - `cache/memcache` — Memcached 缓存
  - `cache/bigcache` — BigCache 内存缓存
  - `cache/freecache` — FreeCache 内存缓存
  - `cache/ristretto` — Ristretto 内存缓存
- **cache-aside 模式**: `GetOrLoad` / `GetOrLoadJSON` 封装缓存空值 + 双重检查锁，防缓存穿透/击穿
  - 支持 `WithLocker` 分布式锁、`WithNullMarker` 空值标记、`WithCacheTTL`/`WithNullTTL` 配置
  - `Invalidate` / `InvalidateBatch` 失效缓存

## 鉴权

### common/jwtutil
- **import**: `github.com/LingByte/ling-base/common/jwtutil`
- JWT 鉴权：Access/Refresh token 对、刷新流程、黑名单/吊销
- HTTP 辅助：Bearer token 提取、上下文 claims 检索、中间件包装
- 用法: `auth, _ := jwtutil.New(jwtutil.Config{Secret: []byte("..."), Issuer: "my-app", AccessTTL: 15*time.Minute, RefreshTTL: 7*24*time.Hour}); pair, _ := auth.Login("user-123", jwtutil.Roles("admin"))`

### common/authcontext
- **import**: `github.com/LingByte/ling-base/common/authcontext`
- 统一安全上下文传播层：从请求提取 Identity → 存入 context → 向下游服务透传
- Identity 包含 UserID/UserName/Roles/Permissions/TenantID/Token
- AuthExtractor 接口：HeaderExtractor（网关透传头）、JWTExtractor（Bearer JWT）
- PropagatingTransport：自动向下游 HTTP 调用注入身份头
- 子模块: `common/authcontext/gin`（Gin 中间件：Middleware/RequireAuth/RequireRole/RequirePermission）
- 用法: `r.Use(authctxgin.Middleware(extractor)); id := authcontext.FromContext(c.Request.Context())`

### common/crypto
- **import**: `github.com/LingByte/ling-base/common/crypto`
- 底层加密原语：AES、RSA、HMAC、JWT 签名/验证

### captcha
- **import**: `github.com/LingByte/ling-base/captcha`
- 验证码：图形验证码、短信验证码

## 任务调度

### scheduler
- **import**: `github.com/LingByte/ling-base/scheduler`
- 分布式定时任务（分布式锁 + 任务分发，区别于单机 cron）
- 用法: `s := scheduler.New(locker); s.AddFunc("*/5 * * * *", "task-name", func() { ... })`

### common/cron
- **import**: `github.com/LingByte/ling-base/common/cron`
- 单机 cron 定时任务封装

## 消息队列

### mq
- **import**: `github.com/LingByte/ling-base/mq`
- 统一消息队列接口，多种实现:
  - `mq/kafka` — Kafka
  - `mq/rabbitmq` — RabbitMQ
  - `mq/redisstream` — Redis Stream
  - `mq/rocketmq` — RocketMQ
  - `mq/activemq` — ActiveMQ
  - `mq/factory` — 工厂模式创建

### queue
- **import**: `github.com/LingByte/ling-base/queue`
- 轻量级任务队列
- 子模块: `queue/memory`、`queue/redis`

### eventbus
- **import**: `github.com/LingByte/ling-base/eventbus`
- 进程内事件发布/订阅，支持通配符匹配

## 存储

### stores
- **import**: `github.com/LingByte/ling-base/stores`
- 统一对象存储接口，多种实现:
  - `stores/s3` — AWS S3
  - `stores/oss` — 阿里云 OSS
  - `stores/cos` — 腾讯云 COS
  - `stores/obs` — 华为云 OBS
  - `stores/minio` — MinIO
  - `stores/kodo` — 七牛云 Kodo
  - `stores/ks3` — 金山云 KS3
  - `stores/tos` — 火山引擎 TOS
  - `stores/local` — 本地文件存储

### search
- **import**: `github.com/LingByte/ling-base/search`
- 统一搜索接口
- 子模块: `search/elasticsearch`、`search/bleve`

## 统计 / 监控

### common/stats
- **import**: `github.com/LingByte/ling-base/common/stats`
- 统计采集抽象：Counter、Gauge、Set、HLL（HyperLogLog）、Timer
- 子模块: `common/stats/memory`（内存实现）、`common/stats/redis`（Redis 实现）、`common/stats/file`（文件实现）、`common/stats/gin`（Gin 中间件）
- 支持 TTL 过期回调持久化（MySQL/PostgreSQL/Kafka 等）

### metrics
- **import**: `github.com/LingByte/ling-base/metrics`
- Prometheus 指标采集

### opentelemetry
- **import**: `github.com/LingByte/ling-base/opentelemetry`
- OpenTelemetry 链路追踪 + Metrics + Logs

### tracing
- **import**: `github.com/LingByte/ling-base/tracing`
- 轻量级链路追踪（不依赖 OpenTelemetry）

## 通知

### notification
- **import**: `github.com/LingByte/ling-base/notification`
- 多渠道通知统一接口
- 子模块: `notification/email`、`notification/sms`、`notification/push`、`notification/webhook`、`notification/im`、`notification/inbox`

## 布隆过滤器

### bloom
- **import**: `github.com/LingByte/ling-base/bloom`
- 布隆过滤器接口
- 子模块: `bloom/memory`（内存）、`bloom/redis`（Redis）、`bloom/redisbloom`（RedisBloom 模块）、`bloom/counting`（计数布隆）、`bloom/scalable`（可扩展布隆）

## 数据库迁移

### common/migration
- **import**: `github.com/LingByte/ling-base/common/migration`
- 数据库迁移接口
- 子模块: `common/migration/gormmigrator`（GORM 迁移器）

## 对象池

### common/pool
- **import**: `github.com/LingByte/ling-base/common/pool`
- 通用对象池封装

## 配置

### common/config
- **import**: `github.com/LingByte/ling-base/common/config`
- 多格式、多环境配置加载（YAML + .env，环境覆盖）

## 通用工具

| 模块 | import 路径 | 说明 |
|------|-------------|------|
| common/convert | `github.com/LingByte/ling-base/common/convert` | 类型转换 |
| common/hash | `github.com/LingByte/ling-base/common/hash` | 哈希工具 |
| common/random | `github.com/LingByte/ling-base/common/random` | 随机数/字符串 |
| common/idgen | `github.com/LingByte/ling-base/common/idgen` | ID 生成（Snowflake/UUID） |
| common/idempotency | `github.com/LingByte/ling-base/common/idempotency` | 幂等状态机（MQ/HTTP 防重放，Redis+Memory） |
| common/uaparse | `github.com/LingByte/ling-base/common/uaparse` | User-Agent 解析（OS/Browser/Device/Engine/Bot） |
| common/probe | `github.com/LingByte/ling-base/common/probe` | 合成 HTTP 探测器（主动监控、响应校验、变量提取、序列探测） |
| common/alerting | `github.com/LingByte/ling-base/common/alerting` | 状态变更告警引擎（OK↔Fail 转换、去重、阈值、通知集成） |
| common/wsutil (EnhancedHub) | `github.com/LingByte/ling-base/common/wsutil` | WebSocket 连接管理（元数据、登录回调、最大连接数、广播过滤） |
| common/rbac | `github.com/LingByte/ling-base/common/rbac` | Casbin RBAC 封装（策略CRUD、角色继承、Gin中间件） |
| common/ssh | `github.com/LingByte/ling-base/common/ssh` | WebSSH 桥接（WebSocket→SSH PTY、终端调整、命令执行） |
| common/codegen | `github.com/LingByte/ling-base/common/codegen` | Go AST 代码生成辅助（import管理、函数查找、语句构建） |
| common/registry | `github.com/LingByte/ling-base/common/registry` | 服务注册与发现（Registry接口、内存实现、Consul实现、Watch监听） |
| common/registry/consul | `github.com/LingByte/ling-base/common/registry/consul` | Consul服务注册发现后端（健康检查、自动注销、出站IP检测） |
| common/dbs_json (StringArray/IntArray/JSONMap/JSONRaw) | `github.com/LingByte/ling-base/common` | GORM/SQL JSON自定义类型（字符串数组、整数数组、JSONMap、原始JSON） |
| common/grpc | `github.com/LingByte/ling-base/common/grpc` | gRPC客户端/服务端辅助（Dial、连接池、拦截器、服务发现resolver） |
| common/shutdown | `github.com/LingByte/ling-base/common/shutdown` | 优雅关闭编排器（信号监听、LIFO清理、超时、编程触发） |
| common/batch | `github.com/LingByte/ling-base/common/batch` | 批处理聚合器（按数量/时间/手动触发flush、泛型、错误回调） |
| common/timeutil | `github.com/LingByte/ling-base/common/timeutil` | 时间工具 |
| common/mathutil | `github.com/LingByte/ling-base/common/mathutil` | 数学工具 |
| common/netutil | `github.com/LingByte/ling-base/common/netutil` | 网络工具 |
| common/validate | `github.com/LingByte/ling-base/common/validate` | 数据校验 |
| common/pinyin | `github.com/LingByte/ling-base/common/pinyin` | 拼音转换 |
| common/nltime | `github.com/LingByte/ling-base/common/nltime` | 自然语言时间解析 |
| common/compress | `github.com/LingByte/ling-base/common/compress` | 压缩/解压 |
| common/imageutil | `github.com/LingByte/ling-base/common/imageutil` | 图像处理（resize/crop/watermark） |
| common/videoutil | `github.com/LingByte/ling-base/common/videoutil` | 视频处理 |
| common/audioutil | `github.com/LingByte/ling-base/common/audioutil` | 音频处理 |
| common/barcode | `github.com/LingByte/ling-base/common/barcode` | 条形码生成 |
| common/embedder | `github.com/LingByte/ling-base/common/embedder` | 文本向量化（OpenAI/Aliyun/Volcengine/Jina/Azure/Nvidia/Gemini/Zhipu/Ollama/Dashscope/Local，工厂模式，缓存，多模态） |
| common/retrieve | `github.com/LingByte/ling-base/common/retrieve` | 检索策略（混合检索、重排序、去重、RRF融合、观测） |
| common/chunk | `github.com/LingByte/ling-base/common/chunk` | 文本分块（LLM分块/规则分块/路由分块/标题层级/启发式/递归/Token，通过relay调用LLM） |
| voice/vad | `github.com/LingByte/ling-base/voice/vad` | 语音活动检测（统一Detector接口：Energy RMS+ZCR / WebRTC GMM / Silero神经网络纯Go / Hybrid Energy预筛+Silero确认；Streamer状态机+hangover+PreSpeechBuffer；barge-in；sync.Pool高并发复用；工厂模式） |
| voice/voiceprint | `github.com/LingByte/ling-base/voice/voiceprint` | 声纹识别（火山引擎/讯飞，注册/验证/1:N搜索） |
| voice/voiceclone | `github.com/LingByte/ling-base/voice/voiceclone` | 语音克隆（火山引擎/讯飞，TTS音色合成） |
| common/qrcode | `github.com/LingByte/ling-base/common/qrcode` | 二维码生成 |

## 服务树 / 资源层级

### common/tree
- **import**: `github.com/LingByte/ling-base/common/tree`
- 服务树（g.p.a 三段式层级）核心接口与逻辑：[Tree] 接口（Add/Get/Query/Exists/Delete/ListChildren/SubTree）+ [Store] 持久化抽象
- 物化路径存储，子树查询用 path 前缀扫描，无需递归 CTE
- 子模块（每个独立 go.mod）:
  - `common/tree/memory` — 进程内 Store，零依赖，用于测试/单机 agent
  - `common/tree/gormstore` — 通用 GORM Store（MySQL/PostgreSQL/SQLite 通用）
  - `common/tree/mysql` — MySQL 薄包装（接受 DSN）
  - `common/tree/postgres` — PostgreSQL 薄包装
  - `common/tree/sqlite` — SQLite 薄包装（支持 `:memory:`）
- 用法: `tr, _ := tree.New(memory.New()); leaf, _ := tr.Add(ctx, "inf.monitor.prometheus")`
- 查询模式: `tree.QueryChildren`（直接子节点名）、`tree.QueryLeaves`（子树下所有 g.p.a 全路径）、`tree.QueryExists`（g.p.a 存在性）
- 删除: `tr.Delete(ctx, "inf", true)` force=true 级联删整棵子树，force=false 有子节点时返回 `ErrHasChildren`


## AI/语音模块

| 模块 | import 路径 | 说明 |
|------|-------------|------|
| recognizer | `github.com/LingByte/ling-base/voice/recognizer` | 语音识别（ASR）统一接口 |
| recognizer/whisper | `github.com/LingByte/ling-base/voice/recognizer/whisper` | OpenAI Whisper |
| recognizer/google | `github.com/LingByte/ling-base/voice/recognizer/google` | Google Speech-to-Text |
| recognizer/baidu | `github.com/LingByte/ling-base/voice/recognizer/baidu` | 百度语音识别 |
| recognizer/volcengine | `github.com/LingByte/ling-base/voice/recognizer/volcengine` | 火山引擎语音识别 |
| synthesizer | `github.com/LingByte/ling-base/synthesizer` | 语音合成（TTS）统一接口 |
| synthesizer/openai | `github.com/LingByte/ling-base/synthesizer/openai` | OpenAI TTS |
| synthesizer/azure | `github.com/LingByte/ling-base/synthesizer/azure` | Azure TTS |
| realtime | `github.com/LingByte/ling-base/realtime` | 实时语音对话统一接口 |
| realtime/openai | `github.com/LingByte/ling-base/realtime/openai` | OpenAI Realtime API |

## 内容审核

| 模块 | import 路径 | 说明 |
|------|-------------|------|
| censor | `github.com/LingByte/ling-base/censor` | 内容审核统一接口 |
| censor/aliyun | `github.com/LingByte/ling-base/censor/aliyun` | 阿里云内容安全 |
| censor/qcloud | `github.com/LingByte/ling-base/censor/qcloud` | 腾讯云内容安全 |
| ocr | `github.com/LingByte/ling-base/providers/ocr` | OCR 统一接口 |

## 渗透测试

### pentest
- **import**: `github.com/LingByte/ling-base/pentest`
- Web 渗透测试工具库，27 个独立的主动安全测试工具，全部仅依赖 Go 标准库
- 工具分类：
  - 注入类：SQL 注入、XSS、命令注入、模板注入、XXE、SSRF、反序列化
  - 遍历类：路径遍历、文件包含、文件上传
  - 扫描类：端口扫描、目录扫描、子域名枚举、信息收集
  - 认证类：JWT 安全、密码爆破、会话安全、CSRF、IDOR
  - 分析类：HTTP 安全头、加密强度、API 安全、业务逻辑、速率限制绕过、敏感信息扫描、安全基线、日志分析
- 用法: `tester := pentest.NewSQLInjectionTester(10*time.Second); result, _ := tester.TestURL(ctx, "https://example.com/page?id=1")`

## 项目工具

### lingcli
- **import**: `github.com/LingByte/ling-base/lingcli`
- 项目脚手架 CLI，一键生成完整 Go 项目骨架
- 模板: web-api、grpc-service、cli-tool、library、worker
- 支持选择集成 ling-base 模块（apidocs、limiter、circuitbreaker、jwt 等）
- 用法: `lingcli create myapp --template web-api --module github.com/me/myapp --modules apidocs,limiter,jwt`

## 注意事项

1. **import 路径**: 所有模块的 import 路径以 `github.com/LingByte/ling-base/` 开头，后跟子目录路径
2. **独立 module**: 每个子目录是独立 Go module，有自己的 `go.mod`，可以单独 `go get`
3. **版本标签**: 模块版本通过 git tag 管理，格式为 `<子目录>/v<版本号>`，如 `apidocs/v0.2.1`
4. **Go 版本**: 需要 Go 1.26+
5. **依赖关系**: 模块之间通过显式 import 依赖，不存在循环引用
6. **接口模式**: 大多数模块遵循"一个接口 + 多个实现"的模式（limiter、lock、cache、mq、stores 等）
7. **函数式选项**: 配置统一使用函数式选项模式（`WithXxx`）
