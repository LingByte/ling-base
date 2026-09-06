# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Each module is versioned independently. Module-specific changes are tagged with
the module path, e.g. `[common/stats]`, `[cache/redis]`, `[scheduler]`.

## [Unreleased]

### Added
- CI/CD pipeline (`.github/workflows/ci.yml`) — fmt + vet + build + test + coverage + lint + govulncheck
- Release pipeline (`.github/workflows/release.yml`) — tag-triggered GitHub Release
- `.golangci.yml` — 统一代码质量门禁（errcheck, staticcheck, revive, bodyclose, ...）
- `Dockerfile` — 多阶段构建（base / builder / tester / cli）
- `docker-compose.yml` — 本地开发环境（Redis + PostgreSQL + MySQL + test runner）
- `CONTRIBUTING.md` — 贡献指南
- `Makefile` 新增 `vuln` target — 本地运行 govulncheck

---

## [common/qrcode] v0.2.0 — 2026-09-06

### Added
- 命名花式模版库：`Template` / `TemplateCategory`（`simple` / `classic` / `creative` / `custom`）
- `BuiltinTemplates`、`ListTemplates`、`GetTemplate`、`RegisterTemplate`、`UnregisterTemplate`
- `GenerateFromTemplate` / `SaveFromTemplate`（支持 `TemplateOverride` 覆盖 logo / 颜色等）
- 内置约 36 个参数化样式预设（模块形状、定位点、纯色 / 渐变）
- Playground「模版库」页与 WASM：`wasmQRCodeTemplates` / `wasmQRCodeFromTemplate`

---

## P0 基础设施统一 & pentest 工具扩充 — 2026-09-02

本轮发版涵盖 8 个提交（`d50b3ca`..`dd084f1`），涉及 common 审计修复、common 重复实现收敛、pentest 模块完善 + 45 个新工具、以及 P0 基础设施统一（HTTP 客户端 / 日志 / 哈希 / 随机）。

### [common] 审计修复 — v0.3.2

#### Fixed
- `common/auditlog` — 修复审计日志写入竞态
- `common/backup` — 修复备份文件路径校验
- `common/curlutil` — 修复 cURL 命令解析边界情况
- `common/diff` — 修复 diff 输出格式
- `common/dnsutil` — 修复 DNS 查询超时处理
- `common/geocode` — 修复经纬度计算精度
- `common/logger` — 修复日志轮转竞态
- `common/markdown` — 修复 Markdown 渲染边界
- `common/money` — 修复金额计算精度
- `common/queue/redis` — 修复 Redis 队列重连
- `common/sse` — 修复 SSE 连接断开处理
- `common/turnstile` — 修复 Cloudflare Turnstile 校验
- `common/upload` — 修复上传文件大小校验

### [common] 重复实现收敛 — 各子模块 v0.1.1 / v0.2.1

#### Changed
- `common/archive` — 内部实现收敛，委托到规范模块
- `common/curlutil` — 内部实现收敛
- `common/hash` — 作为哈希/HMAC 规范模块，被 crypto/signature/sms 引用
- `common/idgen` — `RandomShortID` 改用 `random.StringWithCharset`（避免模偏差）→ v0.2.1
- `common/middleware` — 内部实现收敛
- `common/notification/httpclient` — HTTP 客户端统一（首次发版 v0.1.0）
- `common/notification/push` — 内部哈希实现改用 `common/hash`
- `common/notification/sms` — `SHA1Hex`/`SHA256Base64`/`SHA256Hex`/`MD5Hex` 改用 `common/hash` → v0.1.1
- `common/notification/webhook` — 内部实现收敛
- `common/validate` — 内部实现收敛
- `common/webhook` — 内部实现收敛

### [common/crypto] v0.2.1

#### Changed
- `SignSHA256`/`VerifySHA256`/`SignSHA512`/`VerifySHA512` 改用 `common/hash` 等价函数

### [common/signature] v0.1.1

#### Changed
- `SignHMACSHA256`/`VerifyHMACSHA256`/`SignHMACSHA512`/`VerifyHMACSHA512`/`SignMD5` 改用 `common/hash` 等价函数

### [common/netutil] v0.2.0

#### Added
- `NewStandardHTTPClient` — 统一标准 HTTP 客户端工厂，支持 `CheckRedirect` 和 `Jar`，自动配置 TLS（环境变量驱动）

### [pentest] v0.2.0

#### Added
- 20 个网络安全测试工具（端口扫描 / Web 指纹 / SQL 注入 / XSS / 目录爆破 等）
- 5 个高级类安全测试工具（SSRF / SSTI / 反序列化 / 权限提升 / 业务逻辑）
- 10 个信息收集 / 认证类工具（子域名 / WHOIS / DNS 枚举 / 凭证喷洒 / JWT 分析 等）
- 10 个 Fuzz / 竞态 / 云安全工具（参数 Fuzz / 竞态条件 / S3 / IAM / Lambda 探测 等）
- 覆盖率从 44.5% 提升至 96.7%

#### Changed
- `pentest/base.go` — HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`
- 模块完善 + 消除重复实现

### [relay] v0.1.1

#### Changed
- `DefaultHTTPClient()` 替换为 `common/netutil.NewStandardHTTPClient`

### [bootstrap] v0.1.6

#### Changed
- 标准库 `log` 调用替换为 `common/logger`

### [voice/recognizer] 全模块 v0.1.2

#### Changed
- `logrus` 日志替换为 `common/logger`（aws/baidu/deepgram/funasr/gladia/google/local/qcloud/voiceapi/volcengine/volcengine_llm/whisper）
- HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`

### [voice/synthesizer] 全模块 v0.1.2

#### Changed
- `logrus` 日志替换为 `common/logger`（aliyun/aws/azure/baidu/coqui/elevenlabs/fishaudio/fishspeech/google/local/minimax/openai/qcloud/qiniu/volcengine/xunfei）
- HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`

### [providers] v0.1.1

#### Changed
- `providers/censor/qiniu` — HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`
- `providers/ocr/azure` — HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`
- `providers/ocr/baidu` — HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`

### [agentkit] v0.1.0

#### Changed
- HTTP 客户端改用 `common/netutil.NewStandardHTTPClient`（codeexecutor/e2b、knowledge/embedder、knowledge/document/reader/pdf、memory/chromadb、memory/mem0、memory/tencentdb）
- 首次发版

## [common/bitmap] v0.1.0 — 2026-08-29

### Added
- `common/bitmap` — 精确位图统一接口（`Bitmap` / `Batcher` / `Snapshotter`）
- `common/bitmap/memory` — 稠密本地实现（可增长 / `WithFixed`，支持快照）
- `common/bitmap/roaring` — Roaring 压缩实现（稀疏大 offset，交并运算）
- `common/bitmap/redis` — Redis SETBIT/GETBIT/BITCOUNT/BITOP 分布式实现
- 单测、压力测试与基准；文档页 `/docs/common/bitmap`

## [lingcli] v0.1.0 — 2026-08-19

### Added
- `lingcli` 项目脚手架工具 — 一键生成完整 Go 项目骨架
- 交互模式 + 非交互模式（`--template`/`--module`/`--modules`/`--port`/`--author`）
- 5 个项目模板：`web-api` / `grpc-service` / `cli-tool` / `library` / `worker`
- 19 个可选 ling-base 模块集成：apidocs / limiter / circuitbreaker / middleware / jwt / cache / lock / retry / scheduler / eventbus / stats / notification / mq / stores / search / bloom / captcha / opentelemetry / i18n
- `web-api` 模板特性：
  - Gin + GORM + Bootstrap 生命周期管理
  - 分层架构：Handler → Service → Repository
  - 请求 DTO 校验（Gin binding）
  - 健康检查分级：`/health` / `/live` / `/ready`
  - 配置：YAML + 环境变量覆盖（`APP_` 前缀）
  - 条件渲染：未选模块不生成对应代码和依赖
  - 单元测试：handler / config / repository
  - CI：`.github/workflows/ci.yml`（test + lint + build）
  - Docker：多阶段构建 + ldflags 注入 + HEALTHCHECK
  - Makefile：build / test / test-race / test-cover / benchmark / lint / vet / fmt
  - README：完整 API 端点文档 + 环境变量表 + Docker 部署指南
- `.devin/skills/ling-base-modules/SKILL.md` — AI 辅助集成参考文档

## [common/stats] v0.3.0 — 2026-08-19

### Added
- `stats.ExpireFunc` 纯回调接口 — 过期数据落盘到任意数据库，无需实现类
- `stats.ExpiredKey` / `stats.TimerSummary` 结构
- `memory.WithTTL(TTLConfig)` — TTL 过期自动清理，内存只保留最近 N 天
- `memory.WithReservoirTimer(4096)` — 蓄水池采样，Timer 固定 32KB
- `memory.WithBloomSet(1M, 0.001)` — Bloom filter Set，大规模去重固定 1.4MB
- `ginstats.Middleware` — Gin 中间件，自动采集 PV/UV/IP/响应时间/错误率
- `ginstats.NormalizePath` — 动态路径归一化，防止 key 爆炸
- `common/stats/README.md` — 完整使用文档

### Changed
- 持久化从 `ArchiveStore` 接口改为纯回调 `stats.ExpireFunc`
- 删除 `common/stats/sqlite` 和 `common/stats/mysql` 实现包

### Performance
- 内存优化效果：30天×10万用户+100万Timer样本，161.6MB → 5.5MB（节省 97%）
- Gin 中间件开销：+3.1µs/请求

## [common/stats] v0.2.0 — 2026-08-19

### Added
- TTL 过期清理机制
- SQLite / MySQL 持久化实现（后在 v0.3.0 中删除，改为纯回调）

## [common/stats] v0.1.0 — 2026-08-18

### Added
- `stats.Collector` 抽象接口（Counter / Gauge / Set / HLL / Timer）
- `stats.WebsiteMetrics` 便捷层（PV/UV/VV/IP/跳出率/会话时长/CTR/CVR/留存/DAU/MAU/响应时间/QPS/错误率）
- `memory` 后端实现（线程安全，基于 atomic + mutex + hyperloglog）
- `redis` 后端实现（INCR/PFADD/ZSET 等原生 Redis 操作）
- `file` 后端实现（JSON 持久化 + HLL 二进制序列化）
- 双后端一致性测试
- 性能基准测试（并发量 + 计算效率）

## [lingcli] v0.3.0 — 2026-08-17

### Added
- 模板集成 logger 独立模块
- bootstrap banner 集成

## [lingcli] v0.2.0 — 2026-08-17

### Changed
- 使用 `.tmpl` 模板文件 + `go:embed` 替代字符串拼接

## [lingcli] v0.1.0 — 2026-08-16

### Added
- 项目脚手架工具 — 一键生成完整 Go 项目（cmd/internal/pkg/README/Dockerfile）
- 支持 web-api / worker / gRPC 项目模板
- `go install` 全局安装支持

## [logger] v0.1.0 — 2026-08-16

### Added
- 结构化日志库（zap + lumberjack）
- `logger/gin` Gin 中间件子模块
- 独立 Go module

## [bootstrap] v0.1.0 — 2026-08-15

### Added
- 应用启动框架（生命周期管理 + banner）
- 优雅关闭（graceful shutdown）
- 事件系统
- 配置 profile

---

## 历史模块（按发布顺序）

以下模块在 v0.1.0 阶段发布，后续按需迭代：

| 模块 | 首版 | 说明 |
|------|------|------|
| `cache` | v0.1.0 | 缓存抽象 + LRU/memory/noop/multilevel |
| `cache/bigcache` | v0.1.0 | BigCache 后端 |
| `cache/redis` | v0.1.0 | Redis 后端 |
| `cache/freecache` | v0.1.0 | FreeCache 后端 |
| `cache/ristretto` | v0.1.0 | Ristretto 后端 |
| `cache/memcache` | v0.1.0 | Memcache 后端 |
| `lock` | v0.1.0 | 分布式锁抽象 + memory |
| `lock/redis` | v0.1.0 | Redis 锁 |
| `lock/redlock` | v0.1.0 | Redlock 算法 |
| `lock/etcd` | v0.1.0 | Etcd 锁 |
| `lock/zookeeper` | v0.1.0 | Zookeeper 锁 |
| `lock/consul` | v0.1.0 | Consul 锁 |
| `lock/mysql` | v0.1.0 | MySQL 锁 |
| `lock/postgres` | v0.1.0 | PostgreSQL 锁 |
| `bloom` | v0.1.0 | Bloom filter 抽象 + 估算 |
| `bloom/memory` | v0.1.0 | 内存 Bloom filter |
| `bloom/counting` | v0.1.0 | 计数 Bloom filter |
| `bloom/scalable` | v0.1.0 | 可扩展 Bloom filter |
| `bloom/redis` | v0.1.0 | Redis Bloom filter |
| `bloom/redisbloom` | v0.1.0 | RedisBloom 模块 |
| `limiter` | v0.1.0 | 限流器抽象 |
| `limiter/tokenbucket` | v0.1.0 | 令牌桶 |
| `limiter/count` | v0.1.0 | 并发数限流 |
| `limiter/redis` | v0.1.0 | Redis 限流 |
| `limiter/etcd` | v0.1.0 | Etcd 限流 |
| `circuitbreaker` | v0.1.0 | 熔断器 |
| `retry` | v0.1.0 | 重试策略 |
| `pool` | v0.1.0 | 连接池 |
| `captcha` | v0.1.0 | 验证码（滑块/点选/拼图/算术/旋转） |
| `censor` | v0.1.0 | 内容审核抽象 |
| `censor/aliyun` | v0.1.0 | 阿里云内容审核 |
| `censor/qcloud` | v0.1.0 | 腾讯云内容审核 |
| `censor/qiniu` | v0.1.0 | 七牛云内容审核 |
| `stores` | v0.1.0 | 对象存储抽象 |
| `stores/local` | v0.1.0 | 本地文件存储 |
| `stores/s3` | v0.1.0 | AWS S3 |
| `stores/oss` | v0.1.0 | 阿里云 OSS |
| `stores/cos` | v0.1.0 | 腾讯云 COS |
| `stores/minio` | v0.1.0 | MinIO |
| `stores/kodo` | v0.1.0 | 七牛云 Kodo |
| `stores/tos` | v0.1.0 | 火山引擎 TOS |
| `stores/obs` | v0.1.0 | 华为云 OBS |
| `stores/ks3` | v0.1.0 | 金山云 KS3 |
| `search` | v0.1.0 | 全文搜索抽象 |
| `search/bleve` | v0.1.0 | Bleve 本地索引 |
| `search/elasticsearch` | v0.1.0 | Elasticsearch 8.x |
| `mq` | v0.1.0 | 消息队列抽象 |
| `mq/kafka` | v0.1.0 | Kafka |
| `mq/rabbitmq` | v0.1.0 | RabbitMQ |
| `mq/activemq` | v0.1.0 | ActiveMQ |
| `mq/rocketmq` | v0.1.0 | RocketMQ |
| `mq/redisstream` | v0.1.0 | Redis Stream |
| `queue` | v0.1.0 | 任务队列抽象 |
| `queue/memory` | v0.1.0 | 内存队列 |
| `queue/redis` | v0.1.0 | Redis 队列 |
| `eventbus` | v0.1.0 | 本地事件总线 |
| `notification` | v0.1.0 | 通知调度抽象 |
| `notification/email` | v0.1.0 | 邮件通知 |
| `notification/sms` | v0.1.0 | 短信通知 |
| `notification/im` | v0.1.0 | IM 通知 |
| `notification/inbox` | v0.1.0 | 站内信 |
| `notification/webhook` | v0.1.0 | Webhook |
| `scheduler` | v0.1.0 | 分布式定时任务 |
| `i18n` | v0.1.0 | 国际化 |
| `i18n/gin` | v0.1.0 | Gin i18n 中间件 |
| `i18n/mymemory` | v0.1.0 | MyMemory 机器翻译 |
| `common/jwtutil` | v0.1.0 | JWT 鉴权 |
| `common/imageutil` | v0.1.0 | 图像处理（resize/crop/watermark/格式转换） |
| `common/barcode` | v0.1.0 | 条形码生成 |
| `common/qrcode` | v0.2.0 | 二维码生成（含花式模版库） |
| `common/videoutil` | v0.1.0 | 视频处理 |
| `common/audioutil` | v0.1.0 | 音频处理 |
| `common/migration` | v0.1.0 | 数据库迁移支持 |
| `synthesizer` | v0.1.0 | TTS 语音合成抽象 |
| `recognizer` | v0.1.0 | ASR 语音识别抽象 |
| `realtime` | v0.1.0 | 实时多模态语音对话 |
| `system` | v0.1.0 | 系统信息 + pprof + 健康检查 |
| `version` | v0.1.0 | 版本信息 |
