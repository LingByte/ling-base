# ling-base

Go 基础工具库。采用**多 module 按需引入**：业务只 import 用到的驱动，不会把无关 SDK 拉进依赖树。

## 模块结构

```
ling-base/
├─ go.mod / go.work          # 仓库锚点 + 本地多 module 开发
├─ logger/                   # 结构化日志（zap + lumberjack）
├─ constants/                # 全局常量
├─ common/                   # 通用工具（数组/音频/字符串/时间等）
├─ bootstrap/                # 应用启动框架（生命周期管理 + banner）
├─ version/                  # 版本信息
│
├─ cache/                    # module: .../cache  （接口 + 纯标准库实现）
│  ├─ lru / memory / noop / multilevel
│  ├─ bigcache/              # 独立 module，依赖 allegro/bigcache
│  ├─ redis/                 # 独立 module，依赖 go-redis
│  ├─ memcache / freecache / ristretto
│
├─ lock/                     # module: .../lock   （接口 + memory）
│  ├─ redis / redlock / etcd / zookeeper / consul / mysql / postgres
│
├─ bloom/                    # module: .../bloom  （接口 + 估算 + 共享哈希）
│  ├─ memory / counting / scalable   # 独立 module，纯标准库
│  └─ redis / redisbloom      # 独立 module，依赖 go-redis
│
├─ limiter/                  # module: .../limiter （限流器：令牌桶/滑动窗口/并发数）
│  ├─ count / memory / redis
│  └─ etcd / zookeeper / consul
│
├─ circuitbreaker/           # module: .../circuitbreaker （熔断器）
│
├─ retry/                    # module: .../retry   （重试策略：指数退避/固定间隔）
│
├─ pool/                     # module: .../pool    （连接池）
│
├─ captcha/                  # module: .../captcha （验证码：滑块/点选/拼图/算术/旋转）
│
├─ censor/                   # module: .../censor  （内容审核接口）
│  ├─ aliyun/                # 独立 module，依赖阿里云 SDK
│  ├─ qcloud/                # 独立 module，依赖腾讯云 SDK
│  └─ qiniu/                 # 独立 module，依赖七牛 SDK
│
├─ stores/                   # module: .../stores  （对象存储接口）
│  ├─ local/                 # 本地文件系统（零云 SDK）
│  ├─ s3/ oss/ cos/ minio/   # 独立 module，各引各的 SDK
│  └─ kodo/ tos/ obs/ ks3/
│
├─ search/                   # module: .../search  （全文搜索接口）
│  ├─ bleve/                 # 独立 module，本地 Bleve 索引
│  └─ elasticsearch/         # 独立 module，Elasticsearch 8.x 后端
│
├─ mq/                       # module: .../mq     （消息队列接口）
│  ├─ factory/               # 工厂注册
│  ├─ kafka / rabbitmq / activemq / rocketmq / redisstream
│
├─ queue/                    # module: .../queue   （任务队列：内存/Redis + 容量调度）
│  ├─ memory / redis
│
├─ eventbus/                 # module: .../eventbus （本地事件总线）
│
├─ notification/             # module: .../notification （通知调度：邮件/短信/IM/Webhook）
│  ├─ email / sms / im / inbox / webhook
│
├─ middleware/               # HTTP 中间件（API 版本/限流/熔断/CORS 等）
│
├─ parser/                   # module: .../parser  （文档解析：PDF/DOCX/XLSX/HTML/EPUB/...）
│  └─ ocr/                   # OCR 子模块（aws / google）
│
├─ sandbox/                  # module: .../sandbox （代码沙箱：Docker 隔离执行）
│
├─ system/                   # module: .../system  （系统信息：磁盘缓存/pprof/健康检查）
│
├─ synthesizer/              # module: .../synthesizer （TTS 语音合成接口）
│  ├─ aliyun / baidu / qcloud / volcengine / xunfei
│  ├─ aws / azure / google / openai
│  ├─ elevenlabs / fishaudio / fishspeech / coqui / minimax / qiniu
│  └─ local/                 # 本地 PCM 合成
│
├─ recognizer/               # module: .../recognizer （ASR 语音识别接口）
│  ├─ aliyun / baidu / qcloud / volcengine / volcengine_llm
│  ├─ aws / google / deepgram / gladia / funasr / whisper
│  ├─ voiceapi / local
│
├─ realtime/                 # module: .../realtime （实时多模态语音对话：端到端 ASR+LLM+TTS）
│  ├─ openai/                # OpenAI Realtime API (gpt-4o-realtime-preview)
│  ├─ gemini/                # Google Gemini Live API (gemini-2.0-flash-live)
│  ├─ aliyunomni/            # Qwen-Omni / DashScope 实时对话
│  └─ volcdialogue/          # 豆包 / Volcengine 实时对话（二进制协议）
│
└─ i18n/                     # module: .../i18n   （国际化：翻译 + 格式化 + locale 检测）
   ├─ gin/                   # 独立 module，Gin 中间件
   └─ mymemory/              # 独立 module，MyMemory 机器翻译
```

本地开发使用已提交的 `go.work`；发布后消费者**不需要** `go.work`，直接 `go get` 子模块即可。

## 按需安装示例

```bash
# 只要 LRU（零第三方依赖）
go get github.com/LingByte/ling-base/cache

# 只要 BigCache 驱动（只会拉 bigcache + cache 抽象）
go get github.com/LingByte/ling-base/cache/bigcache

# 只要 Redis 分布式锁
go get github.com/LingByte/ling-base/lock/redis

# 只要 S3 对象存储
go get github.com/LingByte/ling-base/stores/s3

# 只要 Elasticsearch 搜索后端
go get github.com/LingByte/ling-base/search/elasticsearch

# 只要 i18n 核心（零外部依赖）
go get github.com/LingByte/ling-base/i18n

# 要 Gin i18n 中间件
go get github.com/LingByte/ling-base/i18n/gin

# 语音合成 — 只要阿里云 TTS
go get github.com/LingByte/ling-base/synthesizer/aliyun

# 语音识别 — 只要 Whisper ASR
go get github.com/LingByte/ling-base/recognizer/whisper

# 实时多模态语音对话 — 只要 OpenAI Realtime
go get github.com/LingByte/ling-base/realtime/openai
```

```go
import (
    "github.com/LingByte/ling-base/cache"
    "github.com/LingByte/ling-base/cache/bigcache" // 不会间接引入 redis/etcd/...
)
```

### realtime 快速上手

```go
import (
    base "github.com/LingByte/ling-base/realtime"
    _ "github.com/LingByte/ling-base/realtime/openai" // 注册 provider
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
