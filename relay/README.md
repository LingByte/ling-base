# relay

Unified LLM relay layer for calling 40+ AI/LLM providers through a single,
framework-agnostic API with integrated usage metering, retry, and circuit
breaking.

## Structure

```
relay/
├── client.go          # Client: unified entry point (Chat, Embed, Image, ...)
├── go.mod             # Separate Go module
├── channel/           # 39 provider adaptors (see below)
├── common/            # Adaptor interface + RelayInfo + shared types
├── common_handler/    # Shared response handlers (rerank, etc.)
├── constant/          # API type constants
├── helper/            # SSE streaming + response ID helpers
├── meter/             # Usage metering (tokens, images, audio seconds)
├── realtime/          # WebSocket realtime API connectors
├── relaykit/          # DTO types, reason maps, relay converters
├── relaymode/         # Relay mode constants (chat, embed, image, ...)
├── service/           # Adaptor-to-relaykit bridge utilities
├── setting/           # Global relay settings stubs
├── task/              # Async task types
└── types/             # Price data and shared types
```

## Key Types

```go
// Client is the unified entry point for all AI API calls.
type Client struct { ... }

// Provider is the high-level interface for AI API providers.
type Provider interface {
    Name() string
    ApiType() int
    Adaptor() common.Adaptor
}

// ChatRequest is the unified chat completion request.
type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature *float64  `json:"temperature,omitempty"`
    MaxTokens   *int      `json:"max_tokens,omitempty"`
    Stream      bool      `json:"stream,omitempty"`
    Tools       []Tool    `json:"tools,omitempty"`
    // ...
}

// ChatResponse is the unified chat completion response.
type ChatResponse struct {
    ID       string       `json:"id"`
    Model    string       `json:"model"`
    Choices  []ChatChoice `json:"choices"`
    Usage    meter.Usage  `json:"usage"`
    Provider string       `json:"provider"`
}

// ChatStreamResult holds the stream channel and final usage.
type ChatStreamResult struct {
    Ch    chan ChatStreamChunk
    Usage meter.Usage
}
```

## Client Methods

| Method              | Description                          |
|---------------------|--------------------------------------|
| `Chat`              | Synchronous chat completion          |
| `ChatStream`        | Streaming chat completion (channel)  |
| `Embed`             | Text embeddings                      |
| `Image`             | Image generation                     |
| `Audio`             | Audio transcription                  |
| `AudioTranslation`  | Audio translation                    |
| `Rerank`            | Document reranking                   |
| `Responses`         | OpenAI Responses API                 |
| `Completions`       | Legacy completions                   |
| `Moderations`       | Content moderation                   |
| `SubmitTask`        | Async task submission (Midjourney)   |
| `FetchTask`         | Async task polling                   |
| `MidjourneySubmit`  | Midjourney image generation          |
| `SubmitSunoTask`    | Suno music generation                |

## Supported Channels (39)

```
advancedcustom  ai360       ali          aws          baidu
baidu_v2        claude      cloudflare   codex        cohere
coze            deepseek    dify         gemini       jimeng
jina            lingyiwanwu minimax     mistral      mokaai
moonshot        newapi      ollama       openai       openrouter
palm            perplexity  replicate    siliconflow  sub2api
submodel        tencent     vertex       volcengine   xai
xinference      xunfei      zhipu        zhipu_4v
```

## Quick Start

```go
import (
    "github.com/LingByte/ling-base/relay"
    "github.com/LingByte/ling-base/relay/meter"
    "github.com/LingByte/ling-base/relay/channel/openai"
)

client := relay.New(
    relay.WithProvider(openai.New("sk-xxx")),
    relay.WithMeter(meter.NewMemoryMeter()),
)

// Synchronous chat
resp, err := client.Chat(ctx, &relay.ChatRequest{
    Model:    "gpt-4o",
    Messages: []relay.Message{{Role: "user", Content: json.RawMessage(`"Hello"`)}},
})

// Streaming chat
result, err := client.ChatStream(ctx, &relay.ChatRequest{
    Model:    "gpt-4o",
    Messages: []relay.Message{{Role: "user", Content: json.RawMessage(`"Tell me a story"`)}},
    Stream:   true,
})
for chunk := range result.Ch {
    fmt.Print(chunk.Delta)
}
```

## Sub-packages

| Package         | Description                                              |
|-----------------|----------------------------------------------------------|
| `channel`       | 39 provider adaptors (OpenAI, Claude, Gemini, etc.)     |
| `common`        | Adaptor interface, RelayInfo, shared request types       |
| `common_handler`| Shared response handlers (rerank, etc.)                  |
| `constant`      | API type and mode constants                              |
| `helper`        | SSE streaming, response ID generation helpers            |
| `meter`         | Usage metering (tokens, images, audio/video seconds)     |
| `realtime`      | WebSocket realtime API connectors (OpenAI, etc.)         |
| `relaykit`      | DTO types, reason maps, relay format converters          |
| `relaymode`     | Relay mode constants (chat, embed, image, audio, ...)    |
| `service`       | Adaptor-to-relaykit bridge and response utilities        |
| `setting`       | Global relay settings stubs (overridable by app)         |
| `task`          | Async task types                                         |
| `types`         | Price data and shared types                              |
