# common

Core types shared across all relay submodules: the `Adaptor` and `TaskAdaptor` interfaces, `RelayInfo` per-request context, and adaptor registries.

## Key Types

- **`RelayInfo`** -- per-request context carried through the relay pipeline. Holds stream flag, relay mode, model names, request DTO, conversion chain, and channel metadata.
- **`ChannelMeta`** -- channel/provider configuration embedded in `RelayInfo` (API key, base URL, channel type, parameter overrides, etc.).
- **`Adaptor`** -- interface every provider channel implements (see `relay/channel/README.md`).
- **`TaskAdaptor`** -- interface for async task providers (video/music generation) with a submit-then-poll lifecycle.
- **`TaskInfo`** / **`TaskError`** -- result and error types for async task polling.
- **`TaskSubmitReq`** -- stub for task submission request data.

## Registries

- **`AdaptorRegistry`** / **`DefaultRegistry`** -- maps API type ints to adaptor factories. Use `Register` and `GetAdaptor`.
- **`TaskAdaptorRegistry`** / **`DefaultTaskRegistry`** -- maps API type ints to task adaptor factories. Use `Register` and `GetTaskAdaptor`.

## Usage

```go
info := common.NewRelayInfo()
info.ChannelMeta = &common.ChannelMeta{
    ChannelType: constant.ChannelTypeOpenAI,
    ApiType:     constant.APITypeOpenAI,
    ApiKey:      "sk-...",
}

adaptor := common.GetAdaptor(info.ApiType)
adaptor.Init(info)
```
