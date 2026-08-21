# voice

Voice recognition (ASR) and synthesis (TTS) abstraction layer for
ling-base, plus a realtime voice-agent framework. Each sub-package
defines a unified interface with pluggable vendor implementations that
self-register via blank imports.

## Structure

```
voice/
├── recognizer/      # ASR: speech-to-text recognition
├── synthesizer/     # TTS: text-to-speech synthesis
└── realtime/        # Realtime voice agents (WebSocket-based)
```

## Sub-packages

### recognizer

High-level ASR (Automatic Speech Recognition) abstraction. Wraps a
WebSocket-based `Client` with audio buffering, callbacks, and result
conversion. Vendor implementations embed `BaseEngine` and override
only vendor-specific methods.

Supported vendors:

```
qcloud  google  aliyun  funasr  volcengine  volcengine_llm
xfyun_mul  gladia  funasr_realtime  whisper  deepgram
aws  baidu  voiceapi  local
```

```go
type Vendor string  // "qcloud", "google", "aliyun", ...

type Recognizer struct { ... }  // wraps Client with buffering + callbacks

// Factory resolves vendors by name.
func GetSupportedVendors() []Vendor
func IsVendorSupported(vendor Vendor) bool
```

### synthesizer

TTS (Text-to-Speech) synthesis with a unified `Engine` interface and
pluggable `Factory`. Supports streaming PCM delivery via chunk emitters
with configurable frame sizes.

Supported providers:

```
aliyun  aws  azure  baidu  coqui  elevenlabs  fishaudio
fishspeech  google  local  minimax  openai  qcloud  qiniu
volcengine  xunfei
```

```go
type Engine interface { ... }
type Factory interface {
    CreateEngine(config Config) (Engine, error)
    GetSupportedProviders() []Provider
    RegisterCreator(provider Provider, creator Creator)
}
```

### realtime

WebSocket-based realtime voice-agent framework. `BaseAgent` provides
common infrastructure (connection management, tool calling, audio
streaming) that vendor implementations embed. Providers self-register
via blank import and are resolved through `NewAgentFromCredential`.

Supported providers:

```
aliyunomni  gemini  iflytek  minimax  openai
stepfun  tencentsts  volcdialogue
```

```go
// Credential-driven factory: no provider import needed at call site.
func NewAgentFromCredential(cfg map[string]any, opts Options) (Agent, error)

type Agent interface { ... }  // realtime conversation agent
```

## Quick Start

```go
import (
    "github.com/LingByte/ling-base/voice/recognizer"
    _ "github.com/LingByte/ling-base/voice/recognizer/qcloud"  // register vendor
)

// Create an ASR recognizer
rec, err := recognizer.NewRecognizer(recognizer.Config{
    Vendor:   recognizer.VendorQCloud,
    AppID:    "your-app-id",
    SecretID: "your-secret-id",
    SecretKey: "your-secret-key",
})
if err != nil {
    panic(err)
}

// Start recognition session
rec.Start(ctx)
rec.SendAudio(ctx, audioBytes)
result := rec.GetResult()
```
