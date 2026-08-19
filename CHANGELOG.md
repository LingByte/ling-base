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
| `common/qrcode` | v0.1.0 | 二维码生成 |
| `common/videoutil` | v0.1.0 | 视频处理 |
| `common/audioutil` | v0.1.0 | 音频处理 |
| `common/migration` | v0.1.0 | 数据库迁移支持 |
| `synthesizer` | v0.1.0 | TTS 语音合成抽象 |
| `recognizer` | v0.1.0 | ASR 语音识别抽象 |
| `realtime` | v0.1.0 | 实时多模态语音对话 |
| `system` | v0.1.0 | 系统信息 + pprof + 健康检查 |
| `version` | v0.1.0 | 版本信息 |
