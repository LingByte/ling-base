# task

Async task adaptor implementations for providers that use a submit-then-poll lifecycle (video generation, music generation, image generation). Each subdirectory implements `common.TaskAdaptor` for one provider.

## Subdirectories

| Directory | Provider |
|-----------|----------|
| `ali/` | Alibaba Tongyi Wanxiang (video) |
| `doubao/` | ByteDance Doubao (video) |
| `gemini/` | Google Gemini (image/video) |
| `hailuo/` | Hailuo (MiniMax video) |
| `jimeng/` | Jimeng (ByteDance image) |
| `kling/` | Kuaishou Kling (video) |
| `sora/` | OpenAI Sora (video) |
| `suno/` | Suno (music) |
| `vertex/` | Google Vertex AI (video) |
| `vidu/` | Vidu (video) |
| `taskcommon/` | Shared helpers (`UnmarshalMetadata`, `DefaultString`, `EncodeLocalTaskID`) |
| `taskmodel/` | `Task` struct and `TaskStatus` constants |

## TaskAdaptor Lifecycle

1. `Init` -- initialize with `RelayInfo`
2. `ValidateRequestAndSetAction` -- validate and set action type
3. `BuildRequestURL` / `BuildRequestHeader` / `BuildRequestBody` -- construct the upstream request
4. `DoRequest` / `DoResponse` -- submit task, receive task ID
5. `FetchTask` / `ParseTaskResult` -- poll for completion

## Usage

```go
adaptor := common.GetTaskAdaptor(constant.APITypeGemini)
adaptor.Init(info)
adaptor.ValidateRequestAndSetAction(ctx, info)
// ... build and send request, then poll with FetchTask
```
