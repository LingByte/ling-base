# channel

Provider adaptor implementations for the relay system. Each subdirectory implements one AI provider (OpenAI, Claude, Gemini, etc.) behind the `common.Adaptor` interface.

## Adaptor Interface

Every channel implements `common.Adaptor` (defined in `relay/common/adaptor.go`):

```go
type Adaptor interface {
    Init(info *RelayInfo)
    GetRequestURL(info *RelayInfo) (string, error)
    SetupRequestHeader(ctx context.Context, header *http.Header, info *RelayInfo) error
    ConvertOpenAIRequest(ctx, info, request) (any, error)
    ConvertClaudeRequest(ctx, info, request) (any, error)
    ConvertGeminiRequest(ctx, info, request) (any, error)
    ConvertImageRequest(ctx, info, request) (any, error)
    ConvertEmbeddingRequest(ctx, info, request) (any, error)
    ConvertAudioRequest(ctx, info, request) (io.Reader, error)
    ConvertRerankRequest(ctx, relayMode, request) (any, error)
    ConvertOpenAIResponsesRequest(ctx, info, request) (any, error)
    DoRequest(ctx, info, body) (*http.Response, error)
    DoResponse(ctx, resp, info, w) (usage any, err *types.NewAPIError)
    GetModelList() []string
    GetChannelName() string
}
```

The request flow is: `Init` -> `ConvertXxxRequest` -> `GetRequestURL` -> `SetupRequestHeader` -> `DoRequest` -> `DoResponse`.

## Shared Helpers (channel.go)

- `SetupApiRequestHeader` -- sets Content-Type and Accept headers.
- `DoApiRequest` -- builds and sends a JSON POST to the upstream provider.
- `DoFormRequest` -- sends a multipart form request.
- `DoTaskApiRequest` -- sends a request for async task adaptors.
- `GetFullRequestURL` -- concatenates a base URL and path.

## Available Channels

`advancedcustom`, `ai360`, `ali`, `aws`, `baidu`, `baidu_v2`, `claude`,
`cloudflare`, `codex`, `cohere`, `coze`, `deepseek`, `dify`, `gemini`,
`huggingface`, `jimeng`, `jina`, `lingyiwanwu`, `minimax`, `mistral`, `mokaai`,
`moonshot`, `newapi`, `ollama`, `openai`, `openrouter`, `palm`,
`perplexity`, `replicate`, `siliconflow`, `sub2api`, `submodel`,
`tencent`, `vertex`, `volcengine`, `xai`, `xinference`, `xunfei`,
`zhipu`, `zhipu_4v`.

## Adding a New Channel

1. Create a subdirectory named after the provider (e.g. `myprovider/`).
2. Implement `common.Adaptor` in an `adaptor.go` or `<name>.go` file.
3. Register the adaptor with `common.DefaultRegistry.Register(apiType, factory)`.
4. Add the channel type constant in `relay/constant/channel.go` and the API type in `relay/constant/api_type.go`.
5. Provide a `NewProvider` wrapper if the channel should be usable via the high-level `relay.Client`.

```go
// myprovider/adaptor.go
func New(apiKey string) *Adaptor {
    return &Adaptor{APIKey: apiKey, BaseURL: "https://api.example.com"}
}
```
