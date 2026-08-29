# ling-base 架构文档

## 概述

ling-base 是一个 Go 多模块基础库，为 LingByte 系列项目（LingRein、LingEchoX 等）提供可复用的通用能力：AI provider 中继、语音处理、对象存储、消息队列、内容审核、OCR 以及 80+ 通用工具包。

## 仓库结构

```
ling-base/
├── common/          # 80+ 通用工具包（每个独立 go.mod）
├── relay/           # AI provider 中继（60+ 适配器）
├── voice/           # 语音识别 / 合成 / 实时
├── stores/          # 对象存储（9 个 provider）
├── mq/              # 消息队列（5 个 broker）
├── censor/          # 内容审核（3 个 provider）
├── ocr/             # OCR（6 个 provider）
├── bootstrap/       # 应用生命周期框架
├── lingcli/         # 项目脚手架 CLI
├── example/         # 示例应用
├── version/         # 版本信息
├── go.work          # 工作区（175 个模块）
└── ARCHITECTURE.md  # 本文档
```

## 核心设计原则

1. **每个模块独立 go.mod** — 消费者可以只引用需要的模块，不必拉入整个仓库
2. **接口与实现分离** — 每个领域定义统一接口，provider 各自实现
3. **零循环依赖** — 模块间依赖单向流动
4. **workspace 模式开发** — go.work 管理本地开发，发布时各模块独立打 tag

## relay 模块架构

relay 是最核心的模块，实现了统一的 AI provider 中继层。

### 分层

```
relay/
├── client.go            # 高层 Client API（Chat/ChatStream/Embed/Image/Audio/Responses）
├── channel/             # Provider 适配器（40 个 channel + 10 个 task）
│   ├── openai/          # OpenAI（基础适配器，其他 OpenAI 兼容 provider 继承）
│   ├── claude/          # Anthropic Claude
│   ├── gemini/          # Google Gemini
│   ├── aws/             # AWS Bedrock
│   ├── vertex/          # Google Vertex AI
│   ├── ali/             # 阿里通义
│   ├── baidu/           # 百度文心
│   ├── ...              # 更多 provider
│   └── channel.go       # Adaptor 接口定义
├── task/                # 异步任务 provider（视频/音乐生成）
├── relaykit/            # 协议层 DTO 和格式转换
│   ├── dto/             # 请求/响应数据结构
│   ├── types/           # 类型定义和错误码
│   └── relayconvert/    # 格式转换（OpenAI ↔ Claude ↔ Gemini）
├── common/              # relay 内部共享类型（RelayInfo 等）
├── relaymode/           # 中继模式常量
├── meter/               # 用量计量
├── realtime/            # WebSocket 实时 API
└── helper/              # 辅助函数
```

### Adaptor 接口

每个 channel provider 实现 `common.Adaptor` 接口：

```go
type Adaptor interface {
    Init(info *RelayInfo)
    GetRequestURL(info *RelayInfo) (string, error)
    SetupRequestHeader(ctx, header, info) error
    ConvertOpenAIRequest(ctx, info, request) (any, error)      // Chat
    ConvertClaudeRequest(ctx, info, request) (any, error)      // Claude format
    ConvertGeminiRequest(ctx, info, request) (any, error)      // Gemini format
    ConvertEmbeddingRequest(ctx, info, request) (any, error)   // Embeddings
    ConvertRerankRequest(ctx, relayMode, request) (any, error) // Rerank
    ConvertImageRequest(ctx, info, request) (any, error)       // Image gen
    ConvertAudioRequest(ctx, info, request) (io.Reader, error) // TTS/STT
    ConvertOpenAIResponsesRequest(ctx, info, request) (any, error) // Responses API
    DoRequest(ctx, info, body) (*http.Response, error)
    DoResponse(ctx, resp, info, w) (usage, error)
    GetModelList() []string
    GetChannelName() string
}
```

### 支持的 RelayMode

| Mode | 说明 | 支持的 provider 数 |
|------|------|-------------------|
| ChatCompletions | OpenAI chat 格式 | 37+ |
| Embeddings | 向量嵌入 | 21+ |
| Rerank | 重排序 | 9+ |
| ImagesGenerations | 图片生成 | 12+ |
| AudioSpeech | TTS | 6+ |
| Responses | OpenAI Responses API | 8+ |
| Realtime | WebSocket 实时 | 7+ |
| Gemini | Gemini 原生格式 | 3+ |
| Claude | Claude 原生格式 | 11+ |

### Client API

`relay.Client` 是面向使用者的统一入口：

```go
client := relay.New(
    relay.WithProvider(openai.NewProvider("sk-xxx", openai.WithBaseURL("https://..."))),
    relay.WithMeter(meter.NewMemoryMeter()),
)

// 非流式 chat
resp, err := client.Chat(ctx, &relay.ChatRequest{...})

// 流式 chat
result, err := client.ChatStream(ctx, &relay.ChatRequest{Stream: true, ...})

// Embedding
resp, err := client.Embed(ctx, &relay.EmbedRequest{...})

// Responses API
resp, err := client.Responses(ctx, &relay.ResponsesRequest{...})
```

## voice 模块架构

```
voice/
├── recognizer/    # 语音识别（12 个 provider）
├── synthesizer/   # 语音合成（16 个 provider）
└── realtime/      # 实时语音（7 个 provider）
```

每个子模块独立 go.mod，provider 之间互不依赖。

## common 模块

80+ 独立工具包，按功能分类：

| 分类 | 模块 |
|------|------|
| 并发控制 | limiter（10 种）, lock（5 种）, circuitbreaker, retry |
| 缓存 | cache（redis/ristretto）, bloom（5 种）, bitmap（memory/roaring/redis）, queue（3 种） |
| 安全 | totp, passkey, password, jwtutil, captcha |
| IO | compress, imageutil, videoutil, audioutil |
| 网络 | netutil, httpclient, opentelemetry, tracing |
| 工具 | convert, parser, pinyin, qrcode, barcode, random, hash |

## 日志

所有模块使用 `common/logger` 进行结构化日志：

```go
import "github.com/LingByte/ling-base/common/logger"

logger.Warn("message", zap.String("key", "value"))
logger.Error("message", zap.Error(err))
```

relay 模块的 channel 适配器统一使用 logger，不直接调用 fmt.Println。

## 发布

每个模块独立打 tag 发布：

```bash
git tag common/totp/v0.1.0
git tag relay/v0.1.0
git push origin --tags
```

消费者在 go.mod 中引用具体版本：
```
require github.com/LingByte/ling-base/relay v0.1.0
```
